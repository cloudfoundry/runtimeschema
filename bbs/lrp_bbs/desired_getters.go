package lrp_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/pivotal-golang/lager"
)

const maxDesiredGetterWorkPoolSize = 50

func (bbs *LRPBBS) LegacyDesiredLRPByProcessGuid(logger lager.Logger, processGuid string) (models.DesiredLRP, error) {
	logger = logger.WithData(lager.Data{"process-guid": processGuid})
	lrp, _, err := bbs.desiredLRPByProcessGuidWithIndex(logger, processGuid)
	return lrp, err
}

func (bbs *LRPBBS) desiredLRPByProcessGuidWithIndex(logger lager.Logger, processGuid string) (models.DesiredLRP, uint64, error) {
	if len(processGuid) == 0 {
		return models.DesiredLRP{}, 0, bbserrors.ErrNoProcessGuid
	}

	logger.Debug("fetching-desired-lrp-from-bbs")
	node, err := bbs.store.Get(shared.DesiredLRPSchemaPath(models.DesiredLRP{ProcessGuid: processGuid}))
	if err != nil {
		logger.Error("failed-fetching-desired-lrp-from-bbs", err)
		return models.DesiredLRP{}, 0, shared.ConvertStoreError(err)
	}
	logger.Debug("succeeded-fetching-desired-lrp-from-bbs")

	var lrp models.DesiredLRP
	err = models.FromJSON(node.Value, &lrp)

	return lrp, node.Index, err
}
