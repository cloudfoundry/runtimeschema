package lrp_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/pivotal-golang/lager"
)

func (bbs *LRPBBS) WatchForDesiredLRPChanges(logger lager.Logger) (<-chan models.DesiredLRP, <-chan models.DesiredLRP, <-chan error) {
	logger = logger.Session("watching-for-desired-lrp-changes")

	createsAndUpdates := make(chan models.DesiredLRP)
	deletes := make(chan models.DesiredLRP)

	events, _, err := bbs.store.Watch(shared.DesiredLRPSchemaRoot)

	go func() {
		for event := range events {
			switch {
			case event.Node != nil:
				logger.Debug("received-create-or-update")

				var desiredLRP models.DesiredLRP
				err := models.FromJSON(event.Node.Value, &desiredLRP)
				if err != nil {
					logger.Error("failed-to-unmarshal-desired-lrp", err, lager.Data{"value": event.Node.Value})
				} else {
					logger.Debug("sending-create-or-update", lager.Data{"desired-lrp": desiredLRP})
					createsAndUpdates <- desiredLRP
				}

			case event.PrevNode != nil:
				logger.Debug("received-delete")

				var desiredLRP models.DesiredLRP
				err := models.FromJSON(event.PrevNode.Value, &desiredLRP)
				if err != nil {
					logger.Error("failed-to-unmarshal-desired-lrp", err, lager.Data{"value": event.PrevNode.Value})
				} else {
					logger.Debug("sending-delete", lager.Data{"desired-lrp": desiredLRP})
					deletes <- desiredLRP
				}

			default:
				logger.Debug("received-event-with-both-nodes-nil")
			}
		}
	}()

	return createsAndUpdates, deletes, err
}

func (bbs *LRPBBS) WatchForActualLRPChanges(logger lager.Logger) (<-chan models.ActualLRP, <-chan models.ActualLRP, <-chan error) {
	logger = logger.Session("watching-for-actual-lrp-changes")

	createsAndUpdates := make(chan models.ActualLRP)
	deletes := make(chan models.ActualLRP)

	events, _, err := bbs.store.Watch(shared.ActualLRPSchemaRoot)

	go func() {
		for event := range events {
			switch {
			case event.Node != nil:
				logger.Debug("received-create-or-update")

				var actualLRP models.ActualLRP
				err := models.FromJSON(event.Node.Value, &actualLRP)
				if err != nil {
					logger.Error("failed-to-unmarshal-actual-lrp", err, lager.Data{"value": event.Node.Value})
				} else {
					logger.Debug("sending-create-or-update", lager.Data{"actual-lrp": actualLRP})
					createsAndUpdates <- actualLRP
				}

			case event.PrevNode != nil:
				logger.Debug("received-delete")

				var actualLRP models.ActualLRP
				err := models.FromJSON(event.PrevNode.Value, &actualLRP)
				if err != nil {
					logger.Error("failed-to-unmarshal-actual-lrp", err, lager.Data{"value": event.PrevNode.Value})
				} else {
					logger.Debug("sending-delete", lager.Data{"actual-lrp": actualLRP})
					deletes <- actualLRP
				}

			default:
				logger.Debug("received-event-with-both-nodes-nil")
			}
		}
	}()

	return createsAndUpdates, deletes, err
}
