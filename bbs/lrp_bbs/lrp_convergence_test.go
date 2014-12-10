package lrp_bbs_test

import (
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/dropsonde/metric_sender/fake"
	"github.com/cloudfoundry/dropsonde/metrics"
	"github.com/cloudfoundry/storeadapter"

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

		a1 := models.NewActualLRP(processGuid, "instance-guid-1", cellPresence.CellID, "domain", 0, models.ActualLRPStateUnclaimed)
		createAndClaim(a1)

		a2 := models.NewActualLRP(processGuid, "instance-guid-2", cellPresence.CellID, "domain", 1, models.ActualLRPStateUnclaimed)
		createAndClaim(a2)

		_, err := bbs.CreateActualLRP(models.NewActualLRP(unclaimedProcessGuid, "instance-guid-3", "", "domain", 0, models.ActualLRPStateUnclaimed))
		Ω(err).ShouldNot(HaveOccurred())
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

		Context("when an cell is missing", func() {
			BeforeEach(func() {
				etcdClient.Delete(shared.CellSchemaPath(cellPresence.CellID))
			})

			It("should delete LRPs associated with said cell but not the unclaimed LRP", func() {
				lrps, err := bbs.ActualLRPs()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(lrps).Should(HaveLen(1))
				Ω(lrps[0].ProcessGuid).Should(Equal(unclaimedProcessGuid))
			})

			It("should prune LRP directories for apps that are no longer running", func() {
				actual, err := etcdClient.ListRecursively(shared.ActualLRPSchemaRoot)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(actual.ChildNodes).Should(HaveLen(1))
				Ω(actual.ChildNodes[0].Key).Should(Equal(shared.ActualLRPProcessDir(unclaimedProcessGuid)))
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
		})

		Context("when the desired LRP has all its actual LRPs, and there are no extras", func() {
			BeforeEach(func() {
				bbs.DesireLRP(desiredLRP)
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
				bbs.DesireLRP(desiredLRP)
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
		})

		Context("when the desired LRP has only unclaimed actuals", func() {
			BeforeEach(func() {
				desiredLRP.ProcessGuid = unclaimedProcessGuid
				desiredLRP.Instances = 1
				bbs.DesireLRP(desiredLRP)
			})

			It("does not kick the desired LRP", func() {
				commenceWatching()
				bbs.ConvergeLRPs(pollingInterval)

				Consistently(desiredEvents).ShouldNot(Receive())
			})

			It("bumps the compare-and-swapped LRPs convergence counter", func() {
				Ω(sender.GetCounter("ConvergenceLRPsKicked")).Should(Equal(uint64(0)))
				bbs.ConvergeLRPs(pollingInterval)
				Ω(sender.GetCounter("ConvergenceLRPsKicked")).Should(Equal(uint64(0)))
			})

			Context("and the unclaimed actual is stale", func() {
				BeforeEach(func() {
					timeProvider.Increment(pollingInterval + 1*time.Second)
				})

				It("resends a start auction for the actual", func() {
					Ω(startAuctionBBS.LRPStartAuctions()).Should(HaveLen(0))

					commenceWatching()
					bbs.ConvergeLRPs(pollingInterval)

					Ω(startAuctionBBS.LRPStartAuctions()).Should(HaveLen(1))
				})
			})
		})

		Context("when there are extra actual LRPs", func() {
			BeforeEach(func() {
				desiredLRP.Instances = 1
				bbs.DesireLRP(desiredLRP)
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

			Ω([]string{stop1.InstanceGuid, stop2.InstanceGuid}).Should(ConsistOf([]string{
				"instance-guid-1",
				"instance-guid-2",
			}))
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
				Ω(startAuctionBBS.LRPStartAuctions()).Should(HaveLen(0))

				bbs.ConvergeLRPs(pollingInterval)

				Ω(startAuctionBBS.LRPStartAuctions()).Should(HaveLen(0))
			})
		})
	})
})
