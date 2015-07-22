package lrp_bbs

import (
	"fmt"
	"reflect"
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/workpool"
	"github.com/nu7hatch/gouuid"
	"github.com/pivotal-golang/lager"
)

const CrashResetTimeout = 5 * time.Minute
const retireActualThrottlerSize = 20
const RetireActualLRPRetryAttempts = 5

func (bbs *LRPBBS) LegacyClaimActualLRP(
	logger lager.Logger,
	key models.ActualLRPKey,
	instanceKey models.ActualLRPInstanceKey,
) error {
	logger = logger.Session("claim-actual-lrp", lager.Data{"lrp-key": key, "instance-key": instanceKey})
	logger.Info("starting")

	lrp, storeIndex, err := bbs.actualLRPRepo.ActualLRPWithIndex(logger, key.ProcessGuid, key.Index)
	if err == bbserrors.ErrStoreResourceNotFound {
		logger.Error("failed-actual-lrp-not-found", err)
		return bbserrors.ErrActualLRPCannotBeClaimed
	} else if err != nil {
		logger.Error("failed-to-get-actual-lrp", err)
		return err
	}

	if lrp.ActualLRPKey == key &&
		lrp.ActualLRPInstanceKey == instanceKey &&
		lrp.State == models.ActualLRPStateClaimed {
		logger.Info("succeeded")
		return nil
	}

	if !lrp.AllowsTransitionTo(key, instanceKey, models.ActualLRPStateClaimed) {
		logger.Error("failed-to-transition-actual-lrp-to-claimed", nil)
		return bbserrors.ErrActualLRPCannotBeClaimed
	}

	lrp.Since = bbs.clock.Now().UnixNano()
	lrp.State = models.ActualLRPStateClaimed
	lrp.ActualLRPInstanceKey = instanceKey
	lrp.ActualLRPNetInfo = models.ActualLRPNetInfo{}
	lrp.PlacementError = ""

	err = bbs.actualLRPRepo.CompareAndSwapRawActualLRP(logger, lrp, storeIndex)
	if err != nil {
		return err
	}

	logger.Info("succeeded")
	return nil
}

func (bbs *LRPBBS) StartActualLRP(
	logger lager.Logger,
	key models.ActualLRPKey,
	instanceKey models.ActualLRPInstanceKey,
	netInfo models.ActualLRPNetInfo,
) error {
	logger = logger.Session("start-actual-lrp", lager.Data{"lrp-key": key, "instance-key": instanceKey})
	logger.Info("starting")
	lrp, storeIndex, err := bbs.actualLRPRepo.ActualLRPWithIndex(logger, key.ProcessGuid, key.Index)
	if err == bbserrors.ErrStoreResourceNotFound {
		newLRP, err := bbs.newRunningActualLRP(key, instanceKey, netInfo)
		if err != nil {
			return err
		}

		return bbs.actualLRPRepo.CreateRawActualLRP(logger, &newLRP)
	} else if err != nil {
		logger.Error("failed-to-get-actual-lrp", err)
		return err
	}

	if lrp.ActualLRPKey == key &&
		lrp.ActualLRPInstanceKey == instanceKey &&
		lrp.Address == netInfo.Address &&
		reflect.DeepEqual(lrp.Ports, netInfo.Ports) &&
		lrp.State == models.ActualLRPStateRunning {
		logger.Info("succeeded")
		return nil
	}

	if !lrp.AllowsTransitionTo(key, instanceKey, models.ActualLRPStateRunning) {
		logger.Error("failed-to-transition-actual-lrp-to-started", nil)
		return bbserrors.ErrActualLRPCannotBeStarted
	}

	lrp.State = models.ActualLRPStateRunning
	lrp.Since = bbs.clock.Now().UnixNano()
	lrp.ActualLRPInstanceKey = instanceKey
	lrp.ActualLRPNetInfo = netInfo
	lrp.PlacementError = ""

	err = bbs.actualLRPRepo.CompareAndSwapRawActualLRP(logger, lrp, storeIndex)
	if err != nil {
		return err
	}

	logger.Info("succeeded")
	return nil
}

func (bbs *LRPBBS) CrashActualLRP(
	logger lager.Logger,
	key models.ActualLRPKey,
	instanceKey models.ActualLRPInstanceKey,
	crashReason string,
) error {
	logger = logger.Session("crash-actual-lrp", lager.Data{"lrp-key": key, "instance-key": instanceKey})
	logger.Info("starting")

	lrp, storeIndex, err := bbs.actualLRPRepo.ActualLRPWithIndex(logger, key.ProcessGuid, key.Index)
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

	logger.Debug("retrieved-lrp")
	if !lrp.AllowsTransitionTo(key, instanceKey, models.ActualLRPStateCrashed) {
		err := fmt.Errorf("cannot transition crashed lrp from state %s to state %s", lrp.State, models.ActualLRPStateCrashed)
		logger.Error("failed-to-transition-actual", err)
		return bbserrors.ErrActualLRPCannotBeCrashed
	}

	if lrp.State == models.ActualLRPStateUnclaimed || lrp.State == models.ActualLRPStateCrashed ||
		((lrp.State == models.ActualLRPStateClaimed || lrp.State == models.ActualLRPStateRunning) && lrp.ActualLRPInstanceKey != instanceKey) {
		return bbserrors.ErrActualLRPCannotBeCrashed
	}

	lrp.State = models.ActualLRPStateCrashed
	lrp.Since = bbs.clock.Now().UnixNano()
	lrp.CrashCount = newCrashCount
	lrp.ActualLRPInstanceKey = models.ActualLRPInstanceKey{}
	lrp.ActualLRPNetInfo = models.EmptyActualLRPNetInfo()
	lrp.CrashReason = crashReason

	var immediateRestart bool
	if lrp.ShouldRestartImmediately(models.NewDefaultRestartCalculator()) {
		lrp.State = models.ActualLRPStateUnclaimed
		immediateRestart = true
	}

	err = bbs.actualLRPRepo.CompareAndSwapRawActualLRP(logger, lrp, storeIndex)
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
	instanceKey models.ActualLRPInstanceKey,
) error {
	lrp, storeIndex, err := bbs.actualLRPRepo.ActualLRPWithIndex(logger, key.ProcessGuid, key.Index)
	if err != nil {
		return err
	}

	if lrp.ActualLRPKey != key || lrp.ActualLRPInstanceKey != instanceKey {
		logger.Error("failed-to-match-existing-actual-lrp", err, lager.Data{"existing-actual-lrp": lrp})
		return bbserrors.ErrStoreComparisonFailed
	}

	return bbs.removeActualLRPWithIndex(logger, key, storeIndex)
}

func (bbs *LRPBBS) removeActualLRPWithIndex(
	logger lager.Logger,
	key models.ActualLRPKey,
	index uint64,
) error {
	logger = logger.Session("remove-actual-lrp", lager.Data{"lrp-key": key})
	logger.Info("starting")

	err := bbs.actualLRPRepo.CompareAndDeleteRawActualLRPKey(logger, &key, index)
	if err != nil {
		return err
	}

	logger.Info("succeeded")
	return nil
}

func (bbs *LRPBBS) RetireActualLRPs(
	logger lager.Logger,
	lrpKeys []models.ActualLRPKey,
) {
	logger = logger.Session("retire-actual-lrps")

	works := make([]func(), len(lrpKeys))

	for i, lrpKey := range lrpKeys {
		lrpKey := lrpKey

		works[i] = func() {
			err := bbs.retireActualLRP(lrpKey, logger)
			if err != nil {
				logger.Error("failed-to-retire", err, lager.Data{
					"lrp-key": lrpKey,
				})
			}
		}
	}

	throttler, err := workpool.NewThrottler(retireActualThrottlerSize, works)
	if err != nil {
		logger.Error("failed-constructing-throttler", err, lager.Data{"max-workers": retireActualThrottlerSize, "num-works": len(works)})
		return
	}

	throttler.Work()
}

func (bbs *LRPBBS) FailActualLRP(
	logger lager.Logger,
	key models.ActualLRPKey,
	errorMessage string,
) error {
	logger = logger.Session("set-placement-error-actual-lrp", lager.Data{"lrp-key": key})
	logger.Info("starting")

	lrp, storeIndex, err := bbs.actualLRPRepo.ActualLRPWithIndex(logger, key.ProcessGuid, key.Index)
	if err == bbserrors.ErrStoreResourceNotFound {
		logger.Error("failed-actual-lrp-not-found", err)
		return bbserrors.ErrActualLRPCannotBeFailed
	} else if err != nil {
		logger.Error("failed-to-get-actual-lrp", err)
		return err
	}

	if lrp.ActualLRPKey == key && lrp.State != models.ActualLRPStateUnclaimed {
		logger.Error("failed-to-set-placement-error", bbserrors.ErrActualLRPCannotBeFailed)
		return bbserrors.ErrActualLRPCannotBeFailed
	}

	lrp.Since = bbs.clock.Now().UnixNano()
	lrp.PlacementError = errorMessage

	err = bbs.actualLRPRepo.CompareAndSwapRawActualLRP(logger, lrp, storeIndex)
	if err != nil {
		return err
	}

	logger.Info("succeeded")
	return nil
}

func (bbs *LRPBBS) createAndStartActualLRPsForDesired(logger lager.Logger, lrp models.DesiredLRP, indices []uint) {
	createdIndices := bbs.actualLRPRepo.CreateActualLRPsForDesired(logger, lrp, indices)
	start := models.NewLRPStartRequest(lrp, createdIndices...)

	err := bbs.requestLRPAuctions([]models.LRPStartRequest{start})
	if err != nil {
		logger.Error("failed-to-request-start-auctions", err, lager.Data{"lrp-start": start})
	}
}

type stateChange bool

const (
	stateDidChange    stateChange = true
	stateDidNotChange stateChange = false
)

func (bbs *LRPBBS) unclaimActualLRP(
	logger lager.Logger,
	actualLRPKey models.ActualLRPKey,
	actualLRPInstanceKey models.ActualLRPInstanceKey,
) (stateChange, error) {
	logger = logger.Session("unclaim-actual-lrp")
	logger.Info("starting")

	lrp, storeIndex, err := bbs.actualLRPRepo.ActualLRPWithIndex(logger, actualLRPKey.ProcessGuid, actualLRPKey.Index)
	if err != nil {
		logger.Error("failed-to-get-actual-lrp", err)
		return stateDidNotChange, err
	}

	changed, err := bbs.unclaimActualLRPWithIndex(logger, lrp, storeIndex, actualLRPKey, actualLRPInstanceKey)
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
	actualLRPInstanceKey models.ActualLRPInstanceKey,
) (stateChange, error) {
	if lrp.ActualLRPKey != actualLRPKey {
		logger.Error("failed-actual-lrp-key-differs", bbserrors.ErrActualLRPCannotBeUnclaimed)
		return stateDidNotChange, bbserrors.ErrActualLRPCannotBeUnclaimed
	}

	if lrp.State == models.ActualLRPStateUnclaimed {
		logger.Info("succeeded")
		return stateDidNotChange, nil
	}

	if lrp.ActualLRPInstanceKey != actualLRPInstanceKey {
		logger.Error("failed-actual-lrp-instance-key-differs", bbserrors.ErrActualLRPCannotBeUnclaimed)
		return stateDidNotChange, bbserrors.ErrActualLRPCannotBeUnclaimed
	}

	lrp.Since = bbs.clock.Now().UnixNano()
	lrp.State = models.ActualLRPStateUnclaimed
	lrp.ActualLRPInstanceKey = models.ActualLRPInstanceKey{}
	lrp.ActualLRPNetInfo = models.EmptyActualLRPNetInfo()

	err := bbs.actualLRPRepo.CompareAndSwapRawActualLRP(logger, lrp, storeIndex)

	if err != nil {
		return stateDidNotChange, err
	}

	return stateDidChange, nil
}

func (bbs *LRPBBS) retireActualLRP(actualLRPKey models.ActualLRPKey, logger lager.Logger) error {
	var err error
	var lrp *models.ActualLRP
	var index uint64

	for i := 0; i < RetireActualLRPRetryAttempts; i++ {
		lrp, index, err = bbs.actualLRPRepo.ActualLRPWithIndex(logger, actualLRPKey.ProcessGuid, actualLRPKey.Index)
		if err != nil {
			logger.Error("failed-fetching-actual-lrp", err, lager.Data{
				"actual-lrp-key": actualLRPKey,
			})
			break
		}

		switch lrp.State {
		case models.ActualLRPStateUnclaimed, models.ActualLRPStateCrashed:
			err = bbs.removeActualLRPWithIndex(logger, lrp.ActualLRPKey, index)
		default:
			err = bbs.requestStopLRPInstanceWithIndex(logger, lrp.ActualLRPKey, lrp.ActualLRPInstanceKey, index)
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
	instanceKey models.ActualLRPInstanceKey,
	netInfo models.ActualLRPNetInfo,
) (models.ActualLRP, error) {
	guid, err := uuid.NewV4()
	if err != nil {
		return models.ActualLRP{}, err
	}

	actualLRP := models.ActualLRP{
		ActualLRPKey:         key,
		ActualLRPInstanceKey: instanceKey,
		ActualLRPNetInfo:     netInfo,
		Since:                bbs.clock.Now().UnixNano(),
		State:                models.ActualLRPStateRunning,
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
	desiredLRP, err := bbs.LegacyDesiredLRPByProcessGuid(logger, key.ProcessGuid)
	if err == bbserrors.ErrStoreResourceNotFound {
		return bbs.actualLRPRepo.DeleteRawActualLRPKey(logger, &key)
	}

	if err != nil {
		return err
	}

	lrpStart := models.NewLRPStartRequest(desiredLRP, uint(key.Index))
	return bbs.requestLRPAuctions([]models.LRPStartRequest{lrpStart})
}

func (bbs *LRPBBS) requestStopLRPInstanceWithIndex(
	logger lager.Logger,
	key models.ActualLRPKey,
	instanceKey models.ActualLRPInstanceKey,
	index uint64,
) error {
	cell, err := bbs.services.CellById(instanceKey.CellID)
	if err != nil {
		if err == bbserrors.ErrStoreResourceNotFound {
			return bbs.removeActualLRPWithIndex(logger, key, index)
		}
		return err
	}

	logger.Info("stopping-lrp-instance", lager.Data{
		"actual-lrp-key": key,
	})
	err = bbs.cellClient.StopLRPInstance(cell.RepAddress, key, instanceKey)
	if err != nil {
		return err
	}

	return nil
}
