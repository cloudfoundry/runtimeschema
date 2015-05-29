package lrp_bbs

import (
	"fmt"
	"sync"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/workpool"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

const maxDesiredGetterWorkPoolSize = 50

func (bbs *LRPBBS) DesiredLRPs(logger lager.Logger) ([]models.DesiredLRP, error) {
	logger.Debug("fetching-desired-lrps-from-bbs")
	root, err := bbs.store.ListRecursively(shared.DesiredLRPSchemaRoot)
	if err == storeadapter.ErrorKeyNotFound {
		logger.Debug("no-desired-lrps-to-fetch")
		return []models.DesiredLRP{}, nil
	} else if err != nil {
		logger.Error("failed-fetching-desired-lrps-from-bbs", err)
		return []models.DesiredLRP{}, shared.ConvertStoreError(err)
	}
	logger.Debug("succeeded-fetching-desired-lrps-from-bbs", lager.Data{"num-lrps": len(root.ChildNodes)})

	if len(root.ChildNodes) == 0 {
		return []models.DesiredLRP{}, nil
	}

	var lrps = []models.DesiredLRP{}
	lrpLock := sync.Mutex{}
	var workErr error
	workErrLock := sync.Mutex{}

	works := []func(){}

	for _, node := range root.ChildNodes {
		node := node

		works = append(works, func() {
			var lrp models.DesiredLRP
			deserializeErr := models.FromJSON(node.Value, &lrp)
			if deserializeErr != nil {
				logger.Error("invalid-node", deserializeErr)
				workErrLock.Lock()
				workErr = fmt.Errorf("cannot parse lrp JSON for key %s: %s", node.Key, deserializeErr.Error())
				workErrLock.Unlock()
			} else {
				lrpLock.Lock()
				lrps = append(lrps, lrp)
				lrpLock.Unlock()
			}
		})
	}

	throttler, err := workpool.NewThrottler(maxDesiredGetterWorkPoolSize, works)
	if err != nil {
		logger.Error("failed-constructing-throttler", err, lager.Data{"max-workers": maxDesiredGetterWorkPoolSize, "num-works": len(works)})
		return []models.DesiredLRP{}, err
	}

	logger.Debug("performing-deserialization-work")
	throttler.Work()
	if workErr != nil {
		logger.Error("failed-performing-deserialization-work", workErr)
		return []models.DesiredLRP{}, workErr
	}
	logger.Debug("succeeded-performing-deserialization-work", lager.Data{"num-desired-lrps": len(lrps)})

	return lrps, nil
}

func (bbs *LRPBBS) DesiredLRPsByDomain(logger lager.Logger, domain string) ([]models.DesiredLRP, error) {
	if len(domain) == 0 {
		return nil, bbserrors.ErrNoDomain
	}
	logger = logger.WithData(lager.Data{"domain": domain})

	logger.Debug("fetching-desired-lrps-from-bbs")
	root, err := bbs.store.ListRecursively(shared.DesiredLRPSchemaRoot)
	if err == storeadapter.ErrorKeyNotFound {
		logger.Debug("no-desired-lrps-to-fetch")
		return []models.DesiredLRP{}, nil
	} else if err != nil {
		logger.Error("failed-fetching-desired-lrps-from-bbs", err)
		return []models.DesiredLRP{}, shared.ConvertStoreError(err)
	}
	logger.Debug("succeeded-fetching-desired-lrps-from-bbs", lager.Data{"num-lrps": len(root.ChildNodes)})

	if len(root.ChildNodes) == 0 {
		return []models.DesiredLRP{}, nil
	}

	var lrps = []models.DesiredLRP{}
	lrpLock := sync.Mutex{}
	var workErr error
	workErrLock := sync.Mutex{}

	works := []func(){}

	for _, node := range root.ChildNodes {
		node := node

		works = append(works, func() {
			var lrp models.DesiredLRP
			deserializeErr := models.FromJSON(node.Value, &lrp)
			switch {
			case deserializeErr != nil:
				logger.Error("invalid-node", deserializeErr)
				workErrLock.Lock()
				workErr = fmt.Errorf("cannot parse lrp JSON for key %s: %s", node.Key, deserializeErr.Error())
				workErrLock.Unlock()
			case lrp.Domain == domain:
				lrpLock.Lock()
				lrps = append(lrps, lrp)
				lrpLock.Unlock()
			default:
			}
		})
	}

	throttler, err := workpool.NewThrottler(maxDesiredGetterWorkPoolSize, works)
	if err != nil {
		logger.Error("failed-constructing-throttler", err, lager.Data{"max-workers": maxDesiredGetterWorkPoolSize, "num-works": len(works)})
		return []models.DesiredLRP{}, err
	}

	logger.Debug("performing-deserialization-work")
	throttler.Work()
	if workErr != nil {
		logger.Error("failed-performing-deserialization-work", workErr)
		return []models.DesiredLRP{}, workErr
	}
	logger.Debug("succeeded-performing-deserialization-work", lager.Data{"num-desired-lrps": len(lrps)})

	return lrps, nil
}

func (bbs *LRPBBS) DesiredLRPByProcessGuid(logger lager.Logger, processGuid string) (models.DesiredLRP, error) {
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
