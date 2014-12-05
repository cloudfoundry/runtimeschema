package lrp_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

func (bbs *LRPBBS) RequestStopLRPInstance(stopInstance models.StopLRPInstance) error {
	return bbs.RequestStopLRPInstances([]models.StopLRPInstance{stopInstance})
}

func (bbs *LRPBBS) RequestStopLRPInstances(stopInstances []models.StopLRPInstance) error {
	for _, stopInstance := range stopInstances {
		err := shared.RetryIndefinitelyOnStoreTimeout(func() error {
			return bbs.stop(stopInstance)
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (bbs *LRPBBS) stop(stopInstance models.StopLRPInstance) error {
	lrps, err := bbs.ActualLRPsByProcessGuidAndIndex(stopInstance.ProcessGuid, stopInstance.Index)
	if err != nil {
		return err
	}

	for _, lrp := range lrps {
		if lrp.InstanceGuid != stopInstance.InstanceGuid {
			continue
		}

		cell, err := bbs.services.CellById(lrp.CellID)
		if err != nil {
			return err
		}

		err = bbs.cellClient.StopLRPInstance(cell.RepAddress, stopInstance)
		if err != nil {
			return err
		}

		break
	}

	return nil
}
