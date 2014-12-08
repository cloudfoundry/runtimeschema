package task_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/services_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/cb"
	"github.com/cloudfoundry/gunk/timeprovider"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

type TaskBBS struct {
	store        storeadapter.StoreAdapter
	timeProvider timeprovider.TimeProvider
	logger       lager.Logger
	taskClient   cb.TaskClient
	services     *services_bbs.ServicesBBS
}

func New(
	store storeadapter.StoreAdapter,
	timeProvider timeprovider.TimeProvider,
	taskClient cb.TaskClient,
	services *services_bbs.ServicesBBS,
	logger lager.Logger,
) *TaskBBS {
	return &TaskBBS{
		store:        store,
		timeProvider: timeProvider,
		taskClient:   taskClient,
		services:     services,
		logger:       logger,
	}
}
