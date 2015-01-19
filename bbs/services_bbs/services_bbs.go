package services_bbs

import (
	"github.com/cloudfoundry/gunk/timeprovider"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

type ServicesBBS struct {
	store        storeadapter.StoreAdapter
	logger       lager.Logger
	timeProvider timeprovider.TimeProvider
}

func New(store storeadapter.StoreAdapter, timeProvider timeprovider.TimeProvider, logger lager.Logger) *ServicesBBS {
	return &ServicesBBS{
		store:        store,
		logger:       logger,
		timeProvider: timeProvider,
	}
}
