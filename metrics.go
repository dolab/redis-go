package redis

import (
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/dolab/redis-go/metrics"
)

var (
	gometrics     *metrics.Metrics
	gometricsOnce sync.Once
)

// ServeMetrics exports prometheus metrics of internal server.
func ServeMetrics(w http.ResponseWriter, r *http.Request) {
	gometricsOnce.Do(func() {
		if gometrics == nil {
			return
		}

		prometheus.MustRegister(gometrics)
	})

	promhttp.Handler().ServeHTTP(w, r)
}

// NewCounterVec returns a *prometheus.CounterVec for handler usage.
func NewCounterVec(name, help string, labels []string) *prometheus.CounterVec {
	return gometrics.Counter(name, help, labels)
}

// NewGaugeVec returns a *prometheus.GaugeVec for handler usage.
func NewGaugeVec(name, help string, labels []string) *prometheus.GaugeVec {
	return gometrics.Gauge(name, help, labels)
}

// NewHistogramVec returns a *prometheus.HistogramVec for handler usage.
func NewHistogramVec(name, help string, labels []string) *prometheus.HistogramVec {
	return gometrics.Histogram(name, help, labels)
}
