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

func (bbs *LRPBBS) ConvergeLRPs(pollingInterval time.Duration) {
	logger := bbs.logger.Session("converge-lrps")
	logger.Info("starting-convergence")
	defer logger.Info("finished-convergence")

	convergeLRPRunsCounter.Increment()

	convergeStart := bbs.timeProvider.Now()

	// make sure to get funcy here otherwise the time will be precomputed
	defer func() {
		convergeLRPDuration.Send(bbs.timeProvider.Now().Sub(convergeStart))
	}()

	actualsByProcessGuid, err := bbs.pruneActualsWithMissingCells(logger)
	if err != nil {
		logger.Error("failed-to-fetch-and-prune-actual-lrps", err)
		return
	}

	desiredLRPRoot, err := bbs.store.ListRecursively(shared.DesiredLRPSchemaRoot)
	if err != nil && err != storeadapter.ErrorKeyNotFound {
		logger.Error("failed-to-fetch-desired-lrps", err)
		return
	}

	var desiredLRPsToCAS []compareAndSwappableDesiredLRP
	var desiredLRPKeysToDelete []string
	desiredLRPsByProcessGuid := map[string]models.DesiredLRP{}

	for _, node := range desiredLRPRoot.ChildNodes {
		var desiredLRP models.DesiredLRP
		err := models.FromJSON(node.Value, &desiredLRP)

		if err != nil {
			logger.Info("pruning-invalid-desired-lrp-json", lager.Data{
				"error":   err.Error(),
				"payload": node.Value,
			})
			desiredLRPKeysToDelete = append(desiredLRPKeysToDelete, node.Key)
			continue
		}

		desiredLRPsByProcessGuid[desiredLRP.ProcessGuid] = desiredLRP
		actualLRPsForDesired := actualsByProcessGuid[desiredLRP.ProcessGuid]

		if bbs.needsReconciliation(desiredLRP, actualLRPsForDesired, logger) {
			desiredLRPsToCAS = append(desiredLRPsToCAS, compareAndSwappableDesiredLRP{
				OldIndex:      node.Index,
				NewDesiredLRP: desiredLRP,
			})
		}
	}

	stopLRPInstances := bbs.instancesToStop(desiredLRPsByProcessGuid, actualsByProcessGuid, logger)

	lrpsDeletedCounter.Add(uint64(len(desiredLRPKeysToDelete)))
	bbs.store.Delete(desiredLRPKeysToDelete...)

	lrpsKickedCounter.Add(uint64(len(desiredLRPsToCAS)))
	bbs.batchCompareAndSwapDesiredLRPs(desiredLRPsToCAS, logger)

	bbs.resendStartAuctions(desiredLRPsByProcessGuid, actualsByProcessGuid, pollingInterval, logger)

	lrpsStoppedCounter.Add(uint64(len(stopLRPInstances)))

	err = bbs.RequestStopLRPInstances(stopLRPInstances)
	if err != nil {
		logger.Error("failed-to-request-stops", err)
	}
}

func (bbs *LRPBBS) instancesToStop(
	desiredLRPsByProcessGuid map[string]models.DesiredLRP,
	actualsByProcessGuid map[string][]models.ActualLRP,
	logger lager.Logger,
) []models.ActualLRP {
	var actualsToStop []models.ActualLRP

	for processGuid, actuals := range actualsByProcessGuid {
		if _, found := desiredLRPsByProcessGuid[processGuid]; !found {
			for _, actual := range actuals {
				if actual.State == models.ActualLRPStateUnclaimed {
					continue
				}

				logger.Info("detected-undesired-instance", lager.Data{
					"process-guid":  processGuid,
					"instance-guid": actual.InstanceGuid,
					"index":         actual.Index,
				})

				actualsToStop = append(actualsToStop, actual)
			}
		}
	}

	return actualsToStop
}

func (bbs *LRPBBS) resendStartAuctions(
	desiredLRPsByProcessGuid map[string]models.DesiredLRP,
	actualsByProcessGuid map[string][]models.ActualLRP,
	pollingInterval time.Duration,
	logger lager.Logger,
) {
	for processGuid, actuals := range actualsByProcessGuid {
		for _, actual := range actuals {
			if actual.State == models.ActualLRPStateUnclaimed && bbs.timeProvider.Now().After(time.Unix(0, actual.Since).Add(pollingInterval)) {
				desiredLRP, found := desiredLRPsByProcessGuid[processGuid]
				if !found {
					logger.Info("failed-to-find-desired-lrp-for-stale-unclaimed-actual-lrp", lager.Data{"actual-lrp": actual})
					continue
				}

				lrpStart := models.LRPStart{
					DesiredLRP: desiredLRP,
					Index:      actual.Index,
				}

				logger.Info("resending-start-auction", lager.Data{"process-guid": processGuid, "index": actual.Index})
				err := bbs.requestLRPStartAuction(lrpStart)
				if err != nil {
					logger.Error("failed-resending-start-auction", err, lager.Data{
						"lrp-start-auction": lrpStart,
					})
				}
			}
		}
	}
}

func (bbs *LRPBBS) needsReconciliation(
	desiredLRP models.DesiredLRP,
	actualLRPsForDesired []models.ActualLRP,
	logger lager.Logger,
) bool {
	var actuals delta_force.ActualInstances
	for _, actual := range actualLRPsForDesired {
		actuals = append(actuals, delta_force.ActualInstance{
			Index: actual.Index,
			Guid:  actual.InstanceGuid,
		})
	}

	result := delta_force.Reconcile(desiredLRP.Instances, actuals)

	if len(result.IndicesToStart) > 0 {
		logger.Info("detected-missing-instance", lager.Data{
			"process-guid":      desiredLRP.ProcessGuid,
			"desired-instances": desiredLRP.Instances,
			"missing-indices":   result.IndicesToStart,
		})
	}

	if len(result.GuidsToStop) > 0 {
		logger.Info("detected-extra-instance", lager.Data{
			"process-guid":      desiredLRP.ProcessGuid,
			"desired-instances": desiredLRP.Instances,
			"extra-guids":       result.GuidsToStop,
		})
	}

	return !result.Empty()
}

func (bbs *LRPBBS) pruneActualsWithMissingCells(logger lager.Logger) (map[string][]models.ActualLRP, error) {
	actualsByProcessGuid := map[string][]models.ActualLRP{}

	cellRoot, err := bbs.store.ListRecursively(shared.CellSchemaRoot)
	if err == storeadapter.ErrorKeyNotFound {
		cellRoot = storeadapter.StoreNode{}
	} else if err != nil {
		logger.Error("failed-to-get-cells", err)
		return nil, err
	}

	err = prune.Prune(bbs.store, shared.ActualLRPSchemaRoot, func(node storeadapter.StoreNode) (shouldKeep bool) {
		var actual models.ActualLRP
		err := models.FromJSON(node.Value, &actual)
		if err != nil {
			return false
		}

		if actual.State != models.ActualLRPStateUnclaimed {
			if _, ok := cellRoot.Lookup(actual.CellID); !ok {
				logger.Info("detected-actual-with-missing-cell", lager.Data{
					"actual":  actual,
					"cell-id": actual.CellID,
				})
				return false
			}
		}

		actualsByProcessGuid[actual.ProcessGuid] = append(actualsByProcessGuid[actual.ProcessGuid], actual)
		return true
	})

	if err != nil {
		logger.Error("failed-to-prune-actual-lrps", err)
		return nil, err
	}

	return actualsByProcessGuid, nil
}

func (bbs *LRPBBS) batchCompareAndSwapDesiredLRPs(
	desiredLRPsToCAS []compareAndSwappableDesiredLRP,
	logger lager.Logger,
) {
	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(len(desiredLRPsToCAS))
	for _, desiredLRPToCAS := range desiredLRPsToCAS {
		desiredLRP := desiredLRPToCAS.NewDesiredLRP
		value, err := models.ToJSON(desiredLRP)
		if err != nil {
			panic(err)
		}
		newStoreNode := storeadapter.StoreNode{
			Key:   shared.DesiredLRPSchemaPath(desiredLRP),
			Value: value,
		}

		go func(desiredLRPToCAS compareAndSwappableDesiredLRP, newStoreNode storeadapter.StoreNode) {
			defer waitGroup.Done()
			logger.Info("kicking-desired-lrp", lager.Data{"key": newStoreNode.Key})
			err := bbs.store.CompareAndSwapByIndex(desiredLRPToCAS.OldIndex, newStoreNode)
			if err != nil {
				logger.Error("failed-to-compare-and-swap", err)
			}
		}(desiredLRPToCAS, newStoreNode)
	}

	waitGroup.Wait()
}
