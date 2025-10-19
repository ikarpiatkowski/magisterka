package main

import (
	"log/slog"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func main() {
	cfg := new(Config)
	cfg.loadConfig("config.yaml")

	if cfg.Debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	var wg sync.WaitGroup
	wg.Add(3)

	stageCh := make(chan int)

	go func() {
		defer wg.Done()
		reg := prometheus.NewRegistry()
		m := NewMetrics(reg, "pg")
		StartPrometheusServer(cfg.Postgres.MetricsPort, reg)
		runTest(cfg, "pg", m, stageCh)
	}()

	go func() {
		defer wg.Done()
		reg := prometheus.NewRegistry()
		m := NewMetrics(reg, "mg")
		StartPrometheusServer(cfg.Mongo.MetricsPort, reg)
		runTest(cfg, "mg", m, stageCh)
	}()

	go func() {
		defer wg.Done()
		reg := prometheus.NewRegistry()
		m := NewMetrics(reg, "es")
		StartPrometheusServer(cfg.Elasticsearch.MetricsPort, reg)
		runTest(cfg, "es", m, stageCh)
	}()

	go func() {
		for currentClients := cfg.Test.MinClients; currentClients <= cfg.Test.MaxClients; currentClients++ {
			stageCh <- currentClients
			time.Sleep(time.Duration(cfg.Test.StageIntervalS) * time.Second)
		}
		close(stageCh)
	}()

	wg.Wait()
	slog.Info("All tests finished.")
}