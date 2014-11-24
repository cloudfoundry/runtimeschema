package lrp_bbs

import (
	"fmt"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
)

func (bbs *LRPBBS) RequestStopLRPInstance(stopInstance models.StopLRPInstance) error {
	return bbs.RequestStopLRPInstances([]models.StopLRPInstance{stopInstance})
}

func (bbs *LRPBBS) RequestStopLRPInstances(stopInstances []models.StopLRPInstance) error {
	return shared.RetryIndefinitelyOnStoreTimeout(func() error {
		var nodes []storeadapter.StoreNode

		for _, stopInstance := range stopInstances {
			value, err := models.ToJSON(stopInstance)
			if err != nil {
				return err
			}
			nodes = append(nodes, storeadapter.StoreNode{
				Key:   shared.StopLRPInstanceSchemaPath(stopInstance),
				Value: value,
				TTL:   60,
			})
		}
		return bbs.store.SetMulti(nodes)
	})
}

func (bbs *LRPBBS) StopLRPInstances() ([]models.StopLRPInstance, error) {
	stopInstances := []models.StopLRPInstance{}

	node, err := bbs.store.ListRecursively(shared.StopLRPInstanceSchemaRoot)
	if err == storeadapter.ErrorKeyNotFound {
		return stopInstances, nil
	}

	if err != nil {
		return stopInstances, err
	}

	for _, node := range node.ChildNodes {
		var stopInstance models.StopLRPInstance
		err = models.FromJSON(node.Value, &stopInstance)
		if err != nil {
			return stopInstances, fmt.Errorf("cannot parse stop instance JSON for key %s: %s", node.Key, err.Error())
		} else {
			stopInstances = append(stopInstances, stopInstance)
		}
	}

	return stopInstances, nil
}

func (bbs *LRPBBS) ResolveStopLRPInstance(stopInstance models.StopLRPInstance) error {
	return shared.RetryIndefinitelyOnStoreTimeout(func() error {
		err := bbs.store.Delete(shared.StopLRPInstanceSchemaPath(stopInstance))
		if err == storeadapter.ErrorKeyNotFound {
			err = nil
		}
		return err
	})
}

func (bbs *LRPBBS) WatchForStopLRPInstance() (<-chan models.StopLRPInstance, chan<- bool, <-chan error) {
	stopInstances := make(chan models.StopLRPInstance)

	filter := func(event storeadapter.WatchEvent) (models.StopLRPInstance, bool) {
		switch event.Type {
		case storeadapter.CreateEvent, storeadapter.UpdateEvent:
			var stopInstance models.StopLRPInstance
			err := models.FromJSON(event.Node.Value, &stopInstance)
			if err != nil {
				return models.StopLRPInstance{}, false
			}
			return stopInstance, true
		}
		return models.StopLRPInstance{}, false
	}

	stop, errs := shared.WatchWithFilter(bbs.store, shared.StopLRPInstanceSchemaRoot, stopInstances, filter)

	return stopInstances, stop, errs
}
