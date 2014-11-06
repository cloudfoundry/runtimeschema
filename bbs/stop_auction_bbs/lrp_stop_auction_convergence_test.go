package stop_auction_bbs_test

import (
	"path"
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/dropsonde/metric_sender/fake"
	"github.com/cloudfoundry/dropsonde/metrics"
	"github.com/cloudfoundry/storeadapter"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	processGuid               = "process-guid"
	pendingKickDuration       = 30 * time.Second
	claimedExpirationDuration = 5 * time.Minute
)

var _ = Describe("LrpAuctionConvergence", func() {
	var (
		sender *fake.FakeMetricSender

		stopAuctionEvents <-chan models.LRPStopAuction
	)

	BeforeEach(func() {
		sender = fake.NewFakeMetricSender()
		metrics.Initialize(sender)
	})

	JustBeforeEach(func() {
		stopAuctionEvents, _, _ = bbs.WatchForLRPStopAuction()
		bbs.ConvergeLRPStopAuctions(pendingKickDuration, claimedExpirationDuration)
	})

	It("bumps the convergence counter", func() {
		Ω(sender.GetCounter("ConvergenceLRPStopAuctionRuns")).Should(Equal(uint64(1)))
	})

	It("reports the duration that it took to converge", func() {
		reportedDuration := sender.GetValue("ConvergenceLRPStopAuctionDuration")
		Ω(reportedDuration.Unit).Should(Equal("nanos"))
		Ω(reportedDuration.Value).ShouldNot(BeZero())
	})

	Context("when the LRPAuction has invalid JSON", func() {
		var key = path.Join(shared.LRPStopAuctionSchemaRoot, "process-guid", "1")

		BeforeEach(func() {
			etcdClient.Create(storeadapter.StoreNode{
				Key:   key,
				Value: []byte("ß"),
			})
		})

		It("should be removed", func() {
			_, err := etcdClient.Get(key)
			Ω(err).Should(MatchError(storeadapter.ErrorKeyNotFound))
		})

		It("bumps the pruned counter", func() {
			Ω(sender.GetCounter("ConvergenceLRPStopAuctionsPrunedInvalid")).Should(Equal(uint64(1)))
		})
	})

	Describe("Kicking pending auctions", func() {
		Context("up until the pending duration has passed", func() {
			BeforeEach(func() {
				newPendingStopAuction(processGuid)
				timeProvider.Increment(pendingKickDuration)
			})

			It("does not kick the auctions", func() {
				Consistently(stopAuctionEvents).ShouldNot(Receive())
			})
		})

		Context("when the pending duration has passed", func() {
			var auction models.LRPStopAuction

			BeforeEach(func() {
				auction = newPendingStopAuction(processGuid)
				timeProvider.Increment(pendingKickDuration + time.Second)
				newPendingStopAuction(processGuid)
			})

			It("Kicks auctions that haven't been updated in the given amount of time", func() {
				var noticedOnce models.LRPStopAuction
				Eventually(stopAuctionEvents).Should(Receive(&noticedOnce))
				Ω(noticedOnce.Index).Should(Equal(auction.Index))

				Consistently(stopAuctionEvents).ShouldNot(Receive())
			})

			It("bumps the compare-and-swap counter", func() {
				Ω(sender.GetCounter("ConvergenceLRPStopAuctionsKicked")).Should(Equal(uint64(1)))
			})
		})
	})

	Describe("Deleting very old claimed events", func() {
		Context("up until the claimedExpiration duration", func() {
			BeforeEach(func() {
				newClaimedStopAuction(processGuid)
				timeProvider.Increment(claimedExpirationDuration)
			})

			It("should not delete claimed events", func() {
				Ω(bbs.GetAllLRPStopAuctions()).Should(HaveLen(1))
			})
		})

		Context("when we are past the claimedExpiration duration", func() {
			BeforeEach(func() {
				newClaimedStopAuction(processGuid)
				newClaimedStopAuction("other-process")
				newClaimedStopAuction("process-to-delete")
				timeProvider.Increment(claimedExpirationDuration + time.Second)
				newClaimedStopAuction(processGuid)
				newPendingStopAuction("other-process")
			})

			It("should delete claimed events that have expired", func() {
				Ω(bbs.GetAllLRPStopAuctions()).Should(HaveLen(2))
			})

			It("should prune stop auction directories for events that have expired", func() {
				stopAuctionRoot, err := etcdClient.ListRecursively(shared.LRPStopAuctionSchemaRoot)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(stopAuctionRoot.ChildNodes).Should(HaveLen(2))
			})

			It("bumps the pruned counter", func() {
				Ω(sender.GetCounter("ConvergenceLRPStopAuctionsPrunedExpired")).Should(Equal(uint64(3)))
			})
		})
	})
})

var auctionIndex = 0

func newStopAuction(processGuid string) models.LRPStopAuction {
	auctionIndex += 1
	return models.LRPStopAuction{
		Index:       auctionIndex,
		ProcessGuid: processGuid,
	}
}

func newPendingStopAuction(processGuid string) models.LRPStopAuction {
	auction := newStopAuction(processGuid)

	err := bbs.RequestLRPStopAuction(auction)
	Ω(err).ShouldNot(HaveOccurred())
	auction.State = models.LRPStopAuctionStatePending
	auction.UpdatedAt = timeProvider.Time().UnixNano()

	return auction
}

func newClaimedStopAuction(processGuid string) models.LRPStopAuction {
	auction := newPendingStopAuction(processGuid)

	err := bbs.ClaimLRPStopAuction(auction)
	Ω(err).ShouldNot(HaveOccurred())
	auction.State = models.LRPStopAuctionStateClaimed
	auction.UpdatedAt = timeProvider.Time().UnixNano()

	return auction
}
