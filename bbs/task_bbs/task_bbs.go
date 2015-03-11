package task_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/repositories"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/services_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/cb"
	"github.com/cloudfoundry/storeadapter"
	"github.com/go-gorp/gorp"
	"github.com/pivotal-golang/clock"
)

type TaskBBS struct {
	store            storeadapter.StoreAdapter
	clock            clock.Clock
	taskClient       cb.TaskClient
	auctioneerClient cb.AuctioneerClient
	cellClient       cb.CellClient
	services         *services_bbs.ServicesBBS
	dbmap            *gorp.DbMap
	repository       repositories.TaskRepository
}

func New(
	store storeadapter.StoreAdapter,
	clock clock.Clock,
	taskClient cb.TaskClient,
	auctioneerClient cb.AuctioneerClient,
	cellClient cb.CellClient,
	services *services_bbs.ServicesBBS,
	dbmap *gorp.DbMap,
	repository repositories.TaskRepository,
) *TaskBBS {
	return &TaskBBS{
		store:            store,
		clock:            clock,
		taskClient:       taskClient,
		auctioneerClient: auctioneerClient,
		cellClient:       cellClient,
		services:         services,
		dbmap:            dbmap,
		repository:       repository,
	}
}
