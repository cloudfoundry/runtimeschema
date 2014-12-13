package lrp_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/workpool"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

func (bbs *LRPBBS) CreateActualLRP(key models.ActualLRPKey) (*models.ActualLRP, error) {
	lrp := models.ActualLRP{
		ActualLRPKey:          key,
		ActualLRPContainerKey: models.NewActualLRPContainerKey("", ""),
		State: models.ActualLRPStateUnclaimed,
		Since: bbs.timeProvider.Now().UnixNano(),
	}

	return bbs.createRawActualLRP(&lrp)
}

func (bbs *LRPBBS) createRawActualLRP(lrp *models.ActualLRP) (*models.ActualLRP, error) {
	err := lrp.Validate()
	if err != nil {
		return nil, err
	}

	value, err := models.ToJSON(lrp)
	if err != nil {
		return nil, err
	}

	err = shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return bbs.store.Create(storeadapter.StoreNode{
			Key:   shared.ActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index),
			Value: value,
		})
	})

	return lrp, err
}

func (bbs *LRPBBS) ClaimActualLRP(key models.ActualLRPKey, containerKey models.ActualLRPContainerKey) (*models.ActualLRP, error) {
	lrp, index, err := bbs.getActualLRP(key.ProcessGuid, key.Index)
	if err == bbserrors.ErrStoreResourceNotFound {
		return nil, bbserrors.ErrActualLRPCannotBeClaimed
	} else if err != nil {
		return nil, err
	}

	if lrp.ActualLRPKey == key &&
		lrp.ActualLRPContainerKey == containerKey &&
		lrp.State == models.ActualLRPStateClaimed {
		return lrp, nil
	}

	if !lrp.AllowsTransitionTo(key, containerKey, models.ActualLRPStateClaimed) {
		return lrp, bbserrors.ErrActualLRPCannotBeClaimed
	}

	lrp.Since = bbs.timeProvider.Now().UnixNano()
	lrp.State = models.ActualLRPStateClaimed
	lrp.ActualLRPContainerKey = containerKey
	lrp.ActualLRPNetInfo = models.NewActualLRPNetInfo("", nil)

	err = lrp.Validate()
	if err != nil {
		return nil, err
	}

	value, err := models.ToJSON(lrp)
	if err != nil {
		return nil, err
	}

	return lrp, shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return bbs.store.CompareAndSwapByIndex(index, storeadapter.StoreNode{
			Key:   shared.ActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index),
			Value: value,
		})
	})
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
		Since:                 bbs.timeProvider.Now().UnixNano(),
		State:                 models.ActualLRPStateRunning,
	}
}

func (bbs *LRPBBS) StartActualLRP(
	key models.ActualLRPKey,
	containerKey models.ActualLRPContainerKey,
	netInfo models.ActualLRPNetInfo,
) (*models.ActualLRP, error) {
	lrp, index, err := bbs.getActualLRP(key.ProcessGuid, key.Index)
	if err == bbserrors.ErrStoreResourceNotFound {
		newLRP := bbs.newRunningActualLRP(key, containerKey, netInfo)
		createdLRP, err := bbs.createRawActualLRP(&newLRP)
		if err != nil {
			return nil, err
		}

		return createdLRP, nil

	} else if err != nil {
		return nil, err
	}

	if lrp.ActualLRPKey == key &&
		lrp.ActualLRPContainerKey == containerKey &&
		lrp.State == models.ActualLRPStateRunning {
		return lrp, nil
	}

	if !lrp.AllowsTransitionTo(key, containerKey, models.ActualLRPStateRunning) {
		return lrp, bbserrors.ErrActualLRPCannotBeStarted
	}

	lrp.State = models.ActualLRPStateRunning
	lrp.Since = bbs.timeProvider.Now().UnixNano()
	lrp.ActualLRPContainerKey = containerKey
	lrp.ActualLRPNetInfo = netInfo

	err = lrp.Validate()
	if err != nil {
		return nil, err
	}

	value, err := models.ToJSON(lrp)
	if err != nil {
		return nil, err
	}

	err = shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return bbs.store.CompareAndSwapByIndex(index, storeadapter.StoreNode{
			Key:   shared.ActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index),
			Value: value,
		})
	})

	if err != nil {
		return nil, err
	}

	return lrp, nil
}

func (bbs *LRPBBS) RemoveActualLRP(key models.ActualLRPKey, containerKey models.ActualLRPContainerKey) error {
	return shared.RetryIndefinitelyOnStoreTimeout(func() error {
		lrp, storeIndex, err := bbs.getActualLRP(key.ProcessGuid, key.Index)
		if err != nil {
			return err
		}

		if lrp.ActualLRPKey == key &&
			lrp.ActualLRPContainerKey == containerKey {
			return bbs.store.CompareAndDeleteByIndex(storeadapter.StoreNode{
				Key:   shared.ActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index),
				Index: storeIndex,
			})
		} else {
			return bbserrors.ErrStoreComparisonFailed
		}
	})
}

func (bbs *LRPBBS) RetireActualLRPs(lrps []models.ActualLRP, logger lager.Logger) error {
	pool := workpool.NewWorkPool(workerPoolSize)

	errs := make(chan error, len(lrps))
	for _, lrp := range lrps {
		lrp := lrp
		pool.Submit(func() {
			errs <- bbs.retireActualLRP(lrp, logger)
		})
	}

	pool.Stop()

	for i := 0; i < len(lrps); i++ {
		err := <-errs
		if err != nil {
			return err
		}
	}

	return nil

}

func (bbs *LRPBBS) retireActualLRP(lrp models.ActualLRP, logger lager.Logger) error {
	var err error

	if lrp.State == models.ActualLRPStateUnclaimed {
		err = bbs.RemoveActualLRP(lrp.ActualLRPKey, lrp.ActualLRPContainerKey)
	} else {
		err = bbs.RequestStopLRPInstance(lrp)
	}

	if err != nil {
		logger.Error("request-remove-instance-failed", err, lager.Data{
			"instance-guid": lrp.InstanceGuid,
		})
		return err
	}

	return nil
}

func (bbs *LRPBBS) getActualLRP(processGuid string, index int) (*models.ActualLRP, uint64, error) {
	var node storeadapter.StoreNode
	err := shared.RetryIndefinitelyOnStoreTimeout(func() error {
		var err error
		node, err = bbs.store.Get(shared.ActualLRPSchemaPath(processGuid, index))
		return err
	})

	if err != nil {
		return nil, 0, err
	}

	var lrp models.ActualLRP
	err = models.FromJSON(node.Value, &lrp)

	return &lrp, node.Index, err

}
