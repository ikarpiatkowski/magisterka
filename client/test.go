package main

import (
	"context"
	"log/slog"
	"sync"

	"time"
)

func runTest(cfg *Config, db string, m *metrics) {
	slog.Info("Starting a test", "db", db)

	ctx, done := context.WithCancel(context.Background())
	defer done()

	var pg *postgres
	var mg *mongodb

	if db == "pg" {
		pg = NewPostgres(ctx, cfg)
	} else {
		mg = NewMongo(ctx, cfg)
	}

	sleepInterval := cfg.Test.RequestDelayMs
	currentClients := cfg.Test.MinClients

	var wg sync.WaitGroup
	for {
		slog.Info("New", "clients", currentClients)
		m.clients.Set(float64(currentClients))

		ticker := time.NewTicker(time.Duration(cfg.Test.StageIntervalS) * time.Second)
		defer ticker.Stop()

		for i := 0; i < currentClients; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				// Create Product 10 products
				var p product
				for range 9 {
					p = product{
						Name:        genString(20),
						Description: genString(100),
						Price:       float32(random(1, 100)),
						Stock:       100,
						Colors:      []string{genString(5), genString(5)},
					}
					warn(p.create(pg, mg, db, m), "create product failed")
				}

				p2 := product{
					Name:        genString(20),
					Description: genString(100),
					Price:       float32(random(1, 100)),
					Stock:       100,
					Colors:      []string{genString(5), genString(5)},
				}
				warn(p2.create(pg, mg, db, m), "create product failed")

				// Update stock quantity of the product
				p2.Stock = random(1, 100)
				warn(p2.update(pg, mg, db, m), "update product failed")

				// Search for products with low price
				warn(p.search(pg, mg, db, m, cfg.Debug), "search product failed")

				// Delete product
				warn(p.delete(pg, mg, db, m), "delete product failed")

				if sleepInterval > 0 {
					sleep(sleepInterval)
				}
			}()
		}

		<-ticker.C
		wg.Wait()

		if currentClients >= cfg.Test.MaxClients {
			break
		}
		currentClients++
	}
}