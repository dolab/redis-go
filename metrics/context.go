package metrics

import (
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func (m *Metrics) Dialer(localAddr, remoteAddr string) {
	if !m.Enabled() {
		return
	}

	labels := prometheus.Labels{
		"local_addr":  TrimPort(localAddr),
		"remote_addr": TrimPort(remoteAddr),
	}

	m.monitor.dialer.With(labels).Inc()
}

func (m *Metrics) IncConnection(remoteAddr, localAddr string) {
	if !m.Enabled() {
		return
	}

	labels := prometheus.Labels{
		"local_addr":  TrimPort(localAddr),
		"remote_addr": TrimPort(remoteAddr),
	}

	m.monitor.server.connections.With(labels).Inc()
}

func (m *Metrics) DecConnection(remoteAddr, localAddr string) {
	if !m.Enabled() {
		return
	}

	labels := prometheus.Labels{
		"local_addr":  TrimPort(localAddr),
		"remote_addr": TrimPort(remoteAddr),
	}

	m.monitor.server.connections.With(labels).Dec()
}

func (m *Metrics) IncRequest(remoteAddr, localAddr string) {
	if !m.Enabled() {
		return
	}

	labels := prometheus.Labels{
		"remote_addr": remoteAddr,
		"local_addr":  localAddr,
	}

	// for request processing
	m.monitor.server.requests.With(labels).Inc()

	// for request processed
	m.monitor.server.requestsTotal.With(labels).Inc()
}

func (m *Metrics) DecRequest(remoteAddr, localAddr string) {
	if !m.Enabled() {
		return
	}

	labels := prometheus.Labels{
		"remote_addr": remoteAddr,
		"local_addr":  localAddr,
	}

	// for request processing
	m.monitor.server.requests.With(labels).Dec()
}

func (m *Metrics) IncCommands(remoteAddr, localAddr string, cmds []string) {
	if !m.Enabled() {
		return
	}

	for _, cmd := range cmds {
		labels := prometheus.Labels{
			"remote_addr": remoteAddr,
			"local_addr":  localAddr,
			"cmd":         cmd,
		}

		// for commands processing
		m.monitor.server.commands.With(labels).Inc()

		// for commands processed
		m.monitor.server.commandsTotal.With(labels).Inc()
	}
}

func (m *Metrics) DecCommands(remoteAddr, localAddr string, cmds []string) {
	if !m.Enabled() {
		return
	}

	for _, cmd := range cmds {
		labels := prometheus.Labels{
			"remote_addr": remoteAddr,
			"local_addr":  localAddr,
			"cmd":         cmd,
		}

		// for commands processing
		m.monitor.server.commands.With(labels).Dec()
	}
}

func (m *Metrics) IncBytesReceived(remoteAddr, localAddr string, size float64) {
	if !m.Enabled() {
		return
	}

	labels := prometheus.Labels{
		"remote_addr": remoteAddr,
		"local_addr":  localAddr,
	}

	// for bytes received from client
	m.monitor.server.bytesReceived.With(labels).Add(size)
}

func (m *Metrics) IncBytesSend(remoteAddr, localAddr string, size float64) {
	if !m.Enabled() {
		return
	}

	labels := prometheus.Labels{
		"remote_addr": remoteAddr,
		"local_addr":  localAddr,
	}

	// for bytes send to client
	m.monitor.server.bytesSend.With(labels).Add(size)
}

func (m *Metrics) IncBytesWrite(localAddr, remoteAddr string, size float64) {
	if !m.Enabled() {
		return
	}

	labels := prometheus.Labels{
		"remote_addr": remoteAddr,
		"local_addr":  localAddr,
	}

	// for bytes write to redis
	m.monitor.server.bytesWrite.With(labels).Add(size)
}

func (m *Metrics) IncBytesRead(localAddr, remoteAddr string, size float64) {
	if !m.Enabled() {
		return
	}

	labels := prometheus.Labels{
		"remote_addr": remoteAddr,
		"local_addr":  localAddr,
	}

	// for bytes read from redis
	m.monitor.server.bytesRead.With(labels).Add(size)
}

func (m *Metrics) ObserveProxy(remoteAddr, localAddr string, issuedAt time.Time) {
	if !m.Enabled() {
		return
	}

	labels := prometheus.Labels{
		"remote_addr": remoteAddr,
		"local_addr":  localAddr,
	}

	m.monitor.server.proxyDuration.With(labels).Observe(time.Since(issuedAt).Seconds())
}

func (m *Metrics) ObserveRedis(remoteAddr, localAddr string, issuedAt time.Time) {
	if !m.Enabled() {
		return
	}

	labels := prometheus.Labels{
		"remote_addr": TrimPort(remoteAddr),
		"local_addr":  TrimPort(localAddr),
	}

	m.monitor.server.redisDuration.With(labels).Observe(time.Since(issuedAt).Seconds())
}

func (m *Metrics) ObserveRequest(remoteAddr, localAddr string, issuedAt time.Time) {
	if !m.Enabled() {
		return
	}

	labels := prometheus.Labels{
		"remote_addr": remoteAddr,
		"local_addr":  localAddr,
	}

	m.monitor.server.requestDuration.With(labels).Observe(time.Since(issuedAt).Seconds())
}

func (m *Metrics) IncErrors(remoteAddr, localAddr string, cmds []string) {
	if !m.Enabled() {
		return
	}

	labels := prometheus.Labels{
		"remote_addr": remoteAddr,
		"local_addr":  localAddr,
		"cmds":        strings.Join(cmds, ","),
	}

	m.monitor.server.errors.With(labels).Inc()
}
