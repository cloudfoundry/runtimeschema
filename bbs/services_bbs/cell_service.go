package services_bbs

import (
	"errors"
	"path"
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/heartbeater"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/ifrit"
)

const CellPresenceTTL = 10 * time.Second

func (bbs *ServicesBBS) NewCellHeartbeat(cellPresence models.CellPresence, ttl, retryInterval time.Duration) ifrit.Runner {
	payload, err := models.ToJSON(cellPresence)
	if err != nil {
		panic(err)
	}

	return heartbeater.New(bbs.consul, shared.CellSchemaPath(cellPresence.CellID), payload, ttl, bbs.clock, retryInterval, bbs.logger)
}

func (bbs *ServicesBBS) CellById(cellId string) (models.CellPresence, error) {
	cellPresence := models.CellPresence{}

	value, err := bbs.consul.GetValue(shared.CellSchemaPath(cellId))
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
	cells, err := bbs.consul.ListPairsExtending(shared.CellSchemaRoot)
	if err != nil {
		return nil, shared.ConvertConsulError(err)
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

func (bbs *ServicesBBS) WaitForCellEvent() (CellEvent, error) {
	disappeared, stop, errChan := bbs.consul.WatchForDisappearancesUnder(shared.CellSchemaRoot)
	defer close(stop)

	for {
		select {
		case err, ok := <-errChan:
			if !ok {
				return nil, errors.New("wait-for-cell-event-err-chan-closed")
			}

			return nil, shared.ConvertConsulError(err)

		case keys, ok := <-disappeared:
			if !ok {
				return nil, errors.New("wait-for-cell-event-disappearance-chan-closed")
			}

			cellIDs := make([]string, len(keys))
			for i, key := range keys {
				cellIDs[i] = path.Base(key)
			}
			e := CellDisappearedEvent{cellIDs}

			bbs.logger.Info("cell-disappeared", lager.Data{"cell-ids": e.CellIDs()})
			return e, nil
		}
	}

	panic("unreachable")
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
