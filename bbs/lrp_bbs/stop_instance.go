package lrp_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
)

func (bbs *LRPBBS) RequestStopLRPInstance(stopInstance models.StopLRPInstance) error {
	return shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return bbs.store.SetMulti([]storeadapter.StoreNode{
			{
				Key:   shared.StopLRPInstanceSchemaPath(stopInstance),
				Value: stopInstance.ToJSON(),
			},
		})
	})
}

func (bbs *LRPBBS) RemoveStopLRPInstance(stopInstance models.StopLRPInstance) error {
	return shared.RetryIndefinitelyOnStoreTimeout(func() error {
		err := bbs.store.Delete(shared.StopLRPInstanceSchemaPath(stopInstance))
		if err == storeadapter.ErrorKeyNotFound {
			return nil
		}
		return err
	})
}

func (bbs *LRPBBS) WatchForStopLRPInstance() (<-chan models.StopLRPInstance, chan<- bool, <-chan error) {
	stopInstances := make(chan models.StopLRPInstance)
	stopOuter := make(chan bool)
	errsOuter := make(chan error)

	events, stopInner, errsInner := bbs.store.Watch(shared.StopLRPInstanceSchemaRoot)

	go func() {
		defer close(stopInstances)
		defer close(errsOuter)

		for {
			select {
			case <-stopOuter:
				close(stopInner)
				return

			case event, ok := <-events:
				if !ok {
					return
				}

				switch event.Type {
				case storeadapter.CreateEvent, storeadapter.UpdateEvent:
					stopInstance, err := models.NewStopLRPInstanceFromJSON(event.Node.Value)
					if err != nil {
						continue
					}

					stopInstances <- stopInstance
				}

			case err, ok := <-errsInner:
				if ok {
					errsOuter <- err
				}
				return
			}
		}
	}()

	return stopInstances, stopOuter, errsOuter
}
