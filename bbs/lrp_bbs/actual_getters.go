package lrp_bbs

import (
	"fmt"
	"path"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
)

func (bbs *LRPBBS) ActualLRPGroups() ([]models.ActualLRPGroup, error) {
	groups := []models.ActualLRPGroup{}

	node, err := bbs.store.ListRecursively(shared.ActualLRPSchemaRoot)
	if err == storeadapter.ErrorKeyNotFound {
		return groups, nil
	} else if err != nil {
		return groups, shared.ConvertStoreError(err)
	}

	for _, node := range node.ChildNodes {
		for _, indexNode := range node.ChildNodes {
			group := models.ActualLRPGroup{}
			for _, instanceNode := range indexNode.ChildNodes {
				var lrp models.ActualLRP
				err = models.FromJSON(instanceNode.Value, &lrp)
				if err != nil {
					return groups, fmt.Errorf("cannot parse lrp JSON for key %s: %s", instanceNode.Key, err.Error())
				}

				if isInstanceActualLRPNode(instanceNode) {
					group.Instance = &lrp
				}

				if isEvacuatingActualLRPNode(instanceNode) {
					group.Evacuating = &lrp
				}
			}

			if group.Instance != nil || group.Evacuating != nil {
				groups = append(groups, group)
			}
		}
	}

	return groups, nil
}

func (bbs *LRPBBS) ActualLRPs() ([]models.ActualLRP, error) {
	lrps := []models.ActualLRP{}

	node, err := bbs.store.ListRecursively(shared.ActualLRPSchemaRoot)
	if err == storeadapter.ErrorKeyNotFound {
		return lrps, nil
	} else if err != nil {
		return lrps, shared.ConvertStoreError(err)
	}

	for _, node := range node.ChildNodes {
		for _, indexNode := range node.ChildNodes {
			for _, instanceNode := range indexNode.ChildNodes {
				if !isInstanceActualLRPNode(instanceNode) {
					continue
				}

				var lrp models.ActualLRP
				err = models.FromJSON(instanceNode.Value, &lrp)
				if err != nil {
					return lrps, fmt.Errorf("cannot parse lrp JSON for key %s: %s", instanceNode.Key, err.Error())
				} else {
					lrps = append(lrps, lrp)
				}
			}
		}
	}

	return lrps, nil
}

func (bbs *LRPBBS) ActualLRPGroupsByDomain(domain string) ([]models.ActualLRPGroup, error) {
	if len(domain) == 0 {
		return nil, bbserrors.ErrNoDomain
	}

	groups := []models.ActualLRPGroup{}

	node, err := bbs.store.ListRecursively(shared.ActualLRPSchemaRoot)
	if err == storeadapter.ErrorKeyNotFound {
		return groups, nil
	} else if err != nil {
		return groups, shared.ConvertStoreError(err)
	}

	for _, node := range node.ChildNodes {
		for _, indexNode := range node.ChildNodes {
			group := models.ActualLRPGroup{}
			for _, instanceNode := range indexNode.ChildNodes {
				var lrp models.ActualLRP
				err = models.FromJSON(instanceNode.Value, &lrp)
				if err != nil {
					return groups, fmt.Errorf("cannot parse lrp JSON for key %s: %s", instanceNode.Key, err.Error())
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
				groups = append(groups, group)
			}
		}
	}

	return groups, nil
}

func (bbs *LRPBBS) ActualLRPsByProcessGuid(processGuid string) (models.ActualLRPsByIndex, error) {
	if len(processGuid) == 0 {
		return models.ActualLRPsByIndex{}, bbserrors.ErrNoProcessGuid
	}

	lrps := models.ActualLRPsByIndex{}

	node, err := bbs.store.ListRecursively(shared.ActualLRPProcessDir(processGuid))
	if err == storeadapter.ErrorKeyNotFound {
		return lrps, nil
	} else if err != nil {
		return lrps, shared.ConvertStoreError(err)
	}

	for _, indexNode := range node.ChildNodes {
		for _, instanceNode := range indexNode.ChildNodes {
			if !isInstanceActualLRPNode(instanceNode) {
				continue
			}

			var lrp models.ActualLRP
			err = models.FromJSON(instanceNode.Value, &lrp)
			if err != nil {
				return lrps, fmt.Errorf("cannot parse lrp JSON for key %s: %s", instanceNode.Key, err.Error())
			} else {
				lrps[lrp.Index] = lrp
			}
		}
	}

	return lrps, nil
}

func (bbs *LRPBBS) ActualLRPGroupsByProcessGuid(processGuid string) (models.ActualLRPGroupsByIndex, error) {
	if len(processGuid) == 0 {
		return models.ActualLRPGroupsByIndex{}, bbserrors.ErrNoProcessGuid
	}

	groups := models.ActualLRPGroupsByIndex{}

	node, err := bbs.store.ListRecursively(shared.ActualLRPProcessDir(processGuid))
	if err == storeadapter.ErrorKeyNotFound {
		return groups, nil
	} else if err != nil {
		return groups, shared.ConvertStoreError(err)
	}

	for _, indexNode := range node.ChildNodes {
		for _, instanceNode := range indexNode.ChildNodes {
			var lrp models.ActualLRP
			err = models.FromJSON(instanceNode.Value, &lrp)
			if err != nil {
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

func (bbs *LRPBBS) ActualLRPGroupsByCellID(cellID string) ([]models.ActualLRPGroup, error) {
	if len(cellID) == 0 {
		return nil, bbserrors.ErrNoCellID
	}

	groups := []models.ActualLRPGroup{}

	node, err := bbs.store.ListRecursively(shared.ActualLRPSchemaRoot)
	if err == storeadapter.ErrorKeyNotFound {
		return groups, nil
	} else if err != nil {
		return groups, shared.ConvertStoreError(err)
	}

	for _, node := range node.ChildNodes {
		for _, indexNode := range node.ChildNodes {
			group := models.ActualLRPGroup{}
			for _, instanceNode := range indexNode.ChildNodes {
				var lrp models.ActualLRP
				err = models.FromJSON(instanceNode.Value, &lrp)
				if err != nil {
					return groups, fmt.Errorf("cannot parse lrp JSON for key %s: %s", instanceNode.Key, err.Error())
				}
				if lrp.CellID != cellID {
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
				groups = append(groups, group)
			}
		}
	}

	return groups, nil
}

func (bbs *LRPBBS) ActualLRPByProcessGuidAndIndex(processGuid string, index int) (models.ActualLRP, error) {
	if len(processGuid) == 0 {
		return models.ActualLRP{}, bbserrors.ErrNoProcessGuid
	}

	node, err := bbs.store.Get(shared.ActualLRPSchemaPath(processGuid, index))
	if err != nil {
		return models.ActualLRP{}, shared.ConvertStoreError(err)
	}

	var lrp models.ActualLRP
	err = models.FromJSON(node.Value, &lrp)
	if err != nil {
		return models.ActualLRP{}, fmt.Errorf("cannot parse lrp JSON for key %s: %s", node.Key, err.Error())
	}

	return lrp, err
}

func (bbs *LRPBBS) ActualLRPGroupByProcessGuidAndIndex(processGuid string, index int) (models.ActualLRPGroup, error) {
	if len(processGuid) == 0 {
		return models.ActualLRPGroup{}, bbserrors.ErrNoProcessGuid
	}

	indexNode, err := bbs.store.ListRecursively(shared.ActualLRPIndexDir(processGuid, index))
	if err != nil {
		return models.ActualLRPGroup{}, shared.ConvertStoreError(err)
	}

	group := models.ActualLRPGroup{}
	for _, instanceNode := range indexNode.ChildNodes {
		var lrp models.ActualLRP
		err = models.FromJSON(instanceNode.Value, &lrp)
		if err != nil {
			return group, fmt.Errorf("cannot parse lrp JSON for key %s: %s", instanceNode.Key, err.Error())
		}

		if isInstanceActualLRPNode(instanceNode) {
			group.Instance = &lrp
		}

		if isEvacuatingActualLRPNode(instanceNode) {
			group.Evacuating = &lrp
		}
	}

	return group, err
}

func (bbs *LRPBBS) EvacuatingActualLRPByProcessGuidAndIndex(processGuid string, index int) (models.ActualLRP, error) {
	if len(processGuid) == 0 {
		return models.ActualLRP{}, bbserrors.ErrNoProcessGuid
	}

	node, err := bbs.store.Get(shared.EvacuatingActualLRPSchemaPath(processGuid, index))
	if err != nil {
		return models.ActualLRP{}, shared.ConvertStoreError(err)
	}

	var lrp models.ActualLRP
	err = models.FromJSON(node.Value, &lrp)
	if err != nil {
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
