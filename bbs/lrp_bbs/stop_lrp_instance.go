package lrp_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/workpool"
)

const workerPoolSize = 20

func (bbs *LRPBBS) RequestStopLRPInstance(lrp models.ActualLRP) error {
	return shared.RetryIndefinitelyOnStoreTimeout(func() error {
		cell, err := bbs.services.CellById(lrp.CellID)
		if err != nil {
			return err
		}

		err = bbs.cellClient.StopLRPInstance(cell.RepAddress, lrp)
		if err != nil {
			return err
		}

		return nil
	})
}

func (bbs *LRPBBS) RequestStopLRPInstances(lrps []models.ActualLRP) error {
	pool := workpool.NewWorkPool(workerPoolSize)

	errs := make(chan error, len(lrps))
	for _, lrp := range lrps {
		lrp := lrp
		pool.Submit(func() {
			errs <- bbs.RequestStopLRPInstance(lrp)
		})
	}

	pool.Stop()

	for i := 0; i < len(lrps); i++ {
		err := <-errs
		if err != nil {
			return err
		}
	}

	return nil
}
