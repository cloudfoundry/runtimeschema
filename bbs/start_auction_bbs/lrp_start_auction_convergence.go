package start_auction_bbs

import (
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

type compareAndSwappableLRPStartAuction struct {
	OldIndex           uint64
	NewLRPStartAuction models.LRPStartAuction
}

func (bbs *StartAuctionBBS) ConvergeLRPStartAuctions(kickPendingDuration time.Duration, expireClaimedDuration time.Duration) {
	keysToDelete := []string{}
	auctionsToCAS := []compareAndSwappableLRPStartAuction{}

	dirsToKeep := map[string]struct{}{}
	processDirsToPossiblyDelete := map[string]struct{}{}

	auctionRoot, err := bbs.store.ListRecursively(shared.LRPStartAuctionSchemaRoot)
	if err != nil && err != storeadapter.ErrorKeyNotFound {
		bbs.logger.Error("failed-to-get-start-auctions", err)
		return
	}

	for _, processDir := range auctionRoot.ChildNodes {
		if len(processDir.ChildNodes) == 0 {
			keysToDelete = append(keysToDelete, processDir.Key)
			continue
		}

		for _, auctionNode := range processDir.ChildNodes {
			auction, err := models.NewLRPStartAuctionFromJSON(auctionNode.Value)
			if err != nil {
				bbs.logger.Info("detected-invalid-start-auction-json", lager.Data{
					"error":   err.Error(),
					"payload": auctionNode.Value,
				})

				keysToDelete = append(keysToDelete, auctionNode.Key)
				processDirsToPossiblyDelete[processDir.Key] = struct{}{}
				continue
			}

			updatedAt := time.Unix(0, auction.UpdatedAt)

			switch auction.State {
			case models.LRPStartAuctionStatePending:
				if bbs.timeProvider.Time().Sub(updatedAt) > kickPendingDuration {
					bbs.logger.Info("detected-pending-auction", lager.Data{
						"auction":       auction,
						"kick-duration": kickPendingDuration,
					})

					auctionsToCAS = append(auctionsToCAS, compareAndSwappableLRPStartAuction{
						OldIndex:           auctionNode.Index,
						NewLRPStartAuction: auction,
					})
				}

				dirsToKeep[processDir.Key] = struct{}{}

			case models.LRPStartAuctionStateClaimed:
				if bbs.timeProvider.Time().Sub(updatedAt) > expireClaimedDuration {
					bbs.logger.Info("detected-expired-claim", lager.Data{
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
	bbs.batchCompareAndSwapLRPStartAuctions(auctionsToCAS)
}

func (bbs *StartAuctionBBS) batchCompareAndSwapLRPStartAuctions(auctionsToCAS []compareAndSwappableLRPStartAuction) {
	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(len(auctionsToCAS))
	for _, auctionToCAS := range auctionsToCAS {
		auction := auctionToCAS.NewLRPStartAuction
		newStoreNode := storeadapter.StoreNode{
			Key:   shared.LRPStartAuctionSchemaPath(auction),
			Value: auction.ToJSON(),
		}

		go func(auctionToCAS compareAndSwappableLRPStartAuction, newStoreNode storeadapter.StoreNode) {
			err := bbs.store.CompareAndSwapByIndex(auctionToCAS.OldIndex, newStoreNode)
			if err != nil {
				bbs.logger.Error("failed-to-compare-and-swap", err)
			}

			waitGroup.Done()
		}(auctionToCAS, newStoreNode)
	}

	waitGroup.Wait()
}
