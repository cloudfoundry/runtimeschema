package lrp_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/services_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/cb"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/timeprovider"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

type LRPBBS struct {
	store            storeadapter.StoreAdapter
	timeProvider     timeprovider.TimeProvider
	cellClient       cb.CellClient
	auctioneerClient cb.AuctioneerClient
	services         *services_bbs.ServicesBBS
	logger           lager.Logger
}

func New(
	store storeadapter.StoreAdapter,
	timeProvider timeprovider.TimeProvider,
	cellClient cb.CellClient,
	auctioneerClient cb.AuctioneerClient,
	services *services_bbs.ServicesBBS,
	logger lager.Logger,
) *LRPBBS {
	return &LRPBBS{
		store:            store,
		timeProvider:     timeProvider,
		cellClient:       cellClient,
		auctioneerClient: auctioneerClient,
		services:         services,
		logger:           logger,
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
		existingLRP, index, err := bbs.desiredLRPByProcessGuidWithIndex(lrp.ProcessGuid)
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

		err = shared.RetryIndefinitelyOnStoreTimeout(func() error {
			return bbs.store.CompareAndSwapByIndex(index, storeadapter.StoreNode{
				Key:   shared.DesiredLRPSchemaPath(lrp),
				Value: value,
			})
		})
		if err != nil {
			return err
		}

		bbs.processDesiredChange(models.DesiredLRPChange{
			Before: &existingLRP,
			After:  &lrp,
		}, bbs.logger)

		return nil

	case nil:
		bbs.processDesiredChange(models.DesiredLRPChange{
			Before: nil,
			After:  &lrp,
		}, bbs.logger)

		return nil

	default:
		return err
	}
}

func (bbs *LRPBBS) RemoveDesiredLRPByProcessGuid(processGuid string) error {
	lrp, err := bbs.DesiredLRPByProcessGuid(processGuid)
	if err != nil {
		return err
	}

	err = shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return bbs.store.Delete(shared.DesiredLRPSchemaPathByProcessGuid(processGuid))
	})
	if err != nil {
		return err
	}

	bbs.processDesiredChange(models.DesiredLRPChange{
		Before: &lrp,
		After:  nil,
	}, bbs.logger)

	return nil
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
		err := bbs.store.SetMulti([]storeadapter.StoreNode{
			{
				Key:   shared.DesiredLRPSchemaPath(updatedLRP),
				Value: value,
			},
		})
		if err != nil {
			return err
		}

		bbs.processDesiredChange(models.DesiredLRPChange{
			Before: &existing,
			After:  &updatedLRP,
		}, bbs.logger)

		return nil
	})
}
