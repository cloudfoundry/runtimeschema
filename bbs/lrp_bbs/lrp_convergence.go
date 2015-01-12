package lrp_bbs

import (
	"time"

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
)

func (bbs *LRPBBS) ConvergeLRPs(logger lager.Logger, resendStartAuctionTimeout time.Duration) {
	logger = logger.Session("converge-lrps")
	logger.Info("starting-convergence")
	defer logger.Info("finished-convergence")

	convergeLRPRunsCounter.Increment()

	convergeStart := bbs.timeProvider.Now()

	// make sure to get funcy here otherwise the time will be precomputed
	defer func() {
		convergeLRPDuration.Send(bbs.timeProvider.Now().Sub(convergeStart))
	}()

	domainRoot, err := bbs.store.ListRecursively(shared.DomainSchemaRoot)
	if err != nil && err != storeadapter.ErrorKeyNotFound {
		logger.Error("failed-to-fetch-domains", err)
		return
	}

	// Obtain Actuals before Desired to guarantee the most current state of
	// Desired when performing convergence -- minimize any extraneous stops

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

	var malformedDesiredLRPs []string
	desiredLRPsByProcessGuid := map[string]models.DesiredLRP{}

	infos := []reconcileInfo{}
	for _, node := range desiredLRPRoot.ChildNodes {
		var desiredLRP models.DesiredLRP
		err := models.FromJSON(node.Value, &desiredLRP)

		if err != nil {
			logger.Info("pruning-invalid-desired-lrp-json", lager.Data{
				"error":   err.Error(),
				"payload": node.Value,
			})
			malformedDesiredLRPs = append(malformedDesiredLRPs, node.Key)
			continue
		}

		desiredLRPsByProcessGuid[desiredLRP.ProcessGuid] = desiredLRP
		actualLRPsForDesired := actualsByProcessGuid[desiredLRP.ProcessGuid]

		delta := Reconcile(desiredLRP.Instances, actualLRPsForDesired)
		if len(delta.IndicesToStop) > 0 {
			if _, found := domainRoot.Lookup(desiredLRP.Domain); !found {
				for _, index := range delta.IndicesToStop {
					actual := actualLRPsForDesired[index]
					logger.Info("not-stopping-actual-instance-domain-not-fresh", lager.Data{
						"process-guid":  actual.ProcessGuid,
						"instance-guid": actual.InstanceGuid,
						"index":         actual.Index,
					})
				}
				delta.IndicesToStop = []int{}
			}
		}
		if !delta.Empty() {
			infos = append(infos, reconcileInfo{desiredLRP, actualLRPsForDesired, delta})
		}
	}

	bbs.reconcile(infos, logger)

	actualLRPsToStop := bbs.instancesToStop(desiredLRPsByProcessGuid, actualsByProcessGuid, domainRoot, logger)

	for _, actual := range actualLRPsToStop {
		logger.Info("detected-undesired-instance", lager.Data{
			"process-guid":  actual.ProcessGuid,
			"instance-guid": actual.InstanceGuid,
			"index":         actual.Index,
		})
	}

	lrpsDeletedCounter.Add(uint64(len(malformedDesiredLRPs)))
	bbs.store.Delete(malformedDesiredLRPs...)

	lrpStopInstanceCounter.Add(uint64(len(actualLRPsToStop)))
	bbs.RetireActualLRPs(actualLRPsToStop, logger)

	bbs.resendStartAuctions(desiredLRPsByProcessGuid, actualsByProcessGuid, resendStartAuctionTimeout, logger)
}

func (bbs *LRPBBS) instancesToStop(
	desiredLRPsByProcessGuid map[string]models.DesiredLRP,
	actualsByProcessGuid map[string]models.ActualLRPsByIndex,
	domainRoot storeadapter.StoreNode,
	logger lager.Logger,
) []models.ActualLRP {
	var actualsToStop []models.ActualLRP

	for processGuid, actuals := range actualsByProcessGuid {
		if _, found := desiredLRPsByProcessGuid[processGuid]; !found {
			_, domainFound := domainRoot.Lookup(actuals[0].Domain)
			for _, actual := range actuals {
				if domainFound {
					actualsToStop = append(actualsToStop, actual)
				} else {
					logger.Info("not-stopping-actual-instance-domain-not-fresh", lager.Data{
						"process-guid":  actual.ProcessGuid,
						"instance-guid": actual.InstanceGuid,
						"index":         actual.Index,
					})
				}
			}
		}
	}

	return actualsToStop
}

type startRequests struct {
	desiredMap map[string]models.DesiredLRP
	startMap   map[string]models.LRPStartRequest
}

func newStartRequests(desiredMap map[string]models.DesiredLRP) *startRequests {
	return &startRequests{
		desiredMap: desiredMap,
		startMap:   make(map[string]models.LRPStartRequest),
	}
}

func (s startRequests) Add(logger lager.Logger, actual models.ActualLRP) {
	desiredLRP, found := s.desiredMap[actual.ProcessGuid]
	if !found {
		logger.Info("failed-to-find-desired-lrp-for-stale-unclaimed-actual-lrp", lager.Data{"actual-lrp": actual})
		return
	}

	start, found := s.startMap[desiredLRP.ProcessGuid]
	if !found {
		start = models.LRPStartRequest{
			DesiredLRP: desiredLRP,
			Indices:    []uint{uint(actual.Index)},
		}
	} else {
		start.Indices = append(start.Indices, uint(actual.Index))
	}

	logger.Info("adding-start-auction", lager.Data{"process-guid": desiredLRP.ProcessGuid, "index": actual.Index})
	s.startMap[desiredLRP.ProcessGuid] = start
}

func (s startRequests) Slice() []models.LRPStartRequest {
	starts := make([]models.LRPStartRequest, 0, len(s.startMap))
	for _, start := range s.startMap {
		starts = append(starts, start)
	}
	return starts
}

func (bbs *LRPBBS) resendStartAuctions(
	desiredLRPsByProcessGuid map[string]models.DesiredLRP,
	actualsByProcessGuid map[string]models.ActualLRPsByIndex,
	resendStartAuctionTimeout time.Duration,
	logger lager.Logger,
) {
	logger = logger.Session("resending")
	startsByProcessGuid := newStartRequests(desiredLRPsByProcessGuid)

	now := bbs.timeProvider.Now().UnixNano()
	lastPoll := now - pollingInterval.Nanoseconds()
	for _, actuals := range actualsByProcessGuid {
		for _, actual := range actuals {
			switch {
			case actual.ShouldRestartCrash(now):
				unclaimedActual, err := bbs.unclaimCrashedActualLRP(logger, actual.ActualLRPKey)
				if err != nil {
					logger.Info("failed-to-unclaim-crashed-actual-lrp", lager.Data{"actual-lrp": actual})
					continue
				}
				startsByProcessGuid.Add(logger, unclaimedActual)

			case actual.ShouldStartUnclaimed(lastPoll):
				startsByProcessGuid.Add(logger, actual)
			}
		}
	}

	starts := startsByProcessGuid.Slice()
	if len(starts) == 0 {
		return
	}

	err := bbs.requestLRPAuctions(starts)
	if err != nil {
		logger.Error("failed-resending-start-auctions", err, lager.Data{
			"lrp-start-auctions": starts,
		})
	}
}

func (bbs *LRPBBS) pruneActualsWithMissingCells(logger lager.Logger) (map[string]models.ActualLRPsByIndex, error) {
	actualsByProcessGuid := map[string]models.ActualLRPsByIndex{}

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

		if !(actual.State == models.ActualLRPStateUnclaimed || actual.State == models.ActualLRPStateCrashed) {
			if _, ok := cellRoot.Lookup(actual.CellID); !ok {
				logger.Info("detected-actual-with-missing-cell", lager.Data{
					"actual":  actual,
					"cell-id": actual.CellID,
				})
				return false
			}
		}

		actuals, found := actualsByProcessGuid[actual.ProcessGuid]
		if !found {
			actuals = models.ActualLRPsByIndex{}
			actualsByProcessGuid[actual.ProcessGuid] = actuals
		}

		actuals[actual.Index] = actual

		return true
	})

	if err != nil {
		logger.Error("failed-to-prune-actual-lrps", err)
		return nil, err
	}

	return actualsByProcessGuid, nil
}
