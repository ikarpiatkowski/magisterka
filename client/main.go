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

	// Używamy WaitGroup, aby poczekać na zakończenie trzech testów
	var wg sync.WaitGroup
	wg.Add(3)

	// Wspólny kanał etapów (stageCh) — broadcastuje liczbę klientów dla każdego etapu
	stageCh := make(chan int)

	// Uruchomienie testu dla PostgreSQL w osobnej gorutynie
	go func() {
		defer wg.Done()
		reg := prometheus.NewRegistry()
		m := NewMetrics(reg, "pg")
		StartPrometheusServer(cfg.Postgres.MetricsPort, reg)
		runTest(cfg, "pg", m, stageCh)
	}()

	// Uruchomienie testu dla MongoDB w osobnej gorutynie
	go func() {
		defer wg.Done()
		reg := prometheus.NewRegistry()
		m := NewMetrics(reg, "mg")
		StartPrometheusServer(cfg.Mongo.MetricsPort, reg)
		runTest(cfg, "mg", m, stageCh)
	}()

	// Uruchomienie testu dla Elasticsearch w osobnej gorutynie
	go func() {
		defer wg.Done()
		reg := prometheus.NewRegistry()
		m := NewMetrics(reg, "es")
		StartPrometheusServer(cfg.Elasticsearch.MetricsPort, reg)
		runTest(cfg, "es", m, stageCh)
	}()

	// Kontroler etapów — wysyła kolejne liczby klientów do stageCh
	go func() {
		// Gdy wszystkie etapy się wykonają, zamkniemy stageCh aby zakończyć runTest
		for currentClients := cfg.Test.MinClients; currentClients <= cfg.Test.MaxClients; currentClients++ {
			stageCh <- currentClients
			// Poczekaj na czas trwania etapu zanim prześlesz kolejny
			time.Sleep(time.Duration(cfg.Test.StageIntervalS) * time.Second)
		}
		close(stageCh)
	}()

	// Czekaj na zakończenie wszystkich gorutyn (testów)
	wg.Wait()
	slog.Info("All tests finished.")
}