package lrp_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/pivotal-golang/lager"
)

func (bbs *LRPBBS) EvacuateClaimedActualLRP(
	logger lager.Logger,
	actualLRPKey models.ActualLRPKey,
	actualLRPContainerKey models.ActualLRPContainerKey,
) error {
	err := bbs.unclaimActualLRP(logger, actualLRPKey, actualLRPContainerKey)
	if err != nil {
		return err
	}

	err = bbs.requestLRPAuctionForLRPKey(actualLRPKey)
	if err != nil {
		return err
	}

	return nil
}
