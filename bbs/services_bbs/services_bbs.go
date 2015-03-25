package services_bbs

import (
	"github.com/cloudfoundry-incubator/consuladapter"
	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"
)

type ServicesBBS struct {
	consul consuladapter.Adapter
	logger lager.Logger
	clock  clock.Clock
}

func New(consul consuladapter.Adapter, clock clock.Clock, logger lager.Logger) *ServicesBBS {
	return &ServicesBBS{
		consul: consul,
		logger: logger,
		clock:  clock,
	}
}
