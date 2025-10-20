package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"log/slog"

	es9 "github.com/elastic/go-elasticsearch/v9"
)

func genLocalID() string {
	return fmt.Sprintf("local-%d", time.Now().UnixNano())
}

type elasticsearchStore struct {
	client  *es9.Client
	context context.Context
	Cfg     *ElasticsearchConfig
	m       *metrics
	bulkCh      chan *bulkItem
	bulkSize    int
	bulkTimeout time.Duration
	bulkWG      sync.WaitGroup
	pendingMu sync.Mutex
	pending   map[string]chan struct{}
}

type bulkItem struct {
	op    string
	index string
	id    string
	body  []byte
	done  chan bulkResult
}

type bulkResult struct {
	id  string
	err error
}

func NewElasticsearch(ctx context.Context, c *Config, m *metrics) (*elasticsearchStore, error) {
	addr := c.Elasticsearch.Host
	if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		addr = fmt.Sprintf("http://%s", addr)
	}
	c.Elasticsearch.Host = addr

	cfg := es9.Config{Addresses: []string{addr}}
	client, err := es9.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create es client: %w", err)
	}

	es := &elasticsearchStore{
		client:      client,
		context:     ctx,
		Cfg:         &c.Elasticsearch,
		m:           m,
		bulkCh:      make(chan *bulkItem, 2000),
		bulkSize:    500,
		bulkTimeout: 1 * time.Millisecond,
		pending:     make(map[string]chan struct{}),
	}

	es.bulkWG.Add(1)
	go func() {
		defer es.bulkWG.Done()
		es.runBulkProcessor()
	}()

	var lastErr error
	for i := 0; i < 10; i++ {
		res, err := client.Info(client.Info.WithContext(ctx))
		if res != nil && res.Body != nil {
			res.Body.Close()
		}
		if err == nil && res != nil && res.StatusCode >= 200 && res.StatusCode < 300 {
			return es, nil
		}
		lastErr = err
		slog.Warn("Elasticsearch connection ping failed, retrying...", "error", err)
		time.Sleep(2 * time.Second)
	}

	close(es.bulkCh)
	es.bulkWG.Wait()

	return nil, fmt.Errorf("elasticsearch connection failed after retries: %w", lastErr)
}

func (es *elasticsearchStore) runBulkProcessor() {
	var batch []*bulkItem
	timer := time.NewTimer(es.bulkTimeout)
	defer timer.Stop()
	flush := func(items []*bulkItem) {
		if len(items) == 0 {
			return
		}
		es.flushBulk(items)
	}

	for {
		select {
		case it, ok := <-es.bulkCh:
			if !ok {
				flush(batch)
				return
			}
			batch = append(batch, it)
			if len(batch) >= es.bulkSize {
				flush(batch)
				batch = nil
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(es.bulkTimeout)
			}
		case <-timer.C:
			if len(batch) > 0 {
				flush(batch)
				batch = nil
			}
			timer.Reset(es.bulkTimeout)
		case <-es.context.Done():
			flush(batch)
			return
		}
	}
}

func (es *elasticsearchStore) flushBulk(items []*bulkItem) {
	var buf bytes.Buffer
	for _, it := range items {
		switch it.op {
		case "index":
			meta := map[string]map[string]string{"index": {"_index": it.index, "_id": it.id}}
			if it.id == "" {
				meta = map[string]map[string]string{"index": {"_index": it.index}}
			}
			mline, _ := json.Marshal(meta)
			buf.Write(mline)
			buf.WriteByte('\n')
			buf.Write(it.body)
			buf.WriteByte('\n')
		case "update":
			meta := map[string]map[string]string{"update": {"_index": it.index, "_id": it.id}}
			mline, _ := json.Marshal(meta)
			buf.Write(mline)
			buf.WriteByte('\n')
			buf.Write(it.body)
			buf.WriteByte('\n')
		case "delete":
			meta := map[string]map[string]string{"delete": {"_index": it.index, "_id": it.id}}
			mline, _ := json.Marshal(meta)
			buf.Write(mline)
			buf.WriteByte('\n')
		}
	}

	ctx, cancel := context.WithTimeout(es.context, 15*time.Second)
	defer cancel()
	res, err := es.client.Bulk(bytes.NewReader(buf.Bytes()), es.client.Bulk.WithContext(ctx))
	if err != nil {
		slog.Warn("bulk request failed", "error", err)
		for _, it := range items {
			if it.done != nil {
				it.done <- bulkResult{err: err}
			}
		}
		return
	}
	defer res.Body.Close()
	var resp map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		slog.Warn("failed to decode bulk response", "error", err)
		for _, it := range items {
			if it.done != nil {
				it.done <- bulkResult{err: fmt.Errorf("failed to decode bulk response: %w", err)}
			}
		}
		return
	}

	itms, _ := resp["items"].([]interface{})
	for i, it := range items {
		var resultErr error
		var gotID string
		if i < len(itms) {
			if entry, ok := itms[i].(map[string]interface{}); ok {
				for _, v := range entry {
					if m, ok := v.(map[string]interface{}); ok {
						if sid, ok := m["_id"].(string); ok {
							gotID = sid
						}
						if e, ok := m["error"]; ok {
							resultErr = fmt.Errorf("bulk item error: %v", e)
						}
						if statusF, ok := m["status"].(float64); ok {
							status := int(statusF)
							if status >= 400 {
								if resultErr == nil {
									resultErr = fmt.Errorf("bulk item failed with status %d", status)
								}
							}
						}
					}
				}
			}
		}
		if it.done != nil {
			it.done <- bulkResult{id: gotID, err: resultErr}
		}
		if it.op == "index" && resultErr == nil && gotID != "" {
			es.pendingMu.Lock()
			if ch, ok := es.pending[gotID]; ok {
				close(ch)
				delete(es.pending, gotID)
			}
			es.pendingMu.Unlock()
		}
	}
}

func (es *elasticsearchStore) EnqueueBulk(op, index, id string, body []byte) (string, error) {
	if op == "index" && id == "" {
		id = genLocalID()
	}

	it := &bulkItem{op: op, index: index, id: id, body: body, done: make(chan bulkResult, 1)}
	if op == "index" {
		es.pendingMu.Lock()
		if _, exists := es.pending[id]; !exists {
			es.pending[id] = make(chan struct{})
		}
		es.pendingMu.Unlock()
	}

	select {
		case es.bulkCh <- it:
		default:
			es.flushBulk([]*bulkItem{it})
	}

	select {
		case res := <-it.done:
			if res.id != "" {
				return res.id, res.err
			}
			return id, res.err
		case <-time.After(8 * time.Second):
			return "", fmt.Errorf("bulk enqueue timeout")
		}
}

func (es *elasticsearchStore) WaitForIndex(id string, timeout time.Duration) bool {
	es.pendingMu.Lock()
	ch, ok := es.pending[id]
	es.pendingMu.Unlock()
	if !ok {
		return true
	}
	select {
	case <-ch:
		return true
	case <-time.After(timeout):
		return false
	}
}