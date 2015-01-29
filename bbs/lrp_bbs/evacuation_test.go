package lrp_bbs_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Evacuation", func() {
	Describe("EvacuateClaimedActualLRP", func() {
		var evacuateErr error
		var lrpKey models.ActualLRPKey
		var containerKey models.ActualLRPContainerKey

		JustBeforeEach(func() {
			evacuateErr = bbs.EvacuateClaimedActualLRP(logger, lrpKey, containerKey)
		})

		Context("when the actual LRP exists", func() {
			var (
				processGuid        string
				cellID             string
				domain             string
				index              int
				createdLRP         models.ActualLRP
				desiredLRP         models.DesiredLRP
				auctioneerPresence models.AuctioneerPresence
			)

			BeforeEach(func() {
				processGuid = "process-guid"
				index = 0
				domain = "domain"
				cellID = "cell-id"

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

				containerKey = createdLRP.ActualLRPContainerKey
				lrpKey = createdLRP.ActualLRPKey

				auctioneerPresence = models.NewAuctioneerPresence("the-auctioneer-id", "the-address")
				registerAuctioneer(auctioneerPresence)
			})

			It("does not return an error", func() {
				Ω(evacuateErr).ShouldNot(HaveOccurred())
			})

			It("sets the ActualLRP state to UNCLAIMED", func() {
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

			It("requests an auction for the ActualLRP", func() {
				Ω(fakeAuctioneerClient.RequestLRPAuctionsCallCount()).Should(Equal(1))

				requestAddress, requestedAuctions := fakeAuctioneerClient.RequestLRPAuctionsArgsForCall(0)
				Ω(requestAddress).Should(Equal(auctioneerPresence.AuctioneerAddress))
				Ω(requestedAuctions).Should(HaveLen(1))

				Ω(requestedAuctions[0].DesiredLRP).Should(Equal(desiredLRP))
				Ω(requestedAuctions[0].Indices).Should(ConsistOf(uint(index)))
			})

			Context("when the lrp container key is different", func() {
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

			Context("when the lrp key is different", func() {
				BeforeEach(func() {
					lrpKey.Domain = "some-other-domain"
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
	})

})
