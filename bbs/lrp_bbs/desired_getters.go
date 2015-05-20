package lrp_bbs

import (
	"fmt"
	"sync"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
)

const maxDesiredGetterWorkPoolSize = 50

func (bbs *LRPBBS) DesiredLRPs() ([]models.DesiredLRP, error) {
	lrps := []models.DesiredLRP{}

	root, err := bbs.store.ListRecursively(shared.DesiredLRPSchemaRoot)
	if err == storeadapter.ErrorKeyNotFound {
		return lrps, nil
	} else if err != nil {
		return lrps, shared.ConvertStoreError(err)
	}

	if len(root.ChildNodes) == 0 {
		return lrps, nil
	}

	lrpLock := sync.Mutex{}
	errLock := sync.Mutex{}
	workPool, err := constructWorkPool(len(root.ChildNodes), maxDesiredGetterWorkPoolSize)
	if err != nil {
		return lrps, err
	}

	wg := sync.WaitGroup{}
	for _, node := range root.ChildNodes {
		node := node

		wg.Add(1)
		workPool.Submit(func() {
			defer wg.Done()

			var lrp models.DesiredLRP
			deserializeErr := models.FromJSON(node.Value, &lrp)
			if deserializeErr != nil {
				errLock.Lock()
				err = fmt.Errorf("cannot parse lrp JSON for key %s: %s", node.Key, deserializeErr.Error())
				errLock.Unlock()
			} else {
				lrpLock.Lock()
				lrps = append(lrps, lrp)
				lrpLock.Unlock()
			}
		})
	}
	wg.Wait()

	if err != nil {
		return []models.DesiredLRP{}, err
	}

	return lrps, nil
}

func (bbs *LRPBBS) DesiredLRPsByDomain(domain string) ([]models.DesiredLRP, error) {
	if len(domain) == 0 {
		return nil, bbserrors.ErrNoDomain
	}

	lrps := []models.DesiredLRP{}
	root, err := bbs.store.ListRecursively(shared.DesiredLRPSchemaRoot)
	if err == storeadapter.ErrorKeyNotFound {
		return lrps, nil
	} else if err != nil {
		return lrps, shared.ConvertStoreError(err)
	}

	if len(root.ChildNodes) == 0 {
		return lrps, nil
	}

	lrpLock := sync.Mutex{}
	errLock := sync.Mutex{}
	workPool, err := constructWorkPool(len(root.ChildNodes), maxDesiredGetterWorkPoolSize)
	if err != nil {
		return lrps, err
	}

	wg := sync.WaitGroup{}
	for _, node := range root.ChildNodes {
		node := node

		wg.Add(1)
		workPool.Submit(func() {
			defer wg.Done()

			var lrp models.DesiredLRP
			deserializeErr := models.FromJSON(node.Value, &lrp)
			switch {
			case deserializeErr != nil:
				errLock.Lock()
				err = fmt.Errorf("cannot parse lrp JSON for key %s: %s", node.Key, deserializeErr.Error())
				errLock.Unlock()
			case lrp.Domain == domain:
				lrpLock.Lock()
				lrps = append(lrps, lrp)
				lrpLock.Unlock()
			default:
			}
		})
	}
	wg.Wait()

	if err != nil {
		return []models.DesiredLRP{}, err
	}

	return lrps, nil
}

func (bbs *LRPBBS) DesiredLRPByProcessGuid(processGuid string) (models.DesiredLRP, error) {
	lrp, _, err := bbs.desiredLRPByProcessGuidWithIndex(processGuid)
	return lrp, err
}

func (bbs *LRPBBS) desiredLRPByProcessGuidWithIndex(processGuid string) (models.DesiredLRP, uint64, error) {
	if len(processGuid) == 0 {
		return models.DesiredLRP{}, 0, bbserrors.ErrNoProcessGuid
	}

	node, err := bbs.store.Get(shared.DesiredLRPSchemaPath(models.DesiredLRP{ProcessGuid: processGuid}))
	if err != nil {
		return models.DesiredLRP{}, 0, shared.ConvertStoreError(err)
	}

	var lrp models.DesiredLRP
	err = models.FromJSON(node.Value, &lrp)

	return lrp, node.Index, err
}
