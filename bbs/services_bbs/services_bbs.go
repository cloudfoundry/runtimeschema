package services_bbs

import (
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"
)

type ServicesBBS struct {
	store  storeadapter.StoreAdapter
	logger lager.Logger
	clock  clock.Clock
}

func New(store storeadapter.StoreAdapter, clock clock.Clock, logger lager.Logger) *ServicesBBS {
	return &ServicesBBS{
		store:  store,
		logger: logger,
		clock:  clock,
	}
}
