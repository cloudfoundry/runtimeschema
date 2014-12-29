package lrp_bbs_test

import (
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
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
	var dummyAction = &models.DownloadAction{
		From: "http://example.com",
		To:   "/tmp/internet",
	}

	var (
		sender *fake.FakeMetricSender
	)

	BeforeEach(func() {
		sender = fake.NewFakeMetricSender()
		metrics.Initialize(sender)
	})

	Describe("converging missing actual LRPs", func() {
		var processGuid string
		var desiredLRP models.DesiredLRP

		BeforeEach(func() {
			processGuid = "process-guid-for-missing"
		})

		Context("when there are no actuals for desired LRP", func() {
			BeforeEach(func() {
				desiredLRP = models.DesiredLRP{
					ProcessGuid: processGuid,
					Instances:   1,
					Domain:      "some-domain",
					Stack:       "pancake",
					Action:      dummyAction,
				}

				err := bbs.DesireLRP(desiredLRP)
				Ω(err).ShouldNot(HaveOccurred())

				actuals, err := bbs.ActualLRPsByProcessGuid(desiredLRP.ProcessGuid)
				Ω(err).ShouldNot(HaveOccurred())

				for _, actual := range actuals {
					err := bbs.RemoveActualLRP(actual.ActualLRPKey, actual.ActualLRPContainerKey, logger)
					Ω(err).ShouldNot(HaveOccurred())
				}
			})

			Context("when an auctioneer is present", func() {
				BeforeEach(func() {
					auctioneerPresence := models.NewAuctioneerPresence("auctioneer-id", "example.com")
					registerAuctioneer(auctioneerPresence)
				})

				It("emits start auction requests", func() {
					originalAuctionCallCount := fakeAuctioneerClient.RequestLRPStartAuctionsCallCount()

					bbs.ConvergeLRPs(pollingInterval)

					Consistently(fakeAuctioneerClient.RequestLRPStartAuctionsCallCount).Should(Equal(originalAuctionCallCount + 1))

					_, startAuctions := fakeAuctioneerClient.RequestLRPStartAuctionsArgsForCall(originalAuctionCallCount)
					Ω(startAuctions).Should(HaveLen(1))
					Ω(startAuctions[0].DesiredLRP).Should(Equal(desiredLRP))
					Ω(startAuctions[0].Index).Should(Equal(0))
				})

				It("bumps the compare-and-swapped LRPs convergence counter", func() {
					requestsBefore := sender.GetCounter("LRPInstanceStartRequests")
					bbs.ConvergeLRPs(pollingInterval)
					Ω(sender.GetCounter("LRPInstanceStartRequests")).Should(Equal(requestsBefore + 1))
				})
			})

			It("logs", func() {
				bbs.ConvergeLRPs(pollingInterval)
				Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-lrps.request-start"))
			})

			It("logs the convergence", func() {
				bbs.ConvergeLRPs(pollingInterval)
				logMessages := logger.TestSink.LogMessages()
				Ω(logMessages).Should(ContainElement(
					"test.converge-lrps.starting-convergence",
				))
				Ω(logMessages).Should(ContainElement(
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
		})

		Context("when there are fewer actuals for desired LRP", func() {
			BeforeEach(func() {
				desiredLRP = models.DesiredLRP{
					ProcessGuid: processGuid,
					Instances:   2,
					Domain:      "some-domain",
					Stack:       "pancake",
					Action:      dummyAction,
				}

				err := bbs.DesireLRP(desiredLRP)
				Ω(err).ShouldNot(HaveOccurred())

				actualLRPs, err := bbs.ActualLRPsByProcessGuid(processGuid)
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.RemoveActualLRP(actualLRPs[1].ActualLRPKey, actualLRPs[1].ActualLRPContainerKey, logger)
				Ω(err).ShouldNot(HaveOccurred())
			})

			Context("when an auctioneer is present", func() {
				BeforeEach(func() {
					auctioneerPresence := models.NewAuctioneerPresence("auctioneer-id", "example.com")
					registerAuctioneer(auctioneerPresence)
				})

				It("emits start auction requests", func() {
					originalAuctionCallCount := fakeAuctioneerClient.RequestLRPStartAuctionsCallCount()

					bbs.ConvergeLRPs(pollingInterval)

					Consistently(fakeAuctioneerClient.RequestLRPStartAuctionsCallCount).Should(Equal(originalAuctionCallCount + 1))

					_, startAuctions := fakeAuctioneerClient.RequestLRPStartAuctionsArgsForCall(originalAuctionCallCount)
					Ω(startAuctions).Should(HaveLen(1))
					Ω(startAuctions[0].DesiredLRP).Should(Equal(desiredLRP))
					Ω(startAuctions[0].Index).Should(Equal(1))
				})
			})

			It("bumps the LRP start request counter", func() {
				requestsBefore := sender.GetCounter("LRPInstanceStartRequests")
				bbs.ConvergeLRPs(pollingInterval)
				Ω(sender.GetCounter("LRPInstanceStartRequests")).Should(Equal(requestsBefore + 1))
			})

			It("logs", func() {
				bbs.ConvergeLRPs(pollingInterval)
				Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-lrps.request-start"))
			})

			It("logs the convergence", func() {
				bbs.ConvergeLRPs(pollingInterval)
				logMessages := logger.TestSink.LogMessages()
				Ω(logMessages).Should(ContainElement(
					"test.converge-lrps.starting-convergence",
				))
				Ω(logMessages).Should(ContainElement(
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
	})

	Describe("pruning LRPs by cell", func() {
		var cellPresence models.CellPresence
		var processGuid string
		var desiredLRP models.DesiredLRP
		var index int

		JustBeforeEach(func() {
			bbs.ConvergeLRPs(pollingInterval)
		})

		BeforeEach(func() {
			processGuid = "process-guid-for-pruning"

			index = 0

			desiredLRP = models.DesiredLRP{
				ProcessGuid: processGuid,
				Instances:   2,
				Domain:      "some-domain",
				Stack:       "pancake",
				Action:      dummyAction,
			}

			err := bbs.DesireLRP(desiredLRP)
			Ω(err).ShouldNot(HaveOccurred())

			cellPresence = models.NewCellPresence("cell-id", "the-stack", "cell.example.com")
			registerCell(cellPresence)

			actualLRP, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.ClaimActualLRP(
				actualLRP.ActualLRPKey,
				models.NewActualLRPContainerKey("instance-guid", cellPresence.CellID),
				logger,
			)
			Ω(err).ShouldNot(HaveOccurred())
		})

		Context("when the cell is present", func() {
			It("should not prune any LRPs", func() {
				Ω(bbs.ActualLRPs()).Should(HaveLen(2))
			})
		})

		Context("when the cell goes away", func() {
			BeforeEach(func() {
				etcdClient.Delete(shared.CellSchemaPath(cellPresence.CellID))
			})

			It("should delete LRPs associated with said cell but not the unclaimed LRP", func() {
				lrps, err := bbs.ActualLRPs()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(lrps).Should(HaveLen(2))

				indices := make([]int, len(lrps))
				for i, lrp := range lrps {
					Ω(lrp.ProcessGuid).Should(Equal(processGuid))
					Ω(lrp.State).Should(Equal(models.ActualLRPStateUnclaimed))

					indices[i] = lrp.Index
				}

				Ω(indices).Should(ConsistOf([]int{0, 1}))
			})

			It("should prune LRP directories for apps that are no longer running", func() {
				actual, err := etcdClient.ListRecursively(shared.ActualLRPSchemaRoot)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(actual.ChildNodes).Should(HaveLen(1))
				Ω(actual.ChildNodes[0].Key).Should(Equal(shared.ActualLRPProcessDir(processGuid)))
			})

			It("logs", func() {
				Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-lrps.detected-actual-with-missing-cell"))
			})
		})
	})

	Describe("converging extra actual LRPs", func() {
		var processGuid string
		var index int

		Context("when the actual LRP has no corresponding desired LRP", func() {
			Context("when the actual LRP is UNCLAIMED", func() {
				BeforeEach(func() {
					processGuid = "process-guid"
					index = 0
					nonPersistedDesiredLRP := models.DesiredLRP{
						ProcessGuid: processGuid,
						Instances:   1,
						Domain:      "some-domain",
						Stack:       "pancake",
						Action:      dummyAction,
					}

					err := bbs.CreateActualLRP(nonPersistedDesiredLRP, index, logger)
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("removes the actual LRP", func() {
					bbs.ConvergeLRPs(pollingInterval)

					_, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
					Ω(err).Should(Equal(bbserrors.ErrStoreResourceNotFound))
				})

				It("logs", func() {
					bbs.ConvergeLRPs(pollingInterval)

					Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-lrps.detected-undesired-instance"))
				})

				It("bumps the stopped LRPs convergence counter", func() {
					Ω(sender.GetCounter("LRPInstanceStopRequests")).Should(Equal(uint64(0)))
					bbs.ConvergeLRPs(pollingInterval)
					Ω(sender.GetCounter("LRPInstanceStopRequests")).Should(Equal(uint64(1)))
				})
			})

			Context("when the actual LRP is CLAIMED", func() {
				var cellPresence models.CellPresence

				BeforeEach(func() {
					cellPresence = models.NewCellPresence("cell-id", "the-stack", "cell.example.com")
					registerCell(cellPresence)

					processGuid = "process-guid"
					index = 0
					nonPersistedDesiredLRP := models.DesiredLRP{
						ProcessGuid: processGuid,
						Instances:   1,
						Domain:      "some-domain",
						Stack:       "pancake",
						Action:      dummyAction,
					}

					err := bbs.CreateActualLRP(nonPersistedDesiredLRP, index, logger)
					Ω(err).ShouldNot(HaveOccurred())

					actualLRP, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())

					err = bbs.ClaimActualLRP(
						actualLRP.ActualLRPKey,
						models.NewActualLRPContainerKey("instance-guid", cellPresence.CellID),
						logger,
					)
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("sends a stop request to the corresponding cell", func() {
					bbs.ConvergeLRPs(pollingInterval)

					Ω(fakeCellClient.StopLRPInstanceCallCount()).Should(Equal(1))

					addr, stop := fakeCellClient.StopLRPInstanceArgsForCall(0)
					Ω(addr).Should(Equal(cellPresence.RepAddress))
					Ω(stop.ProcessGuid).Should(Equal(processGuid))
					Ω(stop.Index).Should(Equal(index))
				})

				It("logs", func() {
					bbs.ConvergeLRPs(pollingInterval)
					Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-lrps.detected-undesired-instance"))
				})

				It("bumps the stopped LRPs convergence counter", func() {
					Ω(sender.GetCounter("LRPInstanceStopRequests")).Should(Equal(uint64(0)))
					bbs.ConvergeLRPs(pollingInterval)
					Ω(sender.GetCounter("LRPInstanceStopRequests")).Should(Equal(uint64(1)))
				})
			})

			Context("when the actual LRP is RUNNING", func() {
				var cellPresence models.CellPresence

				BeforeEach(func() {
					cellPresence = models.NewCellPresence("cell-id", "the-stack", "cell.example.com")
					registerCell(cellPresence)

					processGuid = "process-guid"
					index = 0
					nonPersistedDesiredLRP := models.DesiredLRP{
						ProcessGuid: processGuid,
						Instances:   1,
						Domain:      "some-domain",
						Stack:       "pancake",
						Action:      dummyAction,
					}

					err := bbs.CreateActualLRP(nonPersistedDesiredLRP, index, logger)
					Ω(err).ShouldNot(HaveOccurred())

					actualLRP, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())

					err = bbs.ClaimActualLRP(
						actualLRP.ActualLRPKey,
						models.NewActualLRPContainerKey("instance-guid", cellPresence.CellID),
						logger,
					)
					Ω(err).ShouldNot(HaveOccurred())

					err = bbs.StartActualLRP(
						actualLRP.ActualLRPKey,
						models.NewActualLRPContainerKey("instance-guid", cellPresence.CellID),
						models.NewActualLRPNetInfo("host", []models.PortMapping{{HostPort: 1234, ContainerPort: 5678}}),
						logger,
					)
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("sends a stop request to the corresponding cell", func() {
					bbs.ConvergeLRPs(pollingInterval)

					Ω(fakeCellClient.StopLRPInstanceCallCount()).Should(Equal(1))

					addr, stop := fakeCellClient.StopLRPInstanceArgsForCall(0)
					Ω(addr).Should(Equal(cellPresence.RepAddress))
					Ω(stop.ProcessGuid).Should(Equal(processGuid))
					Ω(stop.Index).Should(Equal(index))
				})

				It("logs", func() {
					bbs.ConvergeLRPs(pollingInterval)
					Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-lrps.detected-undesired-instance"))
				})

				It("bumps the stopped LRPs convergence counter", func() {
					Ω(sender.GetCounter("LRPInstanceStopRequests")).Should(Equal(uint64(0)))
					bbs.ConvergeLRPs(pollingInterval)
					Ω(sender.GetCounter("LRPInstanceStopRequests")).Should(Equal(uint64(1)))
				})
			})
		})

		Context("when the actual LRP index is too large for its corresponding desired LRP", func() {
			var desiredLRP models.DesiredLRP
			var numInstances int

			BeforeEach(func() {
				processGuid = "process-guid"
				numInstances = 2

				desiredLRP = models.DesiredLRP{
					ProcessGuid: processGuid,
					Instances:   numInstances,
					Domain:      "domain",
					Stack:       "pancake",
					Action:      dummyAction,
				}

				err := bbs.DesireLRP(desiredLRP)
				Ω(err).ShouldNot(HaveOccurred())
			})

			Context("when the actual LRP is UNCLAIMED", func() {
				BeforeEach(func() {
					index = numInstances

					fakeBiggerLRP := desiredLRP
					fakeBiggerLRP.Instances++

					err := bbs.CreateActualLRP(fakeBiggerLRP, index, logger)
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("removes the actual LRP", func() {
					bbs.ConvergeLRPs(pollingInterval)

					_, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
					Ω(err).Should(Equal(bbserrors.ErrStoreResourceNotFound))
				})

				It("logs", func() {
					bbs.ConvergeLRPs(pollingInterval)

					Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-lrps.request-stop"))
				})

				It("bumps the stopped LRPs convergence counter", func() {
					Ω(sender.GetCounter("LRPInstanceStopRequests")).Should(Equal(uint64(0)))
					bbs.ConvergeLRPs(pollingInterval)
					Ω(sender.GetCounter("LRPInstanceStopRequests")).Should(Equal(uint64(1)))
				})
			})

			Context("when the actual LRP is CLAIMED", func() {
				var cellPresence models.CellPresence

				BeforeEach(func() {
					cellPresence = models.NewCellPresence("cell-id", "the-stack", "cell.example.com")
					registerCell(cellPresence)

					index = numInstances

					fakeBiggerLRP := desiredLRP
					fakeBiggerLRP.Instances++

					err := bbs.CreateActualLRP(fakeBiggerLRP, index, logger)
					Ω(err).ShouldNot(HaveOccurred())

					actualLRP, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())

					err = bbs.ClaimActualLRP(
						actualLRP.ActualLRPKey,
						models.NewActualLRPContainerKey("instance-guid", cellPresence.CellID),
						logger,
					)
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("sends a stop request to the corresponding cell", func() {
					bbs.ConvergeLRPs(pollingInterval)

					Ω(fakeCellClient.StopLRPInstanceCallCount()).Should(Equal(1))

					addr, stop := fakeCellClient.StopLRPInstanceArgsForCall(0)
					Ω(addr).Should(Equal(cellPresence.RepAddress))
					Ω(stop.ProcessGuid).Should(Equal(processGuid))
					Ω(stop.Index).Should(Equal(index))
				})

				It("logs", func() {
					bbs.ConvergeLRPs(pollingInterval)
					Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-lrps.request-stop"))
				})

				It("bumps the stopped LRPs convergence counter", func() {
					Ω(sender.GetCounter("LRPInstanceStopRequests")).Should(Equal(uint64(0)))
					bbs.ConvergeLRPs(pollingInterval)
					Ω(sender.GetCounter("LRPInstanceStopRequests")).Should(Equal(uint64(1)))
				})
			})

			Context("when the actual LRP is RUNNING", func() {
				var cellPresence models.CellPresence

				BeforeEach(func() {
					cellPresence = models.NewCellPresence("cell-id", "the-stack", "cell.example.com")
					registerCell(cellPresence)

					index = numInstances

					fakeBiggerLRP := desiredLRP
					fakeBiggerLRP.Instances++

					err := bbs.CreateActualLRP(fakeBiggerLRP, index, logger)
					Ω(err).ShouldNot(HaveOccurred())

					actualLRP, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())

					err = bbs.ClaimActualLRP(
						actualLRP.ActualLRPKey,
						models.NewActualLRPContainerKey("instance-guid", cellPresence.CellID),
						logger,
					)
					Ω(err).ShouldNot(HaveOccurred())

					err = bbs.StartActualLRP(
						actualLRP.ActualLRPKey,
						models.NewActualLRPContainerKey("instance-guid", cellPresence.CellID),
						models.NewActualLRPNetInfo("host", []models.PortMapping{{HostPort: 1234, ContainerPort: 5678}}),
						logger,
					)
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("sends a stop request to the corresponding cell", func() {
					bbs.ConvergeLRPs(pollingInterval)

					Ω(fakeCellClient.StopLRPInstanceCallCount()).Should(Equal(1))

					addr, stop := fakeCellClient.StopLRPInstanceArgsForCall(0)
					Ω(addr).Should(Equal(cellPresence.RepAddress))
					Ω(stop.ProcessGuid).Should(Equal(processGuid))
					Ω(stop.Index).Should(Equal(index))
				})

				It("logs", func() {
					bbs.ConvergeLRPs(pollingInterval)
					Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-lrps.request-stop"))
				})

				It("bumps the stopped LRPs convergence counter", func() {
					Ω(sender.GetCounter("LRPInstanceStopRequests")).Should(Equal(uint64(0)))
					bbs.ConvergeLRPs(pollingInterval)
					Ω(sender.GetCounter("LRPInstanceStopRequests")).Should(Equal(uint64(1)))
				})
			})
		})
	})

	Describe("converging actual LRPs that are UNCLAIMED for too long", func() {
		var desiredLRP models.DesiredLRP

		BeforeEach(func() {
			desiredLRP = models.DesiredLRP{
				ProcessGuid: "process-guid-for-unclaimed",
				Domain:      "some-domain",
				Instances:   1,
				Stack:       "pancake",
				Action:      dummyAction,
			}

			auctioneerPresence := models.NewAuctioneerPresence("auctioneer-id", "example.com")
			registerAuctioneer(auctioneerPresence)

			err := bbs.CreateActualLRP(desiredLRP, 0, lagertest.NewTestLogger("test"))
			Ω(err).ShouldNot(HaveOccurred())

			// make sure created (UNCLAIMED) actual LRP has been in that state for longer than
			// the staleness threshhold of pollingInterval
			timeProvider.Increment(pollingInterval + 1*time.Second)

			err = bbs.DesireLRP(desiredLRP)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("logs", func() {
			bbs.ConvergeLRPs(pollingInterval)

			Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-lrps.resending-start-auction"))
		})

		It("re-emits start auction requests", func() {
			originalAuctionCallCount := fakeAuctioneerClient.RequestLRPStartAuctionsCallCount()
			bbs.ConvergeLRPs(pollingInterval)

			Consistently(fakeAuctioneerClient.RequestLRPStartAuctionsCallCount).Should(Equal(originalAuctionCallCount + 1))

			_, startAuctions := fakeAuctioneerClient.RequestLRPStartAuctionsArgsForCall(originalAuctionCallCount)
			Ω(startAuctions).Should(HaveLen(1))
			Ω(startAuctions[0].DesiredLRP).Should(Equal(desiredLRP))
			Ω(startAuctions[0].Index).Should(Equal(0))
		})
	})
})
