package stop_auction_bbs

import (
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

type compareAndSwappableLRPStopAuction struct {
	OldIndex          uint64
	NewLRPStopAuction models.LRPStopAuction
}

func (bbs *StopAuctionBBS) ConvergeLRPStopAuctions(kickPendingDuration time.Duration, expireClaimedDuration time.Duration) {
	keysToDelete := []string{}
	auctionsToCAS := []compareAndSwappableLRPStopAuction{}

	dirsToKeep := map[string]struct{}{}
	processDirsToPossiblyDelete := map[string]struct{}{}

	auctionRoot, err := bbs.store.ListRecursively(shared.LRPStopAuctionSchemaRoot)
	if err != nil && err != storeadapter.ErrorKeyNotFound {
		bbs.logger.Error("failed-to-get-stop-auctions", err)
		return
	}

	for _, processDir := range auctionRoot.ChildNodes {
		if len(processDir.ChildNodes) == 0 {
			keysToDelete = append(keysToDelete, processDir.Key)
			continue
		}

		for _, auctionNode := range processDir.ChildNodes {
			auction, err := models.NewLRPStopAuctionFromJSON(auctionNode.Value)
			if err != nil {
				bbs.logger.Info("detected-invalid-stop-auction-json", lager.Data{
					"error":   err.Error(),
					"payload": auctionNode.Value,
				})

				keysToDelete = append(keysToDelete, auctionNode.Key)
				processDirsToPossiblyDelete[processDir.Key] = struct{}{}
				continue
			}

			updatedAt := time.Unix(0, auction.UpdatedAt)

			switch auction.State {
			case models.LRPStopAuctionStatePending:
				if bbs.timeProvider.Time().Sub(updatedAt) > kickPendingDuration {
					bbs.logger.Info("detected-pending-auction", lager.Data{
						"auction":       auction,
						"kick-duration": kickPendingDuration,
					})

					auctionsToCAS = append(auctionsToCAS, compareAndSwappableLRPStopAuction{
						OldIndex:          auctionNode.Index,
						NewLRPStopAuction: auction,
					})
				}

				dirsToKeep[shared.LRPStopAuctionProcessDir(auction)] = struct{}{}

			case models.LRPStopAuctionStateClaimed:
				if bbs.timeProvider.Time().Sub(updatedAt) > expireClaimedDuration {
					bbs.logger.Info("detected-expired-claimed-auction", lager.Data{
						"auction":             auction,
						"expiration-duration": expireClaimedDuration,
					})

					keysToDelete = append(keysToDelete, auctionNode.Key)
					processDirsToPossiblyDelete[processDir.Key] = struct{}{}
				} else {
					dirsToKeep[processDir.Key] = struct{}{}
				}

			default:
				dirsToKeep[processDir.Key] = struct{}{}
			}
		}
	}

	processDirsToDelete := []string{}
	for key := range processDirsToPossiblyDelete {
		if _, ok := dirsToKeep[key]; !ok {
			processDirsToDelete = append(processDirsToDelete, key)
		}
	}

	bbs.store.DeleteLeaves(keysToDelete...)
	bbs.store.DeleteLeaves(processDirsToDelete...)
	bbs.batchCompareAndSwapLRPStopAuctions(auctionsToCAS)
}

func (bbs *StopAuctionBBS) batchCompareAndSwapLRPStopAuctions(auctionsToCAS []compareAndSwappableLRPStopAuction) {
	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(len(auctionsToCAS))
	for _, auctionToCAS := range auctionsToCAS {
		auction := auctionToCAS.NewLRPStopAuction
		newStoreNode := storeadapter.StoreNode{
			Key:   shared.LRPStopAuctionSchemaPath(auction),
			Value: auction.ToJSON(),
		}

		go func(auctionToCAS compareAndSwappableLRPStopAuction, newStoreNode storeadapter.StoreNode) {
			err := bbs.store.CompareAndSwapByIndex(auctionToCAS.OldIndex, newStoreNode)
			if err != nil {
				bbs.logger.Error("failed-to-compare-and-swap", err)
			}

			waitGroup.Done()
		}(auctionToCAS, newStoreNode)
	}

	waitGroup.Wait()
}
