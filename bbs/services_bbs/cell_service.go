package services_bbs

import (
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/heartbeater"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	"github.com/tedsuo/ifrit"
)

func (bbs *ServicesBBS) NewCellHeartbeat(cellPresence models.CellPresence, interval time.Duration) ifrit.Runner {
	payload, err := models.ToJSON(cellPresence)
	if err != nil {
		panic(err)
	}
	return heartbeater.New(bbs.store, shared.CellSchemaPath(cellPresence.CellID), string(payload), interval, bbs.logger)
}

func (bbs *ServicesBBS) CellById(cellId string) (models.CellPresence, error) {
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

func (bbs *ServicesBBS) Cells() ([]models.CellPresence, error) {
	node, err := bbs.store.ListRecursively(shared.CellSchemaRoot)
	if err == storeadapter.ErrorKeyNotFound {
		return []models.CellPresence{}, nil
	}

	if err != nil {
		return nil, err
	}

	cellPresences := []models.CellPresence{}
	for _, node := range node.ChildNodes {
		cellPresence := models.CellPresence{}
		err := models.FromJSON(node.Value, &cellPresence)
		if err != nil {
			bbs.logger.Error("failed-to-unmarshal-cells-json", err)
			continue
		}

		cellPresences = append(cellPresences, cellPresence)
	}

	return cellPresences, nil
}
