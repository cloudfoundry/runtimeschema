package lock_bbs

import (
	"time"

	"github.com/cloudfoundry-incubator/consuladapter"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/heartbeater"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/ifrit"
)

const LockTTL = 60 * time.Second
const RetryInterval = 5 * time.Second

type LockBBS struct {
	consul *consuladapter.Adapter
	logger lager.Logger
	clock  clock.Clock
}

func New(consul *consuladapter.Adapter, clock clock.Clock, logger lager.Logger) *LockBBS {
	return &LockBBS{
		consul: consul,
		logger: logger,
		clock:  clock,
	}
}

func (bbs *LockBBS) NewAuctioneerLock(auctioneerPresence models.AuctioneerPresence, ttl, retryInterval time.Duration) (ifrit.Runner, error) {
	auctionerPresenceJSON, err := models.ToJSON(auctioneerPresence)
	if err != nil {
		return nil, err
	}
	return heartbeater.New(bbs.consul, shared.LockSchemaPath("auctioneer_lock"), auctionerPresenceJSON, ttl, bbs.clock, retryInterval, bbs.logger), nil
}

func (bbs *LockBBS) NewConvergeLock(convergerID string, ttl, retryInterval time.Duration) ifrit.Runner {
	return heartbeater.New(bbs.consul, shared.LockSchemaPath("converge_lock"), []byte(convergerID), ttl, bbs.clock, retryInterval, bbs.logger)
}

func (bbs *LockBBS) NewNsyncBulkerLock(bulkerID string, ttl, retryInterval time.Duration) ifrit.Runner {
	return heartbeater.New(bbs.consul, shared.LockSchemaPath("nsync_bulker_lock"), []byte(bulkerID), ttl, bbs.clock, retryInterval, bbs.logger)
}

func (bbs *LockBBS) NewRouteEmitterLock(emitterID string, ttl, retryInterval time.Duration) ifrit.Runner {
	return heartbeater.New(bbs.consul, shared.LockSchemaPath("route_emitter_lock"), []byte(emitterID), ttl, bbs.clock, retryInterval, bbs.logger)
}

func (bbs *LockBBS) NewRuntimeMetricsLock(runtimeMetricsID string, ttl, retryInterval time.Duration) ifrit.Runner {
	return heartbeater.New(bbs.consul, shared.LockSchemaPath("runtime_metrics_lock"), []byte(runtimeMetricsID), ttl, bbs.clock, retryInterval, bbs.logger)
}
