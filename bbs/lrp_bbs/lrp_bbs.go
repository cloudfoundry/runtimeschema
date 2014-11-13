package lrp_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/timeprovider"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

type LRPBBS struct {
	store        storeadapter.StoreAdapter
	timeProvider timeprovider.TimeProvider
	logger       lager.Logger
}

func New(store storeadapter.StoreAdapter, timeProvider timeprovider.TimeProvider, logger lager.Logger) *LRPBBS {
	return &LRPBBS{
		store:        store,
		timeProvider: timeProvider,
		logger:       logger,
	}
}

func (bbs *LRPBBS) DesireLRP(lrp models.DesiredLRP) error {

	err := shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return bbs.store.Create(storeadapter.StoreNode{
			Key:   shared.DesiredLRPSchemaPath(lrp),
			Value: lrp.ToJSON(),
		})
	})

	switch err {
	case storeadapter.ErrorKeyExists:
		existingLRP, err := bbs.GetDesiredLRPByProcessGuid(lrp.ProcessGuid)
		if err != nil {
			return err
		}

		err = existingLRP.ValidateModifications(lrp)
		if err != nil {
			return err
		}

		return shared.RetryIndefinitelyOnStoreTimeout(func() error {
			return bbs.store.SetMulti([]storeadapter.StoreNode{
				{
					Key:   shared.DesiredLRPSchemaPath(lrp),
					Value: lrp.ToJSON(),
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
	return shared.RetryIndefinitelyOnStoreTimeout(func() error {
		if change.Before != nil && change.After != nil {
			return bbs.store.CompareAndSwap(
				storeadapter.StoreNode{
					Key:   shared.DesiredLRPSchemaPath(*change.Before),
					Value: (*change.Before).ToJSON(),
				},
				storeadapter.StoreNode{
					Key:   shared.DesiredLRPSchemaPath(*change.After),
					Value: (*change.After).ToJSON(),
				},
			)
		}

		if change.Before != nil {
			return bbs.store.CompareAndDelete(
				storeadapter.StoreNode{
					Key:   shared.DesiredLRPSchemaPath(*change.Before),
					Value: (*change.Before).ToJSON(),
				},
			)
		}

		if change.After != nil {
			return bbs.store.Create(
				storeadapter.StoreNode{
					Key:   shared.DesiredLRPSchemaPath(*change.After),
					Value: (*change.After).ToJSON(),
				},
			)
		}

		return nil
	})
}

func (bbs *LRPBBS) UpdateDesiredLRP(processGuid string, update models.DesiredLRPUpdate) error {
	existing, err := bbs.GetDesiredLRPByProcessGuid(processGuid)
	if err != nil {
		return err
	}

	updatedLRP := existing.ApplyUpdate(update)
	err = updatedLRP.Validate()
	if err != nil {
		return err
	}

	return shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return bbs.store.SetMulti([]storeadapter.StoreNode{
			{
				Key:   shared.DesiredLRPSchemaPath(updatedLRP),
				Value: updatedLRP.ToJSON(),
			},
		})
	})
}

func (bbs *LRPBBS) RemoveActualLRP(lrp models.ActualLRP) error {
	return bbs.RemoveActualLRPForIndex(lrp.ProcessGuid, lrp.Index, lrp.InstanceGuid)
}

func (bbs *LRPBBS) RemoveActualLRPForIndex(processGuid string, index int, instanceGuid string) error {
	return shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return bbs.store.Delete(shared.ActualLRPSchemaPath(processGuid, index, instanceGuid))
	})
}

func (bbs *LRPBBS) ReportActualLRPAsStarting(processGuid, instanceGuid, cellID, domain string, index int) (models.ActualLRP, error) {
	lrp, err := models.NewActualLRP(processGuid, instanceGuid, cellID, domain, index)
	if err != nil {
		return lrp, err
	}
	lrp.State = models.ActualLRPStateStarting
	lrp.Since = bbs.timeProvider.Time().UnixNano()

	return lrp, shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return bbs.store.SetMulti([]storeadapter.StoreNode{
			{
				Key:   shared.ActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index, lrp.InstanceGuid),
				Value: lrp.ToJSON(),
			},
		})
	})
}

func (bbs *LRPBBS) ReportActualLRPAsRunning(lrp models.ActualLRP, cellID string) error {
	lrp.State = models.ActualLRPStateRunning
	lrp.Since = bbs.timeProvider.Time().UnixNano()
	lrp.CellID = cellID

	return shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return bbs.store.SetMulti([]storeadapter.StoreNode{
			{
				Key:   shared.ActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index, lrp.InstanceGuid),
				Value: lrp.ToJSON(),
			},
		})
	})
}
