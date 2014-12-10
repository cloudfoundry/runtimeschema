package lrp_bbs_test

import (
	"time"

	"github.com/cloudfoundry/gunk/timeprovider/faketimeprovider"
)

type AdvancingFakeTimeProvider struct {
	*faketimeprovider.FakeTimeProvider
	IntervalToAdvance time.Duration
}

func (p *AdvancingFakeTimeProvider) Now() time.Time {
	timeToReturn := p.FakeTimeProvider.Now()
	p.FakeTimeProvider.Increment(p.IntervalToAdvance)
	return timeToReturn
}
