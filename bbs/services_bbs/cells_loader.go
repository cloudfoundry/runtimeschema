package services_bbs

import (
	"sync"

	"github.com/cloudfoundry-incubator/consuladapter"
	"github.com/pivotal-golang/clock"

	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/pivotal-golang/lager"
)

type CellsLoader struct {
	lock     sync.Mutex
	services *ServicesBBS
	cellSet  models.CellSet
	err      error
	logger   lager.Logger
}

func (loader *CellsLoader) Cells() (models.CellSet, error) {
	loader.lock.Lock()
	if loader.cellSet == nil {
		cells, err := loader.services.Cells()
		loader.err = err

		loader.cellSet = models.CellSet{}
		for _, cell := range cells {
			loader.cellSet.Add(cell)
		}
	}
	loader.lock.Unlock()

	return loader.cellSet, loader.err
}

func NewCellsLoader(logger lager.Logger, consul *consuladapter.Adapter, clock clock.Clock) *CellsLoader {
	return &CellsLoader{
		logger:   logger,
		services: New(consul, clock, logger),
	}
}
