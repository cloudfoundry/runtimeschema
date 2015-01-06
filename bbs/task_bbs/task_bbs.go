package task_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/services_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/cb"
	"github.com/cloudfoundry/gunk/timeprovider"
	"github.com/cloudfoundry/storeadapter"
)

type TaskBBS struct {
	store            storeadapter.StoreAdapter
	timeProvider     timeprovider.TimeProvider
	taskClient       cb.TaskClient
	auctioneerClient cb.AuctioneerClient
	services         *services_bbs.ServicesBBS
}

func New(
	store storeadapter.StoreAdapter,
	timeProvider timeprovider.TimeProvider,
	taskClient cb.TaskClient,
	auctioneerClient cb.AuctioneerClient,
	services *services_bbs.ServicesBBS,
) *TaskBBS {
	return &TaskBBS{
		store:            store,
		timeProvider:     timeProvider,
		taskClient:       taskClient,
		auctioneerClient: auctioneerClient,
		services:         services,
	}
}
