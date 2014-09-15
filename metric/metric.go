package metric

import drop "github.com/cloudfoundry/dropsonde/autowire/metrics"

type Counter string

func (c Counter) Increment() {
	drop.IncrementCounter(string(c))
}

func (c Counter) Add(i uint64) {
	drop.AddToCounter(string(c), i)
}
