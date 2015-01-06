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
}

func New(
	store storeadapter.StoreAdapter,
	timeProvider timeprovider.TimeProvider,
	cellClient cb.CellClient,
	auctioneerClient cb.AuctioneerClient,
	services *services_bbs.ServicesBBS,
) *LRPBBS {
	return &LRPBBS{
		store:            store,
		timeProvider:     timeProvider,
		cellClient:       cellClient,
		auctioneerClient: auctioneerClient,
		services:         services,
	}
}

func (bbs *LRPBBS) DesireLRP(logger lager.Logger, lrp models.DesiredLRP) error {
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
		}, logger)

		return nil

	case nil:
		bbs.processDesiredChange(models.DesiredLRPChange{
			Before: nil,
			After:  &lrp,
		}, logger)

		return nil

	default:
		return err
	}
}

func (bbs *LRPBBS) RemoveDesiredLRPByProcessGuid(logger lager.Logger, processGuid string) error {
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
	}, logger)

	return nil
}

func (bbs *LRPBBS) UpdateDesiredLRP(logger lager.Logger, processGuid string, update models.DesiredLRPUpdate) error {
	existing, index, err := bbs.desiredLRPByProcessGuidWithIndex(processGuid)
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
		err := bbs.store.CompareAndSwapByIndex(index, storeadapter.StoreNode{
			Key:   shared.DesiredLRPSchemaPath(updatedLRP),
			Value: value,
		})
		if err != nil {
			return err
		}

		bbs.processDesiredChange(models.DesiredLRPChange{
			Before: &existing,
			After:  &updatedLRP,
		}, logger)

		return nil
	})
}
