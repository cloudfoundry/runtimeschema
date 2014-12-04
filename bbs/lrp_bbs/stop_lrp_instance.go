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
		lrps, err := bbs.ActualLRPsByProcessGuidAndIndex(stopInstance.ProcessGuid, stopInstance.Index)
		if err != nil {
			return err
		}

		for _, lrp := range lrps {
			if lrp.InstanceGuid == stopInstance.InstanceGuid {
				cell, cellErr := bbs.cellById(lrp.CellID)
				if cellErr != nil {
					return cellErr
				}

				bbs.cellClient.StopLRPInstance(cell.RepAddress, stopInstance)
			}
		}
	}

	return nil
}

func (bbs *LRPBBS) cellById(cellId string) (models.CellPresence, error) {
	cellPresence := models.CellPresence{}

	node, err := bbs.store.Get(shared.CellSchemaPath(cellId))
	if err != nil {
		return cellPresence, err
	}

	err = models.FromJSON(node.Value, &cellPresence)
	if err != nil {
		return cellPresence, err
	}

	return cellPresence, nil
}
