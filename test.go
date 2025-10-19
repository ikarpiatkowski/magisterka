package main

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

func runTest(cfg *Config, dbType string, m *metrics) {
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
		es, err = NewElasticsearch(ctx, cfg, m)
		if err != nil {
			slog.Warn("Elasticsearch initialization failed, skipping ES test", "error", err)
			return
		}
	}

	sleepInterval := time.Duration(cfg.Test.RequestDelayMs) * time.Millisecond

	for currentClients := cfg.Test.MinClients; currentClients <= cfg.Test.MaxClients; currentClients++ {
		slog.Info("Starting new stage", "db", dbType, "clients", currentClients)
		m.clients.Set(float64(currentClients))

		stageCtx, cancelStage := context.WithCancel(ctx)

		var stageWG sync.WaitGroup

		for i := 0; i < currentClients; i++ {
			stageWG.Add(1)
			go func(workerID int) {
				defer stageWG.Done()
				for {
					select {
					case <-stageCtx.Done():
						return
					default:
					}

					p := product{
						Name:        genString(20),
						Description: genString(100),
						Price:       float32(random(1, 100)),
						Stock:       100,
						Colors:      []string{genString(5), genString(5)},
					}

					if err := p.create(pg, mg, es, dbType, m); err != nil {
						m.createErrorsTotal.Inc()
						slog.Warn("create product failed", "db", dbType, "error", err)
						if sleepInterval > 0 {
							select {
							case <-time.After(sleepInterval):
							case <-stageCtx.Done():
							}
						}
						continue
					}

					p.Stock = random(1, 100)
					if err := p.update(pg, mg, es, dbType, m); err != nil {
						m.updateErrorsTotal.Inc()
						slog.Warn("update product failed", "db", dbType, "error", err)
					}

					if err := p.search(pg, mg, es, dbType, m, cfg.Debug); err != nil {
						m.searchErrorsTotal.Inc()
						slog.Warn("search product failed", "db", dbType, "error", err)
					}

					if err := p.delete(pg, mg, es, dbType, m); err != nil {
						m.deleteErrorsTotal.Inc()
						slog.Warn("delete product failed", "db", dbType, "error", err)
					}

					if sleepInterval > 0 {
						select {
						case <-time.After(sleepInterval):
						case <-stageCtx.Done():
						}
					}
				}
			}(i)
		}

		time.Sleep(time.Duration(cfg.Test.StageIntervalS) * time.Second)
		cancelStage()
		stageWG.Wait()
	}
	slog.Info("Test finished", "db", dbType)
}