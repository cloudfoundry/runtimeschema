package lrp_bbs

import (
	"path"
	"sync"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/services_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/metric"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

const (
	desiredLRPsDeleted = metric.Counter("ConvergerDesiredLRPsDeleted")
	actualLRPsDeleted  = metric.Counter("ConvergerActualLRPsDeleted")
)

type ConvergenceInput struct {
	AllProcessGuids map[string]struct{}
	DesiredLRPs     models.DesiredLRPsByProcessGuid
	ActualLRPs      models.ActualLRPsByProcessGuidAndIndex
	Domains         models.DomainSet
	Cells           models.CellSet
}

func (bbs *LRPBBS) GatherAndPruneLRPConvergenceInput(logger lager.Logger, cellsLoader *services_bbs.CellsLoader) (*ConvergenceInput, error) {
	guids := map[string]struct{}{}

	// always fetch actualLRPs before desiredLRPs to ensure correctness
	logger.Info("gathering-and-pruning-actual-lrps")
	actuals, err := bbs.gatherAndPruneActualLRPs(logger, guids)
	if err != nil {
		logger.Error("failed-gathering-and-pruning-actual-lrps", err)
		return &ConvergenceInput{}, err
	}
	logger.Info("succeeded-gathering-and-pruning-actual-lrps")

	domains, err := bbs.domains(logger)
	if err != nil {
		return &ConvergenceInput{}, err
	}

	// always fetch desiredLRPs after actualLRPs to ensure correctness
	logger.Info("gathering-and-pruning-desired-lrps")
	desireds, err := bbs.gatherAndPruneDesiredLRPs(logger, domains, guids)
	if err != nil {
		logger.Error("failed-gathering-and-pruning-desired-lrps", err)
		return &ConvergenceInput{}, err
	}
	logger.Info("succeeded-gathering-and-pruning-desired-lrps")

	cellSet, err := cellsLoader.Cells()
	if err != nil {
		return &ConvergenceInput{}, err
	}

	return &ConvergenceInput{
		AllProcessGuids: guids,
		DesiredLRPs:     desireds,
		ActualLRPs:      actuals,
		Domains:         domains,
		Cells:           cellSet,
	}, nil
}

func (bbs *LRPBBS) gatherAndPruneActualLRPs(logger lager.Logger, guids map[string]struct{}) (models.ActualLRPsByProcessGuidAndIndex, error) {
	rootNode, err := bbs.store.ListRecursively(shared.ActualLRPSchemaRoot)
	if err != nil && err != storeadapter.ErrorKeyNotFound {
		return nil, err
	}
	if err == storeadapter.ErrorKeyNotFound {
		logger.Info("actual-lrp-schema-root-not-found")
		return models.ActualLRPsByProcessGuidAndIndex{}, nil
	}

	actuals := models.ActualLRPsByProcessGuidAndIndex{}
	var guidKeysToDelete, indexKeysToDelete []string
	var actualsToDelete []storeadapter.StoreNode
	var guidsLock, actualsLock, guidKeysToDeleteLock, indexKeysToDeleteLock, actualsToDeleteLock sync.Mutex

	logger.Info("walking-actual-lrp-tree")
	wg := sync.WaitGroup{}
	for _, guidGroup := range rootNode.ChildNodes {
		wg.Add(1)
		go func(guidGroup storeadapter.StoreNode) {
			defer wg.Done()
			guidGroupWillBeEmpty := true

			for _, indexGroup := range guidGroup.ChildNodes {
				indexGroupWillBeEmpty := true

				for _, rawActual := range indexGroup.ChildNodes {
					var actual models.ActualLRP
					err := models.FromJSON(rawActual.Value, &actual)
					if err != nil {
						actualsToDeleteLock.Lock()
						actualsToDelete = append(actualsToDelete, rawActual)
						actualsToDeleteLock.Unlock()

						continue
					}

					indexGroupWillBeEmpty = false
					guidGroupWillBeEmpty = false
					guidsLock.Lock()
					guids[actual.ProcessGuid] = struct{}{}
					guidsLock.Unlock()

					if path.Base(rawActual.Key) == shared.ActualLRPInstanceKey {
						actualsLock.Lock()
						actuals.Add(actual)
						actualsLock.Unlock()
					}
				}

				if indexGroupWillBeEmpty {
					indexKeysToDeleteLock.Lock()
					indexKeysToDelete = append(indexKeysToDelete, indexGroup.Key)
					indexKeysToDeleteLock.Unlock()
				}
			}

			if guidGroupWillBeEmpty {
				guidKeysToDeleteLock.Lock()
				guidKeysToDelete = append(guidKeysToDelete, guidGroup.Key)
				guidKeysToDeleteLock.Unlock()
			}
		}(guidGroup)
	}
	wg.Wait()
	logger.Info("done-walking-actual-lrp-tree")

	logger.Info("deleting-invalid-actual-lrps", lager.Data{"num-lrps": len(actualsToDelete)})
	err = bbs.store.CompareAndDeleteByIndex(actualsToDelete...)
	if err != nil {
		logger.Error("failed-deleting-invalid-actual-lrps", err, lager.Data{"num-lrps": len(actualsToDelete)})
	} else {
		logger.Info("succeeded-deleting-invalid-actual-lrps", lager.Data{"num-lrps": len(actualsToDelete)})
	}
	actualLRPsDeleted.Add(uint64(len(actualsToDelete)))

	logger.Info("deleting-empty-actual-indices", lager.Data{"num-indices": len(indexKeysToDelete)})
	err = bbs.store.DeleteLeaves(indexKeysToDelete...)
	if err != nil {
		logger.Error("failed-deleting-empty-actual-indices", err, lager.Data{"num-indices": len(indexKeysToDelete)})
	} else {
		logger.Info("succeeded-deleting-empty-actual-indices", lager.Data{"num-indices": len(indexKeysToDelete)})
	}

	logger.Info("deleting-empty-actual-guids", lager.Data{"num-guids": len(guidKeysToDelete)})
	err = bbs.store.DeleteLeaves(guidKeysToDelete...)
	if err != nil {
		logger.Error("failed-deleting-empty-actual-guids", err, lager.Data{"num-guids": len(guidKeysToDelete)})
	} else {
		logger.Info("succeeded-deleting-empty-actual-guids", lager.Data{"num-guids": len(guidKeysToDelete)})
	}

	return actuals, nil
}

func (bbs *LRPBBS) domains(logger lager.Logger) (map[string]struct{}, error) {
	domains := map[string]struct{}{}

	domainRoot, err := bbs.store.ListRecursively(shared.DomainSchemaRoot)
	if err != nil && err != storeadapter.ErrorKeyNotFound {
		logger.Error("failed-to-fetch-domains", err)
		return nil, err
	}

	for _, node := range domainRoot.ChildNodes {
		domains[path.Base(node.Key)] = struct{}{}
	}

	return domains, nil
}

func (bbs *LRPBBS) gatherAndPruneDesiredLRPs(logger lager.Logger, domains, guids map[string]struct{}) (models.DesiredLRPsByProcessGuid, error) {
	rootNode, err := bbs.store.ListRecursively(shared.DesiredLRPSchemaRoot)
	if err != nil && err != storeadapter.ErrorKeyNotFound {
		return nil, err
	}
	if err == storeadapter.ErrorKeyNotFound {
		logger.Info("desired-lrp-schema-root-not-found")
		return models.DesiredLRPsByProcessGuid{}, nil
	}

	desireds := models.DesiredLRPsByProcessGuid{}
	var desiredsToDelete []storeadapter.StoreNode
	var guidsLock, desiredsLock, desiredsToDeleteLock sync.Mutex

	logger.Info("walking-desired-lrp-tree")
	wg := sync.WaitGroup{}
	for _, rawDesired := range rootNode.ChildNodes {
		wg.Add(1)
		go func(rawDesired storeadapter.StoreNode) {
			defer wg.Done()
			var desired models.DesiredLRP
			err := models.FromJSON(rawDesired.Value, &desired)
			if err != nil {
				desiredsToDeleteLock.Lock()
				desiredsToDelete = append(desiredsToDelete, rawDesired)
				desiredsToDeleteLock.Unlock()
			} else {
				desiredsLock.Lock()
				desireds.Add(desired)
				desiredsLock.Unlock()

				guidsLock.Lock()
				guids[desired.ProcessGuid] = struct{}{}
				guidsLock.Unlock()
			}
		}(rawDesired)
	}
	wg.Wait()
	logger.Info("done-walking-desired-lrp-tree")

	logger.Info("deleting-invalid-desired-lrps", lager.Data{"num-lrps": len(desiredsToDelete)})
	bbs.store.CompareAndDeleteByIndex(desiredsToDelete...)
	logger.Info("done-deleting-invalid-desired-lrps", lager.Data{"num-lrps": len(desiredsToDelete)})
	desiredLRPsDeleted.Add(uint64(len(desiredsToDelete)))

	return desireds, nil
}
