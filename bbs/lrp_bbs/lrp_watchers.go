package lrp_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
)

func (self *LongRunningProcessBBS) WatchForDesiredLRPChanges() (<-chan models.DesiredLRPChange, chan<- bool, <-chan error) {
	return watchForDesiredLRPChanges(self.store)
}

//XXXX
func (self *LongRunningProcessBBS) WatchForActualLongRunningProcesses() (<-chan models.LRP, chan<- bool, <-chan error) {
	return watchForActualLRPs(self.store)
}

func watchForDesiredLRPChanges(store storeadapter.StoreAdapter) (<-chan models.DesiredLRPChange, chan<- bool, <-chan error) {
	changes := make(chan models.DesiredLRPChange)
	stopOuter := make(chan bool)
	errsOuter := make(chan error)

	events, stopInner, errsInner := store.Watch(shared.DesiredLRPSchemaRoot)

	go func() {
		defer close(changes)
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

				var before *models.DesiredLRP
				var after *models.DesiredLRP

				if event.Node != nil {
					aft, err := models.NewDesiredLRPFromJSON(event.Node.Value)
					if err != nil {
						continue
					}

					after = &aft
				}

				if event.PrevNode != nil {
					bef, err := models.NewDesiredLRPFromJSON(event.PrevNode.Value)
					if err != nil {
						continue
					}

					before = &bef
				}

				changes <- models.DesiredLRPChange{
					Before: before,
					After:  after,
				}

			case err, ok := <-errsInner:
				if ok {
					errsOuter <- err
				}
				return
			}
		}
	}()

	return changes, stopOuter, errsOuter
}

func watchForActualLRPs(store storeadapter.StoreAdapter) (<-chan models.LRP, chan<- bool, <-chan error) {
	lrps := make(chan models.LRP)
	stopOuter := make(chan bool)
	errsOuter := make(chan error)

	events, stopInner, errsInner := store.Watch(shared.ActualLRPSchemaRoot)

	go func() {
		defer close(lrps)
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

				lrp, err := models.NewLRPFromJSON(event.Node.Value)
				if err != nil {
					continue
				}

				lrps <- lrp

			case err, ok := <-errsInner:
				if ok {
					errsOuter <- err
				}
				return
			}
		}
	}()

	return lrps, stopOuter, errsOuter
}
