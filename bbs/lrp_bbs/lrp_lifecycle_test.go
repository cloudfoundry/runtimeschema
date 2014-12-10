package lrp_bbs_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LrpLifecycle", func() {
	const cellID = "some-cell-id"

	Describe("CreateActualLRP", func() {
		var expectedActualLRP, actualLRP models.ActualLRP

		BeforeEach(func() {
			actualLRP = models.ActualLRP{
				Domain:       "tests",
				ProcessGuid:  "some-process-guid",
				InstanceGuid: "some-instance-guid",
				State:        models.ActualLRPStateInvalid,
				Index:        2,
			}

			expectedActualLRP = models.ActualLRP{
				Domain:       "tests",
				ProcessGuid:  "some-process-guid",
				InstanceGuid: "some-instance-guid",
				State:        models.ActualLRPStateUnclaimed,
				Index:        2,
			}
		})

		Context("when the LRP has an invalid process guid", func() {
			BeforeEach(func() {
				actualLRP.ProcessGuid = ""
			})

			It("returns a validation error only about the process guid", func() {
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

		Context("when an LRP is already present at the desired kep", func() {
			BeforeEach(func() {
				_, err := bbs.CreateActualLRP(actualLRP)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("returns an error that the resource already exists", func() {
				_, err := bbs.CreateActualLRP(actualLRP)
				Ω(err).Should(MatchError(bbserrors.ErrStoreResourceExists))
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

			Context("when the instance guid differs", func() {
				BeforeEach(func() {
					lrpToCreate.InstanceGuid = "another-instance-guid"
					lrpToCreate.CellID = ""
				})

				It("returns an error", func() {
					Ω(claimErr).Should(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
				})

				It("does not alter the existing actual", func() {
					Ω(claimedLRP.CellID).ShouldNot(Equal(lrpToClaim.CellID))
					Ω(claimedLRP.CellID).Should(Equal(lrpToCreate.CellID))
				})
			})

			Context("when the domain differs", func() {
				BeforeEach(func() {
					lrpToCreate.Domain = "some-other-domain"
					lrpToCreate.CellID = ""
				})

				It("returns an error", func() {
					Ω(claimErr).Should(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
				})

				It("does not alter the existing actual", func() {
					Ω(claimedLRP.CellID).ShouldNot(Equal(lrpToClaim.CellID))
					Ω(claimedLRP.CellID).Should(Equal(lrpToCreate.CellID))
				})
			})

			Context("when the actual is Unclaimed", func() {
				BeforeEach(func() {
					lrpToCreate.State = models.ActualLRPStateUnclaimed
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

				Context("with the same cell", func() {
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
			})

			Context("when the actual is Running", func() {
				BeforeEach(func() {
					lrpToCreate.State = models.ActualLRPStateRunning
				})

				Context("with the same cell", func() {
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

			Context("when the instance guid differs", func() {
				BeforeEach(func() {
					lrpToCreate.InstanceGuid = "another-instance-guid"
				})

				It("returns an error", func() {
					Ω(startErr).Should(Equal(bbserrors.ErrActualLRPCannotBeStarted))
				})

				It("does not alter the existing actual", func() {
					Ω(runningLRP.State).ShouldNot(Equal(models.ActualLRPStateRunning))
				})
			})

			Context("when the domain differs", func() {
				BeforeEach(func() {
					lrpToCreate.Domain = "some-other-domain"
				})

				It("returns an error", func() {
					Ω(startErr).Should(Equal(bbserrors.ErrActualLRPCannotBeStarted))
				})

				It("does not alter the existing actual", func() {
					Ω(runningLRP.State).ShouldNot(Equal(models.ActualLRPStateRunning))
				})
			})

			Context("when the actual is Unclaimed", func() {
				BeforeEach(func() {
					lrpToCreate.State = models.ActualLRPStateUnclaimed
					lrpToCreate.CellID = ""
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

				Context("with the same cell", func() {
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
						Ω(startErr).Should(Equal(bbserrors.ErrActualLRPCannotBeStarted))
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

	Describe("RemoveActualLRP", func() {
		var runningLRP *models.ActualLRP

		BeforeEach(func() {
			lrp := models.NewActualLRP("some-process-guid", "some-instance-guid", cellID, "some-domain", 1, models.ActualLRPStateRunning)
			var err error
			runningLRP, err = bbs.StartActualLRP(lrp)
			Ω(err).ShouldNot(HaveOccurred())
		})

		Context("when the LRP matches", func() {
			It("removes the LRP", func() {
				err := bbs.RemoveActualLRP(*runningLRP)
				Ω(err).ShouldNot(HaveOccurred())

				_, err = etcdClient.Get(shared.ActualLRPSchemaPath(runningLRP.ProcessGuid, runningLRP.Index))
				Ω(err).Should(MatchError(storeadapter.ErrorKeyNotFound))
			})

			Context("when the store is out of commission", func() {
				itRetriesUntilStoreComesBack(func() error {
					return bbs.RemoveActualLRP(*runningLRP)
				})
			})
		})

		Context("when the LRP differs from the one in the store", func() {
			It("does not delete the LRP", func() {
				outOfDateLRP := *runningLRP
				outOfDateLRP.InstanceGuid = "another-instance-guid"
				err := bbs.RemoveActualLRP(outOfDateLRP)
				Ω(err).Should(Equal(bbserrors.ErrStoreComparisonFailed))
			})
		})
	})
})
