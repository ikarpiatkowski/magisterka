package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"log/slog"

	es8 "github.com/elastic/go-elasticsearch/v8"
)

type elasticsearchStore struct {
	client  *es8.Client
	context context.Context
	Cfg     *ElasticsearchConfig
	m       *metrics
}

// NewElasticsearch creates the ES client and waits for the cluster to become available.
func NewElasticsearch(ctx context.Context, c *Config, m *metrics) (*elasticsearchStore, error) {
	addr := c.Elasticsearch.Host
	if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		addr = fmt.Sprintf("http://%s", addr)
	}
	c.Elasticsearch.Host = addr

	cfg := es8.Config{Addresses: []string{addr}}
	client, err := es8.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create es client: %w", err)
	}

	// use a background context for stored client operations so test cancellation
	// (runTest ctx) doesn't immediately cancel in-flight ES requests
	es := &elasticsearchStore{client: client, context: context.Background(), Cfg: &c.Elasticsearch, m: m}

	var lastErr error
	for i := 0; i < 10; i++ {
		res, err := client.Info(client.Info.WithContext(ctx))
		if err == nil && res != nil && res.StatusCode >= 200 && res.StatusCode < 300 {
			if res.Body != nil {
				res.Body.Close()
			}
			return es, nil
		}
		if res != nil && res.Body != nil {
			res.Body.Close()
		}
		lastErr = err
		slog.Warn("Elasticsearch connection ping failed, retrying...", "error", err)
		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("elasticsearch connection failed after retries: %w", lastErr)
}