package lrp_bbs

import (
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
