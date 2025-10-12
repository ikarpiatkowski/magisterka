package main

import (
	"flag"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
)

func main() {
	db := flag.String("db", "", "database to test")
	metricsPort := flag.Int("metrics-port", 0, "port for Prometheus metrics") // Domyślnie 0
	flag.Parse()
	validFlag(*db)

	cfg := new(Config)
	cfg.loadConfig("config.yaml")

	if *metricsPort != 0 {
		cfg.MetricsPort = *metricsPort // Nadpisz tylko, jeśli flaga została podana
	}

	if cfg.Debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)
	StartPrometheusServer(cfg, reg)

	runTest(cfg, *db, m)
}