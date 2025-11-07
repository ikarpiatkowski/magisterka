package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func observeLatency(m *metrics, op string, start time.Time) {
	if m == nil {
		 return
	}
	elapsed := time.Since(start).Seconds()
	m.crudLatency.WithLabelValues(op).Observe(elapsed)
}

var buckets = []float64{
	0.00001, 0.000014, 0.000021, 0.000024, 0.00003, 0.000036, 0.000041, 0.000044,
    0.00005, 0.000056, 0.000061, 0.000064, 0.00007, 0.000076, 0.000081, 0.000084,
    0.00009, 0.000096, 0.0001, 0.000102, 0.000103, 0.000104, 0.000105, 0.000106,
    0.000107, 0.000108, 0.000109, 0.00011, 0.000111, 0.000112, 0.000113, 0.000114,
    0.000115, 0.000116, 0.000117, 0.000118, 0.000119, 0.00012, 0.000121, 0.000122,
    0.000123, 0.000124, 0.000125, 0.000126, 0.000127, 0.000128, 0.000129, 0.00013,
    0.000131, 0.000132, 0.000133, 0.000134, 0.000135, 0.000136, 0.000137, 0.000138,
    0.000139, 0.00014, 0.000141, 0.000142, 0.000143, 0.000144, 0.000145, 0.000146,
    0.000147, 0.000148, 0.000149, 0.00015, 0.000151, 0.000152, 0.000153, 0.000154,
    0.000155, 0.000156, 0.000157, 0.000158, 0.000159, 0.00016, 0.000161, 0.000162,
    0.000163, 0.000164, 0.000165, 0.000166, 0.000167, 0.000168, 0.000169, 0.00017,
    0.000171, 0.000172, 0.000173, 0.000174, 0.000175, 0.000176, 0.000177, 0.000178,
    0.000179, 0.00018, 0.000181, 0.000182, 0.000183, 0.000184, 0.000185, 0.000186,
    0.000187, 0.000188, 0.000189, 0.00019, 0.000191, 0.000192, 0.000193, 0.000194,
    0.000195, 0.000196, 0.000197, 0.000198, 0.000199, 0.0002, 0.00022, 0.00024,
    0.00026, 0.00028, 0.0003, 0.00032, 0.00034, 0.00036, 0.00038, 0.0004,
    0.00042, 0.00044, 0.00046, 0.00048, 0.0005, 0.00052, 0.00054, 0.00056,
    0.00058, 0.0006, 0.00062, 0.00064, 0.00066, 0.00068, 0.0007, 0.00072,
    0.00074, 0.00076, 0.00078, 0.0008, 0.00082, 0.00084, 0.00086, 0.00088,
    0.0009, 0.00092, 0.00094, 0.00096, 0.00098, 0.001, 0.0015, 0.002,
    0.0025, 0.003, 0.0035, 0.004, 0.0045, 0.005, 0.0055, 0.006, 0.0065,
    0.007, 0.0075, 0.008, 0.0085, 0.009, 0.0095, 0.01, 0.014, 0.021,
    0.024, 0.03, 0.036, 0.041, 0.044, 0.05, 0.056, 0.061, 0.064, 0.07,
    0.076, 0.081, 0.084, 0.09, 0.096, 0.1, 0.14, 0.21, 0.24, 0.3,
    0.36, 0.41, 0.44, 0.5, 0.56, 0.61, 0.64, 0.7, 0.76, 0.81,
    0.84, 0.9, 0.96, 1.0, 1.4, 2.1, 2.4, 3.1, 3.4, 4.1, 4.4, 5.0,
}

type metrics struct {
	clients         *prometheus.GaugeVec
	crudLatency     *prometheus.HistogramVec
}

func NewMetrics(reg prometheus.Registerer, dbLabel string) *metrics {
       m := &metrics{
	       clients: prometheus.NewGaugeVec(prometheus.GaugeOpts{
		       Namespace: "client",
		       Name:      "connected_clients",
		       Help:      "Number of active clients.",
	       }, []string{"db", "method"}),
	       crudLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
		       Namespace: "client",
		       Name:      "crud_latency_seconds",
		       Help:      "Latency of CRUD operations in seconds.",
		       Buckets:   buckets,
		       ConstLabels: prometheus.Labels{"db": dbLabel},
	       }, []string{"op"}),
       }
       reg.MustRegister(m.clients, m.crudLatency)
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