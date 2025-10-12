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

	// Używamy WaitGroup, aby poczekać na zakończenie obu testów
	var wg sync.WaitGroup
	wg.Add(2)

	// Uruchomienie testu dla PostgreSQL w osobnej gorutynie
	go func() {
		defer wg.Done()
		// Każdy test potrzebuje własnego rejestru metryk
		reg := prometheus.NewRegistry()
		m := NewMetrics(reg, "pg") // Dodajemy etykietę "pg"
		StartPrometheusServer(cfg.Postgres.MetricsPort, reg)
		runTest(cfg, "pg", m)
	}()

	// Uruchomienie testu dla MongoDB w osobnej gorutynie
	go func() {
		defer wg.Done()
		// Każdy test potrzebuje własnego rejestru metryk
		reg := prometheus.NewRegistry()
		m := NewMetrics(reg, "mg") // Dodajemy etykietę "mg"
		StartPrometheusServer(cfg.Mongo.MetricsPort, reg)
		runTest(cfg, "mg", m)
	}()

	// Czekaj na zakończenie obu gorutyn (testów)
	wg.Wait()
	slog.Info("All tests finished.")
}