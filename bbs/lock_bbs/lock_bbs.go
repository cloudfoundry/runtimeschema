package lock_bbs

import (
	"time"

	"github.com/cloudfoundry-incubator/consuladapter"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/maintainer"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/ifrit"
)

const LockTTL = 10 * time.Second
const RetryInterval = 5 * time.Second

type LockBBS struct {
	consul *consuladapter.Session
	logger lager.Logger
	clock  clock.Clock
}

func New(consul *consuladapter.Session, clock clock.Clock, logger lager.Logger) *LockBBS {
	return &LockBBS{
		consul: consul,
		logger: logger,
		clock:  clock,
	}
}

func (bbs *LockBBS) NewAuctioneerLock(auctioneerPresence models.AuctioneerPresence, retryInterval time.Duration) (ifrit.Runner, error) {
	auctionerPresenceJSON, err := models.ToJSON(auctioneerPresence)
	if err != nil {
		return nil, err
	}
	return maintainer.NewLock(bbs.consul, shared.LockSchemaPath("auctioneer_lock"), auctionerPresenceJSON, bbs.clock, retryInterval, bbs.logger), nil
}

func (bbs *LockBBS) NewConvergeLock(convergerID string, retryInterval time.Duration) ifrit.Runner {
	return maintainer.NewLock(bbs.consul, shared.LockSchemaPath("converge_lock"), []byte(convergerID), bbs.clock, retryInterval, bbs.logger)
}

func (bbs *LockBBS) NewNsyncBulkerLock(bulkerID string, retryInterval time.Duration) ifrit.Runner {
	return maintainer.NewLock(bbs.consul, shared.LockSchemaPath("nsync_bulker_lock"), []byte(bulkerID), bbs.clock, retryInterval, bbs.logger)
}

func (bbs *LockBBS) NewRouteEmitterLock(emitterID string, retryInterval time.Duration) ifrit.Runner {
	return maintainer.NewLock(bbs.consul, shared.LockSchemaPath("route_emitter_lock"), []byte(emitterID), bbs.clock, retryInterval, bbs.logger)
}

func (bbs *LockBBS) NewRuntimeMetricsLock(runtimeMetricsID string, retryInterval time.Duration) ifrit.Runner {
	return maintainer.NewLock(bbs.consul, shared.LockSchemaPath("runtime_metrics_lock"), []byte(runtimeMetricsID), bbs.clock, retryInterval, bbs.logger)
}

func (bbs *LockBBS) NewTpsWatcherLock(tpsWatcherID string, retryInterval time.Duration) ifrit.Runner {
	return maintainer.NewLock(bbs.consul, shared.LockSchemaPath("tps_watcher_lock"), []byte(tpsWatcherID), bbs.clock, retryInterval, bbs.logger)
}

func (bbs *LockBBS) NewBBSMasterLock(bbsPresence models.BBSPresence, retryInterval time.Duration) (ifrit.Runner, error) {
	bbsPresenceJSON, err := models.ToJSON(bbsPresence)
	if err != nil {
		return nil, err
	}
	return maintainer.NewLock(bbs.consul, shared.LockSchemaPath("bbs_lock"), bbsPresenceJSON, bbs.clock, retryInterval, bbs.logger), nil
}
