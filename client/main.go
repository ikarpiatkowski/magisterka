package main

import (
	"log/slog"
	"sync"

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

	go func() {
		defer wg.Done()
		reg := prometheus.NewRegistry()
		m := NewMetrics(reg, "pg")
		StartPrometheusServer(cfg.Postgres.MetricsPort, reg)
		runTest(cfg, "pg", m)
	}()

	go func() {
		defer wg.Done()
		reg := prometheus.NewRegistry()
		m := NewMetrics(reg, "mg")
		StartPrometheusServer(cfg.Mongo.MetricsPort, reg)
		runTest(cfg, "mg", m)
	}()

	go func() {
		defer wg.Done()
		reg := prometheus.NewRegistry()
		m := NewMetrics(reg, "es")
		StartPrometheusServer(cfg.Elasticsearch.MetricsPort, reg)
		runTest(cfg, "es", m)
	}()

	wg.Wait()
	slog.Info("All tests finished.")
}