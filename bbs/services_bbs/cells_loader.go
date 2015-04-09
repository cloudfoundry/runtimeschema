package services_bbs

import (
	"sync"

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
		if err != nil {
			loader.err = err
		} else {
			loader.cellSet = models.CellSet{}
			for _, cell := range cells {
				loader.cellSet.Add(cell)
			}
		}
	}
	loader.lock.Unlock()

	return loader.cellSet, loader.err
}

func (bbs *ServicesBBS) NewCellsLoader() *CellsLoader {
	return &CellsLoader{
		logger:   bbs.logger,
		services: bbs,
	}
}
