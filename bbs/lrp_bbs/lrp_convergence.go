package lrp_bbs

import (
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/delta_force/delta_force"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/prune"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/metric"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

const (
	convergeLRPRunsCounter = metric.Counter("ConvergenceLRPRuns")
	convergeLRPDuration    = metric.Duration("ConvergenceLRPDuration")

	lrpsDeletedCounter = metric.Counter("ConvergenceLRPsDeleted")
	lrpsKickedCounter  = metric.Counter("ConvergenceLRPsKicked")
	lrpsStoppedCounter = metric.Counter("ConvergenceLRPsStopped")
)

type compareAndSwappableDesiredLRP struct {
	OldIndex      uint64
	NewDesiredLRP models.DesiredLRP
}

func (bbs *LRPBBS) ConvergeLRPs() {
	convergeLRPRunsCounter.Increment()

	convergeStart := time.Now()

	// make sure to get funcy here otherwise the time will be precomputed
	defer func() {
		convergeLRPDuration.Send(time.Since(convergeStart))
	}()

	actualsByProcessGuid, err := bbs.pruneActualsWithMissingCells()
	if err != nil {
		bbs.logger.Error("failed-to-fetch-and-prune-actual-lrps", err)
		return
	}

	desiredLRPRoot, err := bbs.store.ListRecursively(shared.DesiredLRPSchemaRoot)
	if err != nil && err != storeadapter.ErrorKeyNotFound {
		bbs.logger.Error("failed-to-fetch-desired-lrps", err)
		return
	}

	var desiredLRPsToCAS []compareAndSwappableDesiredLRP
	var keysToDelete []string
	knownDesiredProcessGuids := map[string]bool{}

	for _, node := range desiredLRPRoot.ChildNodes {
		desiredLRP, err := models.NewDesiredLRPFromJSON(node.Value)

		if err != nil {
			bbs.logger.Info("pruning-invalid-desired-lrp-json", lager.Data{
				"error":   err.Error(),
				"payload": node.Value,
			})
			keysToDelete = append(keysToDelete, node.Key)
			continue
		}

		knownDesiredProcessGuids[desiredLRP.ProcessGuid] = true
		actualLRPsForDesired := actualsByProcessGuid[desiredLRP.ProcessGuid]

		if bbs.needsReconciliation(desiredLRP, actualLRPsForDesired) {
			desiredLRPsToCAS = append(desiredLRPsToCAS, compareAndSwappableDesiredLRP{
				OldIndex:      node.Index,
				NewDesiredLRP: desiredLRP,
			})
		}
	}

	stopLRPInstances := bbs.instancesToStop(knownDesiredProcessGuids, actualsByProcessGuid)

	lrpsDeletedCounter.Add(uint64(len(keysToDelete)))
	bbs.store.Delete(keysToDelete...)

	lrpsKickedCounter.Add(uint64(len(desiredLRPsToCAS)))
	bbs.batchCompareAndSwapDesiredLRPs(desiredLRPsToCAS)

	lrpsStoppedCounter.Add(uint64(len(stopLRPInstances)))
	err = bbs.RequestStopLRPInstances(stopLRPInstances)
	if err != nil {
		bbs.logger.Error("failed-to-request-stops", err)
	}
}

func (bbs *LRPBBS) instancesToStop(knownDesiredProcessGuids map[string]bool, actualsByProcessGuid map[string][]models.ActualLRP) []models.StopLRPInstance {
	var stopLRPInstances []models.StopLRPInstance

	for processGuid, actuals := range actualsByProcessGuid {
		if !knownDesiredProcessGuids[processGuid] {
			for _, actual := range actuals {
				bbs.logger.Info("detected-undesired-process", lager.Data{
					"process-guid":  processGuid,
					"instance-guid": actual.InstanceGuid,
					"index":         actual.Index,
				})

				stopLRPInstances = append(stopLRPInstances, models.StopLRPInstance{
					ProcessGuid:  processGuid,
					InstanceGuid: actual.InstanceGuid,
					Index:        actual.Index,
				})
			}
		}
	}

	return stopLRPInstances
}

func (bbs *LRPBBS) needsReconciliation(desiredLRP models.DesiredLRP, actualLRPsForDesired []models.ActualLRP) bool {
	var actuals delta_force.ActualInstances
	for _, actualLRP := range actualLRPsForDesired {
		actuals = append(actuals, delta_force.ActualInstance{
			Index: actualLRP.Index,
			Guid:  actualLRP.InstanceGuid,
		})
	}
	result := delta_force.Reconcile(desiredLRP.Instances, actuals)

	if len(result.IndicesToStart) > 0 {
		bbs.logger.Info("detected-missing-instance", lager.Data{
			"process-guid":      desiredLRP.ProcessGuid,
			"desired-instances": desiredLRP.Instances,
			"missing-indices":   result.IndicesToStart,
		})
	}

	if len(result.GuidsToStop) > 0 {
		bbs.logger.Info("detected-extra-instance", lager.Data{
			"process-guid":      desiredLRP.ProcessGuid,
			"desired-instances": desiredLRP.Instances,
			"extra-guids":       result.GuidsToStop,
		})
	}

	if len(result.IndicesToStopAllButOne) > 0 {
		bbs.logger.Info("detected-duplicate-instance", lager.Data{
			"process-guid":       desiredLRP.ProcessGuid,
			"desired-instances":  desiredLRP.Instances,
			"duplicated-indices": result.IndicesToStopAllButOne,
		})
	}

	return !result.Empty()
}

func (bbs *LRPBBS) pruneActualsWithMissingCells() (map[string][]models.ActualLRP, error) {
	actualsByProcessGuid := map[string][]models.ActualLRP{}

	cellRoot, err := bbs.store.ListRecursively(shared.CellSchemaRoot)
	if err == storeadapter.ErrorKeyNotFound {
		cellRoot = storeadapter.StoreNode{}
	} else if err != nil {
		bbs.logger.Error("failed-to-get-cells", err)
		return nil, err
	}

	err = prune.Prune(bbs.store, shared.ActualLRPSchemaRoot, func(node storeadapter.StoreNode) (shouldKeep bool) {
		actual, err := models.NewActualLRPFromJSON(node.Value)
		if err != nil {
			return false
		}

		if _, ok := cellRoot.Lookup(actual.CellID); !ok {
			bbs.logger.Info("detected-actual-with-missing-cell", lager.Data{
				"actual":      actual,
				"cell-id": actual.CellID,
			})
			return false
		}

		actualsByProcessGuid[actual.ProcessGuid] = append(actualsByProcessGuid[actual.ProcessGuid], actual)
		return true
	})

	if err != nil {
		bbs.logger.Error("failed-to-prune-actual-lrps", err)
		return nil, err
	}

	return actualsByProcessGuid, nil
}

func (bbs *LRPBBS) batchCompareAndSwapDesiredLRPs(desiredLRPsToCAS []compareAndSwappableDesiredLRP) {
	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(len(desiredLRPsToCAS))
	for _, desiredLRPToCAS := range desiredLRPsToCAS {
		desiredLRP := desiredLRPToCAS.NewDesiredLRP
		newStoreNode := storeadapter.StoreNode{
			Key:   shared.DesiredLRPSchemaPath(desiredLRP),
			Value: desiredLRP.ToJSON(),
		}

		go func(desiredLRPToCAS compareAndSwappableDesiredLRP, newStoreNode storeadapter.StoreNode) {
			err := bbs.store.CompareAndSwapByIndex(desiredLRPToCAS.OldIndex, newStoreNode)
			if err != nil {
				bbs.logger.Error("failed-to-compare-and-swap", err)
			}

			waitGroup.Done()
		}(desiredLRPToCAS, newStoreNode)
	}

	waitGroup.Wait()
}
