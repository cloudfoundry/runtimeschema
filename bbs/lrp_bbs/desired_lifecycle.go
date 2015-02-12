package lrp_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	"github.com/nu7hatch/gouuid"
	"github.com/pivotal-golang/lager"
)

func (bbs *LRPBBS) DesireLRP(logger lager.Logger, lrp models.DesiredLRP) error {
	logger = logger.Session("create-desired-lrp", lager.Data{"process-guid": lrp.ProcessGuid})
	logger.Info("starting")
	defer logger.Info("complete")

	guid, err := uuid.NewV4()
	if err != nil {
		return err
	}

	lrpForCreate := lrp
	lrpForCreate.ModificationTag = models.ModificationTag{
		Epoch: guid.String(),
		Index: 0,
	}

	value, err := models.ToJSON(lrpForCreate)
	if err != nil {
		return err
	}

	err = bbs.store.Create(storeadapter.StoreNode{
		Key:   shared.DesiredLRPSchemaPath(lrpForCreate),
		Value: value,
	})
	if err != nil {
		logger.Error("failed-to-create-desired-lrp", err)
		return shared.ConvertStoreError(err)
	}

	bbs.startInstanceRange(logger, 0, lrp.Instances, lrpForCreate)
	return nil
}

func (bbs *LRPBBS) RemoveDesiredLRPByProcessGuid(logger lager.Logger, processGuid string) error {
	logger = logger.Session("remove-desired-lrp", lager.Data{"process-guid": processGuid})
	logger.Info("starting")
	defer logger.Info("complete")

	lrp, err := bbs.DesiredLRPByProcessGuid(processGuid)
	if err != nil {
		return err
	}

	err = bbs.store.Delete(shared.DesiredLRPSchemaPathByProcessGuid(processGuid))
	if err != nil {
		return shared.ConvertStoreError(err)
	}

	bbs.stopInstanceRange(logger, 0, lrp.Instances, lrp)
	return nil
}

func (bbs *LRPBBS) UpdateDesiredLRP(logger lager.Logger, processGuid string, desiredUpdate models.DesiredLRPUpdate) error {
	logger = logger.Session("update-desired-lrp", lager.Data{"process-guid": processGuid})
	logger.Info("starting")
	defer logger.Info("complete")

	existing, index, err := bbs.desiredLRPByProcessGuidWithIndex(processGuid)
	if err != nil {
		logger.Error("failed-to-fetch-existing-desired-lrp", err)
		return err
	}

	updated := existing.ApplyUpdate(desiredUpdate)

	updated.ModificationTag.Increment()

	value, err := models.ToJSON(updated)
	if err != nil {
		logger.Error("failed-to-serialize-desired-lrp", err)
		return err
	}

	err = bbs.store.CompareAndSwapByIndex(index, storeadapter.StoreNode{
		Key:   shared.DesiredLRPSchemaPath(updated),
		Value: value,
	})
	if err != nil {
		logger.Error("failed-to-CAS-desired-lrp", err)
		return shared.ConvertStoreError(err)
	}

	diff := updated.Instances - existing.Instances
	switch {
	case diff > 0:
		bbs.startInstanceRange(logger, existing.Instances, updated.Instances, updated)

	case diff < 0:
		bbs.stopInstanceRange(logger, updated.Instances, existing.Instances, updated)

	case diff == 0:
		// this space intentionally left blank
	}

	return nil
}

func (bbs *LRPBBS) startInstanceRange(logger lager.Logger, lower, upper int, desired models.DesiredLRP) {
	logger = logger.Session("start-instance-range", lager.Data{"lower": lower, "upper": upper})
	logger.Info("starting")
	defer logger.Info("complete")

	indices := make([]uint, upper-lower)
	i := 0
	for actualIndex := lower; actualIndex < upper; actualIndex++ {
		indices[i] = uint(actualIndex)
		i++
	}

	err := bbs.createAndStartActualLRPsForDesired(logger, desired, indices)
	if err != nil {
		logger.Error("failed-to-create-and-start-actual-lrps", err)
	}
}

func (bbs *LRPBBS) stopInstanceRange(logger lager.Logger, lower, upper int, desired models.DesiredLRP) {
	logger = logger.Session("stop-instance-range", lager.Data{"lower": lower, "upper": upper})
	logger.Info("starting")
	defer logger.Info("complete")

	actualsMap, err := bbs.ActualLRPsByProcessGuid(desired.ProcessGuid)
	if err != nil {
		logger.Error("failed-to-get-actual-lrps", err)
		return
	}

	actuals := make([]models.ActualLRP, 0)
	for i := lower; i < upper; i++ {
		actual, ok := actualsMap[i]
		if ok {
			actuals = append(actuals, actual)
		}
	}

	bbs.RetireActualLRPs(logger, actuals)
}
