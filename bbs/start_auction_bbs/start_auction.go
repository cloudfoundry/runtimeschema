package start_auction_bbs

import (
	"fmt"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
)

func (bbs *StartAuctionBBS) RequestLRPStartAuction(lrp models.LRPStartAuction) error {
	err := lrp.Validate()
	if err != nil {
		return err
	}

	return shared.RetryIndefinitelyOnStoreTimeout(func() error {
		lrp.State = models.LRPStartAuctionStatePending
		lrp.UpdatedAt = bbs.timeProvider.Time().UnixNano()
		value, err := models.ToJSON(lrp)
		if err != nil {
			return err
		}

		return bbs.store.Create(storeadapter.StoreNode{
			Key:   shared.LRPStartAuctionSchemaPath(lrp),
			Value: value,
		})
	})
}

func (bbs *StartAuctionBBS) ClaimLRPStartAuction(lrp models.LRPStartAuction) error {
	originalValue, err := models.ToJSON(lrp)
	if err != nil {
		return err
	}

	lrp.State = models.LRPStartAuctionStateClaimed
	lrp.UpdatedAt = bbs.timeProvider.Time().UnixNano()
	changedValue, err := models.ToJSON(lrp)
	if err != nil {
		return err
	}

	return shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return bbs.store.CompareAndSwap(storeadapter.StoreNode{
			Key:   shared.LRPStartAuctionSchemaPath(lrp),
			Value: originalValue,
		}, storeadapter.StoreNode{
			Key:   shared.LRPStartAuctionSchemaPath(lrp),
			Value: changedValue,
		})
	})
}

func (s *StartAuctionBBS) ResolveLRPStartAuction(lrp models.LRPStartAuction) error {
	err := shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return s.store.Delete(shared.LRPStartAuctionSchemaPath(lrp))
	})
	return err
}

func (bbs *StartAuctionBBS) LRPStartAuctions() ([]models.LRPStartAuction, error) {
	auctions := []models.LRPStartAuction{}

	node, err := bbs.store.ListRecursively(shared.LRPStartAuctionSchemaRoot)
	if err == storeadapter.ErrorKeyNotFound {
		return auctions, nil
	}

	if err != nil {
		return auctions, err
	}

	for _, node := range node.ChildNodes {
		for _, node := range node.ChildNodes {
			var auction models.LRPStartAuction
			err := models.FromJSON(node.Value, &auction)
			if err != nil {
				return auctions, fmt.Errorf("cannot parse lrp JSON for key %s: %s", node.Key, err.Error())
			} else {
				auctions = append(auctions, auction)
			}
		}
	}

	return auctions, nil
}

func (bbs *StartAuctionBBS) WatchForLRPStartAuction() (<-chan models.LRPStartAuction, chan<- bool, <-chan error) {
	auctions := make(chan models.LRPStartAuction)

	filter := func(event storeadapter.WatchEvent) (models.LRPStartAuction, bool) {
		switch event.Type {
		case storeadapter.CreateEvent, storeadapter.UpdateEvent:
			var auction models.LRPStartAuction
			err := models.FromJSON(event.Node.Value, &auction)
			if err != nil {
				return models.LRPStartAuction{}, false
			}

			if auction.State == models.LRPStartAuctionStatePending {
				return auction, true
			}
		}
		return models.LRPStartAuction{}, false
	}

	stop, errs := shared.WatchWithFilter(bbs.store, shared.LRPStartAuctionSchemaRoot, auctions, filter)

	return auctions, stop, errs
}
