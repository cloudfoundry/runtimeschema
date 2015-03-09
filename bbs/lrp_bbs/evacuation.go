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
) (shared.ContainerRetainment, error) {
	logger = logger.Session("evacuating-claimed-actual-lrp", lager.Data{
		"lrp-key":           actualLRPKey,
		"lrp-container-key": actualLRPContainerKey,
	})
	logger.Info("started")

	bbs.RemoveEvacuatingActualLRP(logger, actualLRPKey, actualLRPContainerKey)
	changed, err := bbs.unclaimActualLRP(logger, actualLRPKey, actualLRPContainerKey)
	if err == bbserrors.ErrStoreResourceNotFound {
		return shared.DeleteContainer, nil
	}
	if err != nil {
		return shared.DeleteContainer, err
	}

	if !changed {
		return shared.DeleteContainer, nil
	}

	logger.Info("requesting-start-lrp-auction")
	err = bbs.requestLRPAuctionForLRPKey(logger, actualLRPKey)
	if err != nil {
		logger.Error("failed-requesting-start-lrp-auction", err)
		return shared.DeleteContainer, err
	}
	logger.Info("succeeded-requesting-start-lrp-auction")

	logger.Info("succeeded")
	return shared.DeleteContainer, nil
}

func (bbs *LRPBBS) EvacuateRunningActualLRP(
	logger lager.Logger,
	actualLRPKey models.ActualLRPKey,
	actualLRPContainerKey models.ActualLRPContainerKey,
	actualLRPNetInfo models.ActualLRPNetInfo,
	evacuationTTLInSeconds uint64,
) (shared.ContainerRetainment, error) {
	logger = logger.Session("evacuating-running-actual-lrp", lager.Data{
		"lrp-key":           actualLRPKey,
		"lrp-container-key": actualLRPContainerKey,
	})
	logger.Info("started")

	instanceLRP, storeIndex, err := bbs.actualLRPWithIndex(logger, actualLRPKey.ProcessGuid, actualLRPKey.Index)
	if err == bbserrors.ErrStoreResourceNotFound {
		err := bbs.RemoveEvacuatingActualLRP(logger, actualLRPKey, actualLRPContainerKey)
		if err == bbserrors.ErrActualLRPCannotBeRemoved {
			return shared.DeleteContainer, nil
		} else if err != nil {
			return shared.KeepContainer, err
		}

		return shared.DeleteContainer, nil

	} else if err != nil {
		return shared.KeepContainer, err
	}

	if (instanceLRP.State == models.ActualLRPStateUnclaimed && instanceLRP.PlacementError == "") ||
		(instanceLRP.State == models.ActualLRPStateClaimed && instanceLRP.ActualLRPContainerKey != actualLRPContainerKey) {
		err = bbs.conditionallyEvacuateActualLRP(logger, actualLRPKey, actualLRPContainerKey, actualLRPNetInfo, evacuationTTLInSeconds)
		if err == bbserrors.ErrStoreResourceExists || err == bbserrors.ErrActualLRPCannotBeEvacuated {
			return shared.DeleteContainer, nil
		}
		if err != nil {
			return shared.KeepContainer, err
		}
		logger.Info("succeeded")
		return shared.KeepContainer, nil
	}

	if (instanceLRP.State == models.ActualLRPStateClaimed || instanceLRP.State == models.ActualLRPStateRunning) &&
		instanceLRP.ActualLRPContainerKey == actualLRPContainerKey {
		err := bbs.unconditionallyEvacuateActualLRP(logger, actualLRPKey, actualLRPContainerKey, actualLRPNetInfo, evacuationTTLInSeconds)
		if err != nil {
			return shared.KeepContainer, err
		}

		changed, err := bbs.unclaimActualLRPWithIndex(logger, instanceLRP, storeIndex, actualLRPKey, actualLRPContainerKey)
		if err != nil {
			return shared.KeepContainer, err
		}

		if !changed {
			logger.Info("succeeded")
			return shared.KeepContainer, nil
		}

		logger.Info("requesting-start-lrp-auction")
		err = bbs.requestLRPAuctionForLRPKey(logger, actualLRPKey)
		if err != nil {
			logger.Error("failed-requesting-start-lrp-auction", err)
		} else {
			logger.Info("succeeded-requesting-start-lrp-auction")
			logger.Info("succeeded")
		}
		return shared.KeepContainer, err
	}

	if (instanceLRP.State == models.ActualLRPStateUnclaimed && instanceLRP.PlacementError != "") ||
		(instanceLRP.State == models.ActualLRPStateRunning && instanceLRP.ActualLRPContainerKey != actualLRPContainerKey) ||
		instanceLRP.State == models.ActualLRPStateCrashed {
		err := bbs.RemoveEvacuatingActualLRP(logger, actualLRPKey, actualLRPContainerKey)
		if err == bbserrors.ErrActualLRPCannotBeRemoved {
			return shared.DeleteContainer, nil
		}
		if err != nil {
			return shared.KeepContainer, err
		}

		return shared.DeleteContainer, nil
	}

	logger.Info("succeeded")
	return shared.KeepContainer, nil
}

func (bbs *LRPBBS) EvacuateStoppedActualLRP(
	logger lager.Logger,
	actualLRPKey models.ActualLRPKey,
	actualLRPContainerKey models.ActualLRPContainerKey,
) (shared.ContainerRetainment, error) {
	logger = logger.Session("evacuating-stopped-actual-lrp", lager.Data{
		"lrp-key":           actualLRPKey,
		"lrp-container-key": actualLRPContainerKey,
	})
	logger.Info("started")

	_ = bbs.RemoveEvacuatingActualLRP(logger, actualLRPKey, actualLRPContainerKey)
	err := bbs.RemoveActualLRP(logger, actualLRPKey, actualLRPContainerKey)
	if err == bbserrors.ErrStoreResourceNotFound {
		return shared.DeleteContainer, nil
	} else if err == bbserrors.ErrStoreComparisonFailed {
		return shared.DeleteContainer, bbserrors.ErrActualLRPCannotBeRemoved
	}
	if err != nil {
		return shared.DeleteContainer, err
	}

	logger.Info("succeeded")
	return shared.DeleteContainer, nil
}

func (bbs *LRPBBS) EvacuateCrashedActualLRP(
	logger lager.Logger,
	actualLRPKey models.ActualLRPKey,
	actualLRPContainerKey models.ActualLRPContainerKey,
) (shared.ContainerRetainment, error) {
	logger = logger.Session("evacuating-crashed-actual-lrp", lager.Data{
		"lrp-key":           actualLRPKey,
		"lrp-container-key": actualLRPContainerKey,
	})
	logger.Info("started")

	_ = bbs.RemoveEvacuatingActualLRP(logger, actualLRPKey, actualLRPContainerKey)
	err := bbs.CrashActualLRP(logger, actualLRPKey, actualLRPContainerKey)

	if err == bbserrors.ErrStoreResourceNotFound {
		return shared.DeleteContainer, nil
	} else if err != nil {
		return shared.DeleteContainer, err
	}

	logger.Info("succeeded")
	return shared.DeleteContainer, nil
}

func (bbs *LRPBBS) RemoveEvacuatingActualLRP(
	logger lager.Logger,
	actualLRPKey models.ActualLRPKey,
	actualLRPContainerKey models.ActualLRPContainerKey,
) error {
	logger = logger.Session("removing-evacuating-actual-lrp", lager.Data{
		"lrp-key":           actualLRPKey,
		"lrp-container-key": actualLRPContainerKey,
	})
	logger.Info("started")

	lrp, storeIndex, err := bbs.evacuatingActualLRPWithIndex(logger, actualLRPKey.ProcessGuid, actualLRPKey.Index)
	if err == bbserrors.ErrStoreResourceNotFound {
		logger.Debug("evacuating-actual-lrp-already-removed", lager.Data{"lrp-key": actualLRPKey, "container-key": actualLRPContainerKey})
		return nil
	} else if err != nil {
		logger.Error("failed-to-get-evacuating-actual-lrp", err)
		return err
	}

	if lrp.ActualLRPKey != actualLRPKey || lrp.ActualLRPContainerKey != actualLRPContainerKey {
		return bbserrors.ErrActualLRPCannotBeRemoved
	}

	err = bbs.compareAndDeleteRawEvacuatingActualLRP(logger, lrp, storeIndex)
	if err != nil {
		return err
	}

	logger.Info("succeeded")
	return nil
}

func (bbs *LRPBBS) unconditionallyEvacuateActualLRP(
	logger lager.Logger,
	actualLRPKey models.ActualLRPKey,
	actualLRPContainerKey models.ActualLRPContainerKey,
	actualLRPNetInfo models.ActualLRPNetInfo,
	evacuationTTLInSeconds uint64,
) error {
	existingLRP, storeIndex, err := bbs.evacuatingActualLRPWithIndex(logger, actualLRPKey.ProcessGuid, actualLRPKey.Index)
	if err == bbserrors.ErrStoreResourceNotFound {
		lrp, err := bbs.newRunningActualLRP(actualLRPKey, actualLRPContainerKey, actualLRPNetInfo)
		if err != nil {
			return err
		}

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
	lrp.ModificationTag.Increment()

	return bbs.compareAndSwapRawEvacuatingActualLRP(logger, &lrp, storeIndex, evacuationTTLInSeconds)
}

func (bbs *LRPBBS) conditionallyEvacuateActualLRP(
	logger lager.Logger,
	actualLRPKey models.ActualLRPKey,
	actualLRPContainerKey models.ActualLRPContainerKey,
	actualLRPNetInfo models.ActualLRPNetInfo,
	evacuationTTLInSeconds uint64,
) error {
	existingLRP, storeIndex, err := bbs.evacuatingActualLRPWithIndex(logger, actualLRPKey.ProcessGuid, actualLRPKey.Index)
	if err == bbserrors.ErrStoreResourceNotFound {
		lrp, err := bbs.newRunningActualLRP(actualLRPKey, actualLRPContainerKey, actualLRPNetInfo)
		if err != nil {
			return err
		}

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
	lrp.ModificationTag.Increment()

	return bbs.compareAndSwapRawEvacuatingActualLRP(logger, &lrp, storeIndex, evacuationTTLInSeconds)
}

func (bbs *LRPBBS) evacuatingActualLRPWithIndex(
	logger lager.Logger,
	processGuid string,
	index int,
) (*models.ActualLRP, uint64, error) {
	node, err := bbs.store.Get(shared.EvacuatingActualLRPSchemaPath(processGuid, index))
	if err != nil {
		if err != storeadapter.ErrorKeyNotFound {
			logger.Error("failed-to-get-evacuating-actual-lrp", err)
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

func (bbs *LRPBBS) createRawEvacuatingActualLRP(
	logger lager.Logger,
	lrp *models.ActualLRP,
	evacuationTTLInSeconds uint64,
) error {
	lrpForUpdate := lrp
	lrpForUpdate.ModificationTag.Increment()

	value, err := models.ToJSON(lrpForUpdate)
	if err != nil {
		logger.Error("failed-to-marshal-actual-lrp", err, lager.Data{"actual-lrp": lrpForUpdate})
		return err
	}

	err = bbs.store.Create(storeadapter.StoreNode{
		Key:   shared.EvacuatingActualLRPSchemaPath(lrpForUpdate.ProcessGuid, lrpForUpdate.Index),
		Value: value,
		TTL:   evacuationTTLInSeconds,
	})

	if err != nil {
		logger.Error("failed-to-create-evacuating-actual-lrp", err, lager.Data{"actual-lrp": lrpForUpdate})
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

func (bbs *LRPBBS) compareAndDeleteRawEvacuatingActualLRP(
	logger lager.Logger,
	lrp *models.ActualLRP,
	storeIndex uint64,
) error {
	err := bbs.store.CompareAndDeleteByIndex(storeadapter.StoreNode{
		Key:   shared.EvacuatingActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index),
		Index: storeIndex,
	})

	if err != nil {
		logger.Error("failed-to-compare-and-delete-evacuating-actual-lrp", err, lager.Data{"actual-lrp": lrp})
		return shared.ConvertStoreError(err)
	}

	return nil
}
