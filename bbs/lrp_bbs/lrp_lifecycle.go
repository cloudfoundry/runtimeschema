package lrp_bbs

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/workpool"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

type actualLRPIndexTooLargeError struct {
	actualIndex      int
	desiredInstances int
}

func (e actualLRPIndexTooLargeError) Error() string {
	return fmt.Sprintf("Index %d too large for desired number %d of instances", e.actualIndex, e.desiredInstances)
}

func NewActualLRPIndexTooLargeError(actualIndex, desiredInstances int) error {
	return actualLRPIndexTooLargeError{actualIndex: actualIndex, desiredInstances: desiredInstances}
}

func (bbs *LRPBBS) CreateActualLRP(desiredLRP models.DesiredLRP, index int, logger lager.Logger) error {
	logger = logger.Session("create-actual-lrp")
	var err error
	if index >= desiredLRP.Instances {
		err = NewActualLRPIndexTooLargeError(index, desiredLRP.Instances)
		logger.Error("actual-lrp-index-too-large", err, lager.Data{"actual-index": index, "desired-instances": desiredLRP.Instances})
		return err
	}

	actualLRP := models.ActualLRP{
		ActualLRPKey: models.NewActualLRPKey(
			desiredLRP.ProcessGuid,
			index,
			desiredLRP.Domain,
		),
		State: models.ActualLRPStateUnclaimed,
		Since: bbs.timeProvider.Now().UnixNano(),
	}

	err = bbs.createRawActualLRP(&actualLRP, logger)
	if err != nil {
		return err
	}

	lrpStart := models.LRPStart{
		DesiredLRP: desiredLRP,
		Index:      index,
	}

	err = bbs.requestLRPStartAuction(lrpStart)
	if err != nil {
		logger.Error("failed-to-request-start-auction", err, lager.Data{"lrp-start": lrpStart})
		// The creation succeeded, the start request error can be dropped
	}

	return nil
}

func (bbs *LRPBBS) createRawActualLRP(lrp *models.ActualLRP, logger lager.Logger) error {
	value, err := models.ToJSON(lrp)
	if err != nil {
		logger.Error("failed-to-marshal-actual-lrp", err, lager.Data{"actual-lrp": lrp})
		return err
	}

	err = shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return bbs.store.Create(storeadapter.StoreNode{
			Key:   shared.ActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index),
			Value: value,
		})
	})

	if err != nil {
		logger.Error("failed-to-create-actual-lrp", err, lager.Data{"actual-lrp": lrp})
	}

	return err
}

func (bbs *LRPBBS) ClaimActualLRP(
	key models.ActualLRPKey,
	containerKey models.ActualLRPContainerKey,
	logger lager.Logger,
) error {
	logger = logger.Session("claim-actual-lrp")
	lrp, index, err := bbs.getActualLRP(key.ProcessGuid, key.Index)
	if err == bbserrors.ErrStoreResourceNotFound {
		return bbserrors.ErrActualLRPCannotBeClaimed
	} else if err != nil {
		return err
	}

	if lrp.ActualLRPKey == key &&
		lrp.ActualLRPContainerKey == containerKey &&
		lrp.State == models.ActualLRPStateClaimed {
		return nil
	}

	if !lrp.AllowsTransitionTo(key, containerKey, models.ActualLRPStateClaimed) {
		return bbserrors.ErrActualLRPCannotBeClaimed
	}

	lrp.Since = bbs.timeProvider.Now().UnixNano()
	lrp.State = models.ActualLRPStateClaimed
	lrp.ActualLRPContainerKey = containerKey
	lrp.ActualLRPNetInfo = models.NewActualLRPNetInfo("", nil)

	value, err := models.ToJSON(lrp)
	if err != nil {
		logger.Error("failed-to-marshal-actual-lrp", err, lager.Data{"actual-lrp": lrp})
		return err
	}

	err = shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return bbs.store.CompareAndSwapByIndex(index, storeadapter.StoreNode{
			Key:   shared.ActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index),
			Value: value,
		})
	})

	if err != nil {
		logger.Error("failed-to-compare-and-swap-actual-lrp", err, lager.Data{"actual-lrp": lrp})
	}

	return err
}

func (bbs *LRPBBS) StartActualLRP(
	key models.ActualLRPKey,
	containerKey models.ActualLRPContainerKey,
	netInfo models.ActualLRPNetInfo,
	logger lager.Logger,
) error {
	logger = logger.Session("start-actual-lrp")
	lrp, index, err := bbs.getActualLRP(key.ProcessGuid, key.Index)
	if err == bbserrors.ErrStoreResourceNotFound {
		newLRP := bbs.newRunningActualLRP(key, containerKey, netInfo)
		return bbs.createRawActualLRP(&newLRP, bbs.logger)

	} else if err != nil {
		return err
	}

	if lrp.ActualLRPKey == key &&
		lrp.ActualLRPContainerKey == containerKey &&
		lrp.Host == netInfo.Host &&
		reflect.DeepEqual(lrp.Ports, netInfo.Ports) &&
		lrp.State == models.ActualLRPStateRunning {
		return nil
	}

	if !lrp.AllowsTransitionTo(key, containerKey, models.ActualLRPStateRunning) {
		return bbserrors.ErrActualLRPCannotBeStarted
	}

	lrp.State = models.ActualLRPStateRunning
	lrp.Since = bbs.timeProvider.Now().UnixNano()
	lrp.ActualLRPContainerKey = containerKey
	lrp.ActualLRPNetInfo = netInfo

	value, err := models.ToJSON(lrp)
	if err != nil {
		logger.Error("failed-to-marshal-actual-lrp", err, lager.Data{"actual-lrp": lrp})
		return err
	}

	err = shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return bbs.store.CompareAndSwapByIndex(index, storeadapter.StoreNode{
			Key:   shared.ActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index),
			Value: value,
		})
	})

	if err != nil {
		logger.Error("failed-to-compare-and-swap-actual-lrp", err, lager.Data{"actual-lrp": lrp})
	}

	return err

}

func (bbs *LRPBBS) RemoveActualLRP(
	key models.ActualLRPKey,
	containerKey models.ActualLRPContainerKey,
	logger lager.Logger,
) error {
	logger = logger.Session("remove-actual-lrp")
	return shared.RetryIndefinitelyOnStoreTimeout(func() error {
		lrp, storeIndex, err := bbs.getActualLRP(key.ProcessGuid, key.Index)
		if err != nil {
			return err
		}

		if lrp.ActualLRPKey == key &&
			lrp.ActualLRPContainerKey == containerKey {
			err = bbs.store.CompareAndDeleteByIndex(storeadapter.StoreNode{
				Key:   shared.ActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index),
				Index: storeIndex,
			})

			if err != nil {
				logger.Error("failed-to-compare-and-delete-actual-lrp", err, lager.Data{"actual-lrp": lrp})
			}

			return err

		} else {
			return bbserrors.ErrStoreComparisonFailed
		}
	})
}

func (bbs *LRPBBS) RetireActualLRPs(lrps []models.ActualLRP, logger lager.Logger) {
	logger = logger.Session("retire-actual-lrps")

	pool := workpool.NewWorkPool(workerPoolSize)

	wg := new(sync.WaitGroup)
	wg.Add(len(lrps))

	for _, lrp := range lrps {
		lrp := lrp
		pool.Submit(func() {
			defer wg.Done()

			err := bbs.retireActualLRP(lrp, logger)
			if err != nil {
				logger.Error("failed-to-retire", err, lager.Data{
					"lrp": lrp,
				})
			}
		})
	}

	wg.Wait()

	pool.Stop()
}

func (bbs *LRPBBS) retireActualLRP(lrp models.ActualLRP, logger lager.Logger) error {
	var err error

	if lrp.State == models.ActualLRPStateUnclaimed {
		err = bbs.RemoveActualLRP(lrp.ActualLRPKey, lrp.ActualLRPContainerKey, logger)
	} else {
		err = bbs.RequestStopLRPInstance(lrp)
	}

	if err != nil {
		logger.Error("failed-to-retire-actual-lrp", err, lager.Data{
			"actual-lrp": lrp,
		})
	}

	return err
}

func (bbs *LRPBBS) getActualLRP(processGuid string, index int) (*models.ActualLRP, uint64, error) {
	var node storeadapter.StoreNode
	err := shared.RetryIndefinitelyOnStoreTimeout(func() error {
		var err error
		node, err = bbs.store.Get(shared.ActualLRPSchemaPath(processGuid, index))
		return err
	})

	if err != nil {
		return nil, 0, err
	}

	var lrp models.ActualLRP
	err = models.FromJSON(node.Value, &lrp)

	return &lrp, node.Index, err

}

func (bbs *LRPBBS) newRunningActualLRP(
	key models.ActualLRPKey,
	containerKey models.ActualLRPContainerKey,
	netInfo models.ActualLRPNetInfo,
) models.ActualLRP {
	return models.ActualLRP{
		ActualLRPKey:          key,
		ActualLRPContainerKey: containerKey,
		ActualLRPNetInfo:      netInfo,
		Since:                 bbs.timeProvider.Now().UnixNano(),
		State:                 models.ActualLRPStateRunning,
	}
}

func (bbs *LRPBBS) requestLRPStartAuction(lrpStart models.LRPStart) error {
	auctioneerAddress, err := bbs.services.AuctioneerAddress()
	if err != nil {
		return err
	}

	return bbs.auctioneerClient.RequestLRPStartAuction(auctioneerAddress, lrpStart)
}
