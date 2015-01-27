package lrp_bbs

import (
	"sync"

	"github.com/cloudfoundry-incubator/runtime-schema/metric"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/workpool"
	"github.com/pivotal-golang/lager"
)

var (
	lrpStartInstanceCounter = metric.Counter("LRPInstanceStartRequests")
	lrpStopInstanceCounter  = metric.Counter("LRPInstanceStopRequests")
)

const ConvergencePoolSize = 20

func (bbs *LRPBBS) ResolveConvergence(logger lager.Logger, desiredLRPs models.DesiredLRPsByProcessGuid, changes *ConvergenceChanges) {
	bbs.RetireActualLRPs(changes.ActualLRPsForExtraIndices, logger)
	lrpStopInstanceCounter.Add(uint64(len(changes.ActualLRPsForExtraIndices)))

	startRequests := newStartRequests(desiredLRPs)

	pool := workpool.NewWorkPool(ConvergencePoolSize)
	defer pool.Stop()

	wg := new(sync.WaitGroup)

	for _, actual := range changes.StaleUnclaimedActualLRPs {
		startRequests.Add(logger, actual.ActualLRPKey)
	}

	for _, actual := range changes.ActualLRPsWithMissingCells {
		pool.Submit(bbs.resolveActualsWithMissingCells(logger, wg, desiredLRPs[actual.ProcessGuid], actual, startRequests))
	}

	for _, actualKey := range changes.ActualLRPKeysForMissingIndices {
		pool.Submit(bbs.resolveActualsWithMissingIndices(logger, wg, desiredLRPs[actualKey.ProcessGuid], actualKey, startRequests))
	}

	for _, actual := range changes.RestartableCrashedActualLRPs {
		pool.Submit(bbs.resolveRestartableCrashedActualLRPS(logger, wg, actual, startRequests))
	}

	wg.Wait()

	bbs.startActualLRPs(logger, startRequests)
}

func (bbs *LRPBBS) resolveActualsWithMissingCells(logger lager.Logger, wg *sync.WaitGroup, desired models.DesiredLRP, actual models.ActualLRP, starts *startRequests) func() {
	wg.Add(1)

	return func() {
		defer wg.Done()
		logger = logger.Session("start-missing-actual", lager.Data{
			"process-guid": actual.ProcessGuid,
			"index":        actual.Index,
		})

		err := bbs.RemoveActualLRP(actual.ActualLRPKey, actual.ActualLRPContainerKey, logger)
		if err != nil {
			logger.Error("failed-to-remove-actual-lrp", err)
			return
		}

		err = bbs.createActualLRP(desired, actual.Index, logger)
		if err != nil {
			logger.Error("failed-to-create-actual-lrp", err)
			return
		}

		starts.Add(logger, actual.ActualLRPKey)
	}
}

func (bbs *LRPBBS) resolveActualsWithMissingIndices(logger lager.Logger, wg *sync.WaitGroup, desired models.DesiredLRP, actualKey models.ActualLRPKey, starts *startRequests) func() {
	wg.Add(1)

	return func() {
		defer wg.Done()

		logger = logger.Session("start-missing-actual", lager.Data{
			"process-guid": actualKey.ProcessGuid,
			"index":        actualKey.Index,
		})

		err := bbs.createActualLRP(desired, actualKey.Index, logger)
		if err != nil {
			logger.Error("failed-to-create-actual-lrp", err)
			return
		}

		starts.Add(logger, actualKey)
	}
}

func (bbs *LRPBBS) resolveRestartableCrashedActualLRPS(logger lager.Logger, wg *sync.WaitGroup, actualLRP models.ActualLRP, starts *startRequests) func() {
	wg.Add(1)

	return func() {
		defer wg.Done()

		actualKey := actualLRP.ActualLRPKey

		logger = logger.Session("restart-crash", lager.Data{
			"process-guid": actualKey.ProcessGuid,
			"index":        actualKey.Index,
		})

		if actualLRP.State != models.ActualLRPStateCrashed {
			logger.Error("failed-actual-lrp-state-is-not-crashed", nil)
			return
		}

		err := bbs.UnclaimActualLRP(logger, actualLRP.ActualLRPKey, actualLRP.ActualLRPContainerKey)
		if err != nil {
			logger.Error("failed-to-unclaim-crash", err)
			return
		}

		starts.Add(logger, actualKey)
	}
}

func (bbs *LRPBBS) startActualLRPs(logger lager.Logger, starts *startRequests) {
	count := starts.InstanceCount()
	if count == 0 {
		return
	}

	err := bbs.requestLRPAuctions(starts.Slice())
	if err != nil {
		logger.Error("failed-to-request-starts", err, lager.Data{
			"lrp-start-auctions": starts,
		})
	}

	lrpStartInstanceCounter.Add(count)
}

type startRequests struct {
	desiredMap    models.DesiredLRPsByProcessGuid
	startMap      map[string]models.LRPStartRequest
	instanceCount uint64
	*sync.Mutex
}

func newStartRequests(desiredMap models.DesiredLRPsByProcessGuid) *startRequests {
	return &startRequests{
		desiredMap: desiredMap,
		startMap:   make(map[string]models.LRPStartRequest),
		Mutex:      new(sync.Mutex),
	}
}

func (s *startRequests) Add(logger lager.Logger, actual models.ActualLRPKey) {
	s.Lock()
	defer s.Unlock()

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
	s.instanceCount++
}

func (s *startRequests) Slice() []models.LRPStartRequest {
	s.Lock()
	defer s.Unlock()

	starts := make([]models.LRPStartRequest, 0, len(s.startMap))
	for _, start := range s.startMap {
		starts = append(starts, start)
	}
	return starts
}

func (s *startRequests) InstanceCount() uint64 {
	s.Lock()
	defer s.Unlock()

	return s.instanceCount
}
