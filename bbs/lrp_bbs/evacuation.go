package lrp_bbs

import (
	"reflect"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

func (bbs *LRPBBS) EvacuateClaimedActualLRP(
	logger lager.Logger,
	actualLRPKey models.ActualLRPKey,
	actualLRPContainerKey models.ActualLRPContainerKey,
) error {
	_ = bbs.removeEvacuatingActualLRP(logger, actualLRPKey, actualLRPContainerKey)
	changed, err := bbs.unclaimActualLRP(logger, actualLRPKey, actualLRPContainerKey)
	if err == bbserrors.ErrStoreResourceNotFound {
		return nil
	}
	if err != nil {
		return err
	}

	if !changed {
		return nil
	}

	err = bbs.requestLRPAuctionForLRPKey(actualLRPKey)
	if err != nil {
		return err
	}

	return nil
}

func (bbs *LRPBBS) EvacuateRunningActualLRP(
	logger lager.Logger,
	actualLRPKey models.ActualLRPKey,
	actualLRPContainerKey models.ActualLRPContainerKey,
	actualLRPNetInfo models.ActualLRPNetInfo,
	evacuationTTLInSeconds uint64,
) error {
	instanceLRP, storeIndex, err := bbs.getActualLRP(actualLRPKey.ProcessGuid, actualLRPKey.Index)
	if err == bbserrors.ErrStoreResourceNotFound {
		err := bbs.removeEvacuatingActualLRP(logger, actualLRPKey, actualLRPContainerKey)
		if err == bbserrors.ErrActualLRPCannotBeRemoved {
			return bbserrors.ErrActualLRPCannotBeEvacuated
		}
		if err != nil {
			return err
		}

		return bbserrors.ErrActualLRPCannotBeEvacuated
	}
	if err != nil {
		return err
	}

	if instanceLRP.State == models.ActualLRPStateUnclaimed ||
		(instanceLRP.State == models.ActualLRPStateClaimed && instanceLRP.ActualLRPContainerKey != actualLRPContainerKey) {
		err = bbs.conditionallyEvacuateActualLRP(logger, actualLRPKey, actualLRPContainerKey, actualLRPNetInfo, evacuationTTLInSeconds)
		if err == bbserrors.ErrStoreResourceExists {
			return bbserrors.ErrActualLRPCannotBeEvacuated
		}
		if err != nil {
			return err
		}
		return nil
	}

	if (instanceLRP.State == models.ActualLRPStateClaimed || instanceLRP.State == models.ActualLRPStateRunning) &&
		instanceLRP.ActualLRPContainerKey == actualLRPContainerKey {
		err := bbs.unconditionallyEvacuateActualLRP(logger, actualLRPKey, actualLRPContainerKey, actualLRPNetInfo, evacuationTTLInSeconds)
		if err != nil {
			return err
		}

		changed, err := bbs.unclaimActualLRPWithIndex(logger, instanceLRP, storeIndex, actualLRPKey, actualLRPContainerKey)
		if err == bbserrors.ErrStoreResourceNotFound {
			return nil
		}
		if err != nil {
			return err
		}

		if !changed {
			return nil
		}

		err = bbs.requestLRPAuctionForLRPKey(actualLRPKey)
		if err != nil {
			return err
		}

		return nil
	}

	if (instanceLRP.State == models.ActualLRPStateRunning && instanceLRP.ActualLRPContainerKey != actualLRPContainerKey) ||
		instanceLRP.State == models.ActualLRPStateCrashed {
		err := bbs.removeEvacuatingActualLRP(logger, actualLRPKey, actualLRPContainerKey)
		if err == bbserrors.ErrActualLRPCannotBeRemoved {
			return bbserrors.ErrActualLRPCannotBeEvacuated
		}
		if err != nil {
			return err
		}

		return bbserrors.ErrActualLRPCannotBeEvacuated
	}

	return nil
}

func (bbs *LRPBBS) unconditionallyEvacuateActualLRP(
	logger lager.Logger,
	actualLRPKey models.ActualLRPKey,
	actualLRPContainerKey models.ActualLRPContainerKey,
	actualLRPNetInfo models.ActualLRPNetInfo,
	evacuationTTLInSeconds uint64,
) error {
	existingLRP, storeIndex, err := bbs.evacuatingActualLRPWithIndex(actualLRPKey.ProcessGuid, actualLRPKey.Index)
	if err == bbserrors.ErrStoreResourceNotFound {
		lrp := bbs.newRunningActualLRP(actualLRPKey, actualLRPContainerKey, actualLRPNetInfo)
		return bbs.createRawEvacuatingActualLRP(logger, &lrp, evacuationTTLInSeconds)
	} else if err != nil {
		return err
	}

	if existingLRP.ActualLRPKey == actualLRPKey &&
		existingLRP.ActualLRPContainerKey == actualLRPContainerKey &&
		existingLRP.Address == actualLRPNetInfo.Address &&
		reflect.DeepEqual(existingLRP.Ports, actualLRPNetInfo.Ports) &&
		existingLRP.State == models.ActualLRPStateRunning {
		return nil
	}

	lrp := *existingLRP

	lrp.State = models.ActualLRPStateRunning
	lrp.Since = bbs.clock.Now().UnixNano()
	lrp.ActualLRPContainerKey = actualLRPContainerKey
	lrp.ActualLRPNetInfo = actualLRPNetInfo
	lrp.PlacementError = ""

	return bbs.compareAndSwapRawEvacuatingActualLRP(logger, &lrp, storeIndex, evacuationTTLInSeconds)
}

func (bbs *LRPBBS) conditionallyEvacuateActualLRP(
	logger lager.Logger,
	actualLRPKey models.ActualLRPKey,
	actualLRPContainerKey models.ActualLRPContainerKey,
	actualLRPNetInfo models.ActualLRPNetInfo,
	evacuationTTLInSeconds uint64,
) error {
	existingLRP, storeIndex, err := bbs.evacuatingActualLRPWithIndex(actualLRPKey.ProcessGuid, actualLRPKey.Index)
	if err == bbserrors.ErrStoreResourceNotFound {
		lrp := bbs.newRunningActualLRP(actualLRPKey, actualLRPContainerKey, actualLRPNetInfo)
		return bbs.createRawEvacuatingActualLRP(logger, &lrp, evacuationTTLInSeconds)
	} else if err != nil {
		return err
	}

	if existingLRP.ActualLRPKey == actualLRPKey &&
		existingLRP.ActualLRPContainerKey == actualLRPContainerKey &&
		existingLRP.Address == actualLRPNetInfo.Address &&
		reflect.DeepEqual(existingLRP.Ports, actualLRPNetInfo.Ports) &&
		existingLRP.State == models.ActualLRPStateRunning {
		return nil
	}

	if existingLRP.ActualLRPKey != actualLRPKey ||
		existingLRP.ActualLRPContainerKey != actualLRPContainerKey {
		return bbserrors.ErrActualLRPCannotBeEvacuated
	}

	lrp := *existingLRP

	lrp.State = models.ActualLRPStateRunning
	lrp.Since = bbs.clock.Now().UnixNano()
	lrp.ActualLRPContainerKey = actualLRPContainerKey
	lrp.ActualLRPNetInfo = actualLRPNetInfo
	lrp.PlacementError = ""

	return bbs.compareAndSwapRawEvacuatingActualLRP(logger, &lrp, storeIndex, evacuationTTLInSeconds)
}

func (bbs *LRPBBS) createRawEvacuatingActualLRP(
	logger lager.Logger,
	lrp *models.ActualLRP,
	evacuationTTLInSeconds uint64,
) error {
	value, err := models.ToJSON(lrp)
	if err != nil {
		logger.Error("failed-to-marshal-actual-lrp", err, lager.Data{"actual-lrp": lrp})
		return err
	}

	err = bbs.store.Create(storeadapter.StoreNode{
		Key:   shared.EvacuatingActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index),
		Value: value,
		TTL:   evacuationTTLInSeconds,
	})

	if err != nil {
		logger.Error("failed-to-create-evacuating-actual-lrp", err, lager.Data{"actual-lrp": lrp})
		return shared.ConvertStoreError(err)
	}

	return nil
}

func (bbs *LRPBBS) compareAndSwapRawEvacuatingActualLRP(
	logger lager.Logger,
	lrp *models.ActualLRP,
	storeIndex uint64,
	evacuationTTLInSeconds uint64,
) error {
	value, err := models.ToJSON(lrp)
	if err != nil {
		logger.Error("failed-to-marshal-actual-lrp", err, lager.Data{"actual-lrp": lrp})
		return err
	}

	err = bbs.store.CompareAndSwapByIndex(storeIndex, storeadapter.StoreNode{
		Key:   shared.EvacuatingActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index),
		Value: value,
		TTL:   evacuationTTLInSeconds,
	})
	if err != nil {
		logger.Error("failed-to-compare-and-swap-evacuating-actual-lrp", err, lager.Data{"actual-lrp": lrp})
		return shared.ConvertStoreError(err)
	}

	return nil
}

func (bbs *LRPBBS) removeEvacuatingActualLRP(
	logger lager.Logger,
	actualLRPKey models.ActualLRPKey,
	actualLRPContainerKey models.ActualLRPContainerKey,
) error {
	lrp, storeIndex, err := bbs.evacuatingActualLRPWithIndex(actualLRPKey.ProcessGuid, actualLRPKey.Index)
	if err == bbserrors.ErrStoreResourceNotFound {
		logger.Debug("evacuating-actual-lrp-already-removed", lager.Data{"lrp-key": actualLRPKey, "container-key": actualLRPContainerKey})
		return nil
	}
	if err != nil {
		return err
	}

	if lrp.ActualLRPKey != actualLRPKey || lrp.ActualLRPContainerKey != actualLRPContainerKey {
		return bbserrors.ErrActualLRPCannotBeRemoved
	}

	err = bbs.store.CompareAndDeleteByIndex(storeadapter.StoreNode{
		Key:   shared.EvacuatingActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index),
		Index: storeIndex,
	})
	if err != nil {
		logger.Error("failed-to-compare-and-delete-actual-lrp", err, lager.Data{"actual-lrp": lrp})
		return shared.ConvertStoreError(err)
	}

	logger.Info("succeeded")
	return nil
}

func (bbs *LRPBBS) evacuatingActualLRPWithIndex(processGuid string, index int) (*models.ActualLRP, uint64, error) {
	node, err := bbs.store.Get(shared.EvacuatingActualLRPSchemaPath(processGuid, index))
	if err != nil {
		return nil, 0, shared.ConvertStoreError(err)
	}

	var lrp models.ActualLRP
	err = models.FromJSON(node.Value, &lrp)

	return &lrp, node.Index, err
}
