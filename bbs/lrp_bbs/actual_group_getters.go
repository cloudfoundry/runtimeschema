package lrp_bbs

import (
	"fmt"
	"path"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

const maxActualGroupGetterWorkPoolSize = 50

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

func (bbs *LRPBBS) LegacyActualLRPGroupByProcessGuidAndIndex(logger lager.Logger, processGuid string, index int) (models.ActualLRPGroup, error) {
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

func isInstanceActualLRPNode(node storeadapter.StoreNode) bool {
	return path.Base(node.Key) == shared.ActualLRPInstanceKey
}

func isEvacuatingActualLRPNode(node storeadapter.StoreNode) bool {
	return path.Base(node.Key) == shared.ActualLRPEvacuatingKey
}
