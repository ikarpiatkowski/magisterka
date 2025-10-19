package main

import (
	"context"
	"log/slog"
	"time"
)

func runTest(cfg *Config, dbType string, m *metrics, stageCh <-chan int) {
	slog.Info("Starting a test", "db", dbType)

	ctx, done := context.WithCancel(context.Background())
	defer done()

	var pg *postgres
	var mg *mongodb
    var es *elasticsearchStore

	switch dbType {
case "pg":
		pg = NewPostgres(ctx, cfg)
	case "mg":
		mg = NewMongo(ctx, cfg)
	case "es":
		var err error
		es, err = NewElasticsearch(ctx, cfg)
		if err != nil {
			slog.Warn("Elasticsearch initialization failed, skipping ES test", "error", err)
			return
		}
	}

	sleepInterval := time.Duration(cfg.Test.RequestDelayMs) * time.Millisecond

	for currentClients := range stageCh {
		slog.Info("Starting new stage", "db", dbType, "clients", currentClients)
		m.clients.Set(float64(currentClients))

		stageCtx, cancelStage := context.WithCancel(ctx)
		for i := 0; i < currentClients; i++ {
			go func() {
				for {
					select {
					case <-stageCtx.Done():
						return
					default:
						p := product{
							Name:        genString(20),
							Description: genString(100),
							Price:       float32(random(1, 100)),
							Stock:       100,
							Colors:      []string{genString(5), genString(5)},
						}

						if err := p.create(pg, mg, es, dbType, m); err != nil {
							m.createErrorsTotal.Inc()
							slog.Warn("create product failed", "error", err)
							continue
						}

						p.Stock = random(1, 100)
						if err := p.update(pg, mg, es, dbType, m); err != nil {
							m.updateErrorsTotal.Inc()
							slog.Warn("update product failed", "error", err)
						}

						if err := p.search(pg, mg, es, dbType, m, cfg.Debug); err != nil {
							m.searchErrorsTotal.Inc()
							slog.Warn("search product failed", "error", err)
						}

						if err := p.delete(pg, mg, es, dbType, m); err != nil {
							m.deleteErrorsTotal.Inc()
							slog.Warn("delete product failed", "error", err)
						}

						if sleepInterval > 0 {
							time.Sleep(sleepInterval)
						}
					}
				}
			}()
		}
		time.Sleep(time.Duration(cfg.Test.StageIntervalS) * time.Second)
		cancelStage()
	}
	slog.Info("Test finished", "db", dbType)
}