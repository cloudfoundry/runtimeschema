package lrp_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/services_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/cell_client"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/timeprovider"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

type LRPBBS struct {
	store        storeadapter.StoreAdapter
	timeProvider timeprovider.TimeProvider
	cellClient   cell_client.Client
	services     *services_bbs.ServicesBBS
	logger       lager.Logger
}

func New(store storeadapter.StoreAdapter, timeProvider timeprovider.TimeProvider, cellClient cell_client.Client, services *services_bbs.ServicesBBS, logger lager.Logger) *LRPBBS {
	return &LRPBBS{
		store:        store,
		timeProvider: timeProvider,
		cellClient:   cellClient,
		services:     services,
		logger:       logger,
	}
}

func (bbs *LRPBBS) DesireLRP(lrp models.DesiredLRP) error {
	err := lrp.Validate()
	if err != nil {
		return err
	}

	value, err := models.ToJSON(lrp)
	if err != nil {
		return err
	}

	err = shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return bbs.store.Create(storeadapter.StoreNode{
			Key:   shared.DesiredLRPSchemaPath(lrp),
			Value: value,
		})
	})

	switch err {
	case bbserrors.ErrStoreResourceExists:
		existingLRP, err := bbs.DesiredLRPByProcessGuid(lrp.ProcessGuid)
		if err != nil {
			return err
		}

		err = existingLRP.ValidateModifications(lrp)
		if err != nil {
			return err
		}

		value, err := models.ToJSON(lrp)
		if err != nil {
			return err
		}

		return shared.RetryIndefinitelyOnStoreTimeout(func() error {
			return bbs.store.SetMulti([]storeadapter.StoreNode{
				{
					Key:   shared.DesiredLRPSchemaPath(lrp),
					Value: value,
				},
			})
		})

	default:
		return err
	}
}

func (bbs *LRPBBS) RemoveDesiredLRPByProcessGuid(processGuid string) error {
	return shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return bbs.store.Delete(shared.DesiredLRPSchemaPathByProcessGuid(processGuid))
	})
}

func (bbs *LRPBBS) ChangeDesiredLRP(change models.DesiredLRPChange) error {
	beforeValue, err := models.ToJSON(change.Before)
	if err != nil {
		return err
	}
	afterValue, err := models.ToJSON(change.After)
	if err != nil {
		return err
	}

	return shared.RetryIndefinitelyOnStoreTimeout(func() error {
		if change.Before != nil && change.After != nil {
			return bbs.store.CompareAndSwap(
				storeadapter.StoreNode{
					Key:   shared.DesiredLRPSchemaPath(*change.Before),
					Value: beforeValue,
				},
				storeadapter.StoreNode{
					Key:   shared.DesiredLRPSchemaPath(*change.After),
					Value: afterValue,
				},
			)
		}

		if change.Before != nil {
			return bbs.store.CompareAndDelete(
				storeadapter.StoreNode{
					Key:   shared.DesiredLRPSchemaPath(*change.Before),
					Value: beforeValue,
				},
			)
		}

		if change.After != nil {
			return bbs.store.Create(
				storeadapter.StoreNode{
					Key:   shared.DesiredLRPSchemaPath(*change.After),
					Value: afterValue,
				},
			)
		}

		return nil
	})
}

func (bbs *LRPBBS) UpdateDesiredLRP(processGuid string, update models.DesiredLRPUpdate) error {
	existing, err := bbs.DesiredLRPByProcessGuid(processGuid)
	if err != nil {
		return err
	}

	updatedLRP := existing.ApplyUpdate(update)
	err = updatedLRP.Validate()
	if err != nil {
		return err
	}

	value, err := models.ToJSON(updatedLRP)
	if err != nil {
		return err
	}

	return shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return bbs.store.SetMulti([]storeadapter.StoreNode{
			{
				Key:   shared.DesiredLRPSchemaPath(updatedLRP),
				Value: value,
			},
		})
	})
}

func (bbs *LRPBBS) RemoveActualLRP(lrp models.ActualLRP) error {
	return bbs.RemoveActualLRPForIndex(lrp.ProcessGuid, lrp.Index)
}

func (bbs *LRPBBS) RemoveActualLRPForIndex(processGuid string, index int) error {
	return shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return bbs.store.Delete(shared.ActualLRPSchemaPath(processGuid, index))
	})
}
