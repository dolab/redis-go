package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// A Matrix defines matrix for both gRPC server and gRPC client
type Matrix struct {
	requestDuration *prometheus.HistogramVec
	connections     *prometheus.GaugeVec
	requests        *prometheus.GaugeVec
	requestsTotal   *prometheus.CounterVec
	commands        *prometheus.GaugeVec
	commandsTotal   *prometheus.CounterVec
	bytesReceived   *prometheus.CounterVec
	bytesSend       *prometheus.CounterVec
	errors          *prometheus.CounterVec
}

// NewServerMatrix creates a new matrix for gRPC server
func NewServerMatrix(labels prometheus.Labels) *Matrix {
	serverConnections := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   "redis",
			Subsystem:   "server",
			Name:        "connections",
			Help:        "Number of currently opened server side connections.",
			ConstLabels: labels,
		},
		[]string{"remote_addr", "local_addr"},
	)
	serverRequests := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   "redis",
			Subsystem:   "server",
			Name:        "requests",
			Help:        "Number of currently processing requests by server.",
			ConstLabels: labels,
		},
		[]string{"remote_addr"},
	)
	serverRequestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "redis",
			Subsystem:   "server",
			Name:        "requests_total",
			Help:        "Total number of requests processed by server.",
			ConstLabels: labels,
		},
		[]string{"remote_addr"},
	)
	serverRequestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:   "redis",
			Subsystem:   "server",
			Name:        "request_duration_seconds",
			Help:        "The request latencies in seconds on server side.",
			ConstLabels: labels,
		},
		[]string{"remote_addr"},
	)
	serverCommands := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   "redis",
			Subsystem:   "server",
			Name:        "commands",
			Help:        "Number of currently processing commands by server.",
			ConstLabels: labels,
		},
		[]string{"remote_addr", "cmd"},
	)
	serverCommandsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "redis",
			Subsystem:   "server",
			Name:        "commands_total",
			Help:        "Number of commands processed by server.",
			ConstLabels: labels,
		},
		[]string{"remote_addr", "cmd"},
	)
	serverRecvBytes := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "redis",
			Subsystem:   "server",
			Name:        "bytes_recv_total",
			Help:        "Total bytes of messages received by server.",
			ConstLabels: labels,
		},
		[]string{"remote_addr"},
	)
	serverSendBytes := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "redis",
			Subsystem:   "server",
			Name:        "bytes_send_total",
			Help:        "Total bytes of messages send by server.",
			ConstLabels: labels,
		},
		[]string{"remote_addr"},
	)
	serverErrors := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "redis",
			Subsystem:   "server",
			Name:        "errors_total",
			Help:        "Total number of errors that happen during process on server side.",
			ConstLabels: labels,
		},
		[]string{"remote_addr", "cmds"},
	)

	return &Matrix{
		connections:     serverConnections,
		requests:        serverRequests,
		requestsTotal:   serverRequestsTotal,
		requestDuration: serverRequestDuration,
		commands:        serverCommands,
		commandsTotal:   serverCommandsTotal,
		bytesReceived:   serverRecvBytes,
		bytesSend:       serverSendBytes,
		errors:          serverErrors,
	}
}

// Describe implements prometheus Collector interface.
func (m *Matrix) Describe(in chan<- *prometheus.Desc) {
	// HistogramVec
	m.requestDuration.Describe(in)

	// Gauge
	m.connections.Describe(in)
	m.requests.Describe(in)
	m.commands.Describe(in)

	// CounterVec
	m.requestsTotal.Describe(in)
	m.commandsTotal.Describe(in)
	m.bytesReceived.Describe(in)
	m.bytesSend.Describe(in)
	m.errors.Describe(in)
}

// Collect implements prometheus Collector interface.
func (m *Matrix) Collect(in chan<- prometheus.Metric) {
	// HistogramVec
	m.requestDuration.Collect(in)

	// Gauge
	m.connections.Collect(in)
	m.requests.Collect(in)
	m.commands.Collect(in)

	// CounterVec
	m.requestsTotal.Collect(in)
	m.commandsTotal.Collect(in)
	m.bytesReceived.Collect(in)
	m.bytesSend.Collect(in)
	m.errors.Collect(in)
}
