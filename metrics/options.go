package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Options wraps settings of metrics.
type Options struct {
	Subsystem           string
	Labels              prometheus.Labels
	EnableServerMetrics bool
}

// FillWithDefaults setups default values of options.
func (opts Options) FillWithDefaults() {
	if len(opts.Subsystem) == 0 {
		opts.Subsystem = "server"
	}
}

func (opts Options) Enabled() bool {
	return opts.EnableServerMetrics
}
