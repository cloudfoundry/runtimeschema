package stop_auction_bbs

import (
	"fmt"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
)

func (bbs *StopAuctionBBS) RequestLRPStopAuction(lrp models.LRPStopAuction) error {
	return shared.RetryIndefinitelyOnStoreTimeout(func() error {
		lrp.State = models.LRPStopAuctionStatePending
		lrp.UpdatedAt = bbs.timeProvider.Time().UnixNano()

		value, err := models.ToJSON(lrp)
		if err != nil {
			return err
		}
		return bbs.store.Create(storeadapter.StoreNode{
			Key:   shared.LRPStopAuctionSchemaPath(lrp),
			Value: value,
		})
	})
}

func (bbs *StopAuctionBBS) ClaimLRPStopAuction(lrp models.LRPStopAuction) error {
	originalValue, err := models.ToJSON(lrp)
	if err != nil {
		return err
	}

	lrp.State = models.LRPStopAuctionStateClaimed
	lrp.UpdatedAt = bbs.timeProvider.Time().UnixNano()

	changedValue, err := models.ToJSON(lrp)
	if err != nil {
		return err
	}

	return shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return bbs.store.CompareAndSwap(storeadapter.StoreNode{
			Key:   shared.LRPStopAuctionSchemaPath(lrp),
			Value: originalValue,
		}, storeadapter.StoreNode{
			Key:   shared.LRPStopAuctionSchemaPath(lrp),
			Value: changedValue,
		})
	})
}

func (s *StopAuctionBBS) ResolveLRPStopAuction(lrp models.LRPStopAuction) error {
	err := shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return s.store.Delete(shared.LRPStopAuctionSchemaPath(lrp))
	})
	return err
}

func (bbs *StopAuctionBBS) LRPStopAuctions() ([]models.LRPStopAuction, error) {
	auctions := []models.LRPStopAuction{}

	node, err := bbs.store.ListRecursively(shared.LRPStopAuctionSchemaRoot)
	if err == storeadapter.ErrorKeyNotFound {
		return auctions, nil
	}

	if err != nil {
		return auctions, err
	}

	for _, node := range node.ChildNodes {
		for _, node := range node.ChildNodes {
			var auction models.LRPStopAuction
			err := models.FromJSON(node.Value, &auction)
			if err != nil {
				return auctions, fmt.Errorf("cannot parse auction JSON for key %s: %s", node.Key, err.Error())
			} else {
				auctions = append(auctions, auction)
			}
		}
	}

	return auctions, nil
}

func (bbs *StopAuctionBBS) WatchForLRPStopAuction() (<-chan models.LRPStopAuction, chan<- bool, <-chan error) {
	auctions := make(chan models.LRPStopAuction)

	filter := func(event storeadapter.WatchEvent) (models.LRPStopAuction, bool) {
		switch event.Type {
		case storeadapter.CreateEvent, storeadapter.UpdateEvent:
			var auction models.LRPStopAuction
			err := models.FromJSON(event.Node.Value, &auction)
			if err != nil {
				return models.LRPStopAuction{}, false
			}

			if auction.State == models.LRPStopAuctionStatePending {
				return auction, true
			}
		}
		return models.LRPStopAuction{}, false
	}

	stop, errs := shared.WatchWithFilter(bbs.store, shared.LRPStopAuctionSchemaRoot, auctions, filter)

	return auctions, stop, errs
}
