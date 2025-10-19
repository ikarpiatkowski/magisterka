package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Zastępujemy implementację es8
type elasticsearchStore struct {
	client  *http.Client
	context context.Context
	Cfg     *ElasticsearchConfig
	m       *metrics // Dodajemy metryki
}

// Nowa funkcja NewElasticsearch
func NewElasticsearch(ctx context.Context, c *Config, m *metrics) (*elasticsearchStore, error) {
	addr := c.Elasticsearch.Host
	if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		addr = fmt.Sprintf("http://%s", addr)
	}
	c.Elasticsearch.Host = addr // Zapisujemy pełny adres

	// Używamy http.Client podobnego do tego z main.go
	httpClient := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        c.Test.MaxClients, // Używamy wartości z config
			MaxIdleConnsPerHost: c.Test.MaxClients, // Używamy wartości z config
			IdleConnTimeout:     90 * time.Second,
		},
		Timeout: 10 * time.Second,
	}

	es := &elasticsearchStore{
		client:  httpClient,
		context: ctx,
		Cfg:     &c.Elasticsearch,
		m:       m,
	}

	// Ping do Elasticsearch (zastępując client.Info)
	var lastErr error
	for i := 0; i < 10; i++ {
		req, err := http.NewRequestWithContext(ctx, "GET", es.Cfg.Host, nil)
		if err != nil {
			lastErr = fmt.Errorf("failed to create ping request: %w", err)
			time.Sleep(2 * time.Second)
			continue
		}

		resp, err := es.client.Do(req)
		if err == nil && resp != nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if resp.Body != nil {
				resp.Body.Close()
			}
			return es, nil // Sukces
		}

		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}

		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("elasticsearch ping failed with status: %s", resp.Status)
		}
		slog.Warn("Elasticsearch connection ping failed, retrying...", "error", lastErr)
		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("elasticsearch connection failed after retries: %w", lastErr)
}

// Poniższe funkcje są przeniesione z main.go i zaadaptowane

func (es *elasticsearchStore) performSearch(workerID int, r *rand.Rand) {
	now := time.Now()
	term := "produkt"
	url := fmt.Sprintf("%s/%s/_search?q=name:%s", es.Cfg.Host, es.Cfg.IndexName, term)

	req, _ := http.NewRequestWithContext(es.context, "GET", url, nil)
	resp, err := es.client.Do(req)

	if err != nil {
		slog.Warn("ES Search: Request failed", "client", workerID, "error", err)
		es.m.searchErrorsTotal.Inc()
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		slog.Warn("ES Search: Status error", "client", workerID, "status", resp.Status)
		es.m.searchErrorsTotal.Inc()
	}

	// Rejestrujemy metrykę latency
	es.m.searchLatency.Observe(time.Since(now).Seconds())
}

func (es *elasticsearchStore) performBulkWrite(workerID int, r *rand.Rand) {
	now := time.Now()
	var body strings.Builder

	for i := 0; i < es.Cfg.OpsPerBulk; i++ {
		docID := r.Intn(50000) + 1
		opType := r.Intn(3)

		switch opType {
		case 0: // index
			body.WriteString(fmt.Sprintf(`{"index":{"_index":"%s", "_id":"%d"}}`, es.Cfg.IndexName, docID))
			body.WriteRune('\n')
			body.WriteString(generateRandomProduct(r))
			body.WriteRune('\n')
		case 1: // update
			body.WriteString(fmt.Sprintf(`{"update":{"_index":"%s", "_id":"%d"}}`, es.Cfg.IndexName, docID))
			body.WriteRune('\n')
			body.WriteString(fmt.Sprintf(`{"doc":{"price":%d, "in_stock":%d}}`, r.Intn(2000), r.Intn(50)))
			body.WriteRune('\n')
		case 2: // delete
			body.WriteString(fmt.Sprintf(`{"delete":{"_index":"%s", "_id":"%d"}}`, es.Cfg.IndexName, docID))
			body.WriteRune('\n')
		}
	}

	url := fmt.Sprintf("%s/_bulk", es.Cfg.Host)
	req, _ := http.NewRequestWithContext(es.context, "POST", url, strings.NewReader(body.String()))
	req.Header.Set("Content-Type", "application/x-ndjson")

	resp, err := es.client.Do(req)
	if err != nil {
		slog.Warn("ES Bulk: Request failed", "client", workerID, "error", err)
		es.m.createErrorsTotal.Inc() // Używamy metryki "create" dla "bulk"
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		slog.Warn("ES Bulk: Status error", "client", workerID, "status", resp.Status)
		es.m.createErrorsTotal.Inc() // Używamy metryki "create" dla "bulk"
	}

	// Rejestrujemy metrykę latency
	es.m.createLatency.Observe(time.Since(now).Seconds()) // Używamy metryki "create" dla "bulk"
}

// Przeniesione z main.go
func generateRandomProduct(r *rand.Rand) string {
	name := "Testowy Produkt " + strconv.Itoa(r.Intn(10000))
	price := r.Intn(1000)
	inStock := r.Intn(100)
	return fmt.Sprintf(`{"name":"%s", "price":%d, "in_stock":%d, "created_at":"%s"}`,
		name, price, inStock, time.Now().Format(time.RFC3339))
}