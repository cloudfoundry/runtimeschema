package services_bbs

import (
	"path"
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/maintainer"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/ifrit"
)

const CellPresenceTTL = 10 * time.Second

func (bbs *ServicesBBS) NewCellPresence(cellPresence models.CellPresence, retryInterval time.Duration) ifrit.Runner {
	payload, err := models.ToJSON(cellPresence)
	if err != nil {
		panic(err)
	}

	return maintainer.NewPresence(bbs.consul, shared.CellSchemaPath(cellPresence.CellID), payload, bbs.clock, retryInterval, bbs.logger)
}

func (bbs *ServicesBBS) CellById(cellId string) (models.CellPresence, error) {
	cellPresence := models.CellPresence{}

	value, err := bbs.consul.GetAcquiredValue(shared.CellSchemaPath(cellId))
	if err != nil {
		return cellPresence, shared.ConvertConsulError(err)
	}

	err = models.FromJSON(value, &cellPresence)
	if err != nil {
		return cellPresence, err
	}

	return cellPresence, nil
}

func (bbs *ServicesBBS) Cells() ([]models.CellPresence, error) {
	cells, err := bbs.consul.ListAcquiredValues(shared.CellSchemaRoot)
	if err != nil {
		err = shared.ConvertConsulError(err)
		if err != bbserrors.ErrStoreResourceNotFound {
			return nil, err
		}
	}

	cellPresences := []models.CellPresence{}
	for _, cell := range cells {
		cellPresence := models.CellPresence{}
		err := models.FromJSON(cell, &cellPresence)
		if err != nil {
			bbs.logger.Error("failed-to-unmarshal-cells-json", err)
			continue
		}

		cellPresences = append(cellPresences, cellPresence)
	}

	return cellPresences, nil
}

func (bbs *ServicesBBS) CellEvents() <-chan CellEvent {
	logger := bbs.logger.Session("cell-events")

	events := make(chan CellEvent)
	go func() {
		disappeared := bbs.consul.WatchForDisappearancesUnder(logger, shared.CellSchemaRoot)

		for {
			select {
			case keys, ok := <-disappeared:
				if !ok {
					return
				}

				cellIDs := make([]string, len(keys))
				for i, key := range keys {
					cellIDs[i] = path.Base(key)
				}
				e := CellDisappearedEvent{cellIDs}

				logger.Info("cell-disappeared", lager.Data{"cell-ids": e.CellIDs()})
				events <- e
			}
		}
	}()

	return events
}

type CellEvent interface {
	EventType() CellEventType
	CellIDs() []string
}

type CellEventType int

const (
	CellEventTypeInvalid CellEventType = iota
	CellDisappeared
)

type CellDisappearedEvent struct {
	IDs []string
}

func (CellDisappearedEvent) EventType() CellEventType {
	return CellDisappeared
}

func (e CellDisappearedEvent) CellIDs() []string {
	return e.IDs
}
