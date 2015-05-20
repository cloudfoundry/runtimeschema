package lrp_bbs

import (
	"fmt"
	"path"
	"sync"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
)

const maxActualGroupGetterWorkPoolSize = 50

func (bbs *LRPBBS) ActualLRPGroups() ([]models.ActualLRPGroup, error) {
	groups := []models.ActualLRPGroup{}

	root, err := bbs.store.ListRecursively(shared.ActualLRPSchemaRoot)
	if err == storeadapter.ErrorKeyNotFound {
		return groups, nil
	} else if err != nil {
		return groups, shared.ConvertStoreError(err)
	}

	if len(root.ChildNodes) == 0 {
		return groups, nil
	}

	groupsLock := sync.Mutex{}
	errLock := sync.Mutex{}
	workPool, err := constructWorkPool(len(root.ChildNodes), maxActualGroupGetterWorkPoolSize)
	if err != nil {
		return groups, err
	}
	defer workPool.Stop()

	wg := sync.WaitGroup{}
	for _, node := range root.ChildNodes {
		node := node

		wg.Add(1)
		workPool.Submit(func() {
			defer wg.Done()

			for _, indexNode := range node.ChildNodes {
				group := models.ActualLRPGroup{}
				for _, instanceNode := range indexNode.ChildNodes {
					var lrp models.ActualLRP
					deserializeErr := models.FromJSON(instanceNode.Value, &lrp)
					if deserializeErr != nil {
						errLock.Lock()
						err = fmt.Errorf("cannot parse lrp JSON for key %s: %s", instanceNode.Key, deserializeErr.Error())
						errLock.Unlock()
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
	wg.Wait()

	if err != nil {
		return []models.ActualLRPGroup{}, err
	}

	return groups, nil
}

func (bbs *LRPBBS) ActualLRPs() ([]models.ActualLRP, error) {
	lrps := []models.ActualLRP{}

	root, err := bbs.store.ListRecursively(shared.ActualLRPSchemaRoot)
	if err == storeadapter.ErrorKeyNotFound {
		return lrps, nil
	} else if err != nil {
		return lrps, shared.ConvertStoreError(err)
	}

	if len(root.ChildNodes) == 0 {
		return lrps, nil
	}

	lrpsLock := sync.Mutex{}
	errLock := sync.Mutex{}
	workPool, err := constructWorkPool(len(root.ChildNodes), maxActualGroupGetterWorkPoolSize)
	if err != nil {
		return lrps, err
	}
	defer workPool.Stop()

	wg := sync.WaitGroup{}
	for _, node := range root.ChildNodes {
		node := node

		wg.Add(1)
		workPool.Submit(func() {
			defer wg.Done()

			for _, indexNode := range node.ChildNodes {
				for _, instanceNode := range indexNode.ChildNodes {
					if !isInstanceActualLRPNode(instanceNode) {
						continue
					}

					var lrp models.ActualLRP
					deserializeErr := models.FromJSON(instanceNode.Value, &lrp)
					if deserializeErr != nil {
						errLock.Lock()
						err = fmt.Errorf("cannot parse lrp JSON for key %s: %s", instanceNode.Key, deserializeErr.Error())
						errLock.Unlock()
						continue
					} else {
						lrpsLock.Lock()
						lrps = append(lrps, lrp)
						lrpsLock.Unlock()
					}
				}
			}
		})
	}
	wg.Wait()

	if err != nil {
		return []models.ActualLRP{}, err
	}

	return lrps, nil
}

func (bbs *LRPBBS) ActualLRPGroupsByDomain(domain string) ([]models.ActualLRPGroup, error) {
	if len(domain) == 0 {
		return nil, bbserrors.ErrNoDomain
	}

	groups := []models.ActualLRPGroup{}

	root, err := bbs.store.ListRecursively(shared.ActualLRPSchemaRoot)
	if err == storeadapter.ErrorKeyNotFound {
		return groups, nil
	} else if err != nil {
		return groups, shared.ConvertStoreError(err)
	}

	if len(root.ChildNodes) == 0 {
		return groups, nil
	}

	groupsLock := sync.Mutex{}
	errLock := sync.Mutex{}
	workPool, err := constructWorkPool(len(root.ChildNodes), maxActualGroupGetterWorkPoolSize)
	if err != nil {
		return groups, err
	}
	defer workPool.Stop()

	wg := sync.WaitGroup{}
	for _, node := range root.ChildNodes {
		node := node

		wg.Add(1)
		workPool.Submit(func() {
			defer wg.Done()

			for _, indexNode := range node.ChildNodes {
				group := models.ActualLRPGroup{}
				for _, instanceNode := range indexNode.ChildNodes {
					var lrp models.ActualLRP
					deserializeErr := models.FromJSON(instanceNode.Value, &lrp)
					if deserializeErr != nil {
						errLock.Lock()
						err = fmt.Errorf("cannot parse lrp JSON for key %s: %s", instanceNode.Key, deserializeErr.Error())
						errLock.Unlock()
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
	wg.Wait()

	if err != nil {
		return []models.ActualLRPGroup{}, err
	}

	return groups, nil
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

	root, err := bbs.store.ListRecursively(shared.ActualLRPSchemaRoot)
	if err == storeadapter.ErrorKeyNotFound {
		return groups, nil
	} else if err != nil {
		return groups, shared.ConvertStoreError(err)
	}

	if len(root.ChildNodes) == 0 {
		return groups, nil
	}

	groupsLock := sync.Mutex{}
	errLock := sync.Mutex{}
	workPool, err := constructWorkPool(len(root.ChildNodes), maxActualGroupGetterWorkPoolSize)
	if err != nil {
		return groups, err
	}
	defer workPool.Stop()

	wg := sync.WaitGroup{}
	for _, node := range root.ChildNodes {
		node := node

		wg.Add(1)
		workPool.Submit(func() {
			defer wg.Done()

			for _, indexNode := range node.ChildNodes {
				group := models.ActualLRPGroup{}
				for _, instanceNode := range indexNode.ChildNodes {
					var lrp models.ActualLRP
					deserializeErr := models.FromJSON(instanceNode.Value, &lrp)
					if deserializeErr != nil {
						errLock.Lock()
						err = fmt.Errorf("cannot parse lrp JSON for key %s: %s", instanceNode.Key, deserializeErr.Error())
						errLock.Unlock()
						continue
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
					groupsLock.Lock()
					groups = append(groups, group)
					groupsLock.Unlock()
				}
			}
		})
	}
	wg.Wait()

	if err != nil {
		return []models.ActualLRPGroup{}, err
	}

	return groups, nil
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

	if group.Evacuating == nil && group.Instance == nil {
		return models.ActualLRPGroup{}, bbserrors.ErrStoreResourceNotFound
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
