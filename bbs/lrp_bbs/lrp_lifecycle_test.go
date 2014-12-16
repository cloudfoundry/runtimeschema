package lrp_bbs_test

import (
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
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

	BeforeEach(func() {
		actualLRPKey = models.NewActualLRPKey("some-process-guid", 2, "tests")
		containerKey = models.NewActualLRPContainerKey("some-instance-guid", cellID)
	})

	Describe("CreateActualLRP", func() {
		var (
			expectedActualLRP models.ActualLRP
		)

		BeforeEach(func() {
			expectedActualLRP = models.ActualLRP{
				ActualLRPKey: actualLRPKey,
				State:        models.ActualLRPStateUnclaimed,
			}
		})

		Context("when the LRP has an invalid process guid", func() {
			BeforeEach(func() {
				actualLRPKey.ProcessGuid = ""
			})

			It("returns a validation error only about the process guid", func() {
				_, err := bbs.CreateActualLRP(actualLRPKey)
				Ω(err).Should(ConsistOf(models.ErrInvalidField{"process_guid"}))
			})
		})

		Context("when the LRP is valid", func() {
			It("creates an unclaimed instance", func() {
				_, err := bbs.CreateActualLRP(actualLRPKey)
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
				_, err := bbs.CreateActualLRP(actualLRPKey)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("returns an error that the resource already exists", func() {
				_, err := bbs.CreateActualLRP(actualLRPKey)
				Ω(err).Should(MatchError(bbserrors.ErrStoreResourceExists))
			})
		})
	})

	Describe("ClaimActualLRP", func() {
		Context("when the actual LRP does not exist", func() {
			It("cannot claim the LRP", func() {
				_, err := bbs.ClaimActualLRP(actualLRPKey, containerKey)
				Ω(err).Should(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
			})
		})

		Context("when the actual LRP exists", func() {
			var existingLRP models.ActualLRP
			var claimedLRP *models.ActualLRP
			var claimErr error

			BeforeEach(func() {
				existingLRP = models.ActualLRP{
					ActualLRPKey:          actualLRPKey,
					ActualLRPContainerKey: models.NewActualLRPContainerKey("", ""),
					State: models.ActualLRPStateUnclaimed,
					Since: timeProvider.Now().UnixNano() - 1337,
				}
			})

			JustBeforeEach(func() {
				err := createRawActualLRP(&existingLRP)
				Ω(err).ShouldNot(HaveOccurred())

				claimedLRP, claimErr = bbs.ClaimActualLRP(actualLRPKey, containerKey)
			})

			Context("when the LRP is invalid", func() {
				BeforeEach(func() {
					containerKey.InstanceGuid = ""
				})

				It("returns a validation error", func() {
					Ω(claimErr).Should(ContainElement(models.ErrInvalidField{"instance_guid"}))
				})
			})

			Context("when the domain differs", func() {
				BeforeEach(func() {
					existingLRP.Domain = "some-other-domain"
				})

				It("returns an error", func() {
					Ω(claimErr).Should(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
				})

				It("does not alter the existing actual", func() {
					storedLRP, err := bbs.ActualLRPByProcessGuidAndIndex(actualLRPKey.ProcessGuid, actualLRPKey.Index)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(storedLRP).Should(Equal(&existingLRP))
				})
			})

			Context("when the existing ActualLRP is Unclaimed", func() {
				BeforeEach(func() {
					existingLRP.State = models.ActualLRPStateUnclaimed
					existingLRP.CellID = ""
				})

				It("claims the LRP", func() {
					Ω(claimErr).ShouldNot(HaveOccurred())

					storedLRP, err := bbs.ActualLRPByProcessGuidAndIndex(actualLRPKey.ProcessGuid, actualLRPKey.Index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(claimedLRP).Should(Equal(storedLRP))
				})

				Context("when the store is out of commission", func() {
					itRetriesUntilStoreComesBack(func() error {
						_, err := bbs.ClaimActualLRP(actualLRPKey, containerKey)
						return err
					})
				})
			})

			Context("when the existing ActualLRP is Claimed", func() {
				BeforeEach(func() {
					existingLRP.State = models.ActualLRPStateClaimed
					existingLRP.ActualLRPContainerKey = containerKey
				})

				Context("with the same cell", func() {
					BeforeEach(func() {
						existingLRP.CellID = cellID
					})

					It("does NOT return an error", func() {
						Ω(claimErr).ShouldNot(HaveOccurred())
					})

					It("does not alter the existing LRP", func() {
						storedLRP, err := bbs.ActualLRPByProcessGuidAndIndex(actualLRPKey.ProcessGuid, actualLRPKey.Index)
						Ω(err).ShouldNot(HaveOccurred())
						Ω(storedLRP).Should(Equal(&existingLRP))
					})
				})

				Context("with a different cell", func() {
					BeforeEach(func() {
						existingLRP.CellID = "another-cell-id"
					})

					It("returns an error", func() {
						Ω(claimErr).Should(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
					})

					It("does not alter the existing LRP", func() {
						storedLRP, err := bbs.ActualLRPByProcessGuidAndIndex(actualLRPKey.ProcessGuid, actualLRPKey.Index)
						Ω(err).ShouldNot(HaveOccurred())
						Ω(storedLRP).Should(Equal(&existingLRP))
					})
				})

				Context("when the instance guid differs", func() {
					BeforeEach(func() {
						existingLRP.InstanceGuid = "another-instance-guid"
					})

					It("returns an error", func() {
						Ω(claimErr).Should(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
					})

					It("does not alter the existing actual", func() {
						storedLRP, err := bbs.ActualLRPByProcessGuidAndIndex(actualLRPKey.ProcessGuid, actualLRPKey.Index)
						Ω(err).ShouldNot(HaveOccurred())
						Ω(storedLRP).Should(Equal(&existingLRP))
					})
				})
			})

			Context("when the existing ActualLRP is Running", func() {
				BeforeEach(func() {
					existingLRP.State = models.ActualLRPStateRunning
					existingLRP.ActualLRPContainerKey = containerKey
				})

				Context("with the same cell", func() {
					BeforeEach(func() {
						existingLRP.CellID = cellID
					})

					It("claims the LRP", func() {
						Ω(claimErr).ShouldNot(HaveOccurred())

						storedLRP, err := bbs.ActualLRPByProcessGuidAndIndex(actualLRPKey.ProcessGuid, actualLRPKey.Index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(claimedLRP).Should(Equal(storedLRP))
					})

					Context("when the store is out of commission", func() {
						itRetriesUntilStoreComesBack(func() error {
							_, err := bbs.ClaimActualLRP(actualLRPKey, containerKey)
							return err
						})
					})
				})

				Context("with a different cell", func() {
					BeforeEach(func() {
						existingLRP.CellID = "another-cell-id"
					})

					It("returns an error", func() {
						Ω(claimErr).Should(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
					})

					It("does not alter the existing LRP", func() {
						storedLRP, err := bbs.ActualLRPByProcessGuidAndIndex(actualLRPKey.ProcessGuid, actualLRPKey.Index)
						Ω(err).ShouldNot(HaveOccurred())
						Ω(storedLRP).Should(Equal(&existingLRP))
					})
				})

				Context("when the instance guid differs", func() {
					BeforeEach(func() {
						existingLRP.InstanceGuid = "another-instance-guid"
					})

					It("returns an error", func() {
						Ω(claimErr).Should(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
					})

					It("does not alter the existing actual", func() {
						storedLRP, err := bbs.ActualLRPByProcessGuidAndIndex(actualLRPKey.ProcessGuid, actualLRPKey.Index)
						Ω(err).ShouldNot(HaveOccurred())
						Ω(storedLRP).Should(Equal(&existingLRP))
					})
				})
			})
		})
	})

	Describe("StartActualLRP", func() {
		var netInfo models.ActualLRPNetInfo

		BeforeEach(func() {
			netInfo = models.NewActualLRPNetInfo("127.0.0.2", []models.PortMapping{{8081, 87}})
		})

		Context("when the LRP is invalid", func() {
			BeforeEach(func() {
				containerKey.InstanceGuid = ""
			})

			It("returns a validation error", func() {
				_, err := bbs.StartActualLRP(actualLRPKey, containerKey, netInfo)
				Ω(err).Should(ContainElement(models.ErrInvalidField{"instance_guid"}))
			})
		})

		Context("when the actual LRP exists", func() {
			var (
				existingLRP    models.ActualLRP
				startLRPResult *models.ActualLRP
				startErr       error

				updatedTime int64
			)

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
					storedLRP, err := bbs.ActualLRPByProcessGuidAndIndex(actualLRPKey.ProcessGuid, actualLRPKey.Index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(storedLRP).Should(Equal(&existingLRP))
				})
			}

			itReturnsTheExistingLRP := func() {
				It("returns the existing LRP", func() {
					Ω(startLRPResult).Should(Equal(&existingLRP))
				})
			}

			itStartsTheLRP := func() {
				It("starts the LRP", func() {
					storedLRP, err := bbs.ActualLRPByProcessGuidAndIndex(actualLRPKey.ProcessGuid, actualLRPKey.Index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(startLRPResult).Should(Equal(storedLRP))
					Ω(startLRPResult.State).Should(Equal(models.ActualLRPStateRunning))
					Ω(startLRPResult.Since).Should(Equal(updatedTime))
				})
			}

			BeforeEach(func() {
				existingLRP = models.ActualLRP{
					ActualLRPKey:          actualLRPKey,
					ActualLRPContainerKey: containerKey,
					State: models.ActualLRPStateClaimed,
					Since: timeProvider.Now().UnixNano(),
				}
			})

			JustBeforeEach(func() {
				err := createRawActualLRP(&existingLRP)
				Ω(err).ShouldNot(HaveOccurred())

				timeProvider.Increment(500 * time.Nanosecond)
				updatedTime = timeProvider.Now().UnixNano()
				startLRPResult, startErr = bbs.StartActualLRP(actualLRPKey, containerKey, netInfo)
			})

			Context("when the domain differs", func() {
				BeforeEach(func() {
					existingLRP.Domain = "some-other-domain"
				})

				itReturnsACannotBeClaimedError()
				itReturnsTheExistingLRP()
				itDoesNotAlterTheExistingLRP()
			})

			Context("when the actual is Unclaimed", func() {
				BeforeEach(func() {
					existingLRP.State = models.ActualLRPStateUnclaimed
					existingLRP.ActualLRPContainerKey = models.NewActualLRPContainerKey("", "")
				})

				itDoesNotReturnAnError()
				itStartsTheLRP()
			})

			Context("when the actual is Claimed", func() {
				BeforeEach(func() {
					existingLRP.State = models.ActualLRPStateClaimed
				})

				itDoesNotReturnAnError()
				itStartsTheLRP()

				Context("with a different instance guid", func() {
					BeforeEach(func() {
						existingLRP.InstanceGuid = "another-instance-guid"
					})

					itDoesNotReturnAnError()
					itStartsTheLRP()
				})

				Context("with a different cell", func() {
					BeforeEach(func() {
						existingLRP.CellID = "another-cell-id"
					})

					itDoesNotReturnAnError()
					itStartsTheLRP()
				})
			})

			Context("when the actual is Running", func() {
				BeforeEach(func() {
					existingLRP.State = models.ActualLRPStateRunning
				})

				Context("with the same cell", func() {
					itDoesNotReturnAnError()
					itReturnsTheExistingLRP()
					itDoesNotAlterTheExistingLRP()

					Context("when the instance guid differs", func() {
						BeforeEach(func() {
							existingLRP.InstanceGuid = "another-instance-guid"
						})

						itReturnsACannotBeClaimedError()
						itDoesNotAlterTheExistingLRP()
						itReturnsTheExistingLRP()
					})
				})

				Context("with a different cell", func() {
					BeforeEach(func() {
						existingLRP.CellID = "another-cell-id"
					})

					itReturnsACannotBeClaimedError()
					itReturnsTheExistingLRP()
					itDoesNotAlterTheExistingLRP()

					Context("when the instance guid differs", func() {
						BeforeEach(func() {
							existingLRP.InstanceGuid = "another-instance-guid"
						})

						itReturnsACannotBeClaimedError()
						itReturnsTheExistingLRP()
						itDoesNotAlterTheExistingLRP()
					})
				})
			})
		})

		Context("when the actual LRP does not exist", func() {
			It("creates the running LRP", func() {
				startLRPResult, err := bbs.StartActualLRP(actualLRPKey, containerKey, netInfo)
				Ω(err).ShouldNot(HaveOccurred())

				node, err := etcdClient.Get(shared.ActualLRPSchemaPath(actualLRPKey.ProcessGuid, actualLRPKey.Index))
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
			netInfo := models.NewActualLRPNetInfo("127.0.0.3", []models.PortMapping{{9090, 90}})
			var err error

			runningLRP, err = bbs.StartActualLRP(actualLRPKey, containerKey, netInfo)
			Ω(err).ShouldNot(HaveOccurred())
		})

		Context("when the LRP matches", func() {
			It("removes the LRP", func() {
				err := bbs.RemoveActualLRP(actualLRPKey, containerKey)
				Ω(err).ShouldNot(HaveOccurred())

				_, err = etcdClient.Get(shared.ActualLRPSchemaPath(runningLRP.ProcessGuid, runningLRP.Index))
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
			var unclaimedActualLRP *models.ActualLRP

			BeforeEach(func() {
				key := models.NewActualLRPKey("some-process-guid", 1, "some-domain")
				var err error
				unclaimedActualLRP, err = bbs.CreateActualLRP(key)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("deletes the LRP", func() {
				err := bbs.RetireActualLRPs([]models.ActualLRP{*unclaimedActualLRP}, logger)
				Ω(err).ShouldNot(HaveOccurred())

				_, err = bbs.ActualLRPByProcessGuidAndIndex(unclaimedActualLRP.ProcessGuid, unclaimedActualLRP.Index)
				Ω(err).Should(Equal(bbserrors.ErrStoreResourceNotFound))
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
				lrpKey1 := models.NewActualLRPKey("some-process-guid-1", 1, "some-domain")
				lrpKey2 := models.NewActualLRPKey("some-process-guid-2", 2, "some-domain")

				lrpContainerKey1 := models.NewActualLRPContainerKey("some-instance-guid-1", cellID)
				lrpContainerKey2 := models.NewActualLRPContainerKey("some-instance-guid-2", cellID)

				var err error
				_, claimedActualLRP1, err = createAndClaim(lrpKey1, lrpContainerKey1)
				Ω(err).ShouldNot(HaveOccurred())

				_, claimedActualLRP2, err = createAndClaim(lrpKey2, lrpContainerKey2)
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
