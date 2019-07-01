package metrics

import (
	"net"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics implements both prometheus Collector interface and
// grpc interceptor interface.
type Metrics struct {
	monitor    *Monitor
	trackPeers bool
}

// NewMetrics creates a new metrics of grpc interceptor for shared usage.
func NewMetrics(labels prometheus.Labels) *Metrics {
	return &Metrics{
		monitor: NewMonitor(labels),
	}
}

// Describe implements prometheus Collector interface.
func (m *Metrics) Describe(in chan<- *prometheus.Desc) {
	m.monitor.Describe(in)
}

// Collect implements prometheus Collector interface.
func (m *Metrics) Collect(in chan<- prometheus.Metric) {
	m.monitor.Collect(in)
}

// Dialer ...
func (m *Metrics) Dialer(f func(string, time.Duration) (net.Conn, error)) func(string, time.Duration) (net.Conn, error) {
	return func(addr string, timeout time.Duration) (net.Conn, error) {
		m.monitor.dialer.WithLabelValues(addr).Inc()

		return f(addr, timeout)
	}
}
