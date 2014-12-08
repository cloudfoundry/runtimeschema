package lrp_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
)

func (bbs *LRPBBS) CreateActualLRP(lrp models.ActualLRP) (*models.ActualLRP, error) {
	lrp.State = models.ActualLRPStateUnclaimed
	lrp.CellID = ""
	lrp.InstanceGuid = ""

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

	return lrp, nil
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

	switch existingLRP.State {
	case models.ActualLRPStateUnclaimed, models.ActualLRPStateRunning:
		if existingLRP.State == models.ActualLRPStateRunning && (existingLRP.CellID != lrp.CellID || existingLRP.InstanceGuid != lrp.InstanceGuid) {
			return existingLRP, bbserrors.ErrActualLRPCannotBeClaimed
		}

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
	case models.ActualLRPStateClaimed:
		if existingLRP.CellID != lrp.CellID || existingLRP.InstanceGuid != lrp.InstanceGuid {
			return existingLRP, bbserrors.ErrActualLRPCannotBeClaimed
		}

		return existingLRP, nil
	default:
		return existingLRP, bbserrors.ErrActualLRPCannotBeClaimed
	}
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

	switch existingLRP.State {
	case models.ActualLRPStateRunning:
		if existingLRP.CellID != lrp.CellID || existingLRP.InstanceGuid != lrp.InstanceGuid {
			return nil, bbserrors.ErrActualLRPCannotBeStarted
		}

		return existingLRP, nil

	case models.ActualLRPStateUnclaimed, models.ActualLRPStateClaimed:
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

	default:
		return nil, bbserrors.ErrActualLRPCannotBeStarted
	}
}

func (bbs *LRPBBS) RemoveActualLRP(lrp models.ActualLRP) error {
	return shared.RetryIndefinitelyOnStoreTimeout(func() error {
		storedLRP, index, err := bbs.getActualLRP(lrp.ProcessGuid, lrp.Index)
		if err != nil {
			return err
		}

		if lrp.ProcessGuid != storedLRP.ProcessGuid || lrp.Index != storedLRP.Index || lrp.Domain != storedLRP.Domain ||
			lrp.InstanceGuid != storedLRP.InstanceGuid || lrp.CellID != storedLRP.CellID || lrp.State != storedLRP.State {
			return bbserrors.ErrStoreComparisonFailed
		}

		return bbs.store.CompareAndDeleteByIndex(storeadapter.StoreNode{
			Key:   shared.ActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index),
			Index: index,
		})
	})
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
