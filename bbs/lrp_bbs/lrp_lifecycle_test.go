package lrp_bbs_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/lrp_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LrpLifecycle", func() {
	const cellID = "some-cell-id"
	var actualLRPKey models.ActualLRPKey
	var containerKey models.ActualLRPContainerKey
	var netInfo models.ActualLRPNetInfo

	BeforeEach(func() {
		actualLRPKey = models.NewActualLRPKey("some-process-guid", 2, "tests")
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
				})

				Context("when able to fetch the auctioneer address", func() {
					var auctioneerPresence models.AuctioneerPresence

					BeforeEach(func() {
						auctioneerPresence = models.NewAuctioneerPresence("the-auctioneer-id", "the-address")
						registerAuctioneer(auctioneerPresence)
					})

					It("requests an auction", func() {
						Ω(fakeAuctioneerClient.RequestLRPStartAuctionCallCount()).Should(Equal(1))

						requestAddress, requestedAuction := fakeAuctioneerClient.RequestLRPStartAuctionArgsForCall(0)
						Ω(requestAddress).Should(Equal(auctioneerPresence.AuctioneerAddress))
						Ω(requestedAuction.DesiredLRP).Should(Equal(desiredLRP))
						Ω(requestedAuction.Index).Should(Equal(index))
					})

					Context("when requesting an auction is successful", func() {
						BeforeEach(func() {
							fakeAuctioneerClient.RequestLRPStartAuctionReturns(nil)
						})

						It("does not return an error", func() {
							Ω(errCreate).ShouldNot(HaveOccurred())
						})
					})

					Context("when requesting an auction is unsuccessful", func() {
						BeforeEach(func() {
							fakeAuctioneerClient.RequestLRPStartAuctionReturns(errors.New("oops"))
						})

						It("does not return an error", func() {
							// The creation succeeded, we can ignore the auction request error (converger will eventually do it)
							Ω(errCreate).ShouldNot(HaveOccurred())
						})
					})
				})

				Context("when unable to fetch the auctioneer address", func() {
					It("does not request an auction", func() {
						Consistently(fakeAuctioneerClient.RequestLRPStartAuctionCallCount).Should(BeZero())
					})

					It("does not return an error", func() {
						// The creation succeeded, we can ignore the auction request error (converger will eventually do it)
						Ω(errCreate).ShouldNot(HaveOccurred())
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
					Consistently(fakeAuctioneerClient.RequestLRPStartAuctionCallCount).Should(BeZero())
				})

				It("returns an error", func() {
					Ω(errCreate).Should(Equal(bbserrors.ErrStoreResourceExists))
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
				Consistently(fakeAuctioneerClient.RequestLRPStartAuctionCallCount).Should(BeZero())
			})

			It("returns an error", func() {
				Ω(errCreate).Should(ContainElement(models.ErrInvalidField{"index"}))
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
				Consistently(fakeAuctioneerClient.RequestLRPStartAuctionCallCount).Should(BeZero())
			})

			It("returns an error", func() {
				Ω(errCreate).Should(Equal(lrp_bbs.NewActualLRPIndexTooLargeError(index, desiredLRP.Instances)))
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
				Consistently(fakeAuctioneerClient.RequestLRPStartAuctionCallCount).Should(BeZero())
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
			claimErr = bbs.ClaimActualLRP(lrpKey, containerKey)
		})

		Context("when the actual LRP exists", func() {
			var processGuid string
			var desiredLRP models.DesiredLRP
			var index int
			var createdLRP *models.ActualLRP

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
					)
					Ω(err).ShouldNot(HaveOccurred())
				})

				Context("with the same cell and instance guid", func() {
					var previousTime int64

					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						containerKey = models.NewActualLRPContainerKey(instanceGuid, cellID)

						previousTime = timeProvider.Now().UnixNano()
						timeProvider.IncrementBySeconds(1)
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

						Ω(lrpInBBS.Host).Should(BeEmpty())
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

	Describe("StartActualLRP", func() {
		var startErr error
		var lrpKey models.ActualLRPKey
		var containerKey models.ActualLRPContainerKey
		var netInfo models.ActualLRPNetInfo

		JustBeforeEach(func() {
			startErr = bbs.StartActualLRP(lrpKey, containerKey, netInfo)
		})

		Context("when the actual LRP exists", func() {
			var processGuid string
			var desiredLRP models.DesiredLRP
			var index int
			var createdLRP *models.ActualLRP

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
			})

			Context("when the existing ActualLRP is Claimed", func() {
				var instanceGuid string

				BeforeEach(func() {
					instanceGuid = "some-instance-guid"
					err := bbs.ClaimActualLRP(
						createdLRP.ActualLRPKey,
						models.NewActualLRPContainerKey(instanceGuid, cellID),
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

							previousTime = timeProvider.Now().UnixNano()
							timeProvider.IncrementBySeconds(1)
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
			err := bbs.StartActualLRP(actualLRPKey, containerKey, netInfo)
			Ω(err).ShouldNot(HaveOccurred())
		})

		Context("when the LRP matches", func() {
			It("removes the LRP", func() {
				err := bbs.RemoveActualLRP(actualLRPKey, containerKey)
				Ω(err).ShouldNot(HaveOccurred())

				_, err = etcdClient.Get(shared.ActualLRPSchemaPath(actualLRPKey.ProcessGuid, actualLRPKey.Index))
				Ω(err).Should(MatchError(storeadapter.ErrorKeyNotFound))
			})

			Context("when the store is out of commission", func() {
				itRetriesUntilStoreComesBack(func() error {
					return bbs.RemoveActualLRP(actualLRPKey, containerKey)
				})
			})
		})

		Context("when the LRP differs from the one in the store", func() {
			It("does not delete the LRP", func() {
				containerKey.InstanceGuid = "another-instance-guid"
				err := bbs.RemoveActualLRP(actualLRPKey, containerKey)
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

				err = bbs.RetireActualLRPs([]models.ActualLRP{*lrpInBBS}, logger)
				Ω(err).ShouldNot(HaveOccurred())

				_, err = bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
				Ω(err).Should(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		})

		Context("when the LRP is not Unclaimed", func() {
			var cellPresence models.CellPresence
			var processGuid string
			var blockStopInstanceChan chan struct{}

			BeforeEach(func() {
				cellPresence = models.NewCellPresence(cellID, "the-stack", "cell.example.com")
				registerCell(cellPresence)

				processGuid = "some-process-guid"
				desiredLRP := models.DesiredLRP{
					ProcessGuid: processGuid,
					Domain:      "some-domain",
					Instances:   2,
				}
				lrpContainerKey1 := models.NewActualLRPContainerKey("some-instance-guid-1", cellID)
				lrpContainerKey2 := models.NewActualLRPContainerKey("some-instance-guid-2", cellID)
				createAndClaim(desiredLRP, 0, lrpContainerKey1)
				createAndClaim(desiredLRP, 1, lrpContainerKey2)

				blockStopInstanceChan = make(chan struct{})

				fakeCellClient.StopLRPInstanceStub = func(string, models.ActualLRP) error {
					<-blockStopInstanceChan
					return nil
				}
			})

			It("stops the LRPs in parallel", func() {
				claimedLRP1, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, 0)
				Ω(err).ShouldNot(HaveOccurred())
				claimedLRP2, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, 1)
				Ω(err).ShouldNot(HaveOccurred())

				errChan := make(chan error)
				go func(lrp1, lrp2 models.ActualLRP) {
					errChan <- bbs.RetireActualLRPs([]models.ActualLRP{lrp1, lrp2}, logger)
				}(*claimedLRP1, *claimedLRP2)

				Eventually(fakeCellClient.StopLRPInstanceCallCount).Should(Equal(2))

				addr1, stop1 := fakeCellClient.StopLRPInstanceArgsForCall(0)
				addr2, stop2 := fakeCellClient.StopLRPInstanceArgsForCall(1)

				Ω(addr1).Should(Equal(cellPresence.RepAddress))
				Ω(addr2).Should(Equal(cellPresence.RepAddress))

				Ω([]models.ActualLRP{stop1, stop2}).Should(ConsistOf([]models.ActualLRP{
					*claimedLRP1,
					*claimedLRP2,
				}))

				Consistently(errChan).ShouldNot(Receive())
				close(blockStopInstanceChan)
				Eventually(errChan).Should(Receive(BeNil()))
			})
		})
	})
})
