package main

import (
	"context"
	"fmt"

	es8 "github.com/elastic/go-elasticsearch/v8"
)

type elasticsearchStore struct {
    client  *es8.Client
    context context.Context
}

func NewElasticsearch(ctx context.Context, c *Config) *elasticsearchStore {
	es := &elasticsearchStore{
		context: ctx,
	}
	es.esConnect(c)
	return es
}

func (es *elasticsearchStore) esConnect(c *Config) {
	cfg := es8.Config{
		Addresses: []string{
			fmt.Sprintf("http://%s", c.Elasticsearch.Host),
		},
	}
	client, err := es8.NewClient(cfg)
	fail(err, "Unable to create Elasticsearch client")

	// Sprawdzenie połączenia
	_, err = client.Info()
	fail(err, "Elasticsearch connection failed")

	es.client = client
}