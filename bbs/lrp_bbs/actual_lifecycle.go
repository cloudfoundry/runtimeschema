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
	"github.com/nu7hatch/gouuid"
	"github.com/pivotal-golang/lager"
)

const CrashResetTimeout = 5 * time.Minute
const retireActualPoolSize = 20
const RetireActualLRPRetryAttempts = 5

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

func (bbs *LRPBBS) ClaimActualLRP(
	logger lager.Logger,
	key models.ActualLRPKey,
	containerKey models.ActualLRPContainerKey,
) error {
	logger = logger.Session("claim-actual-lrp", lager.Data{"lrp-key": key, "container-key": containerKey})
	logger.Info("starting")

	lrp, storeIndex, err := bbs.actualLRPWithIndex(logger, key.ProcessGuid, key.Index)
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

	err = bbs.compareAndSwapRawActualLRP(logger, lrp, storeIndex)
	if err != nil {
		return err
	}

	logger.Info("succeeded")
	return nil
}

func (bbs *LRPBBS) StartActualLRP(
	logger lager.Logger,
	key models.ActualLRPKey,
	containerKey models.ActualLRPContainerKey,
	netInfo models.ActualLRPNetInfo,
) error {
	logger = logger.Session("start-actual-lrp", lager.Data{"lrp-key": key, "container-key": containerKey})
	logger.Info("starting")
	lrp, storeIndex, err := bbs.actualLRPWithIndex(logger, key.ProcessGuid, key.Index)
	if err == bbserrors.ErrStoreResourceNotFound {
		newLRP, err := bbs.newRunningActualLRP(key, containerKey, netInfo)
		if err != nil {
			return err
		}

		return bbs.createRawActualLRP(logger, &newLRP)
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

	err = bbs.compareAndSwapRawActualLRP(logger, lrp, storeIndex)
	if err != nil {
		return err
	}

	logger.Info("succeeded")
	return nil
}

func (bbs *LRPBBS) CrashActualLRP(
	logger lager.Logger,
	key models.ActualLRPKey,
	containerKey models.ActualLRPContainerKey,
) error {
	logger = logger.Session("crash-actual-lrp", lager.Data{"lrp-key": key, "container-key": containerKey})
	logger.Info("starting")

	lrp, storeIndex, err := bbs.actualLRPWithIndex(logger, key.ProcessGuid, key.Index)
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

	logger.Debug("retrieved-lrp", lager.Data{"lrp": lrp})
	if !lrp.AllowsTransitionTo(key, containerKey, models.ActualLRPStateCrashed) {
		err := fmt.Errorf("cannot transition crashed lrp from state %s to state %s", lrp.State, models.ActualLRPStateCrashed)
		logger.Error("failed-to-transition-actual", err)
		return bbserrors.ErrActualLRPCannotBeCrashed
	}

	if lrp.State == models.ActualLRPStateUnclaimed || lrp.State == models.ActualLRPStateCrashed ||
		((lrp.State == models.ActualLRPStateClaimed || lrp.State == models.ActualLRPStateRunning) && lrp.ActualLRPContainerKey != containerKey) {
		return bbserrors.ErrActualLRPCannotBeCrashed
	}

	lrp.State = models.ActualLRPStateCrashed
	lrp.Since = bbs.clock.Now().UnixNano()
	lrp.CrashCount = newCrashCount
	lrp.ActualLRPContainerKey = models.ActualLRPContainerKey{}
	lrp.ActualLRPNetInfo = models.EmptyActualLRPNetInfo()

	var immediateRestart bool
	if lrp.ShouldRestartImmediately(models.NewDefaultRestartCalculator()) {
		lrp.State = models.ActualLRPStateUnclaimed
		immediateRestart = true
	}

	err = bbs.compareAndSwapRawActualLRP(logger, lrp, storeIndex)
	if err != nil {
		return err
	}

	if immediateRestart {
		err = bbs.requestLRPAuctionForLRPKey(logger, key)
		if err != nil {
			logger.Error("failed-to-request-auction", err)
			return err
		}
	}

	logger.Info("succeeded")
	return nil
}

func (bbs *LRPBBS) RemoveActualLRP(
	logger lager.Logger,
	key models.ActualLRPKey,
	containerKey models.ActualLRPContainerKey,
) error {
	logger = logger.Session("remove-actual-lrp", lager.Data{"lrp-key": key, "container-key": containerKey})
	logger.Info("starting")

	lrp, storeIndex, err := bbs.actualLRPWithIndex(logger, key.ProcessGuid, key.Index)
	if err != nil {
		return err
	}

	if lrp.ActualLRPKey != key || lrp.ActualLRPContainerKey != containerKey {
		logger.Error("failed-to-match-existing-actual-lrp", err, lager.Data{"existing-actual-lrp": lrp})
		return bbserrors.ErrStoreComparisonFailed
	}

	err = bbs.compareAndDeleteRawActualLRP(logger, lrp, storeIndex)
	if err != nil {
		return err
	}

	logger.Info("succeeded")
	return nil
}

func (bbs *LRPBBS) RetireActualLRPs(
	logger lager.Logger,
	lrps []models.ActualLRP,
) {
	logger = logger.Session("retire-actual-lrps")

	pool := workpool.NewWorkPool(retireActualPoolSize)

	wg := new(sync.WaitGroup)
	wg.Add(len(lrps))

	for _, lrp := range lrps {
		lrpKey := lrp.ActualLRPKey
		pool.Submit(func() {
			defer wg.Done()

			err := bbs.retireActualLRP(lrpKey, logger)
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

func (bbs *LRPBBS) FailActualLRP(
	logger lager.Logger,
	key models.ActualLRPKey,
	errorMessage string,
) error {
	logger = logger.Session("set-placement-error-actual-lrp", lager.Data{"lrp-key": key})
	logger.Info("starting")

	lrp, storeIndex, err := bbs.actualLRPWithIndex(logger, key.ProcessGuid, key.Index)
	if err == bbserrors.ErrStoreResourceNotFound {
		logger.Error("failed-actual-lrp-not-found", err)
		return bbserrors.ErrActualLRPCannotBeFailed
	} else if err != nil {
		logger.Error("failed-to-get-actual-lrp", err)
		return err
	}

	if lrp.ActualLRPKey == key && lrp.State != models.ActualLRPStateUnclaimed {
		logger.Error("failed-to-set-placement-error", bbserrors.ErrActualLRPCannotBeFailed, lager.Data{"lrp": lrp})
		return bbserrors.ErrActualLRPCannotBeFailed
	}

	lrp.Since = bbs.clock.Now().UnixNano()
	lrp.PlacementError = errorMessage

	err = bbs.compareAndSwapRawActualLRP(logger, lrp, storeIndex)
	if err != nil {
		return err
	}

	logger.Info("succeeded")
	return nil
}

func (bbs *LRPBBS) createAndStartActualLRPsForDesired(logger lager.Logger, lrp models.DesiredLRP, indices []uint) {
	createdIndices := make([]uint, 0, len(indices))

	for _, actualIndex := range indices {
		err := bbs.createActualLRP(lrp, int(actualIndex), logger)
		if err != nil {
			logger.Info("failed-creating-actual-lrp", lager.Data{"index": actualIndex, "err-message": err.Error()})
		} else {
			createdIndices = append(createdIndices, actualIndex)
		}
	}
	start := models.NewLRPStartRequest(lrp, createdIndices...)

	err := bbs.requestLRPAuctions([]models.LRPStartRequest{start})
	if err != nil {
		logger.Error("failed-to-request-start-auctions", err, lager.Data{"lrp-start": start})
	}
}

func (bbs *LRPBBS) createActualLRP(desiredLRP models.DesiredLRP, index int, logger lager.Logger) error {
	logger = logger.Session("create-actual-lrp")
	var err error
	if index >= desiredLRP.Instances {
		err = NewActualLRPIndexTooLargeError(index, desiredLRP.Instances)
		logger.Error("actual-lrp-index-too-large", err, lager.Data{"actual-index": index, "desired-instances": desiredLRP.Instances})
		return err
	}

	guid, err := uuid.NewV4()
	if err != nil {
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
		ModificationTag: models.ModificationTag{
			Epoch: guid.String(),
			Index: 0,
		},
	}

	err = bbs.createRawActualLRP(logger, &actualLRP)
	if err != nil {
		return err
	}

	return nil
}

type stateChange bool

const (
	stateDidChange    stateChange = true
	stateDidNotChange stateChange = false
)

func (bbs *LRPBBS) unclaimActualLRP(
	logger lager.Logger,
	actualLRPKey models.ActualLRPKey,
	actualLRPContainerKey models.ActualLRPContainerKey,
) (stateChange, error) {
	logger = logger.Session("unclaim-actual-lrp")
	logger.Info("starting")

	lrp, storeIndex, err := bbs.actualLRPWithIndex(logger, actualLRPKey.ProcessGuid, actualLRPKey.Index)
	if err != nil {
		logger.Error("failed-to-get-actual-lrp", err)
		return stateDidNotChange, err
	}

	changed, err := bbs.unclaimActualLRPWithIndex(logger, lrp, storeIndex, actualLRPKey, actualLRPContainerKey)
	if err != nil {
		return changed, err
	}

	logger.Info("succeeded")
	return changed, nil
}

func (bbs *LRPBBS) unclaimActualLRPWithIndex(
	logger lager.Logger,
	lrp *models.ActualLRP,
	storeIndex uint64,
	actualLRPKey models.ActualLRPKey,
	actualLRPContainerKey models.ActualLRPContainerKey,
) (stateChange, error) {
	if lrp.ActualLRPKey != actualLRPKey {
		logger.Error("failed-actual-lrp-key-differs", bbserrors.ErrActualLRPCannotBeUnclaimed)
		return stateDidNotChange, bbserrors.ErrActualLRPCannotBeUnclaimed
	}

	if lrp.State == models.ActualLRPStateUnclaimed {
		logger.Info("succeeded")
		return stateDidNotChange, nil
	}

	if lrp.ActualLRPContainerKey != actualLRPContainerKey {
		logger.Error("failed-actual-lrp-container-key-differs", bbserrors.ErrActualLRPCannotBeUnclaimed)
		return stateDidNotChange, bbserrors.ErrActualLRPCannotBeUnclaimed
	}

	lrp.Since = bbs.clock.Now().UnixNano()
	lrp.State = models.ActualLRPStateUnclaimed
	lrp.ActualLRPContainerKey = models.ActualLRPContainerKey{}
	lrp.ActualLRPNetInfo = models.EmptyActualLRPNetInfo()

	err := bbs.compareAndSwapRawActualLRP(logger, lrp, storeIndex)

	if err != nil {
		return stateDidNotChange, err
	}

	return stateDidChange, nil
}

func (bbs *LRPBBS) retireActualLRP(actualLRPKey models.ActualLRPKey, logger lager.Logger) error {
	var err error
	var lrp *models.ActualLRP

	for i := 0; i < RetireActualLRPRetryAttempts; i++ {
		lrp, _, err = bbs.actualLRPWithIndex(logger, actualLRPKey.ProcessGuid, actualLRPKey.Index)
		if err != nil {
			logger.Error("failed-fetching-actual-lrp", err, lager.Data{
				"actual-lrp-key": actualLRPKey,
			})
			break
		}

		switch lrp.State {
		case models.ActualLRPStateUnclaimed, models.ActualLRPStateCrashed:
			logger.Info("removing-actual")
			err = bbs.RemoveActualLRP(logger, lrp.ActualLRPKey, lrp.ActualLRPContainerKey)
		default:
			logger.Info("stopping-actual")
			err = bbs.requestStopLRPInstance(lrp.ActualLRPKey, lrp.ActualLRPContainerKey)
		}

		if err == nil {
			break
		}

		if i+1 < RetireActualLRPRetryAttempts {
			logger.Error("retrying-failed-retire-of-actual-lrp", err, lager.Data{
				"actual-lrp-key": actualLRPKey,
				"attempt":        i + 1,
			})
		}
	}

	return err
}

func (bbs *LRPBBS) newRunningActualLRP(
	key models.ActualLRPKey,
	containerKey models.ActualLRPContainerKey,
	netInfo models.ActualLRPNetInfo,
) (models.ActualLRP, error) {
	guid, err := uuid.NewV4()
	if err != nil {
		return models.ActualLRP{}, err
	}

	actualLRP := models.ActualLRP{
		ActualLRPKey:          key,
		ActualLRPContainerKey: containerKey,
		ActualLRPNetInfo:      netInfo,
		Since:                 bbs.clock.Now().UnixNano(),
		State:                 models.ActualLRPStateRunning,
		ModificationTag: models.ModificationTag{
			Epoch: guid.String(),
			Index: 0,
		},
	}

	return actualLRP, nil
}

func (bbs *LRPBBS) requestLRPAuctions(lrpStarts []models.LRPStartRequest) error {
	auctioneerAddress, err := bbs.services.AuctioneerAddress()
	if err != nil {
		return err
	}

	return bbs.auctioneerClient.RequestLRPAuctions(auctioneerAddress, lrpStarts)
}

func (bbs *LRPBBS) requestLRPAuctionForLRPKey(logger lager.Logger, key models.ActualLRPKey) error {
	desiredLRP, err := bbs.DesiredLRPByProcessGuid(key.ProcessGuid)
	if err == bbserrors.ErrStoreResourceNotFound {
		return bbs.deleteRawActualLRPKey(logger, &key)
	}

	if err != nil {
		return err
	}

	lrpStart := models.NewLRPStartRequest(desiredLRP, uint(key.Index))
	return bbs.requestLRPAuctions([]models.LRPStartRequest{lrpStart})
}

func (bbs *LRPBBS) requestStopLRPInstance(
	key models.ActualLRPKey,
	containerKey models.ActualLRPContainerKey,
) error {
	cell, err := bbs.services.CellById(containerKey.CellID)
	if err != nil {
		return err
	}

	err = bbs.cellClient.StopLRPInstance(cell.RepAddress, key, containerKey)
	if err != nil {
		return err
	}

	return nil
}

func (bbs *LRPBBS) actualLRPWithIndex(
	logger lager.Logger,
	processGuid string,
	index int,
) (*models.ActualLRP, uint64, error) {
	node, err := bbs.store.Get(shared.ActualLRPSchemaPath(processGuid, index))
	if err != nil {
		if err != storeadapter.ErrorKeyNotFound {
			logger.Error("failed-to-get-actual-lrp", err)
		}
		return nil, 0, shared.ConvertStoreError(err)
	}

	var lrp models.ActualLRP
	err = models.FromJSON(node.Value, &lrp)

	if err != nil {
		logger.Error("failed-to-unmarshal-actual-lrp", err)
		return nil, 0, err
	}

	return &lrp, node.Index, err
}

func (bbs *LRPBBS) createRawActualLRP(logger lager.Logger, lrp *models.ActualLRP) error {
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

func (bbs *LRPBBS) compareAndSwapRawActualLRP(
	logger lager.Logger,
	lrp *models.ActualLRP,
	storeIndex uint64,
) error {
	lrpForUpdate := lrp
	lrpForUpdate.ModificationTag.Increment()

	value, err := models.ToJSON(lrpForUpdate)
	if err != nil {
		logger.Error("failed-to-marshal-actual-lrp", err, lager.Data{"actual-lrp": lrpForUpdate})
		return err
	}

	err = bbs.store.CompareAndSwapByIndex(storeIndex, storeadapter.StoreNode{
		Key:   shared.ActualLRPSchemaPath(lrpForUpdate.ProcessGuid, lrpForUpdate.Index),
		Value: value,
	})
	if err != nil {
		logger.Error("failed-to-compare-and-swap-actual-lrp", err, lager.Data{"actual-lrp": lrpForUpdate})
		return shared.ConvertStoreError(err)
	}

	return nil
}

func (bbs *LRPBBS) compareAndDeleteRawActualLRP(
	logger lager.Logger,
	lrp *models.ActualLRP,
	storeIndex uint64,
) error {
	err := bbs.store.CompareAndDeleteByIndex(storeadapter.StoreNode{
		Key:   shared.ActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index),
		Index: storeIndex,
	})

	if err != nil {
		logger.Error("failed-to-compare-and-delete-actual-lrp", err, lager.Data{"actual-lrp": lrp})
		return shared.ConvertStoreError(err)
	}

	return nil
}

func (bbs *LRPBBS) deleteRawActualLRPKey(
	logger lager.Logger,
	lrp *models.ActualLRPKey,
) error {
	err := bbs.store.Delete(shared.ActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index))

	if err != nil {
		logger.Error("failed-to-delete-actual-lrp", err, lager.Data{"actual-lrp": lrp})
		return shared.ConvertStoreError(err)
	}

	return nil
}
