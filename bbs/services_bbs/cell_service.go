package services_bbs

import (
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/heartbeater"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/ifrit"
)

const CELL_HEARTBEAT_INTERVAL = 5 * time.Second

func (bbs *ServicesBBS) NewCellHeartbeat(cellPresence models.CellPresence, interval time.Duration) ifrit.Runner {
	payload, err := models.ToJSON(cellPresence)
	if err != nil {
		panic(err)
	}

	return heartbeater.New(bbs.store, bbs.clock, shared.CellSchemaPath(cellPresence.CellID), string(payload), interval, bbs.logger)
}

func (bbs *ServicesBBS) CellById(cellId string) (models.CellPresence, error) {
	cellPresence := models.CellPresence{}

	node, err := bbs.store.Get(shared.CellSchemaPath(cellId))
	if err != nil {
		return cellPresence, shared.ConvertStoreError(err)
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
	} else if err != nil {
		return nil, shared.ConvertStoreError(err)
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

func (bbs *ServicesBBS) WaitForCellEvent() (CellEvent, error) {
	events, stop, errChan := bbs.store.Watch(shared.CellSchemaRoot)
	defer close(stop)

	for {
		select {
		case err := <-errChan:
			return nil, shared.ConvertStoreError(err)

		case event := <-events:
			switch {
			case event.Node != nil && event.PrevNode == nil:
				e := CellAppearedEvent{}

				err := models.FromJSON(event.Node.Value, &e.Presence)
				if err != nil {
					return nil, err
				}

				bbs.logger.Info("cell-appeared", lager.Data{"cell-id": e.CellID()})
				return e, nil

			case event.Node == nil && event.PrevNode != nil:
				e := CellDisappearedEvent{}

				err := models.FromJSON(event.PrevNode.Value, &e.Presence)
				if err != nil {
					return nil, err
				}

				bbs.logger.Info("cell-disappeared", lager.Data{"cell-id": e.CellID()})
				return e, nil
			}
		}
	}

	panic("unreachable")
}

type CellEvent interface {
	EventType() CellEventType
	CellID() string
}

type CellEventType int

const (
	CellEventTypeInvalid CellEventType = iota
	CellAppeared
	CellDisappeared
)

type CellAppearedEvent struct {
	Presence models.CellPresence
}

func (CellAppearedEvent) EventType() CellEventType {
	return CellAppeared
}

func (e CellAppearedEvent) CellID() string {
	return e.Presence.CellID
}

type CellDisappearedEvent struct {
	Presence models.CellPresence
}

func (CellDisappearedEvent) EventType() CellEventType {
	return CellDisappeared
}

func (e CellDisappearedEvent) CellID() string {
	return e.Presence.CellID
}
