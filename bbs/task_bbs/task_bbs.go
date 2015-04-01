package task_bbs

import (
	"github.com/cloudfoundry-incubator/consuladapter"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/services_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/cb"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/clock"
)

type TaskBBS struct {
	store                  storeadapter.StoreAdapter
	consulAdapter          consuladapter.Adapter
	clock                  clock.Clock
	taskClient             cb.TaskClient
	auctioneerClient       cb.AuctioneerClient
	cellClient             cb.CellClient
	services               *services_bbs.ServicesBBS
	receptorTaskHandlerURL string
}

func New(
	store storeadapter.StoreAdapter,
	consulAdapter consuladapter.Adapter,
	clock clock.Clock,
	taskClient cb.TaskClient,
	auctioneerClient cb.AuctioneerClient,
	cellClient cb.CellClient,
	services *services_bbs.ServicesBBS,
	receptorTaskHandlerURL string,
) *TaskBBS {
	return &TaskBBS{
		store:                  store,
		consulAdapter:          consulAdapter,
		clock:                  clock,
		taskClient:             taskClient,
		auctioneerClient:       auctioneerClient,
		cellClient:             cellClient,
		services:               services,
		receptorTaskHandlerURL: receptorTaskHandlerURL,
	}
}
