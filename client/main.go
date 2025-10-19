package main

import (
	"log/slog"
	"sync"

	// "time" // Już niepotrzebne

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

	// KANAŁ USUNIĘTY - JUŻ NIE JEST POTRZEBNY
	// stageCh := make(chan int)

	go func() {
		defer wg.Done()
		reg := prometheus.NewRegistry()
		m := NewMetrics(reg, "pg")
		StartPrometheusServer(cfg.Postgres.MetricsPort, reg)
		runTest(cfg, "pg", m) // Przekazujemy już tylko 3 argumenty
	}()

	go func() {
		defer wg.Done()
		reg := prometheus.NewRegistry()
		m := NewMetrics(reg, "mg")
		StartPrometheusServer(cfg.Mongo.MetricsPort, reg)
		runTest(cfg, "mg", m) // Przekazujemy już tylko 3 argumenty
	}()

	go func() {
		defer wg.Done()
		reg := prometheus.NewRegistry()
		m := NewMetrics(reg, "es")
		StartPrometheusServer(cfg.Elasticsearch.MetricsPort, reg)
		runTest(cfg, "es", m) // Przekazujemy już tylko 3 argumenty
	}()

	// PĘTLA RAMP-UP USUNIĘTA - ZOSTAŁA PRZENIESIONA DO runTest
	// go func() { ... }()

	wg.Wait()
	slog.Info("All tests finished.")
}