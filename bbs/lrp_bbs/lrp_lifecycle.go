package lrp_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/workpool"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

func (bbs *LRPBBS) CreateActualLRP(lrp models.ActualLRP) (*models.ActualLRP, error) {
	lrp.State = models.ActualLRPStateUnclaimed
	lrp.CellID = ""

	err := lrp.Validate()
	if err != nil {
		return nil, err
	}

	lrp.Since = bbs.timeProvider.Now().UnixNano()

	return bbs.CreateRawActualLRP(&lrp)
}

func (bbs *LRPBBS) CreateRawActualLRP(lrp *models.ActualLRP) (*models.ActualLRP, error) {
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

func (bbs *LRPBBS) ClaimActualLRP(lrp models.ActualLRP) (*models.ActualLRP, error) {
	lrp.State = models.ActualLRPStateClaimed
	lrp.Since = bbs.timeProvider.Now().UnixNano()

	err := lrp.Validate()
	if err != nil {
		return nil, err
	}

	existingLRP, index, err := bbs.getActualLRP(lrp.ProcessGuid, lrp.Index)
	if err == bbserrors.ErrStoreResourceNotFound {
		return nil, bbserrors.ErrActualLRPCannotBeClaimed
	} else if err != nil {
		return nil, err
	}

	if existingLRP.IsEquivalentTo(lrp) {
		return existingLRP, nil
	}

	if existingLRP.AllowsTransitionTo(lrp) {
		value, err := models.ToJSON(lrp)
		if err != nil {
			return nil, err
		}

		return &lrp, shared.RetryIndefinitelyOnStoreTimeout(func() error {
			return bbs.store.CompareAndSwapByIndex(index, storeadapter.StoreNode{
				Key:   shared.ActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index),
				Value: value,
			})
		})
	}

	return existingLRP, bbserrors.ErrActualLRPCannotBeClaimed
}

func (bbs *LRPBBS) StartActualLRP(lrp models.ActualLRP) (*models.ActualLRP, error) {
	lrp.State = models.ActualLRPStateRunning
	lrp.Since = bbs.timeProvider.Now().UnixNano()

	err := lrp.Validate()
	if err != nil {
		return nil, err
	}

	existingLRP, index, err := bbs.getActualLRP(lrp.ProcessGuid, lrp.Index)
	if err == bbserrors.ErrStoreResourceNotFound {
		createdLRP, err := bbs.CreateRawActualLRP(&lrp)

		if err == nil {
			return createdLRP, nil
		}

		if err == bbserrors.ErrStoreResourceExists {
			existingLRP, index, err = bbs.getActualLRP(lrp.ProcessGuid, lrp.Index)
		}

		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	if existingLRP.IsEquivalentTo(lrp) {
		return existingLRP, nil
	}

	if existingLRP.AllowsTransitionTo(lrp) {
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
			return existingLRP, err
		}

		return &lrp, nil
	}

	return existingLRP, bbserrors.ErrActualLRPCannotBeStarted
}

func (bbs *LRPBBS) RemoveActualLRP(lrp models.ActualLRP) error {
	return shared.RetryIndefinitelyOnStoreTimeout(func() error {
		storedLRP, index, err := bbs.getActualLRP(lrp.ProcessGuid, lrp.Index)
		if err != nil {
			return err
		}

		if !storedLRP.IsEquivalentTo(lrp) {
			return bbserrors.ErrStoreComparisonFailed
		}

		return bbs.store.CompareAndDeleteByIndex(storeadapter.StoreNode{
			Key:   shared.ActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index),
			Index: index,
		})
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
		err = bbs.RemoveActualLRP(lrp)
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
