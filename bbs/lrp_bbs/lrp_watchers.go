package lrp_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/pivotal-golang/lager"
)

func (bbs *LRPBBS) WatchForDesiredLRPChanges(logger lager.Logger, created func(models.DesiredLRP), changed func(models.DesiredLRPChange), deleted func(models.DesiredLRP)) (chan<- bool, <-chan error) {
	logger = logger.Session("watching-for-desired-lrp-changes")

	events, stop, err := bbs.store.Watch(shared.DesiredLRPSchemaRoot)

	go func() {
		for event := range events {
			switch {
			case event.Node != nil && event.PrevNode == nil:
				logger.Debug("received-create")

				var desiredLRP models.DesiredLRP
				err := models.FromJSON(event.Node.Value, &desiredLRP)
				if err != nil {
					logger.Error("failed-to-unmarshal-desired-lrp", err, lager.Data{"value": event.Node.Value})
					continue
				}

				logger.Debug("sending-create", lager.Data{"desired-lrp": desiredLRP})
				created(desiredLRP)

			case event.Node != nil && event.PrevNode != nil: // update
				logger.Debug("received-update")

				var before models.DesiredLRP
				err := models.FromJSON(event.PrevNode.Value, &before)
				if err != nil {
					logger.Error("failed-to-unmarshal-desired-lrp", err, lager.Data{"value": event.PrevNode.Value})
					continue
				}

				var after models.DesiredLRP
				err = models.FromJSON(event.Node.Value, &after)
				if err != nil {
					logger.Error("failed-to-unmarshal-desired-lrp", err, lager.Data{"value": event.Node.Value})
					continue
				}

				logger.Debug("sending-update", lager.Data{"before": before, "after": after})
				changed(models.DesiredLRPChange{Before: before, After: after})

			case event.Node == nil && event.PrevNode != nil: // delete
				logger.Debug("received-delete")

				var desiredLRP models.DesiredLRP
				err := models.FromJSON(event.PrevNode.Value, &desiredLRP)
				if err != nil {
					logger.Error("failed-to-unmarshal-desired-lrp", err, lager.Data{"value": event.PrevNode.Value})
					continue
				}

				logger.Debug("sending-delete", lager.Data{"desired-lrp": desiredLRP})
				deleted(desiredLRP)

			default:
				logger.Debug("received-event-with-both-nodes-nil")
			}
		}
	}()

	return stop, err
}

func (bbs *LRPBBS) WatchForActualLRPChanges(logger lager.Logger, created func(models.ActualLRP), changed func(models.ActualLRPChange), deleted func(models.ActualLRP)) (chan<- bool, <-chan error) {
	logger = logger.Session("watching-for-actual-lrp-changes")

	events, stop, err := bbs.store.Watch(shared.ActualLRPSchemaRoot)

	go func() {
		for event := range events {
			logger.Debug("event-node", lager.Data{"event": event})
			switch {
			case event.Node != nil && event.PrevNode == nil:
				logger.Debug("received-create")

				var actualLRP models.ActualLRP
				err := models.FromJSON(event.Node.Value, &actualLRP)
				if err != nil {
					logger.Error("failed-to-unmarshal-actual-lrp-on-create", err, lager.Data{"key": event.Node.Key, "value": event.Node.Value})
					continue
				}

				logger.Debug("sending-create", lager.Data{"actual-lrp": actualLRP})
				created(actualLRP)

			case event.Node != nil && event.PrevNode != nil:
				logger.Debug("received-change")

				var before models.ActualLRP
				err := models.FromJSON(event.PrevNode.Value, &before)
				if err != nil {
					logger.Error("failed-to-unmarshal-prev-actual-lrp-on-change", err, lager.Data{"key": event.PrevNode.Key, "value": event.PrevNode.Value})
					continue
				}

				var after models.ActualLRP
				err = models.FromJSON(event.Node.Value, &after)
				if err != nil {
					logger.Error("failed-to-unmarshal-actual-lrp-on-change", err, lager.Data{"key": event.Node.Key, "value": event.Node.Value})
					continue
				}

				logger.Debug("sending-change", lager.Data{"before": before, "after": after})
				changed(models.ActualLRPChange{Before: before, After: after})

			case event.PrevNode != nil && event.Node == nil:
				logger.Debug("received-delete")

				var actualLRP models.ActualLRP
				if event.PrevNode.Dir {
					continue
				}
				err := models.FromJSON(event.PrevNode.Value, &actualLRP)
				if err != nil {
					logger.Error("failed-to-unmarshal-prev-actual-lrp-on-delete", err, lager.Data{"key": event.PrevNode.Key, "value": event.PrevNode.Value})
				} else {
					logger.Debug("sending-delete", lager.Data{"actual-lrp": actualLRP})
					deleted(actualLRP)
				}

			default:
				logger.Debug("received-event-with-both-nodes-nil")
			}
		}
	}()

	return stop, err
}
