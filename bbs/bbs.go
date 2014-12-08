package bbs

import (
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/lock_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/lrp_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/services_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/start_auction_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/stop_auction_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/task_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/cb"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/timeprovider"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/ifrit"
)

//Bulletin Board System/Store

type ReceptorBBS interface {
	//task
	DesireTask(models.Task) error
	Tasks() ([]models.Task, error)
	TasksByDomain(domain string) ([]models.Task, error)
	TaskByGuid(taskGuid string) (*models.Task, error)
	ResolvingTask(taskGuid string) error
	ResolveTask(taskGuid string) error
	CancelTask(taskGuid string) error

	//desired lrp
	DesireLRP(models.DesiredLRP) error
	UpdateDesiredLRP(processGuid string, update models.DesiredLRPUpdate) error
	RemoveDesiredLRPByProcessGuid(processGuid string) error
	DesiredLRPs() ([]models.DesiredLRP, error)
	DesiredLRPsByDomain(domain string) ([]models.DesiredLRP, error)
	DesiredLRPByProcessGuid(processGuid string) (*models.DesiredLRP, error)

	//actual lrp
	ActualLRPs() ([]models.ActualLRP, error)
	ActualLRPsByDomain(domain string) ([]models.ActualLRP, error)
	ActualLRPsByProcessGuid(string) ([]models.ActualLRP, error)
	ActualLRPByProcessGuidAndIndex(string, int) (*models.ActualLRP, error)
	RequestStopLRPInstance(stopInstances models.ActualLRP) error

	// cells
	Cells() ([]models.CellPresence, error)

	// freshness
	BumpFreshness(models.Freshness) error
	Freshnesses() ([]models.Freshness, error)

	//services
	NewReceptorHeartbeat(models.ReceptorPresence, time.Duration) ifrit.Runner
}

type RepBBS interface {
	//services
	NewCellHeartbeat(cellPresence models.CellPresence, interval time.Duration) ifrit.Runner

	//task
	StartTask(taskGuid string, cellID string) error
	TaskByGuid(taskGuid string) (*models.Task, error)
	TasksByCellID(cellID string) ([]models.Task, error)
	CompleteTask(taskGuid string, failed bool, failureReason string, result string) error

	//lrp
	ActualLRPsByCellID(cellID string) ([]models.ActualLRP, error)
	ClaimActualLRP(models.ActualLRP) (*models.ActualLRP, error)
	StartActualLRP(models.ActualLRP) (*models.ActualLRP, error)
	RemoveActualLRP(lrp models.ActualLRP) error
}

type ConvergerBBS interface {
	//lrp
	ConvergeLRPs()
	ActualLRPsByProcessGuid(string) ([]models.ActualLRP, error)
	RequestStopLRPInstance(lrp models.ActualLRP) error
	WatchForDesiredLRPChanges() (<-chan models.DesiredLRPChange, chan<- bool, <-chan error)
	CreateActualLRP(models.ActualLRP) (*models.ActualLRP, error)

	//start auction
	ConvergeLRPStartAuctions(kickPendingDuration time.Duration, expireClaimedDuration time.Duration)
	RequestLRPStartAuction(models.LRPStartAuction) error

	//stop auction
	ConvergeLRPStopAuctions(kickPendingDuration time.Duration, expireClaimedDuration time.Duration)
	RequestLRPStopAuction(models.LRPStopAuction) error

	//task
	ConvergeTask(timeToClaim, convergenceInterval, timeToResolve time.Duration)

	//lock
	NewConvergeLock(convergerID string, interval time.Duration) ifrit.Runner
}

type TPSBBS interface {
	//lrp
	ActualLRPsByProcessGuid(string) ([]models.ActualLRP, error)
}

type NsyncBBS interface {
	// lrp
	DesireLRP(models.DesiredLRP) error
	RemoveDesiredLRPByProcessGuid(guid string) error
	DesiredLRPsByDomain(domain string) ([]models.DesiredLRP, error)
	ChangeDesiredLRP(change models.DesiredLRPChange) error
	BumpFreshness(freshness models.Freshness) error

	//lock
	NewNsyncBulkerLock(bulkerID string, interval time.Duration) ifrit.Runner
	NewNsyncListenerLock(listenerID string, interval time.Duration) ifrit.Runner
}

type AuctioneerBBS interface {
	//services
	Cells() ([]models.CellPresence, error)

	//start auction
	WatchForLRPStartAuction() (<-chan models.LRPStartAuction, chan<- bool, <-chan error)
	ClaimLRPStartAuction(models.LRPStartAuction) error
	ResolveLRPStartAuction(models.LRPStartAuction) error

	//stop auction
	WatchForLRPStopAuction() (<-chan models.LRPStopAuction, chan<- bool, <-chan error)
	ClaimLRPStopAuction(models.LRPStopAuction) error
	ResolveLRPStopAuction(models.LRPStopAuction) error

	//task
	WatchForDesiredTask() (<-chan models.Task, chan<- bool, <-chan error)

	//lock
	NewAuctioneerLock(auctioneerID string, interval time.Duration) ifrit.Runner
}

type MetricsBBS interface {
	//task
	Tasks() ([]models.Task, error)

	//services
	ServiceRegistrations() (models.ServiceRegistrations, error)

	//lrps
	Freshnesses() ([]models.Freshness, error)
	DesiredLRPs() ([]models.DesiredLRP, error)
	ActualLRPs() ([]models.ActualLRP, error)

	//lock
	NewRuntimeMetricsLock(runtimeMetricsID string, interval time.Duration) ifrit.Runner
}

type RouteEmitterBBS interface {
	// lrp
	WatchForDesiredLRPChanges() (<-chan models.DesiredLRPChange, chan<- bool, <-chan error)
	WatchForActualLRPChanges() (<-chan models.ActualLRPChange, chan<- bool, <-chan error)
	DesiredLRPs() ([]models.DesiredLRP, error)
	RunningActualLRPs() ([]models.ActualLRP, error)

	//lock
	NewRouteEmitterLock(emitterID string, interval time.Duration) ifrit.Runner
}

type VeritasBBS interface {
	//task
	Tasks() ([]models.Task, error)

	//lrp
	DesiredLRPs() ([]models.DesiredLRP, error)
	ActualLRPs() ([]models.ActualLRP, error)
	DesireLRP(models.DesiredLRP) error
	RemoveDesiredLRPByProcessGuid(guid string) error
	Freshnesses() ([]models.Freshness, error)

	//start auctions
	LRPStartAuctions() ([]models.LRPStartAuction, error)

	//stop auctions
	LRPStopAuctions() ([]models.LRPStopAuction, error)

	//services
	Cells() ([]models.CellPresence, error)
}

func NewReceptorBBS(store storeadapter.StoreAdapter, timeProvider timeprovider.TimeProvider, logger lager.Logger) ReceptorBBS {
	return NewBBS(store, timeProvider, logger)
}

func NewRepBBS(store storeadapter.StoreAdapter, timeProvider timeprovider.TimeProvider, logger lager.Logger) RepBBS {
	return NewBBS(store, timeProvider, logger)
}

func NewConvergerBBS(store storeadapter.StoreAdapter, timeProvider timeprovider.TimeProvider, logger lager.Logger) ConvergerBBS {
	return NewBBS(store, timeProvider, logger)
}

func NewNsyncBBS(store storeadapter.StoreAdapter, timeProvider timeprovider.TimeProvider, logger lager.Logger) NsyncBBS {
	return NewBBS(store, timeProvider, logger)
}

func NewAuctioneerBBS(store storeadapter.StoreAdapter, timeProvider timeprovider.TimeProvider, logger lager.Logger) AuctioneerBBS {
	return NewBBS(store, timeProvider, logger)
}

func NewMetricsBBS(store storeadapter.StoreAdapter, timeProvider timeprovider.TimeProvider, logger lager.Logger) MetricsBBS {
	return NewBBS(store, timeProvider, logger)
}

func NewRouteEmitterBBS(store storeadapter.StoreAdapter, timeProvider timeprovider.TimeProvider, logger lager.Logger) RouteEmitterBBS {
	return NewBBS(store, timeProvider, logger)
}

func NewTPSBBS(store storeadapter.StoreAdapter, timeProvider timeprovider.TimeProvider, logger lager.Logger) TPSBBS {
	return NewBBS(store, timeProvider, logger)
}

func NewVeritasBBS(store storeadapter.StoreAdapter, timeProvider timeprovider.TimeProvider, logger lager.Logger) VeritasBBS {
	return NewBBS(store, timeProvider, logger)
}

func NewBBS(store storeadapter.StoreAdapter, timeProvider timeprovider.TimeProvider, logger lager.Logger) *BBS {
	services := services_bbs.New(store, logger.Session("services-bbs"))

	return &BBS{
		LockBBS:         lock_bbs.New(store, logger.Session("lock-bbs")),
		LRPBBS:          lrp_bbs.New(store, timeProvider, cb.NewCellClient(), services, logger.Session("lrp-bbs")),
		StartAuctionBBS: start_auction_bbs.New(store, timeProvider, logger.Session("lrp-start-auction-bbs")),
		StopAuctionBBS:  stop_auction_bbs.New(store, timeProvider, logger.Session("lrp-stop-auction-bbs")),
		ServicesBBS:     services,
		TaskBBS:         task_bbs.New(store, timeProvider, cb.NewTaskClient(), services, logger.Session("task-bbs")),
	}
}

type BBS struct {
	*lock_bbs.LockBBS
	*lrp_bbs.LRPBBS
	*start_auction_bbs.StartAuctionBBS
	*stop_auction_bbs.StopAuctionBBS
	*services_bbs.ServicesBBS
	*task_bbs.TaskBBS
}
