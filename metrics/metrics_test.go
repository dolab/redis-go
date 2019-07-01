package metrics

import (
	"testing"

	"github.com/golib/assert"
)

func Test_Metrics(t *testing.T) {
	it := assert.New(t)

	metrics := NewMetrics(nil)
	it.NotNil(metrics)
}
