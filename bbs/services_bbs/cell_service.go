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
	return heartbeater.New(bbs.store, shared.CellSchemaPath(cellPresence.CellID), string(cellPresence.ToJSON()), interval, bbs.logger)
}

func (bbs *ServicesBBS) Cells() ([]models.CellPresence, error) {
	node, err := bbs.store.ListRecursively(shared.CellSchemaRoot)
	if err == storeadapter.ErrorKeyNotFound {
		return []models.CellPresence{}, nil
	}

	if err != nil {
		return nil, err
	}

	var cellPresences []models.CellPresence
	for _, node := range node.ChildNodes {
		cellPresence, err := models.NewCellPresenceFromJSON(node.Value)
		if err != nil {
			bbs.logger.Error("failed-to-unmarshal-cells-json", err)
			continue
		}

		cellPresences = append(cellPresences, cellPresence)
	}

	return cellPresences, nil
}
