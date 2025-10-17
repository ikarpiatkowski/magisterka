package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	es8 "github.com/elastic/go-elasticsearch/v8"
)

type elasticsearchStore struct {
	client  *es8.Client
	context context.Context
}

// NewElasticsearch attempts to create and verify an Elasticsearch client.
// It retries a few times before returning an error so the calling code can
// decide whether to skip the ES test or fail the whole process.
func NewElasticsearch(ctx context.Context, c *Config) (*elasticsearchStore, error) {
	es := &elasticsearchStore{context: ctx}

	addr := c.Elasticsearch.Host
	// Allow user to specify host as either "host:port" or full URL
	if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		addr = fmt.Sprintf("http://%s", addr)
	}

	cfg := es8.Config{
		Addresses: []string{addr},
	}

	client, err := es8.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create Elasticsearch client: %w", err)
	}

	// Try to contact the cluster a few times before giving up. This helps when
	// Docker Compose starts services concurrently and ES isn't ready yet.
	var lastErr error
	for i := 0; i < 10; i++ {
		res, err := client.Info(client.Info.WithContext(ctx))
		if err == nil && res != nil && res.StatusCode >= 200 && res.StatusCode < 300 {
			if res.Body != nil {
				res.Body.Close()
			}
			es.client = client
			return es, nil
		}
		if res != nil && res.Body != nil {
			res.Body.Close()
		}
		lastErr = err
		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("elasticsearch connection failed after retries: %w", lastErr)
}