package main

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var buckets = []float64{
	0.00001, 0.000025, 0.00005, 0.000075, // < 100 mikrosekund
	0.0001, 0.00025, 0.0005, 0.00075,   // < 1 milisekunda
	0.001, 0.0025, 0.005, 0.0075,     // < 10 milisekund
	0.01, 0.025, 0.05, 0.075,         // < 100 milisekund
	0.1, 0.25, 0.5, 0.75,             // < 1 sekunda
	1.0, 2.5, 5.0,                    // wolne operacje
}

type metrics struct {
	clients           prometheus.Gauge
	createLatency     prometheus.Histogram
	updateLatency     prometheus.Histogram
	searchLatency     prometheus.Histogram
	deleteLatency     prometheus.Histogram
	createErrorsTotal prometheus.Counter
	updateErrorsTotal prometheus.Counter
	searchErrorsTotal prometheus.Counter
	deleteErrorsTotal prometheus.Counter
}

func NewMetrics(reg prometheus.Registerer, dbLabel string) *metrics {
	m := &metrics{
		clients: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace:   "client",
			Name:        "connected_clients",
			Help:        "Number of active clients.",
			ConstLabels: prometheus.Labels{"db": dbLabel},
		}),
		createLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace:   "client",
			Name:        "create_latency_seconds",
			Help:        "Latency of create operations in seconds.",
			Buckets:     buckets,
			ConstLabels: prometheus.Labels{"db": dbLabel},
		}),
		updateLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace:   "client",
			Name:        "update_latency_seconds",
			Help:        "Latency of update operations in seconds.",
			Buckets:     buckets,
			ConstLabels: prometheus.Labels{"db": dbLabel},
		}),
		searchLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace:   "client",
			Name:        "search_latency_seconds",
			Help:        "Latency of search operations in seconds.",
			Buckets:     buckets,
			ConstLabels: prometheus.Labels{"db": dbLabel},
		}),
		deleteLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace:   "client",
			Name:        "delete_latency_seconds",
			Help:        "Latency of delete operations in seconds.",
			Buckets:     buckets,
			ConstLabels: prometheus.Labels{"db": dbLabel},
		}),
		createErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace:   "client",
			Name:        "create_errors_total",
			Help:        "Total number of create errors.",
			ConstLabels: prometheus.Labels{"db": dbLabel},
		}),
		updateErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace:   "client",
			Name:        "update_errors_total",
			Help:        "Total number of update errors.",
			ConstLabels: prometheus.Labels{"db": dbLabel},
		}),
		searchErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace:   "client",
			Name:        "search_errors_total",
			Help:        "Total number of search errors.",
			ConstLabels: prometheus.Labels{"db": dbLabel},
		}),
		deleteErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace:   "client",
			Name:        "delete_errors_total",
			Help:        "Total number of delete errors.",
			ConstLabels: prometheus.Labels{"db": dbLabel},
		}),
	}
	reg.MustRegister(
		m.clients, m.createLatency, m.updateLatency, m.searchLatency, m.deleteLatency,
		m.createErrorsTotal, m.updateErrorsTotal, m.searchErrorsTotal, m.deleteErrorsTotal,
	)
	return m
}


func StartPrometheusServer(port int, reg *prometheus.Registry) {
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

		addr := fmt.Sprintf(":%d", port)
		slog.Info("Starting Prometheus metrics server", "address", addr)

		if err := http.ListenAndServe(addr, mux); err != nil {
			slog.Error("Failed to start Prometheus metrics server", "error", err)
		}
	}()
}