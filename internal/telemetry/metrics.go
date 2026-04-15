package telemetry

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	metricsOnce sync.Once

	httpRequestsTotal    *prometheus.CounterVec
	httpRequestDuration  *prometheus.HistogramVec
	httpRequestsInFlight *prometheus.GaugeVec

	billingOperationTotal   *prometheus.CounterVec
	billingOperationLatency *prometheus.HistogramVec
)

func MetricsHandler() http.Handler {
	ensureMetrics()
	return promhttp.Handler()
}

func ObserveHTTPRequest(app, method, route string, status int, duration time.Duration) {
	ensureMetrics()
	statusCode := strconv.Itoa(status)
	httpRequestsTotal.WithLabelValues(app, method, route, statusCode).Inc()
	httpRequestDuration.WithLabelValues(app, method, route, statusCode).Observe(duration.Seconds())
}

func IncHTTPInFlight(app string) {
	ensureMetrics()
	httpRequestsInFlight.WithLabelValues(app).Inc()
}

func DecHTTPInFlight(app string) {
	ensureMetrics()
	httpRequestsInFlight.WithLabelValues(app).Dec()
}

func ObserveOperation(name string, duration time.Duration, err error) {
	ensureMetrics()
	result := "ok"
	if err != nil {
		result = "error"
	}
	billingOperationTotal.WithLabelValues(name, result).Inc()
	billingOperationLatency.WithLabelValues(name, result).Observe(duration.Seconds())
}

func ensureMetrics() {
	metricsOnce.Do(func() {
		httpRequestsTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "railzway",
				Subsystem: "http",
				Name:      "requests_total",
				Help:      "Total HTTP requests handled by Railzway.",
			},
			[]string{"app", "method", "route", "status"},
		)
		httpRequestDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "railzway",
				Subsystem: "http",
				Name:      "request_duration_seconds",
				Help:      "HTTP request latency in seconds.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"app", "method", "route", "status"},
		)
		httpRequestsInFlight = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "railzway",
				Subsystem: "http",
				Name:      "requests_in_flight",
				Help:      "Current in-flight HTTP requests.",
			},
			[]string{"app"},
		)
		billingOperationTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "railzway",
				Subsystem: "billing",
				Name:      "operations_total",
				Help:      "Total important billing operations by result.",
			},
			[]string{"operation", "result"},
		)
		billingOperationLatency = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "railzway",
				Subsystem: "billing",
				Name:      "operation_duration_seconds",
				Help:      "Latency for important billing operations in seconds.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"operation", "result"},
		)

		prometheus.MustRegister(
			httpRequestsTotal,
			httpRequestDuration,
			httpRequestsInFlight,
			billingOperationTotal,
			billingOperationLatency,
		)
	})
}
