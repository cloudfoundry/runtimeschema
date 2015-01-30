package lrp_bbs

import (
	"errors"
	"fmt"
	"path"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
)

var ErrNoDomain = errors.New("no domain given")

func (bbs *LRPBBS) DesiredLRPs() ([]models.DesiredLRP, error) {
	lrps := []models.DesiredLRP{}

	node, err := bbs.store.ListRecursively(shared.DesiredLRPSchemaRoot)
	if err == storeadapter.ErrorKeyNotFound {
		return lrps, nil
	} else if err != nil {
		return lrps, shared.ConvertStoreError(err)
	}

	for _, node := range node.ChildNodes {
		var lrp models.DesiredLRP
		err = models.FromJSON(node.Value, &lrp)
		if err != nil {
			return lrps, fmt.Errorf("cannot parse lrp JSON for key %s: %s", node.Key, err.Error())
		} else {
			lrps = append(lrps, lrp)
		}
	}

	return lrps, nil
}

func (bbs *LRPBBS) DesiredLRPsByDomain(domain string) ([]models.DesiredLRP, error) {
	if len(domain) == 0 {
		return nil, ErrNoDomain
	}

	lrps := []models.DesiredLRP{}

	node, err := bbs.store.ListRecursively(shared.DesiredLRPSchemaRoot)
	if err == storeadapter.ErrorKeyNotFound {
		return lrps, nil
	} else if err != nil {
		return lrps, shared.ConvertStoreError(err)
	}

	for _, node := range node.ChildNodes {
		var lrp models.DesiredLRP
		err = models.FromJSON(node.Value, &lrp)
		if err != nil {
			return lrps, fmt.Errorf("cannot parse lrp JSON for key %s: %s", node.Key, err.Error())
		} else if lrp.Domain == domain {
			lrps = append(lrps, lrp)
		}
	}

	return lrps, nil
}

func (bbs *LRPBBS) DesiredLRPByProcessGuid(processGuid string) (models.DesiredLRP, error) {
	lrp, _, err := bbs.desiredLRPByProcessGuidWithIndex(processGuid)
	return lrp, err
}

func (bbs *LRPBBS) ActualLRPsByCellID(cellID string) ([]models.ActualLRP, error) {
	lrps := []models.ActualLRP{}

	node, err := bbs.store.ListRecursively(shared.ActualLRPSchemaRoot)
	if err == storeadapter.ErrorKeyNotFound {
		return lrps, nil
	} else if err != nil {
		return lrps, shared.ConvertStoreError(err)
	}

	for _, processNode := range node.ChildNodes {
		for _, indexNode := range processNode.ChildNodes {
			lrp, err := extractDominantActualLRP(indexNode)
			if err != nil {
				return lrps, err
			}
			if lrp.CellID == cellID {
				lrps = append(lrps, lrp)
			}
		}
	}

	return lrps, nil
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
			lrp, err := extractDominantActualLRP(indexNode)
			if err != nil {
				return lrps, err
			}
			lrps = append(lrps, lrp)
		}
	}

	return lrps, nil
}

func (bbs *LRPBBS) ActualLRPsByProcessGuid(processGuid string) (models.ActualLRPsByIndex, error) {
	lrps := models.ActualLRPsByIndex{}

	node, err := bbs.store.ListRecursively(shared.ActualLRPProcessDir(processGuid))
	if err == storeadapter.ErrorKeyNotFound {
		return lrps, nil
	} else if err != nil {
		return lrps, shared.ConvertStoreError(err)
	}

	for _, indexNode := range node.ChildNodes {
		lrp, err := extractDominantActualLRP(indexNode)
		if err != nil {
			return lrps, err
		}
		lrps[lrp.Index] = lrp
	}

	return lrps, nil
}

func (bbs *LRPBBS) ActualLRPByProcessGuidAndIndex(processGuid string, index int) (models.ActualLRP, error) {
	indexNode, indexErr := bbs.store.ListRecursively(shared.ActualLRPIndexDir(processGuid, index))

	if indexErr != nil {
		return models.ActualLRP{}, shared.ConvertStoreError(indexErr)
	}

	return extractDominantActualLRP(indexNode)
}

func (bbs *LRPBBS) ActualLRPsByDomain(domain string) ([]models.ActualLRP, error) {
	lrps := []models.ActualLRP{}

	node, err := bbs.store.ListRecursively(shared.ActualLRPSchemaRoot)
	if err == storeadapter.ErrorKeyNotFound {
		return lrps, nil
	} else if err != nil {
		return lrps, shared.ConvertStoreError(err)
	}

	for _, node := range node.ChildNodes {
		for _, indexNode := range node.ChildNodes {
			lrp, err := extractDominantActualLRP(indexNode)
			if err != nil {
				return lrps, err
			}
			if lrp.Domain == domain {
				lrps = append(lrps, lrp)
			}
		}
	}

	return lrps, nil
}

func (bbs *LRPBBS) desiredLRPByProcessGuidWithIndex(processGuid string) (models.DesiredLRP, uint64, error) {
	node, err := bbs.store.Get(shared.DesiredLRPSchemaPath(models.DesiredLRP{ProcessGuid: processGuid}))
	if err != nil {
		return models.DesiredLRP{}, 0, shared.ConvertStoreError(err)
	}

	var lrp models.DesiredLRP
	err = models.FromJSON(node.Value, &lrp)

	return lrp, node.Index, err
}

func extractDominantActualLRP(indexNode storeadapter.StoreNode) (models.ActualLRP, error) {
	var instanceNode *storeadapter.StoreNode
	var evacuatingNode *storeadapter.StoreNode

	for _, node := range indexNode.ChildNodes {
		node := node
		switch path.Base(node.Key) {
		case shared.ActualLRPInstanceKey:
			instanceNode = &node
		case shared.ActualLRPEvacuatingKey:
			evacuatingNode = &node
		}
	}

	instanceFound := instanceNode != nil
	evacuatingFound := evacuatingNode != nil

	if !instanceFound && !evacuatingFound {
		return models.ActualLRP{}, bbserrors.ErrStoreResourceNotFound
	}

	var err error
	var instanceLRP models.ActualLRP
	var evacuatingLRP models.ActualLRP

	if instanceFound {
		err = models.FromJSON(instanceNode.Value, &instanceLRP)
		if err != nil {
			return models.ActualLRP{}, fmt.Errorf("cannot parse lrp JSON for key %s: %s", instanceNode.Key, err.Error())
		}
	}

	if evacuatingFound {
		err = models.FromJSON(evacuatingNode.Value, &evacuatingLRP)
		if err != nil {
			return models.ActualLRP{}, fmt.Errorf("cannot parse lrp JSON for key %s: %s", evacuatingNode.Key, err.Error())
		}
	}

	if instanceLRP.State == models.ActualLRPStateRunning || instanceLRP.State == models.ActualLRPStateCrashed {
		return instanceLRP, err
	}

	if evacuatingFound {
		return evacuatingLRP, nil
	}

	return instanceLRP, err
}
