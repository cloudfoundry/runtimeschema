package metric

import (
	"time"

	"github.com/cloudfoundry/dropsonde/autowire/metrics"
)

type Counter string

func (c Counter) Increment() {
	metrics.IncrementCounter(string(c))
}

func (c Counter) Add(i uint64) {
	metrics.AddToCounter(string(c), i)
}

type Duration string

func (name Duration) Send(duration time.Duration) {
	metrics.SendValue(string(name), float64(duration), "nanos")
}
