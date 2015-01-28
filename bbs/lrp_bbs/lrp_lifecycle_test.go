package lrp_bbs_test

import (
	"errors"
	"fmt"
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/lrp_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("LrpLifecycle", func() {
	const cellID = "some-cell-id"
	var actualLRPKey models.ActualLRPKey
	var containerKey models.ActualLRPContainerKey
	var netInfo models.ActualLRPNetInfo
	var index int

	BeforeEach(func() {
		index = 2
		actualLRPKey = models.NewActualLRPKey("some-process-guid", index, "tests")
		containerKey = models.NewActualLRPContainerKey("some-instance-guid", cellID)
		netInfo = models.NewActualLRPNetInfo("127.0.0.2", []models.PortMapping{{8081, 87}})
	})

	Describe("CreateActualLRP", func() {
		var (
			desiredLRP models.DesiredLRP
			index      int

			errCreate error
		)

		JustBeforeEach(func() {
			errCreate = bbs.CreateActualLRP(desiredLRP, index, logger)
		})

		Context("when given a desired LRP and valid index", func() {
			BeforeEach(func() {
				desiredLRP = models.DesiredLRP{
					ProcessGuid: "the-process-guid",
					Domain:      "the-domain",
					Instances:   3,
				}
				index = 1
			})

			Context("when an LRP is not already present at the desired key", func() {
				It("persists the actual LRP", func() {
					actualLRPs, err := bbs.ActualLRPs()
					Ω(err).ShouldNot(HaveOccurred())
					Ω(actualLRPs).Should(HaveLen(1))

					actualLRP := actualLRPs[0]
					Ω(actualLRP.ProcessGuid).Should(Equal(desiredLRP.ProcessGuid))
					Ω(actualLRP.Domain).Should(Equal(desiredLRP.Domain))
					Ω(actualLRP.Index).Should(Equal(index))
					Ω(actualLRP.State).Should(Equal(models.ActualLRPStateUnclaimed))
					Ω(actualLRP.PlacementError).Should(BeEmpty())
				})

				Context("when able to fetch the auctioneer address", func() {
					var auctioneerPresence models.AuctioneerPresence

					BeforeEach(func() {
						auctioneerPresence = models.NewAuctioneerPresence("the-auctioneer-id", "the-address")
						registerAuctioneer(auctioneerPresence)
					})

					It("requests an auction", func() {
						Ω(fakeAuctioneerClient.RequestLRPAuctionsCallCount()).Should(Equal(1))

						requestAddress, requestedAuctions := fakeAuctioneerClient.RequestLRPAuctionsArgsForCall(0)
						Ω(requestAddress).Should(Equal(auctioneerPresence.AuctioneerAddress))
						Ω(requestedAuctions).Should(HaveLen(1))
						Ω(requestedAuctions[0].DesiredLRP).Should(Equal(desiredLRP))
						Ω(requestedAuctions[0].Indices).Should(ConsistOf(uint(index)))
					})

					Context("when requesting an auction is successful", func() {
						BeforeEach(func() {
							fakeAuctioneerClient.RequestLRPAuctionsReturns(nil)
						})

						It("does not return an error", func() {
							Ω(errCreate).ShouldNot(HaveOccurred())
						})
					})

					Context("when requesting an auction is unsuccessful", func() {
						BeforeEach(func() {
							fakeAuctioneerClient.RequestLRPAuctionsReturns(errors.New("oops"))
						})

						It("does not return an error", func() {
							// The creation succeeded, we can ignore the auction request error (converger will eventually do it)
							Ω(errCreate).ShouldNot(HaveOccurred())
						})

						It("logs the error", func() {
							Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.failed-to-request-start-auctions"))
						})
					})
				})

				Context("when unable to fetch the auctioneer address", func() {
					It("does not request an auction", func() {
						Consistently(fakeAuctioneerClient.RequestLRPAuctionsCallCount).Should(BeZero())
					})

					It("does not return an error", func() {
						// The creation succeeded, we can ignore the auction request error (converger will eventually do it)
						Ω(errCreate).ShouldNot(HaveOccurred())
					})

					It("logs the error", func() {
						Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.failed-to-request-start-auctions"))
					})
				})
			})

			Context("when an LRP is already present at the desired key", func() {
				BeforeEach(func() {
					desiredLRP = models.DesiredLRP{
						ProcessGuid: "the-process-guid",
						Domain:      "the-domain",
						Instances:   3,
					}
					index = 2

					err := bbs.CreateActualLRP(desiredLRP, index, logger)
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("does not persist an actual LRP", func() {
					Consistently(bbs.ActualLRPs).Should(HaveLen(1))
				})

				It("does not request an auction", func() {
					Consistently(fakeAuctioneerClient.RequestLRPAuctionsCallCount).Should(BeZero())
				})

				It("returns an error", func() {
					Ω(errCreate).Should(Equal(bbserrors.ErrStoreResourceExists))
				})

				It("logs the error", func() {
					Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.create-actual-lrp.failed-to-create-actual-lrp"))
				})
			})
		})

		Context("when given a negative index", func() {
			BeforeEach(func() {
				desiredLRP = models.DesiredLRP{
					ProcessGuid: "the-process-guid",
					Domain:      "the-domain",
					Instances:   3,
				}
				index = -1
			})

			It("does not persist an actual LRP", func() {
				Consistently(bbs.ActualLRPs).Should(BeEmpty())
			})

			It("does not request an auction", func() {
				Consistently(fakeAuctioneerClient.RequestLRPAuctionsCallCount).Should(BeZero())
			})

			It("returns an error", func() {
				Ω(errCreate).Should(ContainElement(models.ErrInvalidField{"index"}))
			})

			It("logs the error", func() {
				Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.create-actual-lrp.failed-to-marshal-actual-lrp"))
			})
		})

		Context("when given an index that is too large relative to the desired instances", func() {
			BeforeEach(func() {
				desiredLRP = models.DesiredLRP{
					ProcessGuid: "the-process-guid",
					Domain:      "the-domain",
					Instances:   3,
				}
				index = 4
			})

			It("does not persist an actual LRP", func() {
				Consistently(bbs.ActualLRPs).Should(BeEmpty())
			})

			It("does not request an auction", func() {
				Consistently(fakeAuctioneerClient.RequestLRPAuctionsCallCount).Should(BeZero())
			})

			It("returns an error", func() {
				Ω(errCreate).Should(Equal(lrp_bbs.NewActualLRPIndexTooLargeError(index, desiredLRP.Instances)))
			})

			It("logs the error", func() {
				Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.create-actual-lrp.actual-lrp-index-too-large"))
			})
		})

		Context("when given a desired LRP with invalid information to construct an actual LRP", func() {
			BeforeEach(func() {
				desiredLRP = models.DesiredLRP{
					ProcessGuid: "the-process-guid",
					Instances:   3,
					// missing Domain
				}
				index = 1
			})

			It("does not persist an actual LRP", func() {
				Consistently(bbs.ActualLRPs).Should(BeEmpty())
			})

			It("does not request an auction", func() {
				Consistently(fakeAuctioneerClient.RequestLRPAuctionsCallCount).Should(BeZero())
			})

			It("returns an error", func() {
				Ω(errCreate).Should(ContainElement(models.ErrInvalidField{"domain"}))
			})
		})
	})

	Describe("ClaimActualLRP", func() {
		var claimErr error
		var lrpKey models.ActualLRPKey
		var containerKey models.ActualLRPContainerKey

		JustBeforeEach(func() {
			claimErr = bbs.ClaimActualLRP(lrpKey, containerKey, logger)
		})

		Context("when the actual LRP exists", func() {
			var processGuid string
			var desiredLRP models.DesiredLRP
			var index int
			var createdLRP models.ActualLRP

			BeforeEach(func() {
				index = 1
				processGuid = "some-process-guid"
				desiredLRP = models.DesiredLRP{
					ProcessGuid: processGuid,
					Domain:      "some-domain",
					Instances:   index + 1,
				}

				err := bbs.CreateActualLRP(desiredLRP, index, logger)
				Ω(err).ShouldNot(HaveOccurred())

				createdLRP, err = bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
				Ω(err).ShouldNot(HaveOccurred())
			})

			Context("when the container key is invalid", func() {
				BeforeEach(func() {
					lrpKey = createdLRP.ActualLRPKey
					containerKey = models.NewActualLRPContainerKey(
						"", // invalid InstanceGuid
						cellID,
					)
				})

				It("returns a validation error", func() {
					Ω(claimErr).Should(ContainElement(models.ErrInvalidField{"instance_guid"}))
				})

				It("does not modify the persisted actual LRP", func() {
					lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrpInBBS.State).Should(Equal(models.ActualLRPStateUnclaimed))
				})

				It("logs the error", func() {
					Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.claim-actual-lrp.failed-to-marshal-actual-lrp"))
				})
			})

			Context("when the domain differs", func() {
				BeforeEach(func() {
					lrpKey = models.NewActualLRPKey(
						createdLRP.ProcessGuid,
						createdLRP.Index,
						"some-other-domain",
					)
					containerKey = models.NewActualLRPContainerKey("some-instance-guid", cellID)
				})

				It("returns an error", func() {
					Ω(claimErr).Should(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
				})

				It("does not modify the persisted actual LRP", func() {
					lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrpInBBS.State).Should(Equal(models.ActualLRPStateUnclaimed))
				})
			})

			Context("when the existing ActualLRP is Unclaimed", func() {
				BeforeEach(func() {
					lrpKey = createdLRP.ActualLRPKey
					containerKey = models.NewActualLRPContainerKey("some-instance-guid", cellID)
				})

				It("does not error", func() {
					Ω(claimErr).ShouldNot(HaveOccurred())
				})

				It("claims the actual LRP", func() {
					lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrpInBBS.State).Should(Equal(models.ActualLRPStateClaimed))
				})
			})

			Context("when the existing ActualLRP is Claimed", func() {
				var instanceGuid string

				BeforeEach(func() {
					instanceGuid = "some-instance-guid"
					err := bbs.ClaimActualLRP(
						createdLRP.ActualLRPKey,
						models.NewActualLRPContainerKey(instanceGuid, cellID),
						logger,
					)
					Ω(err).ShouldNot(HaveOccurred())
				})

				Context("with the same cell and instance guid", func() {
					var previousTime int64

					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						containerKey = models.NewActualLRPContainerKey(instanceGuid, cellID)

						previousTime = clock.Now().UnixNano()
						clock.IncrementBySeconds(1)
					})

					It("does not return an error", func() {
						Ω(claimErr).ShouldNot(HaveOccurred())
					})

					It("does not alter the state of the persisted LRP", func() {
						lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(lrpInBBS.State).Should(Equal(models.ActualLRPStateClaimed))
					})

					It("does not update the timestamp of the persisted actual lrp", func() {
						lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(lrpInBBS.Since).Should(Equal(previousTime))
					})
				})

				Context("with a different cell", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						containerKey = models.NewActualLRPContainerKey(instanceGuid, "another-cell-id")
					})

					It("returns an error", func() {
						Ω(claimErr).Should(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
					})

					It("does not alter the existing LRP", func() {
						lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(lrpInBBS.CellID).Should(Equal(cellID))
					})
				})

				Context("when the instance guid differs", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						containerKey = models.NewActualLRPContainerKey("another-instance-guid", cellID)
					})

					It("returns an error", func() {
						Ω(claimErr).Should(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
					})

					It("does not alter the existing actual", func() {
						lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(lrpInBBS.InstanceGuid).Should(Equal(instanceGuid))
					})
				})
			})

			Context("when the existing ActualLRP is Running", func() {
				var instanceGuid string

				BeforeEach(func() {
					instanceGuid = "some-instance-guid"
					err := bbs.StartActualLRP(
						createdLRP.ActualLRPKey,
						models.NewActualLRPContainerKey(instanceGuid, cellID),
						models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{{ContainerPort: 1234, HostPort: 5678}}),
						logger,
					)
					Ω(err).ShouldNot(HaveOccurred())
				})

				Context("with the same cell and instance guid", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						containerKey = models.NewActualLRPContainerKey(instanceGuid, cellID)
					})

					It("does not return an error", func() {
						Ω(claimErr).ShouldNot(HaveOccurred())
					})

					It("reverts the persisted LRP to the CLAIMED state", func() {
						lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(lrpInBBS.State).Should(Equal(models.ActualLRPStateClaimed))
					})

					It("clears the net info", func() {
						lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(lrpInBBS.Address).Should(BeEmpty())
						Ω(lrpInBBS.Ports).Should(BeEmpty())
					})
				})

				Context("with a different cell", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						containerKey = models.NewActualLRPContainerKey(instanceGuid, "another-cell-id")
					})

					It("returns an error", func() {
						Ω(claimErr).Should(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
					})

					It("does not alter the existing LRP", func() {
						lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(lrpInBBS.CellID).Should(Equal(cellID))
					})
				})

				Context("when the instance guid differs", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						containerKey = models.NewActualLRPContainerKey("another-instance-guid", cellID)
					})

					It("returns an error", func() {
						Ω(claimErr).Should(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
					})

					It("does not alter the existing actual", func() {
						lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(lrpInBBS.InstanceGuid).Should(Equal(instanceGuid))
					})
				})
			})

			Context("when there is a placement error", func() {
				BeforeEach(func() {
					lrpKey = createdLRP.ActualLRPKey
					containerKey = models.NewActualLRPContainerKey("some-instance-guid", cellID)

					err := bbs.FailLRP(logger, lrpKey, "insufficient resources")
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("should clear placement error", func() {
					createdLRP, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(createdLRP.PlacementError).Should(BeEmpty())
				})
			})
		})

		Context("when the actual LRP does not exist", func() {
			BeforeEach(func() {
				lrpKey = models.NewActualLRPKey("process-guid", 1, "domain")
				containerKey = models.NewActualLRPContainerKey("instance-guid", cellID)
			})

			It("cannot claim the LRP", func() {
				Ω(claimErr).Should(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
			})

			It("does not create an actual LRP", func() {
				lrps, err := bbs.ActualLRPs()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(lrps).Should(BeEmpty())
			})
		})
	})

	Describe("UnclaimActualLRP", func() {
		var unclaimErr error
		var actual models.ActualLRP
		var lrpKey models.ActualLRPKey
		var containerKey models.ActualLRPContainerKey

		JustBeforeEach(func() {
			unclaimErr = bbs.UnclaimActualLRP(logger, lrpKey, containerKey)
		})

		Context("when the actual LRP exists", func() {
			var (
				processGuid, cellID string
				createdLRP          models.ActualLRP
				index               int
			)

			BeforeEach(func() {
				cellID = "cell-id"
				processGuid = "process-guid"
				index = 0
				actual = models.ActualLRP{
					ActualLRPKey:          models.NewActualLRPKey(processGuid, index, "domain"),
					ActualLRPContainerKey: models.NewActualLRPContainerKey("instanceGuid", cellID),
					State: models.ActualLRPStateClaimed,
					Since: 777,
				}
				createRawActualLRP(actual)

				var err error
				createdLRP, err = bbs.ActualLRPByProcessGuidAndIndex(actual.ActualLRPKey.ProcessGuid, actual.ActualLRPKey.Index)
				Ω(err).ShouldNot(HaveOccurred())
			})

			Context("when the container key is not the same", func() {
				BeforeEach(func() {
					lrpKey = createdLRP.ActualLRPKey
					containerKey = models.NewActualLRPContainerKey(
						"", // invalid InstanceGuid
						cellID,
					)
				})

				It("returns a validation error", func() {
					Ω(unclaimErr).Should(Equal(bbserrors.ErrActualLRPCannotBeUnclaimed))
				})

				It("does not modify the persisted actual LRP", func() {
					lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrpInBBS.State).Should(Equal(models.ActualLRPStateClaimed))
				})

				It("logs the error", func() {
					Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.unclaim-actual-lrp.failed-actual-lrp-container-key-differs"))
				})
			})

			Context("when the actualLRPKey is not the same", func() {
				BeforeEach(func() {
					lrpKey = createdLRP.ActualLRPKey
					lrpKey.Domain = "some-other-domain"
					containerKey = createdLRP.ActualLRPContainerKey
				})

				It("returns a validation error", func() {
					Ω(unclaimErr).Should(Equal(bbserrors.ErrActualLRPCannotBeUnclaimed))
				})

				It("does not modify the persisted actual LRP", func() {
					lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrpInBBS.State).Should(Equal(models.ActualLRPStateClaimed))
				})

				It("logs the error", func() {
					Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.unclaim-actual-lrp.failed-actual-lrp-key-differs"))
				})
			})

			Context("when the actualLRPKey and actualLRPContainerKey are the same", func() {
				BeforeEach(func() {
					clock.IntervalToAdvance = 0
					lrpKey = createdLRP.ActualLRPKey
					containerKey = createdLRP.ActualLRPContainerKey
				})

				It("sets the State to Unclaimed", func() {
					lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrpInBBS.State).Should(Equal(models.ActualLRPStateUnclaimed))
				})

				It("clears the ActualLRPContainerKey", func() {
					lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrpInBBS.ActualLRPContainerKey).Should(Equal(models.ActualLRPContainerKey{}))
				})

				It("clears the ActualLRPNetInfo", func() {
					lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrpInBBS.ActualLRPNetInfo).Should(Equal(models.ActualLRPNetInfo{}))
				})

				It("updates the Since", func() {
					lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrpInBBS.Since).Should(Equal(clock.Now().UnixNano()))
				})
			})
		})
	})

	Describe("EvacuateActualLRP", func() {
		var (
			evacuateErr  error
			lrpKey       models.ActualLRPKey
			containerKey models.ActualLRPContainerKey
			timeout      time.Duration
		)

		BeforeEach(func() {
			timeout = 30*time.Second + 500*time.Millisecond
		})

		JustBeforeEach(func() {
			evacuateErr = bbs.EvacuateActualLRP(logger, lrpKey, containerKey, timeout)
		})

		Context("when the actual LRP exists", func() {
			var (
				processGuid        string
				cellID             string
				domain             string
				index              int
				instanceGuid       string
				createdLRP         models.ActualLRP
				desiredLRP         models.DesiredLRP
				auctioneerPresence models.AuctioneerPresence
			)

			BeforeEach(func() {
				processGuid = "process-guid"
				index = 0
				domain = "domain"
				cellID = "cell-id"
				instanceGuid = "instance-guid"

				desiredLRP = models.DesiredLRP{
					ProcessGuid: processGuid,
					Domain:      "domain",
					Instances:   index + 1,
					Stack:       "the-stack",
					Action: &models.RunAction{
						Path: "/bin/true",
					},
				}

				createRawDesiredLRP(desiredLRP)

				err := bbs.CreateActualLRP(desiredLRP, index, logger)
				Ω(err).ShouldNot(HaveOccurred())

				createdLRP, err = bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
				Ω(err).ShouldNot(HaveOccurred())

				lrpKey = createdLRP.ActualLRPKey

				containerKey.CellID = cellID
				containerKey.InstanceGuid = instanceGuid

				auctioneerPresence = models.NewAuctioneerPresence("the-auctioneer-id", "the-address")
				registerAuctioneer(auctioneerPresence)
			})

			Context("when the actual LRP is in the CLAIMED state", func() {
				BeforeEach(func() {
					err := bbs.ClaimActualLRP(lrpKey, containerKey, logger)
					Ω(err).ShouldNot(HaveOccurred())

					createdLRP, err = bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(createdLRP.State).Should(Equal(models.ActualLRPStateClaimed))
				})

				It("does not return an error", func() {
					Ω(evacuateErr).ShouldNot(HaveOccurred())
				})

				It("sets the ActualLRP state to UNCLAIMED", func() {
					lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrpInBBS.State).Should(Equal(models.ActualLRPStateUnclaimed))
				})

				It("requests an auction for the ActualLRP", func() {
					Ω(fakeAuctioneerClient.RequestLRPAuctionsCallCount()).Should(Equal(1))

					requestAddress, requestedAuctions := fakeAuctioneerClient.RequestLRPAuctionsArgsForCall(0)
					Ω(requestAddress).Should(Equal(auctioneerPresence.AuctioneerAddress))
					Ω(requestedAuctions).Should(HaveLen(1))

					Ω(requestedAuctions[0].DesiredLRP).Should(Equal(desiredLRP))
					Ω(requestedAuctions[0].Indices).Should(ConsistOf(uint(index)))
				})

				Context("when unclaiming the actualLRP fails", func() {
					BeforeEach(func() {
						containerKey.InstanceGuid = "some-other-guid"
					})

					It("returns an error", func() {
						Ω(evacuateErr).Should(HaveOccurred())
					})

					It("does not request an auction for the ActualLRP", func() {
						Ω(fakeAuctioneerClient.RequestLRPAuctionsCallCount()).Should(Equal(0))
					})
				})

				Context("when requesting start auction fails", func() {
					var err error
					BeforeEach(func() {
						err = errors.New("error")
						fakeAuctioneerClient.RequestLRPAuctionsReturns(err)
					})

					It("returns an error", func() {
						Ω(evacuateErr).Should(Equal(err))
					})
				})
			})

			Context("when the actual LRP is in the RUNNING state", func() {
				BeforeEach(func() {
					err := bbs.StartActualLRP(lrpKey, containerKey, netInfo, logger)
					Ω(err).ShouldNot(HaveOccurred())

					createdLRP, err = bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(createdLRP.State).Should(Equal(models.ActualLRPStateRunning))
				})

				It("moves the LRP instance to the evacuating path", func() {
					_, err := etcdClient.Get(shared.EvacuatingActualLRPSchemaPath(processGuid, index))
					Ω(err).ShouldNot(HaveOccurred())

					_, err = bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
					Ω(err).Should(Equal(bbserrors.ErrStoreResourceNotFound))
				})

				It("sets the ttl on the evacuating instance", func() {
					node, err := etcdClient.Get(shared.EvacuatingActualLRPSchemaPath(processGuid, index))
					Ω(err).ShouldNot(HaveOccurred())

					Ω(node.TTL).Should(Equal(uint64(30)))
				})

				It("requests an auction for the ActualLRP", func() {
					Ω(fakeAuctioneerClient.RequestLRPAuctionsCallCount()).Should(Equal(1))

					requestAddress, requestedAuctions := fakeAuctioneerClient.RequestLRPAuctionsArgsForCall(0)
					Ω(requestAddress).Should(Equal(auctioneerPresence.AuctioneerAddress))
					Ω(requestedAuctions).Should(HaveLen(1))

					Ω(requestedAuctions[0].DesiredLRP).Should(Equal(desiredLRP))
					Ω(requestedAuctions[0].Indices).Should(ConsistOf(uint(index)))
				})

				Context("when moving the LRP to evacuating fails", func() {
					PIt("it returns an error", func() {
						// real bbs makes this hard
					})
				})

				Context("when requesting start auction fails", func() {
					var err error
					BeforeEach(func() {
						err = errors.New("error")
						fakeAuctioneerClient.RequestLRPAuctionsReturns(err)
					})

					It("returns an error", func() {
						Ω(evacuateErr).Should(Equal(err))
					})
				})
			})
		})
	})

	Describe("StartActualLRP", func() {
		var startErr error
		var lrpKey models.ActualLRPKey
		var containerKey models.ActualLRPContainerKey
		var netInfo models.ActualLRPNetInfo

		JustBeforeEach(func() {
			startErr = bbs.StartActualLRP(lrpKey, containerKey, netInfo, logger)
		})

		Context("when the actual LRP exists", func() {
			var processGuid string
			var desiredLRP models.DesiredLRP
			var index int
			var createdLRP models.ActualLRP

			BeforeEach(func() {
				index = 1
				processGuid = "some-process-guid"
				desiredLRP = models.DesiredLRP{
					ProcessGuid: processGuid,
					Domain:      "some-domain",
					Instances:   index + 1,
				}

				err := bbs.CreateActualLRP(desiredLRP, index, logger)
				Ω(err).ShouldNot(HaveOccurred())

				createdLRP, err = bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
				Ω(err).ShouldNot(HaveOccurred())
			})

			Context("when the container key is invalid", func() {
				BeforeEach(func() {
					lrpKey = createdLRP.ActualLRPKey
					containerKey = models.NewActualLRPContainerKey(
						"", // invalid InstanceGuid
						cellID,
					)
					netInfo = models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{{ContainerPort: 1234, HostPort: 5678}})
				})

				It("returns a validation error", func() {
					Ω(startErr).Should(ContainElement(models.ErrInvalidField{"instance_guid"}))
				})

				It("does not modify the persisted actual LRP", func() {
					lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrpInBBS.State).Should(Equal(models.ActualLRPStateUnclaimed))
				})

				It("logs the error", func() {
					Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.start-actual-lrp.failed-to-marshal-actual-lrp"))
				})
			})

			Context("when the domain differs", func() {
				BeforeEach(func() {
					lrpKey = models.NewActualLRPKey(
						createdLRP.ProcessGuid,
						createdLRP.Index,
						"some-other-domain",
					)
					containerKey = models.NewActualLRPContainerKey("some-instance-guid", cellID)
					netInfo = models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{{ContainerPort: 1234, HostPort: 5678}})
				})

				It("returns an error", func() {
					Ω(startErr).Should(Equal(bbserrors.ErrActualLRPCannotBeStarted))
				})

				It("does not modify the persisted actual LRP", func() {
					lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrpInBBS.State).Should(Equal(models.ActualLRPStateUnclaimed))
				})
			})

			Context("when the existing ActualLRP is Unclaimed", func() {
				BeforeEach(func() {
					lrpKey = createdLRP.ActualLRPKey
					containerKey = models.NewActualLRPContainerKey("some-instance-guid", cellID)
					netInfo = models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{{ContainerPort: 1234, HostPort: 5678}})
				})

				It("does not error", func() {
					Ω(startErr).ShouldNot(HaveOccurred())
				})

				It("starts the actual LRP", func() {
					lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrpInBBS.State).Should(Equal(models.ActualLRPStateRunning))
				})

				Context("when there is a placement error", func() {
					BeforeEach(func() {
						err := bbs.FailLRP(logger, lrpKey, "found no compatible cells")
						Ω(err).ShouldNot(HaveOccurred())
					})

					It("should clear placement error", func() {
						createdLRP, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
						Ω(err).ShouldNot(HaveOccurred())
						Ω(createdLRP.PlacementError).Should(BeEmpty())
					})
				})
			})

			Context("when the existing ActualLRP is Claimed", func() {
				var instanceGuid string

				BeforeEach(func() {
					instanceGuid = "some-instance-guid"
					err := bbs.ClaimActualLRP(
						createdLRP.ActualLRPKey,
						models.NewActualLRPContainerKey(instanceGuid, cellID),
						logger,
					)
					Ω(err).ShouldNot(HaveOccurred())
				})

				Context("with the same cell and instance guid", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						containerKey = models.NewActualLRPContainerKey(instanceGuid, cellID)
						netInfo = models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{{ContainerPort: 1234, HostPort: 5678}})
					})

					It("does not return an error", func() {
						Ω(startErr).ShouldNot(HaveOccurred())
					})

					It("promotes the persisted LRP to RUNNING", func() {
						lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(lrpInBBS.State).Should(Equal(models.ActualLRPStateRunning))
					})
				})

				Context("with a different cell", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						containerKey = models.NewActualLRPContainerKey(instanceGuid, "another-cell-id")
						netInfo = models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{{ContainerPort: 1234, HostPort: 5678}})
					})

					It("does not return an error", func() {
						Ω(startErr).ShouldNot(HaveOccurred())
					})

					It("promotes the persisted LRP to RUNNING", func() {
						lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(lrpInBBS.State).Should(Equal(models.ActualLRPStateRunning))
					})
				})

				Context("when the instance guid differs", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						containerKey = models.NewActualLRPContainerKey("another-instance-guid", cellID)
						netInfo = models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{{ContainerPort: 1234, HostPort: 5678}})
					})

					It("does not return an error", func() {
						Ω(startErr).ShouldNot(HaveOccurred())
					})

					It("promotes the persisted LRP to RUNNING", func() {
						lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(lrpInBBS.State).Should(Equal(models.ActualLRPStateRunning))
					})
				})

			})

			Context("when the existing ActualLRP is Running", func() {
				var instanceGuid string

				BeforeEach(func() {
					instanceGuid = "some-instance-guid"
					err := bbs.StartActualLRP(
						createdLRP.ActualLRPKey,
						models.NewActualLRPContainerKey(instanceGuid, cellID),
						models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{{ContainerPort: 1234, HostPort: 5678}}),
						logger,
					)
					Ω(err).ShouldNot(HaveOccurred())
				})

				Context("with the same cell and instance guid", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						containerKey = models.NewActualLRPContainerKey(instanceGuid, cellID)
						netInfo = models.NewActualLRPNetInfo("5.6.7.8", []models.PortMapping{{ContainerPort: 4567, HostPort: 4321}})
					})

					It("does not return an error", func() {
						Ω(startErr).ShouldNot(HaveOccurred())
					})

					It("does not alter the state of the persisted LRP", func() {
						lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(lrpInBBS.State).Should(Equal(models.ActualLRPStateRunning))
					})

					It("updates the net info", func() {
						lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(lrpInBBS.ActualLRPNetInfo).Should(Equal(netInfo))
					})

					Context("and the same net info", func() {
						var previousTime int64
						BeforeEach(func() {
							lrpKey = createdLRP.ActualLRPKey
							containerKey = models.NewActualLRPContainerKey(instanceGuid, cellID)
							netInfo = models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{{ContainerPort: 1234, HostPort: 5678}})

							previousTime = clock.Now().UnixNano()
							clock.IncrementBySeconds(1)
						})

						It("does not update the timestamp of the persisted actual lrp", func() {
							lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
							Ω(err).ShouldNot(HaveOccurred())

							Ω(lrpInBBS.Since).Should(Equal(previousTime))
						})
					})
				})

				Context("with a different cell", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						containerKey = models.NewActualLRPContainerKey(instanceGuid, "another-cell-id")
					})

					It("returns an error", func() {
						Ω(startErr).Should(Equal(bbserrors.ErrActualLRPCannotBeStarted))
					})

					It("does not alter the existing LRP", func() {
						lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(lrpInBBS.CellID).Should(Equal(cellID))
					})
				})

				Context("when the instance guid differs", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						containerKey = models.NewActualLRPContainerKey("another-instance-guid", cellID)
					})

					It("returns an error", func() {
						Ω(startErr).Should(Equal(bbserrors.ErrActualLRPCannotBeStarted))
					})

					It("does not alter the existing actual", func() {
						lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(lrpInBBS.InstanceGuid).Should(Equal(instanceGuid))
					})
				})
			})
		})

		Context("when the actual LRP does not exist", func() {
			BeforeEach(func() {
				lrpKey = models.NewActualLRPKey("process-guid", 1, "domain")
				containerKey = models.NewActualLRPContainerKey("instance-guid", cellID)
				netInfo = models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{{ContainerPort: 1234, HostPort: 5678}})
			})

			It("starts the LRP", func() {
				Ω(startErr).ShouldNot(HaveOccurred())
			})

			It("creates an actual LRP", func() {
				lrp, err := bbs.ActualLRPByProcessGuidAndIndex("process-guid", 1)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(lrp.State).Should(Equal(models.ActualLRPStateRunning))
			})
		})
	})

	Describe("RemoveActualLRP", func() {
		BeforeEach(func() {
			netInfo := models.NewActualLRPNetInfo("127.0.0.3", []models.PortMapping{{9090, 90}})
			err := bbs.StartActualLRP(actualLRPKey, containerKey, netInfo, logger)
			Ω(err).ShouldNot(HaveOccurred())
		})

		Context("when the LRP matches", func() {
			It("removes the LRP", func() {
				err := bbs.RemoveActualLRP(actualLRPKey, containerKey, logger)
				Ω(err).ShouldNot(HaveOccurred())

				_, err = etcdClient.Get(shared.ActualLRPSchemaPath(actualLRPKey.ProcessGuid, actualLRPKey.Index))
				Ω(err).Should(MatchError(storeadapter.ErrorKeyNotFound))
			})
		})

		Context("when the LRP differs from the one in the store", func() {
			It("does not delete the LRP", func() {
				containerKey.InstanceGuid = "another-instance-guid"
				err := bbs.RemoveActualLRP(actualLRPKey, containerKey, logger)
				Ω(err).Should(Equal(bbserrors.ErrStoreComparisonFailed))
			})
		})
	})

	Describe("RetireActualLRPs", func() {
		Context("with an Unclaimed LRP", func() {
			var processGuid string
			var index int

			BeforeEach(func() {
				processGuid = "some-process-guid"
				index = 1

				desiredLRP := models.DesiredLRP{
					ProcessGuid: processGuid,
					Domain:      "some-domain",
					Instances:   2,
				}

				err := bbs.CreateActualLRP(desiredLRP, index, logger)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("deletes the LRP", func() {
				lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
				Ω(err).ShouldNot(HaveOccurred())

				bbs.RetireActualLRPs([]models.ActualLRP{lrpInBBS}, logger)

				_, err = bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
				Ω(err).Should(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		})

		Context("when the LRP is Crashed", func() {
			var actual models.ActualLRP

			BeforeEach(func() {
				actual = models.ActualLRP{
					ActualLRPKey: models.NewActualLRPKey("processGuid", 0, "domain"),
					CrashCount:   1,
					State:        models.ActualLRPStateCrashed,
					Since:        777,
				}
				createRawActualLRP(actual)
			})

			JustBeforeEach(func() {
				bbs.RetireActualLRPs([]models.ActualLRP{actual}, logger)
			})

			It("should remove the actual", func() {
				_, err := bbs.ActualLRPByProcessGuidAndIndex(actual.ProcessGuid, actual.Index)
				Ω(err).Should(Equal(bbserrors.ErrStoreResourceNotFound))
			})

			It("should not log a failure", func() {
				Ω(logger).ShouldNot(gbytes.Say("fail"))
			})
		})

		Context("when the LRP is not Unclaimed", func() {
			var cellPresence models.CellPresence
			var processGuid string
			var blockStopInstanceChan chan struct{}
			var doneRetiring chan struct{}

			var claimedLRP1 models.ActualLRP
			var claimedLRP2 models.ActualLRP

			BeforeEach(func() {
				processGuid = "some-process-guid"
				desiredLRP := models.DesiredLRP{
					ProcessGuid: processGuid,
					Domain:      "some-domain",
					Instances:   2,
				}
				lrpContainerKey1 := models.NewActualLRPContainerKey("some-instance-guid-1", cellID)
				lrpContainerKey2 := models.NewActualLRPContainerKey("some-instance-guid-2", cellID)
				createAndClaim(desiredLRP, 0, lrpContainerKey1, logger)
				createAndClaim(desiredLRP, 1, lrpContainerKey2, logger)

				blockStopInstanceChan = make(chan struct{})

				fakeCellClient.StopLRPInstanceStub = func(string, models.ActualLRPKey, models.ActualLRPContainerKey) error {
					<-blockStopInstanceChan
					return nil
				}
			})

			JustBeforeEach(func() {
				var err error
				claimedLRP1, err = bbs.ActualLRPByProcessGuidAndIndex(processGuid, 0)
				Ω(err).ShouldNot(HaveOccurred())

				claimedLRP2, err = bbs.ActualLRPByProcessGuidAndIndex(processGuid, 1)
				Ω(err).ShouldNot(HaveOccurred())

				doneRetiring = make(chan struct{})

				go func(lrp1, lrp2 models.ActualLRP, doneRetiring chan struct{}, logger lager.Logger) {
					bbs.RetireActualLRPs([]models.ActualLRP{lrp1, lrp2}, logger)
					close(doneRetiring)
				}(claimedLRP1, claimedLRP2, doneRetiring, logger)
			})

			Context("when the cell is present", func() {
				BeforeEach(func() {
					cellPresence = models.NewCellPresence(cellID, "the-stack", "cell.example.com", "the-zone")
					registerCell(cellPresence)
				})

				It("stops the LRPs in parallel", func() {
					Eventually(fakeCellClient.StopLRPInstanceCallCount).Should(Equal(2))

					addr1, key1, cnrKey1 := fakeCellClient.StopLRPInstanceArgsForCall(0)
					addr2, key2, cnrKey2 := fakeCellClient.StopLRPInstanceArgsForCall(1)

					Ω(addr1).Should(Equal(cellPresence.RepAddress))
					Ω(addr2).Should(Equal(cellPresence.RepAddress))

					Ω([]models.ActualLRPKey{key1, key2}).Should(ConsistOf(
						claimedLRP1.ActualLRPKey,
						claimedLRP2.ActualLRPKey,
					))

					Ω([]models.ActualLRPContainerKey{cnrKey1, cnrKey2}).Should(ConsistOf(
						claimedLRP1.ActualLRPContainerKey,
						claimedLRP2.ActualLRPContainerKey,
					))

					Consistently(doneRetiring).ShouldNot(BeClosed())

					close(blockStopInstanceChan)

					Eventually(doneRetiring).Should(BeClosed())
				})

				Context("when stopping any of the LRPs fails", func() {
					BeforeEach(func() {
						fakeCellClient.StopLRPInstanceStub = func(cellAddr string, key models.ActualLRPKey, _ models.ActualLRPContainerKey) error {
							return fmt.Errorf("failed to stop %d", key.Index)
						}
					})

					It("logs the failure", func() {
						Eventually(doneRetiring).Should(BeClosed())

						Ω(logger.LogMessages()).Should(ContainElement("test.retire-actual-lrps.failed-to-retire"))
					})
				})
			})

			Context("when the cell is not present", func() {
				It("does not stop the instances", func() {
					Eventually(fakeCellClient.StopLRPInstanceCallCount).Should(Equal(0))
				})

				It("logs the error", func() {
					Eventually(logger.TestSink.LogMessages).Should(ContainElement("test.retire-actual-lrps.failed-to-retire-actual-lrp"))
				})
			})
		})
	})

	Describe("FailLRP", func() {

		Context("when lrp exists", func() {
			var (
				placementError string
				instanceGuid   string
				processGuid    string
				index          int
			)

			BeforeEach(func() {
				index = 1
				placementError = "insufficient resources"
				processGuid = "process-guid"
				instanceGuid = "instance-guid"

				desiredLRP := models.DesiredLRP{
					ProcessGuid: processGuid,
					Domain:      "the-domain",
					Instances:   3,
				}

				errCreate := bbs.CreateActualLRP(desiredLRP, index, logger)
				Ω(errCreate).ShouldNot(HaveOccurred())

				lrp, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
				Ω(err).ShouldNot(HaveOccurred())

				actualLRPKey = lrp.ActualLRPKey
				containerKey = models.NewActualLRPContainerKey(instanceGuid, cellID)
			})

			Context("in unclaimed state", func() {
				BeforeEach(func() {
					err := bbs.FailLRP(logger, actualLRPKey, placementError)
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("sets the placement error", func() {
					lrp, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(lrp.PlacementError).Should(Equal(placementError))
				})

			})

			Context("not in unclaimed state", func() {
				BeforeEach(func() {
					claimErr := bbs.ClaimActualLRP(actualLRPKey, containerKey, logger)
					Ω(claimErr).ShouldNot(HaveOccurred())
				})

				It("error", func() {
					err := bbs.FailLRP(logger, actualLRPKey, placementError)
					Ω(err).Should(HaveOccurred())
				})
			})
		})

		Context("when lrp does not exist", func() {
			It("error", func() {
				err := bbs.FailLRP(logger, models.NewActualLRPKey("non-existent-process-guid", index, "tests"),
					"non existent resources")
				Ω(err).Should(HaveOccurred())
				Ω(err).Should(Equal(bbserrors.ErrCannotFailLRP))
			})
		})
	})
})
