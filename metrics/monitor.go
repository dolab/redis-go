package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// A Monitor defines metrics for gRPC
type Monitor struct {
	dialer *prometheus.CounterVec
	server *Matrix
}

// NewMonitor creates Monitor for starting
func NewMonitor(labels prometheus.Labels) *Monitor {
	dialer := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "redis",
			Subsystem:   "client",
			Name:        "total_reconnects",
			Help:        "Total number of reconnects made by server to redis back server.",
			ConstLabels: labels,
		},
		[]string{"address"},
	)

	return &Monitor{
		dialer: dialer,
		server: NewServerMatrix(labels),
	}
}

// Describe implements prometheus Collector interface.
func (m *Monitor) Describe(in chan<- *prometheus.Desc) {
	m.dialer.Describe(in)
	m.server.Describe(in)
}

// Collect implements prometheus Collector interface.
func (m *Monitor) Collect(in chan<- prometheus.Metric) {
	m.dialer.Collect(in)
	m.server.Collect(in)
}
