package lrp_bbs_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LrpLifecycle", func() {
	const cellID = "some-cell-id"

	Describe("CreateActualLRP", func() {
		var expectedActualLRP, actualLRP models.ActualLRP

		BeforeEach(func() {
			expectedActualLRP = models.ActualLRP{
				Domain:      "tests",
				ProcessGuid: "some-process-guid",
				State:       models.ActualLRPStateUnclaimed,
				Index:       2,
			}
			actualLRP = models.ActualLRP{
				Domain:      "tests",
				ProcessGuid: "some-process-guid",
				State:       models.ActualLRPStateInvalid,
				Index:       2,
			}
		})

		Context("when the LRP is invalid", func() {
			BeforeEach(func() {
				actualLRP.ProcessGuid = ""
			})

			It("returns a validation error", func() {
				_, err := bbs.CreateActualLRP(actualLRP)
				Ω(err).Should(ConsistOf(models.ErrInvalidField{"process_guid"}))
			})
		})

		Context("when the LRP is valid", func() {
			It("creates an unclaimed instance", func() {
				_, err := bbs.CreateActualLRP(actualLRP)
				Ω(err).ShouldNot(HaveOccurred())

				node, err := etcdClient.Get(shared.ActualLRPSchemaPath(expectedActualLRP.ProcessGuid, expectedActualLRP.Index))
				Ω(err).ShouldNot(HaveOccurred())

				var actualActualLRP models.ActualLRP
				err = models.FromJSON(node.Value, &actualActualLRP)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(actualActualLRP.Since).ShouldNot(BeZero())
				expectedActualLRP.Since = actualActualLRP.Since
				Ω(actualActualLRP).Should(Equal(expectedActualLRP))
			})
		})
	})

	Describe("ClaimActualLRP", func() {
		var lrpToClaim models.ActualLRP

		BeforeEach(func() {
			lrpToClaim = models.NewActualLRP("some-process-guid", "some-instance-guid", cellID, "some-domain", 1, models.ActualLRPStateUnclaimed)
		})

		Context("when the LRP is invalid", func() {
			BeforeEach(func() {
				lrpToClaim.ProcessGuid = ""
			})

			It("returns a validation error", func() {
				_, err := bbs.ClaimActualLRP(lrpToClaim)
				Ω(err).Should(ContainElement(models.ErrInvalidField{"process_guid"}))
			})
		})

		Context("when the actual LRP exists", func() {
			var lrpToCreate models.ActualLRP
			var createdLRP *models.ActualLRP
			var claimedLRP *models.ActualLRP
			var claimErr error

			BeforeEach(func() {
				lrpToCreate = lrpToClaim
			})

			JustBeforeEach(func() {
				var err error
				createdLRP, err = bbs.CreateRawActualLRP(&lrpToCreate)
				Ω(err).ShouldNot(HaveOccurred())

				claimedLRP, claimErr = bbs.ClaimActualLRP(lrpToClaim)
			})

			Context("when the actual is Unclaimed", func() {
				BeforeEach(func() {
					lrpToCreate.State = models.ActualLRPStateUnclaimed
					lrpToCreate.InstanceGuid = ""
					lrpToCreate.CellID = ""
				})

				It("claims the LRP", func() {
					Ω(claimErr).ShouldNot(HaveOccurred())

					existingLRP, err := bbs.ActualLRPByProcessGuidAndIndex(lrpToClaim.ProcessGuid, lrpToClaim.Index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(claimedLRP).Should(Equal(existingLRP))
				})

				Context("when the store is out of commission", func() {
					itRetriesUntilStoreComesBack(func() error {
						_, err := bbs.ClaimActualLRP(lrpToClaim)
						return err
					})
				})
			})

			Context("when the actual is Claimed", func() {
				BeforeEach(func() {
					lrpToCreate.State = models.ActualLRPStateClaimed
				})

				Context("with the same cell and instance guid", func() {
					It("does not alter the existing LRP", func() {
						Ω(claimErr).ShouldNot(HaveOccurred())

						existingLRP, err := bbs.ActualLRPByProcessGuidAndIndex(lrpToClaim.ProcessGuid, lrpToClaim.Index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(createdLRP).Should(Equal(existingLRP))
					})
				})

				Context("with a different cell", func() {
					BeforeEach(func() {
						lrpToCreate.CellID = "another-cell-id"
					})

					It("cannot claim the LRP", func() {
						Ω(claimErr).Should(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
						Ω(claimedLRP.CellID).ShouldNot(Equal(lrpToClaim.CellID))
						Ω(claimedLRP.CellID).Should(Equal(lrpToCreate.CellID))
					})
				})

				Context("with a different instance guid", func() {
					BeforeEach(func() {
						lrpToCreate.InstanceGuid = "another-instance-guid"
					})

					It("cannot claim the LRP", func() {
						Ω(claimErr).Should(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
						Ω(claimedLRP.InstanceGuid).ShouldNot(Equal(lrpToClaim.InstanceGuid))
						Ω(claimedLRP.InstanceGuid).Should(Equal(lrpToCreate.InstanceGuid))
					})
				})
			})

			Context("when the actual is Running", func() {
				BeforeEach(func() {
					lrpToCreate.State = models.ActualLRPStateRunning
				})

				It("cannot claim the LRP", func() {
					Ω(claimErr).Should(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
				})
			})
		})

		Context("when the actual LRP does not exist", func() {
			It("cannot claim the LRP", func() {
				_, err := bbs.ClaimActualLRP(lrpToClaim)
				Ω(err).Should(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
			})
		})
	})

	Describe("StartActualLRP", func() {
		var lrpToStart models.ActualLRP

		BeforeEach(func() {
			lrpToStart = models.NewActualLRP("some-process-guid", "some-instance-guid", cellID, "some-domain", 1, models.ActualLRPStateClaimed)
		})

		Context("when the LRP is invalid", func() {
			BeforeEach(func() {
				lrpToStart.ProcessGuid = ""
			})

			It("returns a validation error", func() {
				_, err := bbs.StartActualLRP(lrpToStart)
				Ω(err).Should(ContainElement(models.ErrInvalidField{"process_guid"}))
			})
		})

		Context("when the actual LRP exists", func() {
			var lrpToCreate models.ActualLRP
			var createdLRP *models.ActualLRP
			var runningLRP *models.ActualLRP
			var startErr error

			BeforeEach(func() {
				lrpToCreate = lrpToStart
			})

			JustBeforeEach(func() {
				var err error
				createdLRP, err = bbs.CreateRawActualLRP(&lrpToCreate)
				Ω(err).ShouldNot(HaveOccurred())

				runningLRP, startErr = bbs.StartActualLRP(lrpToStart)
			})

			Context("when the actual is Unclaimed", func() {
				BeforeEach(func() {
					lrpToCreate.State = models.ActualLRPStateUnclaimed
					lrpToCreate.CellID = ""
					lrpToCreate.InstanceGuid = ""
				})

				It("starts the LRP", func() {
					Ω(startErr).ShouldNot(HaveOccurred())

					existingLRP, err := bbs.ActualLRPByProcessGuidAndIndex(lrpToStart.ProcessGuid, lrpToStart.Index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(runningLRP).Should(Equal(existingLRP))
				})
			})

			Context("when the actual is Claimed", func() {
				BeforeEach(func() {
					lrpToCreate.State = models.ActualLRPStateClaimed
				})

				It("starts the LRP", func() {
					Ω(startErr).ShouldNot(HaveOccurred())

					existingLRP, err := bbs.ActualLRPByProcessGuidAndIndex(lrpToStart.ProcessGuid, lrpToStart.Index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(runningLRP).Should(Equal(existingLRP))
				})

				Context("with a different cell", func() {
					BeforeEach(func() {
						lrpToCreate.CellID = "another-cell-id"
					})

					It("starts the LRP", func() {
						Ω(startErr).ShouldNot(HaveOccurred())

						existingLRP, err := bbs.ActualLRPByProcessGuidAndIndex(lrpToStart.ProcessGuid, lrpToStart.Index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(runningLRP).Should(Equal(existingLRP))
					})
				})
			})

			Context("when the actual is Running", func() {
				BeforeEach(func() {
					lrpToCreate.State = models.ActualLRPStateRunning
				})

				Context("with the same cell and instance guid", func() {
					It("does not alter the existing LRP", func() {
						Ω(startErr).ShouldNot(HaveOccurred())

						existingLRP, err := bbs.ActualLRPByProcessGuidAndIndex(lrpToStart.ProcessGuid, lrpToStart.Index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(createdLRP).Should(Equal(existingLRP))
					})
				})

				Context("with a different cell", func() {
					BeforeEach(func() {
						lrpToCreate.CellID = "another-cell-id"
					})

					It("cannot claim the LRP", func() {
						Ω(startErr).Should(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
					})
				})

				Context("with a different instance guid", func() {
					BeforeEach(func() {
						lrpToCreate.InstanceGuid = "another-instance-guid"
					})

					It("cannot claim the LRP", func() {
						Ω(startErr).Should(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
					})
				})
			})
		})

		Context("when the actual LRP does not exist", func() {
			It("creates the running LRP", func() {
				runningLRP, err := bbs.StartActualLRP(lrpToStart)
				Ω(err).ShouldNot(HaveOccurred())

				node, err := etcdClient.Get(shared.ActualLRPSchemaPath(lrpToStart.ProcessGuid, lrpToStart.Index))
				Ω(err).ShouldNot(HaveOccurred())

				expectedJSON, err := models.ToJSON(runningLRP)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(node.Value).Should(MatchJSON(expectedJSON))
			})
		})
	})
})
