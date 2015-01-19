package lrp_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/metric"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/pivotal-golang/lager"
)

const (
	convergeLRPRunsCounter = metric.Counter("ConvergenceLRPRuns")
	convergeLRPDuration    = metric.Duration("ConvergenceLRPDuration")

	lrpsDeletedCounter = metric.Counter("ConvergenceLRPsDeleted")
)

func (bbs *LRPBBS) ConvergeLRPs(logger lager.Logger) {
	logger = logger.Session("converge-lrps")
	logger.Info("starting-convergence")
	defer logger.Info("finished-convergence")

	convergeLRPRunsCounter.Increment()

	// make sure to get funcy here otherwise the time will be precomputed
	convergeStart := bbs.timeProvider.Now()
	defer func() {
		convergeLRPDuration.Send(bbs.timeProvider.Now().Sub(convergeStart))
	}()

	convergenceInput, err := bbs.GatherAndPruneLRPConvergenceInput(logger)
	if err != nil {
		logger.Error("failed-to-gather-convergence-input", err)
		return
	}

	changes := CalculateConvergence(logger, bbs.timeProvider, convergenceInput)

	bbs.RetireActualLRPs(changes.ActualLRPsForExtraIndices, logger)
	lrpStopInstanceCounter.Add(uint64(len(changes.ActualLRPsForExtraIndices)))

	actualsToStart := []models.ActualLRPKey{}

	for _, actual := range changes.StaleUnclaimedActualLRPs {
		actualsToStart = append(actualsToStart, actual.ActualLRPKey)
	}

	for _, actual := range changes.ActualLRPsWithMissingCells {
		l := logger.Session("start-missing-actual", lager.Data{
			"process-guid": actual.ProcessGuid,
			"index":        actual.Index,
		})

		err = bbs.RemoveActualLRP(actual.ActualLRPKey, actual.ActualLRPContainerKey, logger)
		if err != nil {
			l.Error("failed-to-remove-actual-lrp", err)
			continue
		}

		err := bbs.createActualLRP(convergenceInput.DesiredLRPs[actual.ProcessGuid], actual.Index, l)
		if err != nil {
			l.Error("failed-to-create-actual-lrp", err)
			continue
		}

		actualsToStart = append(actualsToStart, actual.ActualLRPKey)
	}

	for _, actualKey := range changes.ActualLRPKeysForMissingIndices {
		l := logger.Session("start-missing-actual", lager.Data{
			"process-guid": actualKey.ProcessGuid,
			"index":        actualKey.Index,
		})

		err := bbs.createActualLRP(convergenceInput.DesiredLRPs[actualKey.ProcessGuid], actualKey.Index, l)
		if err != nil {
			l.Error("failed-to-create-actual-lrp", err)
			continue
		}

		actualsToStart = append(actualsToStart, actualKey)
	}

	for _, actual := range changes.RestartableCrashedActualLRPs {
		l := logger.Session("restart-crash", lager.Data{
			"process-guid": actual.ProcessGuid,
			"index":        actual.Index,
		})

		unclaimedActualLRP, err := bbs.unclaimCrashedActualLRP(l, actual.ActualLRPKey)
		if err != nil {
			l.Error("failed-to-unclaim-crash", err)
			continue
		}

		actualsToStart = append(actualsToStart, unclaimedActualLRP.ActualLRPKey)
	}

	bbs.startActualLRPs(convergenceInput.DesiredLRPs, actualsToStart, logger)
	lrpStartInstanceCounter.Add(uint64(len(actualsToStart)))
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

func (s startRequests) Add(logger lager.Logger, actual models.ActualLRPKey) {
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

// Should (eventually) build list of:
//   starts             [for eventual requestLRPAuctions]
//   stops              [whatever stopping is]
//   unclaimed crashes  [bulk unclaim crashed actual]
func (bbs *LRPBBS) startActualLRPs(
	desiredLRPsByProcessGuid models.DesiredLRPsByProcessGuid,
	actuals []models.ActualLRPKey,
	logger lager.Logger,
) {
	startsByProcessGuid := newStartRequests(desiredLRPsByProcessGuid)

	for _, actual := range actuals {
		startsByProcessGuid.Add(logger, actual)
	}

	starts := startsByProcessGuid.Slice()
	if len(starts) == 0 {
		return
	}

	err := bbs.requestLRPAuctions(starts)
	if err != nil {
		logger.Error("failed-to-request-starts", err, lager.Data{
			"lrp-start-auctions": starts,
		})
	}
}
