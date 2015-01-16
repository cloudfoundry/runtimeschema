package lrp_bbs

import (
	"errors"
	"fmt"

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
			for _, instanceNode := range indexNode.ChildNodes {
				var lrp models.ActualLRP
				err = models.FromJSON(instanceNode.Value, &lrp)
				if err != nil {
					return lrps, fmt.Errorf("cannot parse lrp JSON for key %s: %s", instanceNode.Key, err.Error())
				} else if lrp.CellID == cellID {
					lrps = append(lrps, lrp)
				}
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
			for _, instanceNode := range indexNode.ChildNodes {
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

func (bbs *LRPBBS) RunningActualLRPs() ([]models.ActualLRP, error) {
	lrps, err := bbs.ActualLRPs()
	if err != nil {
		return []models.ActualLRP{}, err
	}

	return filterActualLRPs(lrps, models.ActualLRPStateRunning), nil
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
		for _, instanceNode := range indexNode.ChildNodes {
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

func (bbs *LRPBBS) ActualLRPByProcessGuidAndIndex(processGuid string, index int) (models.ActualLRP, error) {
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
			for _, instanceNode := range indexNode.ChildNodes {
				var lrp models.ActualLRP
				err = models.FromJSON(instanceNode.Value, &lrp)
				if err != nil {
					return lrps, fmt.Errorf("cannot parse lrp JSON for key %s: %s", instanceNode.Key, err.Error())
				} else if lrp.Domain == domain {
					lrps = append(lrps, lrp)
				}
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

func filterActualLRPs(lrps []models.ActualLRP, state models.ActualLRPState) []models.ActualLRP {
	filteredLRPs := []models.ActualLRP{}
	for _, lrp := range lrps {
		if lrp.State == state {
			filteredLRPs = append(filteredLRPs, lrp)
		}
	}

	return filteredLRPs
}
