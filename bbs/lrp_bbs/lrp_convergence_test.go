package lrp_bbs_test

import (
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/dropsonde/metric_sender/fake"
	"github.com/cloudfoundry/dropsonde/metrics"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LrpConvergence", func() {
	const pollingInterval = 5 * time.Second

	var (
		sender *fake.FakeMetricSender

		cellPresence models.CellPresence
	)

	processGuid := "process-guid"
	unclaimedProcessGuid := "unclaimed-process-guid"

	BeforeEach(func() {
		sender = fake.NewFakeMetricSender()
		metrics.Initialize(sender)

		cellPresence = models.CellPresence{
			CellID:     "the-cell-id",
			Stack:      "the-stack",
			RepAddress: "cell.example.com",
		}
		registerCell(cellPresence)

		auctioneerPresence := models.AuctioneerPresence{
			AuctioneerID:      "auctioneer-id",
			AuctioneerAddress: "example.com",
		}
		registerAuctioneer(auctioneerPresence)

		desiredLRP := models.DesiredLRP{ProcessGuid: processGuid, Domain: "domain", Instances: 2}
		createAndClaim(desiredLRP, 0, models.NewActualLRPContainerKey("instance-guid-1", cellPresence.CellID))
		createAndClaim(desiredLRP, 1, models.NewActualLRPContainerKey("instance-guid-2", cellPresence.CellID))

		unclaimedDesiredLRP := models.DesiredLRP{ProcessGuid: unclaimedProcessGuid, Domain: "another-domain", Instances: 1}
		_, err := bbs.CreateActualLRP(unclaimedDesiredLRP, 0, logger)
		Ω(err).ShouldNot(HaveOccurred())
	})

	It("logs the convergence", func() {
		bbs.ConvergeLRPs(pollingInterval)
		logMessages := logger.TestSink.LogMessages()
		Ω(logMessages[0]).Should(Equal(
			"test.converge-lrps.starting-convergence",
		))
		Ω(logMessages[len(logMessages)-1]).Should(Equal(
			"test.converge-lrps.finished-convergence",
		))
	})

	It("bumps the convergence counter", func() {
		Ω(sender.GetCounter("ConvergenceLRPRuns")).Should(Equal(uint64(0)))
		bbs.ConvergeLRPs(pollingInterval)
		Ω(sender.GetCounter("ConvergenceLRPRuns")).Should(Equal(uint64(1)))
		bbs.ConvergeLRPs(pollingInterval)
		Ω(sender.GetCounter("ConvergenceLRPRuns")).Should(Equal(uint64(2)))
	})

	It("reports the duration that it took to converge", func() {
		timeProvider.IntervalToAdvance = 500 * time.Nanosecond
		bbs.ConvergeLRPs(pollingInterval)

		reportedDuration := sender.GetValue("ConvergenceLRPDuration")
		Ω(reportedDuration.Unit).Should(Equal("nanos"))
		Ω(reportedDuration.Value).ShouldNot(BeZero())
	})

	Describe("pruning LRPs by cell", func() {
		JustBeforeEach(func() {
			bbs.ConvergeLRPs(pollingInterval)
		})

		Context("when no cell is missing", func() {
			It("should not prune any LRPs", func() {
				Ω(bbs.ActualLRPs()).Should(HaveLen(3))
			})
		})

		Context("when a cell is missing", func() {
			BeforeEach(func() {
				etcdClient.Delete(shared.CellSchemaPath(cellPresence.CellID))
			})

			It("should delete LRPs associated with said cell but not the unclaimed LRP", func() {
				lrps, err := bbs.ActualLRPs()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(lrps).Should(HaveLen(1))

				Ω(lrps[0].Index).Should(Equal(0))
				Ω(lrps[0].ProcessGuid).Should(Equal(unclaimedProcessGuid))
				Ω(lrps[0].State).Should(Equal(models.ActualLRPStateUnclaimed))
			})

			It("should prune LRP directories for apps that are no longer running", func() {
				actual, err := etcdClient.ListRecursively(shared.ActualLRPSchemaRoot)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(actual.ChildNodes).Should(HaveLen(1))
				Ω(actual.ChildNodes[0].Key).Should(Equal(shared.ActualLRPProcessDir(unclaimedProcessGuid)))
			})

			It("logs", func() {
				Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-lrps.detected-actual-with-missing-cell"))
			})
		})
	})

	Describe("when there is a desired LRP", func() {
		var desiredEvents <-chan models.DesiredLRPChange
		var desiredLRP models.DesiredLRP

		commenceWatching := func() {
			desiredEvents, _, _ = bbs.WatchForDesiredLRPChanges()
		}

		BeforeEach(func() {
			desiredLRP = models.DesiredLRP{
				Domain:      "tests",
				ProcessGuid: processGuid,
				Instances:   2,
				Stack:       "pancake",
				Action: &models.DownloadAction{
					From: "http://example.com",
					To:   "/tmp/internet",
				},
			}
		})

		Context("when the desired LRP has malformed JSON", func() {
			BeforeEach(func() {
				err := etcdClient.SetMulti([]storeadapter.StoreNode{
					{
						Key:   shared.DesiredLRPSchemaPathByProcessGuid("bogus-desired"),
						Value: []byte("ß"),
					},
				})

				Ω(err).ShouldNot(HaveOccurred())
			})

			It("should delete the bogus entry", func() {
				bbs.ConvergeLRPs(pollingInterval)
				_, err := etcdClient.Get(shared.DesiredLRPSchemaPathByProcessGuid("bogus-desired"))
				Ω(err).Should(MatchError(storeadapter.ErrorKeyNotFound))
			})

			It("bumps the deleted LRPs convergence counter", func() {
				Ω(sender.GetCounter("ConvergenceLRPsDeleted")).Should(Equal(uint64(0)))
				bbs.ConvergeLRPs(pollingInterval)
				Ω(sender.GetCounter("ConvergenceLRPsDeleted")).Should(Equal(uint64(1)))
			})

			It("logs", func() {
				bbs.ConvergeLRPs(pollingInterval)
				Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-lrps.pruning-invalid-desired-lrp-json"))
			})
		})

		Context("when the desired LRP has all its actual LRPs, and there are no extras", func() {
			BeforeEach(func() {
				err := bbs.DesireLRP(desiredLRP)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("should not kick the desired LRP", func() {
				commenceWatching()
				bbs.ConvergeLRPs(pollingInterval)

				Consistently(desiredEvents).ShouldNot(Receive())
			})
		})

		Context("when the desired LRP is missing actuals", func() {
			BeforeEach(func() {
				desiredLRP.Instances = 3
				err := bbs.DesireLRP(desiredLRP)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("should kick the desired LRP", func() {
				commenceWatching()
				bbs.ConvergeLRPs(pollingInterval)

				var noticedOnce models.DesiredLRPChange
				Eventually(desiredEvents).Should(Receive(&noticedOnce))
				Ω(*noticedOnce.After).Should(Equal(desiredLRP))
			})

			It("bumps the compare-and-swapped LRPs convergence counter", func() {
				Ω(sender.GetCounter("ConvergenceLRPsKicked")).Should(Equal(uint64(0)))
				bbs.ConvergeLRPs(pollingInterval)
				Ω(sender.GetCounter("ConvergenceLRPsKicked")).Should(Equal(uint64(1)))
			})

			It("logs", func() {
				bbs.ConvergeLRPs(pollingInterval)
				Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-lrps.detected-missing-instance"))
			})
		})

		Context("when the desired LRP has stale unclaimed actuals", func() {
			BeforeEach(func() {
				desiredLRP.ProcessGuid = unclaimedProcessGuid
				desiredLRP.Instances = 1

				err := bbs.DesireLRP(desiredLRP)
				Ω(err).ShouldNot(HaveOccurred())

				timeProvider.Increment(pollingInterval + 1*time.Second)
			})

			It("resends a start auction for the unclaimed actual", func() {
				originalCallCount := fakeAuctioneerClient.RequestLRPStartAuctionCallCount()

				commenceWatching()
				bbs.ConvergeLRPs(pollingInterval)

				Ω(fakeAuctioneerClient.RequestLRPStartAuctionCallCount()).Should(Equal(originalCallCount + 1))

				_, auction := fakeAuctioneerClient.RequestLRPStartAuctionArgsForCall(originalCallCount)
				Ω(auction.DesiredLRP).Should(Equal(desiredLRP))
				Ω(auction.Index).Should(Equal(0))
			})

			It("logs", func() {
				commenceWatching()
				bbs.ConvergeLRPs(pollingInterval)
				Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-lrps.resending-start-auction"))
			})
		})

		Context("when there are extra actual LRPs", func() {
			BeforeEach(func() {
				desiredLRP.Instances = 1
				err := bbs.DesireLRP(desiredLRP)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("should kick the desired LRP", func() {
				commenceWatching()
				bbs.ConvergeLRPs(pollingInterval)

				var noticedOnce models.DesiredLRPChange
				Eventually(desiredEvents).Should(Receive(&noticedOnce))
				Ω(*noticedOnce.After).Should(Equal(desiredLRP))
			})

			It("bumps the compare-and-swapped LRPs convergence counter", func() {
				Ω(sender.GetCounter("ConvergenceLRPsKicked")).Should(Equal(uint64(0)))
				bbs.ConvergeLRPs(pollingInterval)
				Ω(sender.GetCounter("ConvergenceLRPsKicked")).Should(Equal(uint64(1)))
			})

			It("logs", func() {
				bbs.ConvergeLRPs(pollingInterval)
				Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-lrps.detected-extra-instance"))
				Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-lrps.kicking-desired-lrp"))
			})
		})
	})

	Context("when there is an actual LRP with no matching desired LRP", func() {
		It("should emit a stop for the actual LRP", func() {
			bbs.ConvergeLRPs(pollingInterval)
			Ω(fakeCellClient.StopLRPInstanceCallCount()).Should(Equal(2))

			addr1, stop1 := fakeCellClient.StopLRPInstanceArgsForCall(0)
			addr2, stop2 := fakeCellClient.StopLRPInstanceArgsForCall(1)

			Ω(addr1).Should(Equal(cellPresence.RepAddress))
			Ω(addr2).Should(Equal(cellPresence.RepAddress))

			Ω(stop1.ProcessGuid).Should(Equal(processGuid))
			Ω(stop2.ProcessGuid).Should(Equal(processGuid))
			Ω([]int{stop1.Index, stop2.Index}).Should(ConsistOf([]int{0, 1}))
		})

		It("logs", func() {
			bbs.ConvergeLRPs(pollingInterval)
			Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-lrps.detected-undesired-instance"))
		})

		It("bumps the stopped LRPs convergence counter", func() {
			Ω(sender.GetCounter("ConvergenceLRPsStopped")).Should(Equal(uint64(0)))
			bbs.ConvergeLRPs(pollingInterval)
			Ω(sender.GetCounter("ConvergenceLRPsStopped")).Should(Equal(uint64(2)))
		})

		Context("and there is a stale unclaimed actual", func() {
			BeforeEach(func() {
				timeProvider.Increment(pollingInterval + 1*time.Second)
			})

			It("does not resends a start auction for the actual", func() {
				originalAuctionCallCount := fakeAuctioneerClient.RequestLRPStartAuctionCallCount()
				bbs.ConvergeLRPs(pollingInterval)
				Consistently(fakeAuctioneerClient.RequestLRPStartAuctionCallCount).Should(Equal(originalAuctionCallCount))
			})

			It("logs", func() {
				bbs.ConvergeLRPs(pollingInterval)
				Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-lrps.failed-to-find-desired-lrp-for-stale-unclaimed-actual-lrp"))
			})
		})
	})

	It("Sends start auctions for actual LRPs that are UNCLAIMED for too long", func() {
		desiredLRP := models.DesiredLRP{
			ProcessGuid: "some-process-guid",
			Domain:      "some-domain",
			Instances:   1,
			Stack:       "pancake",
			Action: &models.DownloadAction{
				From: "http://example.com",
				To:   "/tmp/internet",
			},
		}
		_, err := bbs.CreateActualLRP(desiredLRP, 0, lagertest.NewTestLogger("test"))
		Ω(err).ShouldNot(HaveOccurred())

		// make sure created (UNCLAIMED) actual LRP has been in that state for longer than
		// the staelness threshhold of pollingInterval
		timeProvider.Increment(pollingInterval + 1*time.Second)

		err = bbs.DesireLRP(desiredLRP)
		Ω(err).ShouldNot(HaveOccurred())

		originalAuctionCallCount := fakeAuctioneerClient.RequestLRPStartAuctionCallCount()
		bbs.ConvergeLRPs(pollingInterval)
		Consistently(fakeAuctioneerClient.RequestLRPStartAuctionCallCount).Should(Equal(originalAuctionCallCount + 1))

		_, startAuction := fakeAuctioneerClient.RequestLRPStartAuctionArgsForCall(originalAuctionCallCount)
		Ω(startAuction.DesiredLRP).Should(Equal(desiredLRP))
		Ω(startAuction.Index).Should(Equal(0))
	})
})
