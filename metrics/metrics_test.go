package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golib/assert"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func TestMetrics(t *testing.T) {
	it := assert.New(t)

	metrics := NewMetrics(nil, true)
	if it.NotNil(metrics) {
		it.NotNil(metrics.counters)
		it.NotNil(metrics.gauges)
		it.NotNil(metrics.histograms)
	}
}

func TestMetrics_Counter(t *testing.T) {
	it := assert.New(t)

	metrics := NewMetrics(nil, true)

	name := "testing_counter"
	help := "custom testing counter"

	counter := metrics.Counter(name, help, nil)
	counter.With(nil).Inc()

	tmpcounter := metrics.Counter(name, help, nil)
	tmpcounter.With(nil).Inc()

	newcounter := metrics.Counter(name+"_new", help, nil)
	newcounter.With(nil).Inc()

	if it.Nil(prometheus.Register(metrics)) {
		defer prometheus.Unregister(metrics)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "https://example.com", nil)

		promhttp.Handler().ServeHTTP(w, r)

		output := w.Body.String()
		it.Contains(output, `redis_handler_testing_counter 2`)
		it.Contains(output, `redis_handler_testing_counter_new 1`)
	}
}

func TestMetrics_Gauge(t *testing.T) {
	it := assert.New(t)

	metrics := NewMetrics(nil, true)

	name := "testing_gauge"
	help := "custom testing gauge"

	gauge := metrics.Gauge(name, help, nil)
	gauge.With(nil).Inc()

	tmpguage := metrics.Gauge(name, help, nil)
	tmpguage.With(nil).Inc()

	newguage := metrics.Gauge(name+"_new", help, nil)
	newguage.With(nil).Inc()

	if it.Nil(prometheus.Register(metrics)) {
		defer prometheus.Unregister(metrics)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "https://example.com", nil)

		promhttp.Handler().ServeHTTP(w, r)

		output := w.Body.String()
		it.Contains(output, `redis_handler_testing_gauge 2`)
		it.Contains(output, `redis_handler_testing_gauge_new 1`)
	}
}

func TestMetrics_Histogram(t *testing.T) {
	it := assert.New(t)

	metrics := NewMetrics(nil, true)

	name := "testing_histogram"
	help := "custom testing histogram"

	issuedAt := time.Now()

	histogram := metrics.Histogram(name, help, nil)
	histogram.With(nil).Observe(time.Since(issuedAt).Seconds())

	tmphistogram := metrics.Histogram(name, help, nil)
	tmphistogram.With(nil).Observe(time.Since(issuedAt).Seconds())

	newhistogram := metrics.Histogram(name+"_new", help, nil)
	newhistogram.With(nil).Observe(time.Since(issuedAt).Seconds())

	if it.Nil(prometheus.Register(metrics)) {
		defer prometheus.Unregister(metrics)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "https://example.com", nil)

		promhttp.Handler().ServeHTTP(w, r)

		output := w.Body.String()
		it.Contains(output, `redis_handler_testing_histogram_count 2`)
		it.Contains(output, `redis_handler_testing_histogram_new_count 1`)
	}
}
