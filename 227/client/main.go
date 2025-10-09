package main

import (
	"flag"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
)

func main() {
   db := flag.String("db", "", "database to test")
   metricsPort := flag.Int("metrics-port", 8081, "port for Prometheus metrics")
   flag.Parse()
   validFlag(*db)

   cfg := new(Config)
   cfg.loadConfig("config.yaml")
   cfg.MetricsPort = *metricsPort // override config with flag

   if cfg.Debug {
	   slog.SetLogLoggerLevel(slog.LevelDebug)
   }

   reg := prometheus.NewRegistry()
   m := NewMetrics(reg)
   StartPrometheusServer(cfg, reg)

   runTest(cfg, *db, m)
}
