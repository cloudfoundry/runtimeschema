package lrp_bbs_test

import (
	"errors"
	"sync"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LrpLifecycle", func() {
	const cellID = "some-cell-id"

	Describe("CreateActualLRP", func() {
		var (
			desiredLRP models.DesiredLRP
			index      int
			logger     *lagertest.TestLogger

			errCreate error
		)

		BeforeEach(func() {
			logger = lagertest.NewTestLogger("test")
		})

		JustBeforeEach(func() {
			_, errCreate = bbs.CreateActualLRP(desiredLRP, index, logger)
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
				It("persists the acutal LRP", func() {
					actualLRPs, err := bbs.ActualLRPs()
					Ω(err).ShouldNot(HaveOccurred())
					Ω(actualLRPs).Should(HaveLen(1))

					actualLRP := actualLRPs[0]
					Ω(actualLRP.InstanceGuid).ShouldNot(BeEmpty())
					Ω(actualLRP.ProcessGuid).Should(Equal(desiredLRP.ProcessGuid))
					Ω(actualLRP.Domain).Should(Equal(desiredLRP.Domain))
					Ω(actualLRP.Index).Should(Equal(index))
					Ω(actualLRP.State).Should(Equal(models.ActualLRPStateUnclaimed))
				})

				Context("when able to fetch the auctioneer address", func() {
					var auctioneerPresence models.AuctioneerPresence

					BeforeEach(func() {
						auctioneerPresence = models.AuctioneerPresence{
							AuctioneerID:      "the-auctioneer-id",
							AuctioneerAddress: "the-address",
						}

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
						var errRequestAuction error

						BeforeEach(func() {
							errRequestAuction = errors.New("oops")
							fakeAuctioneerClient.RequestLRPStartAuctionReturns(errRequestAuction)
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

					_, err := bbs.CreateActualLRP(desiredLRP, index, logger)
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("does not persist an actual LRP", func() {
					Consistently(bbs.ActualLRPs).Should(HaveLen(1))
				})

				It("does not request an auction", func() {
					Consistently(fakeAuctioneerClient.RequestLRPStartAuctionCallCount).Should(BeZero())
				})

				It("returns an error", func() {
					Ω(errCreate).Should(HaveOccurred())
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
				Ω(errCreate).Should(HaveOccurred())
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
				Ω(errCreate).Should(HaveOccurred())
			})
		})

		Context("when given a desired LRP with a missing domain", func() {
			BeforeEach(func() {
				desiredLRP = models.DesiredLRP{
					ProcessGuid: "the-process-guid",
					Instances:   3,
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
				Ω(errCreate).Should(HaveOccurred())
			})
		})

		Context("when given a desired LRP with a missing process guid", func() {
			BeforeEach(func() {
				desiredLRP = models.DesiredLRP{
					Domain:    "the-domain",
					Instances: 3,
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
				Ω(errCreate).Should(HaveOccurred())
			})
		})
	})

	Describe("ClaimActualLRP", func() {
		var lrpToClaim *models.ActualLRP
		var claimErr error

		JustBeforeEach(func() {
			_, claimErr = bbs.ClaimActualLRP(*lrpToClaim)
		})

		Context("when the actual LRP exists", func() {
			var desiredLRP models.DesiredLRP
			var index int

			BeforeEach(func() {
				index = 1
				desiredLRP = models.DesiredLRP{
					ProcessGuid: "some-process-guid",
					Domain:      "some-domain",
					Instances:   index + 1,
				}
				var err error
				lrpToClaim, err = bbs.CreateActualLRP(desiredLRP, index, logger)
				Ω(err).ShouldNot(HaveOccurred())
				lrpToClaim.CellID = cellID
			})

			Context("when the store is out of commission", func() {
				itRetriesUntilStoreComesBack(func() error {
					_, err := bbs.ClaimActualLRP(*lrpToClaim)
					return err
				})
			})

			Context("when the LRP is invalid", func() {
				BeforeEach(func() {
					lrpToClaim.ProcessGuid = ""
				})

				It("returns a validation error", func() {
					Ω(claimErr).Should(ContainElement(models.ErrInvalidField{"process_guid"}))
				})

				It("does not modify the persisted actual LRP", func() {
					lrps, err := bbs.ActualLRPs()
					Ω(err).ShouldNot(HaveOccurred())
					Ω(lrps).Should(HaveLen(1))
					Ω(lrps[0].ProcessGuid).Should(Equal(desiredLRP.ProcessGuid))
					Ω(lrps[0].Domain).Should(Equal(desiredLRP.Domain))
					Ω(lrps[0].Index).Should(Equal(index))
					Ω(lrps[0].State).Should(Equal(models.ActualLRPStateUnclaimed))
				})
			})

			Context("when the instance guid differs", func() {
				BeforeEach(func() {
					lrpToClaim.InstanceGuid = "another-instance-guid"
				})

				It("returns an error", func() {
					Ω(claimErr).Should(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
				})

				It("does not modify the persisted actual LRP", func() {
					lrps, err := bbs.ActualLRPs()
					Ω(err).ShouldNot(HaveOccurred())
					Ω(lrps).Should(HaveLen(1))
					Ω(lrps[0].ProcessGuid).Should(Equal(desiredLRP.ProcessGuid))
					Ω(lrps[0].Domain).Should(Equal(desiredLRP.Domain))
					Ω(lrps[0].Index).Should(Equal(index))
					Ω(lrps[0].State).Should(Equal(models.ActualLRPStateUnclaimed))
				})
			})

			Context("when the domain differs", func() {
				BeforeEach(func() {
					lrpToClaim.Domain = "some-other-domain"
				})

				It("returns an error", func() {
					Ω(claimErr).Should(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
				})

				It("does not modify the persisted actual LRP", func() {
					lrps, err := bbs.ActualLRPs()
					Ω(err).ShouldNot(HaveOccurred())
					Ω(lrps).Should(HaveLen(1))
					Ω(lrps[0].ProcessGuid).Should(Equal(desiredLRP.ProcessGuid))
					Ω(lrps[0].Domain).Should(Equal(desiredLRP.Domain))
					Ω(lrps[0].Index).Should(Equal(index))
					Ω(lrps[0].State).Should(Equal(models.ActualLRPStateUnclaimed))
				})
			})

			Context("when the actual is Unclaimed", func() {
				It("claims the LRP", func() {
					Ω(claimErr).ShouldNot(HaveOccurred())

					existingLRP, err := bbs.ActualLRPByProcessGuidAndIndex(lrpToClaim.ProcessGuid, lrpToClaim.Index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(existingLRP.State).Should(Equal(models.ActualLRPStateClaimed))
					Ω(existingLRP.CellID).Should(Equal(lrpToClaim.CellID))
					Ω(existingLRP.InstanceGuid).Should(Equal(lrpToClaim.InstanceGuid))
				})
			})

			Context("when the actual is Claimed with the same cell", func() {
				BeforeEach(func() {
					_, err := bbs.ClaimActualLRP(*lrpToClaim)
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("does not alter the existing LRP", func() {
					Ω(claimErr).ShouldNot(HaveOccurred())

					existingLRP, err := bbs.ActualLRPByProcessGuidAndIndex(lrpToClaim.ProcessGuid, lrpToClaim.Index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(existingLRP.State).Should(Equal(models.ActualLRPStateClaimed))
					Ω(existingLRP.CellID).Should(Equal(lrpToClaim.CellID))
					Ω(existingLRP.InstanceGuid).Should(Equal(lrpToClaim.InstanceGuid))
				})
			})

			Context("when the actual is Claimed with a different cell", func() {
				BeforeEach(func() {
					_, err := bbs.ClaimActualLRP(*lrpToClaim)
					Ω(err).ShouldNot(HaveOccurred())

					lrpToClaim.CellID = "another-cell-id"
				})

				It("cannot claim the LRP", func() {
					Ω(claimErr).Should(Equal(bbserrors.ErrActualLRPCannotBeClaimed))

					existingLRP, err := bbs.ActualLRPByProcessGuidAndIndex(lrpToClaim.ProcessGuid, lrpToClaim.Index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(existingLRP.State).Should(Equal(models.ActualLRPStateClaimed))
					Ω(existingLRP.CellID).Should(Equal(cellID))
					Ω(existingLRP.CellID).ShouldNot(Equal(lrpToClaim.CellID))
					Ω(existingLRP.InstanceGuid).Should(Equal(lrpToClaim.InstanceGuid))
				})
			})

			Context("when the actual is Running with the same cell", func() {
				BeforeEach(func() {
					claimedLRP, err := bbs.ClaimActualLRP(*lrpToClaim)
					Ω(err).ShouldNot(HaveOccurred())

					_, err = bbs.StartActualLRP(*claimedLRP)
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("claims the LRP", func() {
					Ω(claimErr).ShouldNot(HaveOccurred())

					existingLRP, err := bbs.ActualLRPByProcessGuidAndIndex(lrpToClaim.ProcessGuid, lrpToClaim.Index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(existingLRP.State).Should(Equal(models.ActualLRPStateClaimed))
					Ω(existingLRP.CellID).Should(Equal(cellID))
					Ω(existingLRP.InstanceGuid).Should(Equal(lrpToClaim.InstanceGuid))
				})
			})

			Context("when the actual is Running with a different cell", func() {
				BeforeEach(func() {
					claimedLRP, err := bbs.ClaimActualLRP(*lrpToClaim)
					Ω(err).ShouldNot(HaveOccurred())

					_, err = bbs.StartActualLRP(*claimedLRP)
					Ω(err).ShouldNot(HaveOccurred())

					lrpToClaim.CellID = "another-cell-id"
				})

				It("cannot claim the LRP", func() {
					Ω(claimErr).Should(Equal(bbserrors.ErrActualLRPCannotBeClaimed))

					existingLRP, err := bbs.ActualLRPByProcessGuidAndIndex(lrpToClaim.ProcessGuid, lrpToClaim.Index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(existingLRP.State).Should(Equal(models.ActualLRPStateRunning))
					Ω(existingLRP.CellID).Should(Equal(cellID))
					Ω(existingLRP.CellID).ShouldNot(Equal(lrpToClaim.CellID))
					Ω(existingLRP.InstanceGuid).Should(Equal(lrpToClaim.InstanceGuid))
				})
			})
		})

		Context("when the actual LRP does not exist", func() {
			BeforeEach(func() {
				lrpToClaim = &models.ActualLRP{
					ProcessGuid:  "process-guid",
					InstanceGuid: "instance-guid",
					CellID:       cellID,
					Domain:       "domain",
					Index:        0,
					State:        models.ActualLRPStateUnclaimed,
				}
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
		var lrpToStart *models.ActualLRP

		var startLRPResult *models.ActualLRP
		var startErr error

		var createdLRP *models.ActualLRP
		var claimedLRP *models.ActualLRP
		var previouslyStartedLRP *models.ActualLRP

		itDoesNotReturnAnError := func() {
			It("does not return an error", func() {
				Ω(startErr).ShouldNot(HaveOccurred())
			})
		}

		itReturnsACannotBeClaimedError := func() {
			It("returns a 'cannot be claimed' error", func() {
				Ω(startErr).Should(Equal(bbserrors.ErrActualLRPCannotBeStarted))
			})
		}

		itDoesNotAlterTheExistingLRP := func() {
			It("does not alter the existing LRP", func() {
				existingLRP, err := bbs.ActualLRPByProcessGuidAndIndex(lrpToStart.ProcessGuid, lrpToStart.Index)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(existingLRP.State).Should(Equal(lrpToStart.State))
			})
		}

		itStartsTheLRP := func() {
			It("starts the LRP", func() {
				existingLRP, err := bbs.ActualLRPByProcessGuidAndIndex(lrpToStart.ProcessGuid, lrpToStart.Index)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(startLRPResult).Should(Equal(existingLRP))
				Ω(startLRPResult.State).Should(Equal(models.ActualLRPStateRunning))
			})
		}

		Context("when the actual LRP exists", func() {
			BeforeEach(func() {
				desiredLRP := models.DesiredLRP{
					ProcessGuid: "process-guid",
					Domain:      "domain",
					Instances:   1,
				}

				var err error
				createdLRP, err = bbs.CreateActualLRP(desiredLRP, 0, lagertest.NewTestLogger("test"))
				Ω(err).ShouldNot(HaveOccurred())
			})

			JustBeforeEach(func() {
				startLRPResult, startErr = bbs.StartActualLRP(*lrpToStart)
			})

			Context("when the actual is Unclaimed", func() {
				BeforeEach(func() {
					lrpToStart = createdLRP
					lrpToStart.CellID = cellID
				})

				itDoesNotReturnAnError()
				itStartsTheLRP()

				Context("with a different instance guid", func() {
					BeforeEach(func() {
						lrpToStart.InstanceGuid = "another-instance-guid"
					})

					itDoesNotReturnAnError()
					itStartsTheLRP()
				})
			})

			Context("when the actual is Claimed", func() {
				BeforeEach(func() {
					var err error

					createdLRP.CellID = cellID
					claimedLRP, err = bbs.ClaimActualLRP(*createdLRP)
					Ω(err).ShouldNot(HaveOccurred())

					lrpToStart = claimedLRP
				})

				itDoesNotReturnAnError()
				itStartsTheLRP()

				Context("with a different instance guid", func() {
					BeforeEach(func() {
						lrpToStart.InstanceGuid = "another-instance-guid"
					})

					itDoesNotReturnAnError()
					itStartsTheLRP()
				})

				Context("with a different cell", func() {
					BeforeEach(func() {
						lrpToStart.CellID = "another-cell-id"
					})

					itDoesNotReturnAnError()
					itStartsTheLRP()
				})
			})

			Context("when the actual is Running", func() {
				BeforeEach(func() {
					var err error

					createdLRP.CellID = cellID
					claimedLRP, err = bbs.ClaimActualLRP(*createdLRP)
					Ω(err).ShouldNot(HaveOccurred())

					previouslyStartedLRP, err = bbs.StartActualLRP(*claimedLRP)
					Ω(err).ShouldNot(HaveOccurred())

					lrpToStart = previouslyStartedLRP
				})

				Context("with the same cell", func() {
					itDoesNotReturnAnError()

					itDoesNotAlterTheExistingLRP()

					Context("when the instance guid differs", func() {
						BeforeEach(func() {
							lrpToStart.InstanceGuid = "another-instance-guid"
						})

						itReturnsACannotBeClaimedError()
						itDoesNotAlterTheExistingLRP()
					})
				})

				Context("with a different cell", func() {
					BeforeEach(func() {
						lrpToStart.CellID = "another-cell-id"
					})

					itReturnsACannotBeClaimedError()
					itDoesNotAlterTheExistingLRP()

					Context("when the instance guid differs", func() {
						BeforeEach(func() {
							lrpToStart.InstanceGuid = "another-instance-guid"
						})

						itReturnsACannotBeClaimedError()
						itDoesNotAlterTheExistingLRP()
					})
				})
			})
		})

		Context("when the actual LRP does not exist", func() {
			It("creates the running LRP", func() {
				startLRPResult, err := bbs.StartActualLRP(*lrpToStart)
				Ω(err).ShouldNot(HaveOccurred())

				node, err := etcdClient.Get(shared.ActualLRPSchemaPath(lrpToStart.ProcessGuid, lrpToStart.Index))
				Ω(err).ShouldNot(HaveOccurred())

				expectedJSON, err := models.ToJSON(startLRPResult)
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

	Describe("RetireActualLRPs", func() {
		Context("with an Unclaimed LRP", func() {
			var unclaimedActualLRP *models.ActualLRP

			BeforeEach(func() {
				desiredLrp := models.DesiredLRP{ProcessGuid: "some-process-guid", Domain: "some-domain", Instances: 1}
				var err error
				unclaimedActualLRP, err = bbs.CreateActualLRP(desiredLrp, 0, lagertest.NewTestLogger("test"))
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("deletes the LRP", func() {
				err := bbs.RetireActualLRPs([]models.ActualLRP{*unclaimedActualLRP}, logger)
				Ω(err).ShouldNot(HaveOccurred())

				lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(unclaimedActualLRP.ProcessGuid, unclaimedActualLRP.Index)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(lrpInBBS).Should(BeNil())
			})
		})

		Context("when the LRP is not Unclaimed", func() {
			var claimedActualLRP1, claimedActualLRP2 *models.ActualLRP
			var cellPresence models.CellPresence

			BeforeEach(func() {
				cellPresence = models.CellPresence{
					CellID:     cellID,
					Stack:      "the-stack",
					RepAddress: "cell.example.com",
				}

				registerCell(cellPresence)

				lrp1 := models.DesiredLRP{ProcessGuid: "some-process-guid-1", Domain: "some-domain", Instances: 1}
				lrp2 := models.DesiredLRP{ProcessGuid: "some-process-guid-2", Domain: "some-domain", Instances: 1}
				logger := lagertest.NewTestLogger("test")
				var err error
				createdActualLRP1, err := bbs.CreateActualLRP(lrp1, 0, logger)
				Ω(err).ShouldNot(HaveOccurred())

				createdActualLRP1.CellID = cellID
				claimedActualLRP1, err = bbs.ClaimActualLRP(*createdActualLRP1)
				Ω(err).ShouldNot(HaveOccurred())

				createdActualLRP2, err := bbs.CreateActualLRP(lrp2, 0, logger)
				Ω(err).ShouldNot(HaveOccurred())

				createdActualLRP2.CellID = cellID
				claimedActualLRP2, err = bbs.ClaimActualLRP(*createdActualLRP2)
				Ω(err).ShouldNot(HaveOccurred())

				wg := new(sync.WaitGroup)
				wg.Add(2)

				fakeCellClient.StopLRPInstanceStub = func(string, models.ActualLRP) error {
					wg.Done()
					wg.Wait()
					return nil
				}
			})

			It("stops the LRPs in parallel", func() {
				err := bbs.RetireActualLRPs(
					[]models.ActualLRP{
						*claimedActualLRP1,
						*claimedActualLRP2,
					},
					logger,
				)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(fakeCellClient.StopLRPInstanceCallCount()).Should(Equal(2))

				addr1, stop1 := fakeCellClient.StopLRPInstanceArgsForCall(0)
				Ω(addr1).Should(Equal(cellPresence.RepAddress))

				addr2, stop2 := fakeCellClient.StopLRPInstanceArgsForCall(1)
				Ω(addr2).Should(Equal(cellPresence.RepAddress))

				Ω([]models.ActualLRP{stop1, stop2}).Should(ConsistOf([]models.ActualLRP{
					*claimedActualLRP1,
					*claimedActualLRP2,
				}))
			})
		})
	})
})
