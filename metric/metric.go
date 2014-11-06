package metric

import (
	"time"

	"github.com/cloudfoundry/dropsonde/metrics"
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

type Mebibytes string

func (name Mebibytes) Send(mebibytes int) {
	metrics.SendValue(string(name), float64(mebibytes), "MiB")
}

type Metric string

func (name Metric) Send(value int) {
	metrics.SendValue(string(name), float64(value), "Metric")
}
