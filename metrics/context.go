package metrics

import (
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func (m *Metrics) Dialer(localAddr, remoteAddr string) {
	localN := len(localAddr)
	if n := strings.IndexByte(localAddr, ':'); n > 0 {
		localN = n
	}

	remoteN := len(remoteAddr)
	if n := strings.IndexByte(remoteAddr, ':'); n > 0 {
		remoteN = n
	}

	labels := prometheus.Labels{
		"local_addr":  localAddr[:localN],
		"remote_addr": remoteAddr[:remoteN],
	}

	m.monitor.dialer.With(labels).Inc()
}

func (m *Metrics) IncConnection(remoteAddr, localAddr string) {
	localN := len(localAddr)
	if n := strings.IndexByte(localAddr, ':'); n > 0 {
		localN = n
	}

	remoteN := len(remoteAddr)
	if n := strings.IndexByte(remoteAddr, ':'); n > 0 {
		remoteN = n
	}

	labels := prometheus.Labels{
		"local_addr":  localAddr[:localN],
		"remote_addr": remoteAddr[:remoteN],
	}

	m.monitor.server.connections.With(labels).Inc()
}

func (m *Metrics) DecConnection(remoteAddr, localAddr string) {
	localN := len(localAddr)
	if n := strings.IndexByte(localAddr, ':'); n > 0 {
		localN = n
	}

	remoteN := len(remoteAddr)
	if n := strings.IndexByte(remoteAddr, ':'); n > 0 {
		remoteN = n
	}

	labels := prometheus.Labels{
		"local_addr":  localAddr[:localN],
		"remote_addr": remoteAddr[:remoteN],
	}

	m.monitor.server.connections.With(labels).Dec()
}

func (m *Metrics) IncRequest(remoteAddr string) {
	labels := prometheus.Labels{
		"remote_addr": remoteAddr,
	}

	// for request processing
	m.monitor.server.requests.With(labels).Inc()

	// for request processed
	m.monitor.server.requestsTotal.With(labels).Inc()
}

func (m *Metrics) DecRequest(remoteAddr string) {
	labels := prometheus.Labels{
		"remote_addr": remoteAddr,
	}

	// for request processing
	m.monitor.server.requests.With(labels).Dec()
}

func (m *Metrics) ObserveRequest(remoteAddr string, issuedAt time.Time) {
	labels := prometheus.Labels{
		"remote_addr": remoteAddr,
	}

	m.monitor.server.requestDuration.With(labels).Observe(time.Since(issuedAt).Seconds())
}

func (m *Metrics) IncCommands(remoteAddr string, cmds []string) {
	for _, cmd := range cmds {
		labels := prometheus.Labels{
			"remote_addr": remoteAddr,
			"cmd":         cmd,
		}

		// for commands processing
		m.monitor.server.commands.With(labels).Inc()

		// for commands processed
		m.monitor.server.commandsTotal.With(labels).Inc()
	}
}

func (m *Metrics) DecCommands(remoteAddr string, cmds []string) {
	for _, cmd := range cmds {
		labels := prometheus.Labels{
			"remote_addr": remoteAddr,
			"cmd":         cmd,
		}

		// for commands processing
		m.monitor.server.commands.With(labels).Dec()
	}
}

func (m *Metrics) IncErrors(remoteAddr string, cmds []string) {
	labels := prometheus.Labels{
		"remote_addr": remoteAddr,
		"cmds":        strings.Join(cmds, ","),
	}

	m.monitor.server.errors.With(labels).Inc()
}
