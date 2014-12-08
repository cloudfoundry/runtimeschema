package lrp_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

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
	for _, lrp := range lrps {
		err := bbs.RequestStopLRPInstance(lrp)
		if err != nil {
			return err
		}
	}

	return nil
}
