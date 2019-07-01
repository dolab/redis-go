package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics implements prometheus Collector interface for server and handler.
type Metrics struct {
	monitor *Monitor

	// for custom metrics with handler
	mutex      sync.RWMutex
	counters   map[string]*prometheus.CounterVec
	gauges     map[string]*prometheus.GaugeVec
	histograms map[string]*prometheus.HistogramVec
}

// NewMetrics creates a new metrics of grpc interceptor for shared usage.
func NewMetrics(labels prometheus.Labels) *Metrics {
	return &Metrics{
		monitor:    NewMonitor(labels),
		counters:   make(map[string]*prometheus.CounterVec),
		gauges:     make(map[string]*prometheus.GaugeVec),
		histograms: make(map[string]*prometheus.HistogramVec),
	}
}

// Describe implements prometheus Collector interface.
func (m *Metrics) Describe(in chan<- *prometheus.Desc) {
	m.monitor.Describe(in)

	for _, vec := range m.counters {
		vec.Describe(in)
	}

	for _, vec := range m.gauges {
		vec.Describe(in)
	}

	for _, vec := range m.histograms {
		vec.Describe(in)
	}
}

// Collect implements prometheus Collector interface.
func (m *Metrics) Collect(in chan<- prometheus.Metric) {
	m.monitor.Collect(in)

	for _, vec := range m.counters {
		vec.Collect(in)
	}

	for _, vec := range m.gauges {
		vec.Collect(in)
	}

	for _, vec := range m.histograms {
		vec.Collect(in)
	}
}

// Counter ...
func (m *Metrics) Counter(name, help string, labels []string) *prometheus.CounterVec {
	// find
	m.mutex.RLock()
	if vec, ok := m.counters[name]; ok {
		m.mutex.RUnlock()

		return vec
	}
	m.mutex.RUnlock()

	// create
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if vec, ok := m.counters[name]; ok {
		return vec
	}

	m.counters[name] = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "redis",
			Subsystem: "handler",
			Name:      name,
			Help:      help,
		},
		labels,
	)

	return m.counters[name]
}

// Gauge ...
func (m *Metrics) Gauge(name, help string, labels []string) *prometheus.GaugeVec {
	// find
	m.mutex.RLock()
	if vec, ok := m.gauges[name]; ok {
		m.mutex.RUnlock()

		return vec
	}
	m.mutex.RUnlock()

	// create
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if vec, ok := m.gauges[name]; ok {
		return vec
	}

	m.gauges[name] = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "redis",
			Subsystem: "handler",
			Name:      name,
			Help:      help,
		},
		labels,
	)

	return m.gauges[name]
}

// Histogram ...
func (m *Metrics) Histogram(name, help string, labels []string) *prometheus.HistogramVec {
	// find
	m.mutex.RLock()
	if vec, ok := m.histograms[name]; ok {
		m.mutex.RUnlock()

		return vec
	}
	m.mutex.RUnlock()

	// create
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if vec, ok := m.histograms[name]; ok {
		return vec
	}

	m.histograms[name] = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "redis",
			Subsystem: "handler",
			Name:      name,
			Help:      help,
		},
		labels,
	)

	return m.histograms[name]
}
