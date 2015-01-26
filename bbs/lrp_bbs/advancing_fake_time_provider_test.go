package lrp_bbs_test

import (
	"time"

	"github.com/pivotal-golang/clock/fakeclock"
)

type AdvancingFakeClock struct {
	*fakeclock.FakeClock
	IntervalToAdvance time.Duration
}

func (p *AdvancingFakeClock) Now() time.Time {
	timeToReturn := p.FakeClock.Now()
	p.FakeClock.Increment(p.IntervalToAdvance)
	return timeToReturn
}
