package lrp_bbs

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/workpool"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

type actualLRPIndexTooLargeError struct {
	actualIndex      int
	desiredInstances int
}

func (e actualLRPIndexTooLargeError) Error() string {
	return fmt.Sprintf("Index %d too large for desired number %d of instances", e.actualIndex, e.desiredInstances)
}

func NewActualLRPIndexTooLargeError(actualIndex, desiredInstances int) error {
	return actualLRPIndexTooLargeError{actualIndex: actualIndex, desiredInstances: desiredInstances}
}

func (bbs *LRPBBS) CreateActualLRP(desiredLRP models.DesiredLRP, index int, logger lager.Logger) error {
	err := bbs.createActualLRP(desiredLRP, index, logger)
	if err != nil {
		return err
	}

	lrpStart := models.LRPStartRequest{
		DesiredLRP: desiredLRP,
		Indices:    []uint{uint(index)},
	}

	err = bbs.requestLRPAuctions([]models.LRPStartRequest{lrpStart})
	if err != nil {
		logger.Error("failed-to-request-start-auctions", err, lager.Data{"lrp-start": lrpStart})
		// The creation succeeded, the start request error can be dropped
	}

	return nil
}

func (bbs *LRPBBS) createAndStartActualLRPsForDesired(logger lager.Logger, lrp models.DesiredLRP, indices []uint) error {
	start := models.NewLRPStartRequest(lrp, indices)

	for _, actualIndex := range indices {
		err := bbs.createActualLRP(lrp, int(actualIndex), logger)
		if err != nil {
			return err
		}
	}

	err := bbs.requestLRPAuctions([]models.LRPStartRequest{start})
	if err != nil {
		logger.Error("failed-to-request-start-auctions", err, lager.Data{"lrp-start": start})
		// The creation succeeded, the start request error can be dropped
	}
	return nil
}

func (bbs *LRPBBS) createActualLRP(desiredLRP models.DesiredLRP, index int, logger lager.Logger) error {
	logger = logger.Session("create-actual-lrp")
	var err error
	if index >= desiredLRP.Instances {
		err = NewActualLRPIndexTooLargeError(index, desiredLRP.Instances)
		logger.Error("actual-lrp-index-too-large", err, lager.Data{"actual-index": index, "desired-instances": desiredLRP.Instances})
		return err
	}

	actualLRP := models.ActualLRP{
		ActualLRPKey: models.NewActualLRPKey(
			desiredLRP.ProcessGuid,
			index,
			desiredLRP.Domain,
		),
		State: models.ActualLRPStateUnclaimed,
		Since: bbs.clock.Now().UnixNano(),
	}

	err = bbs.createRawActualLRP(&actualLRP, logger)
	if err != nil {
		return err
	}

	return nil
}

func (bbs *LRPBBS) unclaimCrashedActualLRP(logger lager.Logger, key models.ActualLRPKey) (models.ActualLRP, error) {
	logger = logger.Session("unlaim-crashed-actual-lrp")
	logger.Info("starting")
	lrp, index, err := bbs.getActualLRP(key.ProcessGuid, key.Index)
	if err != nil {
		logger.Error("failed-to-get-actual-lrp", err)
		return models.ActualLRP{}, err
	}

	if lrp.State != models.ActualLRPStateCrashed {
		logger.Error("failed-actual-lrp-state-is-not-crashed", nil, lager.Data{"actual-lrp": lrp})
		return models.ActualLRP{}, bbserrors.ErrActualLRPCannotBeUnclaimed
	}

	lrp.Since = bbs.clock.Now().UnixNano()
	lrp.State = models.ActualLRPStateUnclaimed
	lrp.ActualLRPContainerKey = models.ActualLRPContainerKey{}
	lrp.ActualLRPNetInfo = models.ActualLRPNetInfo{}

	value, err := models.ToJSON(lrp)
	if err != nil {
		logger.Error("failed-to-marshal-actual-lrp", err, lager.Data{"actual-lrp": lrp})
		return models.ActualLRP{}, err
	}

	err = bbs.store.CompareAndSwapByIndex(index, storeadapter.StoreNode{
		Key:   shared.ActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index),
		Value: value,
	})

	if err != nil {
		logger.Error("failed-to-compare-and-swap-actual-lrp", err, lager.Data{"actual-lrp": lrp})
		return models.ActualLRP{}, shared.ConvertStoreError(err)
	}

	logger.Info("succeeded")
	return *lrp, nil
}

func (bbs *LRPBBS) createRawActualLRP(lrp *models.ActualLRP, logger lager.Logger) error {
	value, err := models.ToJSON(lrp)
	if err != nil {
		logger.Error("failed-to-marshal-actual-lrp", err, lager.Data{"actual-lrp": lrp})
		return err
	}

	err = bbs.store.Create(storeadapter.StoreNode{
		Key:   shared.ActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index),
		Value: value,
	})

	if err != nil {
		logger.Error("failed-to-create-actual-lrp", err, lager.Data{"actual-lrp": lrp})
		return shared.ConvertStoreError(err)
	}

	return nil
}

func (bbs *LRPBBS) ClaimActualLRP(
	key models.ActualLRPKey,
	containerKey models.ActualLRPContainerKey,
	logger lager.Logger,
) error {
	logger = logger.Session("claim-actual-lrp")
	logger.Info("starting")
	lrp, index, err := bbs.getActualLRP(key.ProcessGuid, key.Index)
	if err == bbserrors.ErrStoreResourceNotFound {
		logger.Error("failed-actual-lrp-not-found", err)
		return bbserrors.ErrActualLRPCannotBeClaimed
	} else if err != nil {
		logger.Error("failed-to-get-actual-lrp", err)
		return err
	}

	if lrp.ActualLRPKey == key &&
		lrp.ActualLRPContainerKey == containerKey &&
		lrp.State == models.ActualLRPStateClaimed {
		logger.Info("succeeded")
		return nil
	}

	if !lrp.AllowsTransitionTo(key, containerKey, models.ActualLRPStateClaimed) {
		logger.Error("failed-to-transition-actual-lrp-to-claimed", nil)
		return bbserrors.ErrActualLRPCannotBeClaimed
	}

	lrp.Since = bbs.clock.Now().UnixNano()
	lrp.State = models.ActualLRPStateClaimed
	lrp.ActualLRPContainerKey = containerKey
	lrp.ActualLRPNetInfo = models.ActualLRPNetInfo{}
	lrp.PlacementError = ""

	value, err := models.ToJSON(lrp)
	if err != nil {
		logger.Error("failed-to-marshal-actual-lrp", err, lager.Data{"actual-lrp": lrp})
		return err
	}

	err = bbs.store.CompareAndSwapByIndex(index, storeadapter.StoreNode{
		Key:   shared.ActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index),
		Value: value,
	})
	if err != nil {
		logger.Error("failed-to-compare-and-swap-actual-lrp", err, lager.Data{"actual-lrp": lrp})
		return shared.ConvertStoreError(err)
	}

	logger.Info("succeeded")
	return nil
}

func (bbs *LRPBBS) StartActualLRP(
	key models.ActualLRPKey,
	containerKey models.ActualLRPContainerKey,
	netInfo models.ActualLRPNetInfo,
	logger lager.Logger,
) error {
	logger = logger.Session("start-actual-lrp")
	logger.Info("starting")
	lrp, index, err := bbs.getActualLRP(key.ProcessGuid, key.Index)
	if err == bbserrors.ErrStoreResourceNotFound {
		newLRP := bbs.newRunningActualLRP(key, containerKey, netInfo)
		return bbs.createRawActualLRP(&newLRP, logger)
	} else if err != nil {
		logger.Error("failed-to-get-actual-lrp", err)
		return err
	}

	if lrp.ActualLRPKey == key &&
		lrp.ActualLRPContainerKey == containerKey &&
		lrp.Address == netInfo.Address &&
		reflect.DeepEqual(lrp.Ports, netInfo.Ports) &&
		lrp.State == models.ActualLRPStateRunning {
		logger.Info("succeeded")
		return nil
	}

	if !lrp.AllowsTransitionTo(key, containerKey, models.ActualLRPStateRunning) {
		logger.Error("failed-to-transition-actual-lrp-to-started", nil)
		return bbserrors.ErrActualLRPCannotBeStarted
	}

	lrp.State = models.ActualLRPStateRunning
	lrp.Since = bbs.clock.Now().UnixNano()
	lrp.ActualLRPContainerKey = containerKey
	lrp.ActualLRPNetInfo = netInfo
	lrp.PlacementError = ""

	value, err := models.ToJSON(lrp)
	if err != nil {
		logger.Error("failed-to-marshal-actual-lrp", err, lager.Data{"actual-lrp": lrp})
		return err
	}

	err = bbs.store.CompareAndSwapByIndex(index, storeadapter.StoreNode{
		Key:   shared.ActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index),
		Value: value,
	})
	if err != nil {
		logger.Error("failed-to-compare-and-swap-actual-lrp", err, lager.Data{"actual-lrp": lrp})
		return shared.ConvertStoreError(err)
	}

	logger.Info("succeeded")
	return nil

}

const CrashResetTimeout = 5 * time.Minute

func (bbs *LRPBBS) CrashActualLRP(key models.ActualLRPKey, containerKey models.ActualLRPContainerKey, logger lager.Logger) error {
	logger = logger.Session("crash-actual-lrp", lager.Data{"process-guid": key.ProcessGuid})
	logger.Info("starting")

	lrp, storeIndex, err := bbs.getActualLRP(key.ProcessGuid, key.Index)
	if err != nil {
		logger.Error("failed-to-get-actual-lrp", err)
		return err
	}

	latestChangeTime := time.Duration(bbs.clock.Now().UnixNano() - lrp.Since)

	var newCrashCount int
	if latestChangeTime > CrashResetTimeout && lrp.State == models.ActualLRPStateRunning {
		newCrashCount = 1
	} else {
		newCrashCount = lrp.CrashCount + 1
	}

	if !lrp.AllowsTransitionTo(lrp.ActualLRPKey, lrp.ActualLRPContainerKey, models.ActualLRPStateCrashed) {
		err := fmt.Errorf("cannot transition crashed lrp from state %s to state %s", lrp.State, models.ActualLRPStateCrashed)
		logger.Error("failed-to-transition-actual", err)
		return err
	}

	lrp.State = models.ActualLRPStateCrashed
	lrp.Since = bbs.clock.Now().UnixNano()
	lrp.CrashCount = newCrashCount
	lrp.ActualLRPContainerKey = models.ActualLRPContainerKey{}
	lrp.ActualLRPNetInfo = models.ActualLRPNetInfo{}

	var immediateRestart bool
	if lrp.ShouldRestartImmediately(bbs.restartCalculator) {
		lrp.State = models.ActualLRPStateUnclaimed
		immediateRestart = true
	}

	value, err := models.ToJSON(lrp)
	if err != nil {
		logger.Error("failed-to-marshal-actual-lrp", err, lager.Data{"actual-lrp": lrp})
		return err
	}

	err = bbs.store.CompareAndSwapByIndex(storeIndex, storeadapter.StoreNode{
		Key:   shared.ActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index),
		Value: value,
	})

	if err != nil {
		logger.Error("failed-to-CAS-actual-lrp", err, lager.Data{"actual-lrp": lrp})
		return shared.ConvertStoreError(err)
	}

	if immediateRestart {
		err := bbs.requestLRPAuctionForLRPKey(key)
		if err != nil {
			logger.Error("failed-to-request-auction", err)
			return err
		}
	}

	logger.Info("succeeded")
	return nil
}

func (bbs *LRPBBS) RemoveActualLRP(
	key models.ActualLRPKey,
	containerKey models.ActualLRPContainerKey,
	logger lager.Logger,
) error {
	logger = logger.Session("remove-actual-lrp")
	logger.Info("starting")

	lrp, storeIndex, err := bbs.getActualLRP(key.ProcessGuid, key.Index)
	if err != nil {
		logger.Error("failed-to-get-actual-lrp", err)
		return err
	}

	if lrp.ActualLRPKey != key || lrp.ActualLRPContainerKey != containerKey {
		logger.Error("failed-to-match-existing-actual-lrp", err, lager.Data{"existing-actual-lrp": lrp})
		return bbserrors.ErrStoreComparisonFailed
	}

	err = bbs.store.CompareAndDeleteByIndex(storeadapter.StoreNode{
		Key:   shared.ActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index),
		Index: storeIndex,
	})
	if err != nil {
		logger.Error("failed-to-compare-and-delete-actual-lrp", err, lager.Data{"actual-lrp": lrp})
		return shared.ConvertStoreError(err)
	}

	logger.Info("succeeded")
	return nil
}

const RetireActualPoolSize = 20

func (bbs *LRPBBS) RetireActualLRPs(lrps []models.ActualLRP, logger lager.Logger) {
	logger = logger.Session("retire-actual-lrps")

	pool := workpool.NewWorkPool(RetireActualPoolSize)

	wg := new(sync.WaitGroup)
	wg.Add(len(lrps))

	for _, lrp := range lrps {
		lrp := lrp
		pool.Submit(func() {
			defer wg.Done()

			err := bbs.retireActualLRP(lrp, logger)
			if err != nil {
				logger.Error("failed-to-retire", err, lager.Data{
					"lrp": lrp,
				})
			}
		})
	}

	wg.Wait()

	pool.Stop()
}

func (bbs *LRPBBS) retireActualLRP(lrp models.ActualLRP, logger lager.Logger) error {
	switch lrp.State {

	case models.ActualLRPStateUnclaimed, models.ActualLRPStateCrashed:
		return bbs.RemoveActualLRP(lrp.ActualLRPKey, lrp.ActualLRPContainerKey, logger)

	default:
		logger.Info("stopping-actual")
		err := bbs.RequestStopLRPInstance(lrp.ActualLRPKey, lrp.ActualLRPContainerKey)
		if err != nil {
			logger.Error("failed-to-retire-actual-lrp", err, lager.Data{
				"actual-lrp": lrp,
			})
			return err
		}
		return nil
	}
}

func (bbs *LRPBBS) getActualLRP(processGuid string, index int) (*models.ActualLRP, uint64, error) {
	node, err := bbs.store.Get(shared.ActualLRPSchemaPath(processGuid, index))
	if err != nil {
		return nil, 0, shared.ConvertStoreError(err)
	}

	var lrp models.ActualLRP
	err = models.FromJSON(node.Value, &lrp)

	return &lrp, node.Index, err
}

func (bbs *LRPBBS) newRunningActualLRP(
	key models.ActualLRPKey,
	containerKey models.ActualLRPContainerKey,
	netInfo models.ActualLRPNetInfo,
) models.ActualLRP {
	return models.ActualLRP{
		ActualLRPKey:          key,
		ActualLRPContainerKey: containerKey,
		ActualLRPNetInfo:      netInfo,
		Since:                 bbs.clock.Now().UnixNano(),
		State:                 models.ActualLRPStateRunning,
	}
}

func (bbs *LRPBBS) requestLRPAuctions(lrpStarts []models.LRPStartRequest) error {
	auctioneerAddress, err := bbs.services.AuctioneerAddress()
	if err != nil {
		return err
	}

	return bbs.auctioneerClient.RequestLRPAuctions(auctioneerAddress, lrpStarts)
}

func (bbs *LRPBBS) requestLRPAuctionForLRPKey(key models.ActualLRPKey) error {
	desiredLRP, err := bbs.DesiredLRPByProcessGuid(key.ProcessGuid)
	if err != nil {
		return err
	}

	lrpStart := models.NewLRPStartRequest(desiredLRP, []uint{uint(key.Index)})
	return bbs.requestLRPAuctions([]models.LRPStartRequest{lrpStart})
}

func (bbs *LRPBBS) FailLRP(
	logger lager.Logger,
	key models.ActualLRPKey,
	errorMessage string,
) error {
	logger = logger.Session("set-placement-error-actual-lrp")
	logger.Info("starting")
	lrp, index, err := bbs.getActualLRP(key.ProcessGuid, key.Index)
	if err == bbserrors.ErrStoreResourceNotFound {
		logger.Error("failed-actual-lrp-not-found", err)
		return bbserrors.ErrCannotFailLRP
	} else if err != nil {
		logger.Error("failed-to-get-actual-lrp", err)
		return err
	}

	if lrp.ActualLRPKey == key && lrp.State != models.ActualLRPStateUnclaimed {
		logger.Error("failed-to-set-placement-error", bbserrors.ErrCannotFailLRP, lager.Data{"lrp": lrp})
		return bbserrors.ErrCannotFailLRP
	}

	lrp.Since = bbs.clock.Now().UnixNano()
	lrp.PlacementError = errorMessage

	value, err := models.ToJSON(lrp)
	if err != nil {
		logger.Error("failed-to-marshal-actual-lrp", err, lager.Data{"actual-lrp": lrp})
		return err
	}

	err = bbs.store.CompareAndSwapByIndex(index, storeadapter.StoreNode{
		Key:   shared.ActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index),
		Value: value,
	})
	if err != nil {
		logger.Error("failed-to-compare-and-swap-actual-lrp", err, lager.Data{"actual-lrp": lrp})
		return shared.ConvertStoreError(err)
	}

	logger.Info("succeeded")
	return nil
}
