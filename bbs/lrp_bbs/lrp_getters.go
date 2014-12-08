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
	}

	if err != nil {
		return lrps, err
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
	}

	if err != nil {
		return lrps, err
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

func (bbs *LRPBBS) DesiredLRPByProcessGuid(processGuid string) (*models.DesiredLRP, error) {
	var node storeadapter.StoreNode
	err := shared.RetryIndefinitelyOnStoreTimeout(func() error {
		var err error
		node, err = bbs.store.Get(shared.DesiredLRPSchemaPath(models.DesiredLRP{ProcessGuid: processGuid}))
		return err
	})

	if err != nil {
		return nil, err
	}

	var lrp models.DesiredLRP
	err = models.FromJSON(node.Value, &lrp)

	return &lrp, err
}

func (bbs *LRPBBS) ActualLRPsByCellID(cellID string) ([]models.ActualLRP, error) {
	lrps := []models.ActualLRP{}

	node, err := bbs.store.ListRecursively(shared.ActualLRPSchemaRoot)
	if err == storeadapter.ErrorKeyNotFound {
		return lrps, nil
	}

	if err != nil {
		return lrps, err
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
	}

	if err != nil {
		return lrps, err
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

	filteredLRPs := []models.ActualLRP{}
	for _, lrp := range lrps {
		if lrp.State == models.ActualLRPStateRunning {
			filteredLRPs = append(filteredLRPs, lrp)
		}
	}

	return filteredLRPs, nil
}

func (bbs *LRPBBS) ActualLRPsByProcessGuid(processGuid string) (models.ActualLRPsByIndex, error) {
	lrps := models.ActualLRPsByIndex{}

	node, err := bbs.store.ListRecursively(shared.ActualLRPProcessDir(processGuid))
	if err == storeadapter.ErrorKeyNotFound {
		return lrps, nil
	}

	if err != nil {
		return lrps, err
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

func (bbs *LRPBBS) ActualLRPsByProcessGuidAndIndex(processGuid string, index int) ([]models.ActualLRP, error) {
	lrps := []models.ActualLRP{}

	node, err := bbs.store.ListRecursively(shared.ActualLRPIndexDir(processGuid, index))
	if err == storeadapter.ErrorKeyNotFound {
		return lrps, nil
	}

	if err != nil {
		return lrps, err
	}

	for _, instanceNode := range node.ChildNodes {
		var lrp models.ActualLRP
		err := models.FromJSON(instanceNode.Value, &lrp)
		if err != nil {
			return lrps, fmt.Errorf("cannot parse lrp JSON for key %s: %s", instanceNode.Key, err.Error())
		} else {
			lrps = append(lrps, lrp)
		}
	}

	return lrps, nil
}

func (bbs *LRPBBS) RunningActualLRPsByProcessGuid(processGuid string) (models.ActualLRPsByIndex, error) {
	lrps, err := bbs.ActualLRPsByProcessGuid(processGuid)
	if err != nil {
		return nil, err
	}

	for i, lrp := range lrps {
		if lrp.State != models.ActualLRPStateRunning {
			delete(lrps, i)
		}
	}

	return lrps, nil
}

func (bbs *LRPBBS) ActualLRPsByDomain(domain string) ([]models.ActualLRP, error) {
	lrps := []models.ActualLRP{}

	node, err := bbs.store.ListRecursively(shared.ActualLRPSchemaRoot)
	if err == storeadapter.ErrorKeyNotFound {
		return lrps, nil
	}

	if err != nil {
		return lrps, err
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
