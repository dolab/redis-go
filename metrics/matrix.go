package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// A Matrix defines matrix for both gRPC server and gRPC client
type Matrix struct {
	connections     *prometheus.GaugeVec
	requests        *prometheus.GaugeVec
	requestsTotal   *prometheus.CounterVec
	commands        *prometheus.GaugeVec
	commandsTotal   *prometheus.CounterVec
	bytesReceived   *prometheus.CounterVec
	bytesSend       *prometheus.CounterVec
	bytesWrite      *prometheus.CounterVec
	bytesRead       *prometheus.CounterVec
	redisDuration   *prometheus.HistogramVec
	proxyDuration   *prometheus.HistogramVec
	requestDuration *prometheus.HistogramVec
	errors          *prometheus.CounterVec
}

// NewServerMatrix creates a new matrix for gRPC server
func NewServerMatrix(subsystem string, labels prometheus.Labels) *Matrix {
	serverConnections := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   "redis",
			Subsystem:   subsystem,
			Name:        "connections",
			Help:        "Number of currently opened server side connections.",
			ConstLabels: labels,
		},
		[]string{"remote_addr", "local_addr"},
	)
	serverRequests := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   "redis",
			Subsystem:   subsystem,
			Name:        "requests",
			Help:        "Number of currently processing requests by server.",
			ConstLabels: labels,
		},
		[]string{"remote_addr", "local_addr"},
	)
	serverRequestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "redis",
			Subsystem:   subsystem,
			Name:        "requests_total",
			Help:        "Total number of requests processed by server.",
			ConstLabels: labels,
		},
		[]string{"remote_addr", "local_addr"},
	)
	serverCommands := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   "redis",
			Subsystem:   subsystem,
			Name:        "commands",
			Help:        "Number of currently processing commands by server.",
			ConstLabels: labels,
		},
		[]string{"remote_addr", "local_addr", "cmd"},
	)
	serverCommandsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "redis",
			Subsystem:   subsystem,
			Name:        "commands_total",
			Help:        "Total number of commands processed by server.",
			ConstLabels: labels,
		},
		[]string{"remote_addr", "local_addr", "cmd"},
	)
	serverRecvBytes := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "redis",
			Subsystem:   subsystem,
			Name:        "bytes_recv_total",
			Help:        "Total bytes of messages received from client by server.",
			ConstLabels: labels,
		},
		[]string{"remote_addr", "local_addr"},
	)
	serverSendBytes := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "redis",
			Subsystem:   subsystem,
			Name:        "bytes_send_total",
			Help:        "Total bytes of messages send to client by server.",
			ConstLabels: labels,
		},
		[]string{"remote_addr", "local_addr"},
	)
	serverWriteBytes := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "redis",
			Subsystem:   subsystem,
			Name:        "bytes_write_total",
			Help:        "Total bytes of messages write to redis by server.",
			ConstLabels: labels,
		},
		[]string{"remote_addr", "local_addr"},
	)
	serverReadBytes := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "redis",
			Subsystem:   subsystem,
			Name:        "bytes_read_total",
			Help:        "Total bytes of messages read from redis by server.",
			ConstLabels: labels,
		},
		[]string{"remote_addr", "local_addr"},
	)
	serverProxyDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:   "redis",
			Subsystem:   subsystem,
			Name:        "proxy_duration_seconds",
			Help:        "The request latencies in seconds between client and server.",
			ConstLabels: labels,
		},
		[]string{"remote_addr", "local_addr"},
	)
	serverRedisDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:   "redis",
			Subsystem:   subsystem,
			Name:        "redis_duration_seconds",
			Help:        "The request latencies in seconds between redis and server.",
			ConstLabels: labels,
		},
		[]string{"local_addr", "remote_addr"},
	)
	serverRequestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:   "redis",
			Subsystem:   subsystem,
			Name:        "request_duration_seconds",
			Help:        "The request latencies in seconds on server side.",
			ConstLabels: labels,
		},
		[]string{"remote_addr", "local_addr"},
	)
	serverErrors := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "redis",
			Subsystem:   subsystem,
			Name:        "errors_total",
			Help:        "Total number of errors that happen during process on server side.",
			ConstLabels: labels,
		},
		[]string{"remote_addr", "local_addr", "cmds"},
	)

	return &Matrix{
		connections:     serverConnections,
		requests:        serverRequests,
		requestsTotal:   serverRequestsTotal,
		commands:        serverCommands,
		commandsTotal:   serverCommandsTotal,
		bytesReceived:   serverRecvBytes,
		bytesSend:       serverSendBytes,
		bytesWrite:      serverWriteBytes,
		bytesRead:       serverReadBytes,
		proxyDuration:   serverProxyDuration,
		redisDuration:   serverRedisDuration,
		requestDuration: serverRequestDuration,
		errors:          serverErrors,
	}
}

// Describe implements prometheus Collector interface.
func (m *Matrix) Describe(in chan<- *prometheus.Desc) {
	// HistogramVec
	m.proxyDuration.Describe(in)
	m.redisDuration.Describe(in)
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
	m.proxyDuration.Collect(in)
	m.redisDuration.Collect(in)
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
