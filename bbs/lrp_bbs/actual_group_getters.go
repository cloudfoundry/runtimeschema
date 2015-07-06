package lrp_bbs

import (
	"fmt"
	"path"
	"sync"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/workpool"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

const maxActualGroupGetterWorkPoolSize = 50

func (bbs *LRPBBS) ActualLRPGroupsByDomain(logger lager.Logger, domain string) ([]models.ActualLRPGroup, error) {
	if len(domain) == 0 {
		return nil, bbserrors.ErrNoDomain
	}
	logger = logger.WithData(lager.Data{"domain": domain})

	logger.Debug("fetching-actual-lrps-from-bbs")
	root, err := bbs.store.ListRecursively(shared.ActualLRPSchemaRoot)
	if err == storeadapter.ErrorKeyNotFound {
		logger.Debug("no-actual-lrps-to-fetch")
		return []models.ActualLRPGroup{}, nil
	} else if err != nil {
		logger.Error("failed-fetching-actual-lrps-from-bbs", err)
		return []models.ActualLRPGroup{}, shared.ConvertStoreError(err)
	}
	logger.Debug("succeeded-fetching-actual-lrps-from-bbs", lager.Data{"num-lrps": len(root.ChildNodes)})

	if len(root.ChildNodes) == 0 {
		return []models.ActualLRPGroup{}, nil
	}

	var groups = []models.ActualLRPGroup{}
	groupsLock := sync.Mutex{}
	var workErr error
	workErrLock := sync.Mutex{}

	works := []func(){}

	for _, node := range root.ChildNodes {
		node := node

		works = append(works, func() {
			for _, indexNode := range node.ChildNodes {
				group := models.ActualLRPGroup{}
				for _, instanceNode := range indexNode.ChildNodes {
					var lrp models.ActualLRP
					deserializeErr := models.FromJSON(instanceNode.Value, &lrp)
					if deserializeErr != nil {
						logger.Error("invalid-instance-node", deserializeErr)
						workErrLock.Lock()
						workErr = fmt.Errorf("cannot parse lrp JSON for key %s: %s", instanceNode.Key, deserializeErr.Error())
						workErrLock.Unlock()
						continue
					}
					if lrp.Domain != domain {
						continue
					}

					if isInstanceActualLRPNode(instanceNode) {
						group.Instance = &lrp
					}

					if isEvacuatingActualLRPNode(instanceNode) {
						group.Evacuating = &lrp
					}
				}

				if group.Instance != nil || group.Evacuating != nil {
					groupsLock.Lock()
					groups = append(groups, group)
					groupsLock.Unlock()
				}
			}
		})
	}

	throttler, err := workpool.NewThrottler(maxActualGroupGetterWorkPoolSize, works)
	if err != nil {
		logger.Error("failed-constructing-throttler", err, lager.Data{"max-workers": maxActualGroupGetterWorkPoolSize, "num-works": len(works)})
		return []models.ActualLRPGroup{}, err
	}

	logger.Debug("performing-deserialization-work")
	throttler.Work()
	if workErr != nil {
		logger.Error("failed-performing-deserialization-work", workErr)
		return []models.ActualLRPGroup{}, workErr
	}
	logger.Debug("succeeded-performing-deserialization-work", lager.Data{"num-actual-lrp-groups": len(groups)})

	return groups, nil
}

func (bbs *LRPBBS) ActualLRPGroupsByProcessGuid(logger lager.Logger, processGuid string) (models.ActualLRPGroupsByIndex, error) {
	if len(processGuid) == 0 {
		return models.ActualLRPGroupsByIndex{}, bbserrors.ErrNoProcessGuid
	}
	logger = logger.WithData(lager.Data{"process-guid": processGuid})

	groups := models.ActualLRPGroupsByIndex{}

	logger.Debug("fetching-actual-lrps-from-bbs")
	root, err := bbs.store.ListRecursively(shared.ActualLRPProcessDir(processGuid))
	if err == storeadapter.ErrorKeyNotFound {
		logger.Debug("no-actual-lrps-to-fetch")
		return groups, nil
	} else if err != nil {
		logger.Error("failed-fetching-actual-lrps-from-bbs", err)
		return groups, shared.ConvertStoreError(err)
	}
	logger.Debug("succeeded-fetching-actual-lrps-from-bbs", lager.Data{"num-lrps": len(root.ChildNodes)})

	for _, indexNode := range root.ChildNodes {
		for _, instanceNode := range indexNode.ChildNodes {
			var lrp models.ActualLRP
			err = models.FromJSON(instanceNode.Value, &lrp)
			if err != nil {
				logger.Error("invalid-instance-node", err)
				return groups, fmt.Errorf("cannot parse lrp JSON for key %s: %s", instanceNode.Key, err.Error())
			}

			group := groups[lrp.Index]

			if isInstanceActualLRPNode(instanceNode) {
				group.Instance = &lrp
			}

			if isEvacuatingActualLRPNode(instanceNode) {
				group.Evacuating = &lrp
			}

			groups[lrp.Index] = group
		}
	}

	return groups, nil
}

func (bbs *LRPBBS) ActualLRPGroupByProcessGuidAndIndex(logger lager.Logger, processGuid string, index int) (models.ActualLRPGroup, error) {
	if len(processGuid) == 0 {
		return models.ActualLRPGroup{}, bbserrors.ErrNoProcessGuid
	}
	logger = logger.WithData(lager.Data{"process-guid": processGuid, "index": index})

	logger.Debug("fetching-actual-lrps-from-bbs")
	indexNode, err := bbs.store.ListRecursively(shared.ActualLRPIndexDir(processGuid, index))
	if err != nil {
		logger.Error("failed-fetching-actual-lrps-from-bbs", err)
		return models.ActualLRPGroup{}, shared.ConvertStoreError(err)
	}
	logger.Debug("succeeded-fetching-actual-lrps-from-bbs")

	group := models.ActualLRPGroup{}
	for _, instanceNode := range indexNode.ChildNodes {
		var lrp models.ActualLRP
		err = models.FromJSON(instanceNode.Value, &lrp)
		if err != nil {
			logger.Error("invalid-instance-node", err)
			return group, fmt.Errorf("cannot parse lrp JSON for key %s: %s", instanceNode.Key, err.Error())
		}

		if isInstanceActualLRPNode(instanceNode) {
			group.Instance = &lrp
		}

		if isEvacuatingActualLRPNode(instanceNode) {
			group.Evacuating = &lrp
		}
	}

	if group.Evacuating == nil && group.Instance == nil {
		return models.ActualLRPGroup{}, bbserrors.ErrStoreResourceNotFound
	}

	return group, err
}

func (bbs *LRPBBS) EvacuatingActualLRPByProcessGuidAndIndex(logger lager.Logger, processGuid string, index int) (models.ActualLRP, error) {
	if len(processGuid) == 0 {
		return models.ActualLRP{}, bbserrors.ErrNoProcessGuid
	}
	logger = logger.WithData(lager.Data{"process-guid": processGuid, "index": index})

	logger.Debug("fetching-evacuating-lrp-from-bbs")
	node, err := bbs.store.Get(shared.EvacuatingActualLRPSchemaPath(processGuid, index))
	if err != nil {
		logger.Error("failed-fetching-evacuating-lrp-from-bbs", err)
		return models.ActualLRP{}, shared.ConvertStoreError(err)
	}
	logger.Debug("succeeded-fetching-evacuating-lrp-from-bbs")

	var lrp models.ActualLRP
	err = models.FromJSON(node.Value, &lrp)
	if err != nil {
		logger.Error("invalid-node", err)
		return models.ActualLRP{}, fmt.Errorf("cannot parse lrp JSON for key %s: %s", node.Key, err.Error())
	}

	return lrp, err
}

func isInstanceActualLRPNode(node storeadapter.StoreNode) bool {
	return path.Base(node.Key) == shared.ActualLRPInstanceKey
}

func isEvacuatingActualLRPNode(node storeadapter.StoreNode) bool {
	return path.Base(node.Key) == shared.ActualLRPEvacuatingKey
}
