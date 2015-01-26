package lock_bbs

import (
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/heartbeater"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/ifrit"
)

const HEARTBEAT_INTERVAL = 30 * time.Second

type LockBBS struct {
	store  storeadapter.StoreAdapter
	logger lager.Logger
	clock  clock.Clock
}

func New(store storeadapter.StoreAdapter, clock clock.Clock, logger lager.Logger) *LockBBS {
	return &LockBBS{
		store:  store,
		logger: logger,
		clock:  clock,
	}
}

func (bbs *LockBBS) NewAuctioneerLock(auctioneerPresence models.AuctioneerPresence, interval time.Duration) (ifrit.Runner, error) {
	auctionerPresenceJSON, err := models.ToJSON(auctioneerPresence)
	if err != nil {
		return nil, err
	}
	return heartbeater.New(bbs.store, bbs.clock, shared.LockSchemaPath("auctioneer_lock"), string(auctionerPresenceJSON), interval, bbs.logger), nil
}

func (bbs *LockBBS) NewConvergeLock(convergerID string, interval time.Duration) ifrit.Runner {
	return heartbeater.New(bbs.store, bbs.clock, shared.LockSchemaPath("converge_lock"), convergerID, interval, bbs.logger)
}

func (bbs *LockBBS) NewNsyncBulkerLock(bulkerID string, interval time.Duration) ifrit.Runner {
	return heartbeater.New(bbs.store, bbs.clock, shared.LockSchemaPath("nsync_bulker_lock"), bulkerID, interval, bbs.logger)
}

func (bbs *LockBBS) NewNsyncListenerLock(listenerID string, interval time.Duration) ifrit.Runner {
	return heartbeater.New(bbs.store, bbs.clock, shared.LockSchemaPath("nsync_listener_lock"), listenerID, interval, bbs.logger)
}

func (bbs *LockBBS) NewRouteEmitterLock(emitterID string, interval time.Duration) ifrit.Runner {
	return heartbeater.New(bbs.store, bbs.clock, shared.LockSchemaPath("route_emitter_lock"), emitterID, interval, bbs.logger)
}

func (bbs *LockBBS) NewRuntimeMetricsLock(runtimeMetricsID string, interval time.Duration) ifrit.Runner {
	return heartbeater.New(bbs.store, bbs.clock, shared.LockSchemaPath("runtime_metrics_lock"), runtimeMetricsID, interval, bbs.logger)
}
