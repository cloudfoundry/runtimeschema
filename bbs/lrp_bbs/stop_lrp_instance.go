package lrp_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

const workerPoolSize = 20

func (bbs *LRPBBS) RequestStopLRPInstance(
	key models.ActualLRPKey,
	containerKey models.ActualLRPContainerKey,
) error {
	return shared.RetryIndefinitelyOnStoreTimeout(func() error {
		cell, err := bbs.services.CellById(containerKey.CellID)
		if err != nil {
			return err
		}

		err = bbs.cellClient.StopLRPInstance(cell.RepAddress, key, containerKey)
		if err != nil {
			return err
		}

		return nil
	})
}
