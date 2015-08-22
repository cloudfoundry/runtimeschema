package actuallrprepository

import (
	"fmt"
	"path"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/workpool"
	"github.com/cloudfoundry/storeadapter"
	"github.com/nu7hatch/gouuid"
	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"
)

//go:generate counterfeiter -o fakes/fake_actuallrprepository.go . ActualLRPRepository
type ActualLRPRepository interface {
	ActualLRPsByProcessGuid(logger lager.Logger, processGuid string) (models.ActualLRPsByIndex, error)
	ActualLRPWithIndex(logger lager.Logger, processGuid string, index int) (*models.ActualLRP, uint64, error)
	CreateRawActualLRP(logger lager.Logger, lrp *models.ActualLRP) error
	CompareAndSwapRawActualLRP(logger lager.Logger, lrp *models.ActualLRP, storeIndex uint64) error
	CompareAndDeleteRawActualLRPKey(logger lager.Logger, lrp *models.ActualLRPKey, storeIndex uint64) error
	DeleteRawActualLRPKey(logger lager.Logger, lrp *models.ActualLRPKey) error
	CreateActualLRP(logger lager.Logger, desiredLRP models.DesiredLRP, index int) error
	CreateActualLRPsForDesired(logger lager.Logger, lrp models.DesiredLRP, indices []uint) []uint
}

func NewActualLRPRepository(store storeadapter.StoreAdapter, clock clock.Clock) *actualLRPRepository {
	return &actualLRPRepository{
		store: store,
		clock: clock,
	}
}

const createActualMaxWorkers = 100

type actualLRPIndexTooLargeError struct {
	actualIndex      int
	desiredInstances int
}

func (e actualLRPIndexTooLargeError) Error() string {
	return fmt.Sprintf("Index %d too large for desired number %d of instances", e.actualIndex, e.desiredInstances)
}

type actualLRPRepository struct {
	store storeadapter.StoreAdapter
	clock clock.Clock
}

func (repo *actualLRPRepository) ActualLRPsByProcessGuid(_ lager.Logger, processGuid string) (models.ActualLRPsByIndex, error) {
	if len(processGuid) == 0 {
		return models.ActualLRPsByIndex{}, bbserrors.ErrNoProcessGuid
	}

	lrps := models.ActualLRPsByIndex{}

	node, err := repo.store.ListRecursively(shared.ActualLRPProcessDir(processGuid))
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

func (repo *actualLRPRepository) ActualLRPWithIndex(
	logger lager.Logger,
	processGuid string,
	index int,
) (*models.ActualLRP, uint64, error) {
	node, err := repo.store.Get(shared.ActualLRPSchemaPath(processGuid, index))
	if err != nil {
		if err != storeadapter.ErrorKeyNotFound {
			logger.Error("failed-to-get-actual-lrp", err)
		}
		return nil, 0, shared.ConvertStoreError(err)
	}

	var lrp models.ActualLRP
	err = models.FromJSON(node.Value, &lrp)

	if err != nil {
		logger.Error("failed-to-unmarshal-actual-lrp", err)
		return nil, 0, err
	}

	return &lrp, node.Index, err
}

func (repo *actualLRPRepository) CreateRawActualLRP(logger lager.Logger, lrp *models.ActualLRP) error {
	value, err := models.ToJSON(lrp)
	if err != nil {
		logger.Error("failed-to-marshal-actual-lrp", err, lager.Data{"actual-lrp": lrp})
		return err
	}

	err = repo.store.Create(storeadapter.StoreNode{
		Key:   shared.ActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index),
		Value: value,
	})

	if err != nil {
		logger.Error("failed-to-create-actual-lrp", err, lager.Data{"actual-lrp": lrp})
		return shared.ConvertStoreError(err)
	}

	return nil
}

func (repo *actualLRPRepository) CompareAndSwapRawActualLRP(
	logger lager.Logger,
	lrp *models.ActualLRP,
	storeIndex uint64,
) error {
	lrpForUpdate := lrp
	lrpForUpdate.ModificationTag.Increment()

	value, err := models.ToJSON(lrpForUpdate)
	if err != nil {
		logger.Error("failed-to-marshal-actual-lrp", err, lager.Data{"actual-lrp": lrpForUpdate})
		return err
	}

	err = repo.store.CompareAndSwapByIndex(storeIndex, storeadapter.StoreNode{
		Key:   shared.ActualLRPSchemaPath(lrpForUpdate.ProcessGuid, lrpForUpdate.Index),
		Value: value,
	})
	if err != nil {
		logger.Error("failed-to-compare-and-swap-actual-lrp", err, lager.Data{"actual-lrp": lrpForUpdate})
		return shared.ConvertStoreError(err)
	}

	return nil
}

func (repo *actualLRPRepository) CompareAndDeleteRawActualLRPKey(
	logger lager.Logger,
	lrp *models.ActualLRPKey,
	storeIndex uint64,
) error {
	err := repo.store.CompareAndDeleteByIndex(storeadapter.StoreNode{
		Key:   shared.ActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index),
		Index: storeIndex,
	})

	if err != nil {
		logger.Error("failed-to-compare-and-delete-actual-lrp", err, lager.Data{"actual-lrp": lrp})
		return shared.ConvertStoreError(err)
	}

	return nil
}

func (repo *actualLRPRepository) DeleteRawActualLRPKey(
	logger lager.Logger,
	lrp *models.ActualLRPKey,
) error {
	err := repo.store.Delete(shared.ActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index))

	if err != nil {
		logger.Error("failed-to-delete-actual-lrp", err, lager.Data{"actual-lrp": lrp})
		return shared.ConvertStoreError(err)
	}

	return nil
}

func (repo *actualLRPRepository) CreateActualLRP(logger lager.Logger, desiredLRP models.DesiredLRP, index int) error {
	logger = logger.Session("create-actual-lrp")
	var err error
	if index >= desiredLRP.Instances {
		err = actualLRPIndexTooLargeError{actualIndex: index, desiredInstances: desiredLRP.Instances}
		logger.Error("actual-lrp-index-too-large", err, lager.Data{"actual-index": index, "desired-instances": desiredLRP.Instances})
		return err
	}

	guid, err := uuid.NewV4()
	if err != nil {
		return err
	}

	actualLRP := models.ActualLRP{
		ActualLRPKey: models.NewActualLRPKey(
			desiredLRP.ProcessGuid,
			index,
			desiredLRP.Domain,
		),
		State: models.ActualLRPStateUnclaimed,
		Since: repo.clock.Now().UnixNano(),
		ModificationTag: models.ModificationTag{
			Epoch: guid.String(),
			Index: 0,
		},
	}

	err = repo.CreateRawActualLRP(logger, &actualLRP)
	if err != nil {
		return err
	}

	return nil
}

func (repo *actualLRPRepository) CreateActualLRPsForDesired(logger lager.Logger, lrp models.DesiredLRP, indices []uint) []uint {
	createdIndicesChan := make(chan uint, len(indices))

	works := make([]func(), len(indices))

	for i, actualIndex := range indices {
		actualIndex := actualIndex
		works[i] = func() {
			err := repo.CreateActualLRP(logger, lrp, int(actualIndex))
			if err != nil {
				logger.Info("failed-creating-actual-lrp", lager.Data{"index": actualIndex, "err-message": err.Error()})
			} else {
				createdIndicesChan <- actualIndex
			}
		}
	}

	throttler, err := workpool.NewThrottler(createActualMaxWorkers, works)
	if err != nil {
		logger.Error("failed-constructing-throttler", err, lager.Data{"max-workers": createActualMaxWorkers, "num-works": len(works)})
		return []uint{}
	}

	throttler.Work()
	close(createdIndicesChan)

	createdIndices := make([]uint, 0, len(indices))
	for createdIndex := range createdIndicesChan {
		createdIndices = append(createdIndices, createdIndex)
	}

	return createdIndices
}

func isInstanceActualLRPNode(node storeadapter.StoreNode) bool {
	return path.Base(node.Key) == shared.ActualLRPInstanceKey
}
