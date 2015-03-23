package lrp_bbs_test

import (
	"fmt"

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

var _ = Describe("Actual LRP Lifecycle", func() {
	const cellID = "some-cell-id"
	var actualLRPKey models.ActualLRPKey
	var instanceKey models.ActualLRPInstanceKey
	var netInfo models.ActualLRPNetInfo
	var index int

	BeforeEach(func() {
		index = 2
		actualLRPKey = models.NewActualLRPKey("some-process-guid", index, "tests")
		instanceKey = models.NewActualLRPInstanceKey("some-instance-guid", cellID)
		netInfo = models.NewActualLRPNetInfo("127.0.0.2", []models.PortMapping{{8081, 87}})
	})

	Describe("ClaimActualLRP", func() {
		var claimErr error
		var lrpKey models.ActualLRPKey
		var instanceKey models.ActualLRPInstanceKey

		JustBeforeEach(func() {
			claimErr = bbs.ClaimActualLRP(logger, lrpKey, instanceKey)
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
					Stack:       "some-stack",
					Instances:   index + 1,
					Action: &models.RunAction{
						Path: "true",
					},
				}

				err := bbs.DesireLRP(logger, desiredLRP)
				Ω(err).ShouldNot(HaveOccurred())

				lrpGroup, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
				Ω(err).ShouldNot(HaveOccurred())
				createdLRP = *lrpGroup.Instance
			})

			Context("when the instance key is invalid", func() {
				BeforeEach(func() {
					lrpKey = createdLRP.ActualLRPKey
					instanceKey = models.NewActualLRPInstanceKey(
						"", // invalid InstanceGuid
						cellID,
					)
				})

				It("returns a validation error", func() {
					Ω(claimErr).Should(ContainElement(models.ErrInvalidField{"instance_guid"}))
				})

				It("does not modify the persisted actual LRP", func() {
					lrpGroupInBBS, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrpGroupInBBS.Instance.State).Should(Equal(models.ActualLRPStateUnclaimed))
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
					instanceKey = models.NewActualLRPInstanceKey("some-instance-guid", cellID)
				})

				It("returns an error", func() {
					Ω(claimErr).Should(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
				})

				It("does not modify the persisted actual LRP", func() {
					lrpGroupInBBS, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrpGroupInBBS.Instance.State).Should(Equal(models.ActualLRPStateUnclaimed))
				})
			})

			Context("when the existing ActualLRP is Unclaimed", func() {
				BeforeEach(func() {
					lrpKey = createdLRP.ActualLRPKey
					instanceKey = models.NewActualLRPInstanceKey("some-instance-guid", cellID)
				})

				It("does not error", func() {
					Ω(claimErr).ShouldNot(HaveOccurred())
				})

				It("claims the actual LRP", func() {
					lrpGroupInBBS, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrpGroupInBBS.Instance.State).Should(Equal(models.ActualLRPStateClaimed))
				})

				It("updates the ModificationIndex", func() {
					lrpGroupInBBS, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrpGroupInBBS.Instance.ModificationTag.Index).Should(Equal(createdLRP.ModificationTag.Index + 1))
				})
			})

			Context("when the existing ActualLRP is Claimed", func() {
				var instanceGuid string

				BeforeEach(func() {
					instanceGuid = "some-instance-guid"
					err := bbs.ClaimActualLRP(
						logger,
						createdLRP.ActualLRPKey,
						models.NewActualLRPInstanceKey(instanceGuid, cellID),
					)
					Ω(err).ShouldNot(HaveOccurred())
				})

				Context("with the same cell and instance guid", func() {
					var previousTime int64

					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						instanceKey = models.NewActualLRPInstanceKey(instanceGuid, cellID)

						previousTime = clock.Now().UnixNano()
						clock.IncrementBySeconds(1)
					})

					It("does not return an error", func() {
						Ω(claimErr).ShouldNot(HaveOccurred())
					})

					It("does not alter the state of the persisted LRP", func() {
						lrpGroupInBBS, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(lrpGroupInBBS.Instance.State).Should(Equal(models.ActualLRPStateClaimed))
					})

					It("does not update the timestamp of the persisted actual lrp", func() {
						lrpGroupInBBS, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(lrpGroupInBBS.Instance.Since).Should(Equal(previousTime))
					})
				})

				Context("with a different cell", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						instanceKey = models.NewActualLRPInstanceKey(instanceGuid, "another-cell-id")
					})

					It("returns an error", func() {
						Ω(claimErr).Should(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
					})

					It("does not alter the existing LRP", func() {
						lrpGroupInBBS, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(lrpGroupInBBS.Instance.CellID).Should(Equal(cellID))
					})
				})

				Context("when the instance guid differs", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						instanceKey = models.NewActualLRPInstanceKey("another-instance-guid", cellID)
					})

					It("returns an error", func() {
						Ω(claimErr).Should(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
					})

					It("does not alter the existing actual", func() {
						lrpGroupInBBS, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(lrpGroupInBBS.Instance.InstanceGuid).Should(Equal(instanceGuid))
					})
				})
			})

			Context("when the existing ActualLRP is Running", func() {
				var instanceGuid string

				BeforeEach(func() {
					instanceGuid = "some-instance-guid"
					err := bbs.StartActualLRP(
						logger,
						createdLRP.ActualLRPKey,
						models.NewActualLRPInstanceKey(instanceGuid, cellID),
						models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{{ContainerPort: 1234, HostPort: 5678}}),
					)
					Ω(err).ShouldNot(HaveOccurred())
				})

				Context("with the same cell and instance guid", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						instanceKey = models.NewActualLRPInstanceKey(instanceGuid, cellID)
					})

					It("does not return an error", func() {
						Ω(claimErr).ShouldNot(HaveOccurred())
					})

					It("reverts the persisted LRP to the CLAIMED state", func() {
						lrpGroupInBBS, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(lrpGroupInBBS.Instance.State).Should(Equal(models.ActualLRPStateClaimed))
					})

					It("clears the net info", func() {
						lrpGroupInBBS, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(lrpGroupInBBS.Instance.Address).Should(BeEmpty())
						Ω(lrpGroupInBBS.Instance.Ports).Should(BeEmpty())
					})
				})

				Context("with a different cell", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						instanceKey = models.NewActualLRPInstanceKey(instanceGuid, "another-cell-id")
					})

					It("returns an error", func() {
						Ω(claimErr).Should(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
					})

					It("does not alter the existing LRP", func() {
						lrpGroupInBBS, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(lrpGroupInBBS.Instance.CellID).Should(Equal(cellID))
					})
				})

				Context("when the instance guid differs", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						instanceKey = models.NewActualLRPInstanceKey("another-instance-guid", cellID)
					})

					It("returns an error", func() {
						Ω(claimErr).Should(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
					})

					It("does not alter the existing actual", func() {
						lrpGroupInBBS, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(lrpGroupInBBS.Instance.InstanceGuid).Should(Equal(instanceGuid))
					})
				})
			})

			Context("when there is a placement error", func() {
				BeforeEach(func() {
					lrpKey = createdLRP.ActualLRPKey
					instanceKey = models.NewActualLRPInstanceKey("some-instance-guid", cellID)

					err := bbs.FailActualLRP(logger, lrpKey, "insufficient resources")
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("should clear placement error", func() {
					createdLRP, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(createdLRP.Instance.PlacementError).Should(BeEmpty())
				})
			})
		})

		Context("when the actual LRP does not exist", func() {
			BeforeEach(func() {
				lrpKey = models.NewActualLRPKey("process-guid", 1, "domain")
				instanceKey = models.NewActualLRPInstanceKey("instance-guid", cellID)
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
		var instanceKey models.ActualLRPInstanceKey
		var netInfo models.ActualLRPNetInfo

		JustBeforeEach(func() {
			startErr = bbs.StartActualLRP(logger, lrpKey, instanceKey, netInfo)
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
					Stack:       "some-stack",
					Instances:   index + 1,
					Action: &models.RunAction{
						Path: "true",
					},
				}

				err := bbs.DesireLRP(logger, desiredLRP)
				Ω(err).ShouldNot(HaveOccurred())

				lrpGroup, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
				Ω(err).ShouldNot(HaveOccurred())
				createdLRP = *lrpGroup.Instance
			})

			Context("when the instance key is invalid", func() {
				BeforeEach(func() {
					lrpKey = createdLRP.ActualLRPKey
					instanceKey = models.NewActualLRPInstanceKey(
						"", // invalid InstanceGuid
						cellID,
					)
					netInfo = models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{{ContainerPort: 1234, HostPort: 5678}})
				})

				It("returns a validation error", func() {
					Ω(startErr).Should(ContainElement(models.ErrInvalidField{"instance_guid"}))
				})

				It("does not modify the persisted actual LRP", func() {
					lrpGroupInBBS, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrpGroupInBBS.Instance.State).Should(Equal(models.ActualLRPStateUnclaimed))
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
					instanceKey = models.NewActualLRPInstanceKey("some-instance-guid", cellID)
					netInfo = models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{{ContainerPort: 1234, HostPort: 5678}})
				})

				It("returns an error", func() {
					Ω(startErr).Should(Equal(bbserrors.ErrActualLRPCannotBeStarted))
				})

				It("does not modify the persisted actual LRP", func() {
					lrpGroupInBBS, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrpGroupInBBS.Instance.State).Should(Equal(models.ActualLRPStateUnclaimed))
				})
			})

			Context("when the existing ActualLRP is Unclaimed", func() {
				BeforeEach(func() {
					lrpKey = createdLRP.ActualLRPKey
					instanceKey = models.NewActualLRPInstanceKey("some-instance-guid", cellID)
					netInfo = models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{{ContainerPort: 1234, HostPort: 5678}})
				})

				It("does not error", func() {
					Ω(startErr).ShouldNot(HaveOccurred())
				})

				It("starts the actual LRP", func() {
					lrpGroupInBBS, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrpGroupInBBS.Instance.State).Should(Equal(models.ActualLRPStateRunning))
				})

				Context("when there is a placement error", func() {
					BeforeEach(func() {
						err := bbs.FailActualLRP(logger, lrpKey, "found no compatible cells")
						Ω(err).ShouldNot(HaveOccurred())
					})

					It("should clear placement error", func() {
						createdLRP, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
						Ω(err).ShouldNot(HaveOccurred())
						Ω(createdLRP.Instance.PlacementError).Should(BeEmpty())
					})
				})
			})

			Context("when the existing ActualLRP is Claimed", func() {
				var instanceGuid string

				BeforeEach(func() {
					instanceGuid = "some-instance-guid"
					err := bbs.ClaimActualLRP(
						logger,
						createdLRP.ActualLRPKey,
						models.NewActualLRPInstanceKey(instanceGuid, cellID),
					)
					Ω(err).ShouldNot(HaveOccurred())
				})

				Context("with the same cell and instance guid", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						instanceKey = models.NewActualLRPInstanceKey(instanceGuid, cellID)
						netInfo = models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{{ContainerPort: 1234, HostPort: 5678}})
					})

					It("does not return an error", func() {
						Ω(startErr).ShouldNot(HaveOccurred())
					})

					It("promotes the persisted LRP to RUNNING", func() {
						lrpGroupInBBS, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(lrpGroupInBBS.Instance.State).Should(Equal(models.ActualLRPStateRunning))
					})
				})

				Context("with a different cell", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						instanceKey = models.NewActualLRPInstanceKey(instanceGuid, "another-cell-id")
						netInfo = models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{{ContainerPort: 1234, HostPort: 5678}})
					})

					It("does not return an error", func() {
						Ω(startErr).ShouldNot(HaveOccurred())
					})

					It("promotes the persisted LRP to RUNNING", func() {
						lrpGroupInBBS, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(lrpGroupInBBS.Instance.State).Should(Equal(models.ActualLRPStateRunning))
					})
				})

				Context("when the instance guid differs", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						instanceKey = models.NewActualLRPInstanceKey("another-instance-guid", cellID)
						netInfo = models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{{ContainerPort: 1234, HostPort: 5678}})
					})

					It("does not return an error", func() {
						Ω(startErr).ShouldNot(HaveOccurred())
					})

					It("promotes the persisted LRP to RUNNING", func() {
						lrpGroupInBBS, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(lrpGroupInBBS.Instance.State).Should(Equal(models.ActualLRPStateRunning))
					})
				})

			})

			Context("when the existing ActualLRP is Running", func() {
				var instanceGuid string

				BeforeEach(func() {
					instanceGuid = "some-instance-guid"
					err := bbs.StartActualLRP(
						logger,
						createdLRP.ActualLRPKey,
						models.NewActualLRPInstanceKey(instanceGuid, cellID),
						models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{{ContainerPort: 1234, HostPort: 5678}}),
					)
					Ω(err).ShouldNot(HaveOccurred())
				})

				Context("with the same cell and instance guid", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						instanceKey = models.NewActualLRPInstanceKey(instanceGuid, cellID)
						netInfo = models.NewActualLRPNetInfo("5.6.7.8", []models.PortMapping{{ContainerPort: 4567, HostPort: 4321}})
					})

					It("does not return an error", func() {
						Ω(startErr).ShouldNot(HaveOccurred())
					})

					It("does not alter the state of the persisted LRP", func() {
						lrpGroupInBBS, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(lrpGroupInBBS.Instance.State).Should(Equal(models.ActualLRPStateRunning))
					})

					It("updates the net info", func() {
						lrpGroupInBBS, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(lrpGroupInBBS.Instance.ActualLRPNetInfo).Should(Equal(netInfo))
					})

					Context("and the same net info", func() {
						var previousTime int64
						BeforeEach(func() {
							lrpKey = createdLRP.ActualLRPKey
							instanceKey = models.NewActualLRPInstanceKey(instanceGuid, cellID)
							netInfo = models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{{ContainerPort: 1234, HostPort: 5678}})

							previousTime = clock.Now().UnixNano()
							clock.IncrementBySeconds(1)
						})

						It("does not update the timestamp of the persisted actual lrp", func() {
							lrpGroupInBBS, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
							Ω(err).ShouldNot(HaveOccurred())

							Ω(lrpGroupInBBS.Instance.Since).Should(Equal(previousTime))
						})
					})
				})

				Context("with a different cell", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						instanceKey = models.NewActualLRPInstanceKey(instanceGuid, "another-cell-id")
					})

					It("returns an error", func() {
						Ω(startErr).Should(Equal(bbserrors.ErrActualLRPCannotBeStarted))
					})

					It("does not alter the existing LRP", func() {
						lrpGroupInBBS, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(lrpGroupInBBS.Instance.CellID).Should(Equal(cellID))
					})
				})

				Context("when the instance guid differs", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						instanceKey = models.NewActualLRPInstanceKey("another-instance-guid", cellID)
					})

					It("returns an error", func() {
						Ω(startErr).Should(Equal(bbserrors.ErrActualLRPCannotBeStarted))
					})

					It("does not alter the existing actual", func() {
						lrpGroupInBBS, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(lrpGroupInBBS.Instance.InstanceGuid).Should(Equal(instanceGuid))
					})
				})
			})
		})

		Context("when the actual LRP does not exist", func() {
			BeforeEach(func() {
				lrpKey = models.NewActualLRPKey("process-guid", 1, "domain")
				instanceKey = models.NewActualLRPInstanceKey("instance-guid", cellID)
				netInfo = models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{{ContainerPort: 1234, HostPort: 5678}})
			})

			It("starts the LRP", func() {
				Ω(startErr).ShouldNot(HaveOccurred())
			})

			It("sets the State", func() {
				lrpGroup, err := bbs.ActualLRPGroupByProcessGuidAndIndex("process-guid", 1)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(lrpGroup.Instance.State).Should(Equal(models.ActualLRPStateRunning))
			})

			It("sets the ModificationTag", func() {
				lrpGroup, err := bbs.ActualLRPGroupByProcessGuidAndIndex("process-guid", 1)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(lrpGroup.Instance.ModificationTag.Epoch).ShouldNot(BeEmpty())
				Ω(lrpGroup.Instance.ModificationTag.Index).Should(BeEquivalentTo(0))
			})
		})
	})

	Describe("RemoveActualLRP", func() {
		BeforeEach(func() {
			netInfo := models.NewActualLRPNetInfo("127.0.0.3", []models.PortMapping{{9090, 90}})
			err := bbs.StartActualLRP(logger, actualLRPKey, instanceKey, netInfo)
			Ω(err).ShouldNot(HaveOccurred())
		})

		Context("when the LRP matches", func() {
			It("removes the LRP", func() {
				err := bbs.RemoveActualLRP(logger, actualLRPKey, instanceKey)
				Ω(err).ShouldNot(HaveOccurred())

				_, err = etcdClient.Get(shared.ActualLRPSchemaPath(actualLRPKey.ProcessGuid, actualLRPKey.Index))
				Ω(err).Should(MatchError(storeadapter.ErrorKeyNotFound))
			})
		})

		Context("when the LRP differs from the one in the store", func() {
			It("does not delete the LRP", func() {
				instanceKey.InstanceGuid = "another-instance-guid"
				err := bbs.RemoveActualLRP(logger, actualLRPKey, instanceKey)
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
					Stack:       "some-stack",
					Instances:   2,
					Action: &models.RunAction{
						Path: "true",
					},
				}

				err := bbs.DesireLRP(logger, desiredLRP)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("deletes the LRP", func() {
				lrpGroupInBBS, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
				Ω(err).ShouldNot(HaveOccurred())

				bbs.RetireActualLRPs(logger, []models.ActualLRPKey{lrpGroupInBBS.Instance.ActualLRPKey})

				_, err = bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
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
				setRawActualLRP(actual)
			})

			JustBeforeEach(func() {
				bbs.RetireActualLRPs(logger, []models.ActualLRPKey{actual.ActualLRPKey})
			})

			It("should remove the actual", func() {
				_, err := bbs.ActualLRPGroupByProcessGuidAndIndex(actual.ProcessGuid, actual.Index)
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
					Stack:       "some-stack",
					Instances:   2,
					Action: &models.RunAction{
						Path: "true",
					},
				}

				lrpInstanceKey1 := models.NewActualLRPInstanceKey("some-instance-guid-1", cellID)
				lrpInstanceKey2 := models.NewActualLRPInstanceKey("some-instance-guid-2", cellID)

				errDesire := bbs.DesireLRP(logger, desiredLRP)
				Ω(errDesire).ShouldNot(HaveOccurred())

				claimDesireLRPByIndex(desiredLRP, 0, lrpInstanceKey1, logger)
				claimDesireLRPByIndex(desiredLRP, 1, lrpInstanceKey2, logger)

				blockStopInstanceChan = make(chan struct{})

				fakeCellClient.StopLRPInstanceStub = func(string, models.ActualLRPKey, models.ActualLRPInstanceKey) error {
					<-blockStopInstanceChan
					return nil
				}
			})

			JustBeforeEach(func() {
				lrpGroup1, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, 0)
				Ω(err).ShouldNot(HaveOccurred())
				claimedLRP1 = *lrpGroup1.Instance

				lrpGroup2, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, 1)
				Ω(err).ShouldNot(HaveOccurred())
				claimedLRP2 = *lrpGroup2.Instance

				doneRetiring = make(chan struct{})

				go func(bbs *lrp_bbs.LRPBBS, lrp1, lrp2 models.ActualLRP, doneRetiring chan struct{}, logger lager.Logger) {
					bbs.RetireActualLRPs(logger, []models.ActualLRPKey{lrp1.ActualLRPKey, lrp2.ActualLRPKey})
					close(doneRetiring)
				}(bbs, claimedLRP1, claimedLRP2, doneRetiring, logger)
			})

			Context("when the cell is present", func() {
				BeforeEach(func() {
					cellPresence = models.NewCellPresence(cellID, "cell.example.com", "the-zone", models.NewCellCapacity(128, 1024, 6))
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

					Ω([]models.ActualLRPInstanceKey{cnrKey1, cnrKey2}).Should(ConsistOf(
						claimedLRP1.ActualLRPInstanceKey,
						claimedLRP2.ActualLRPInstanceKey,
					))

					Consistently(doneRetiring).ShouldNot(BeClosed())

					close(blockStopInstanceChan)

					Eventually(doneRetiring).Should(BeClosed())
				})

				Context("when stopping any of the LRPs fails", func() {
					BeforeEach(func() {
						fakeCellClient.StopLRPInstanceStub = func(cellAddr string, key models.ActualLRPKey, _ models.ActualLRPInstanceKey) error {
							return fmt.Errorf("failed to stop %d", key.Index)
						}
					})

					It("logs the failure", func() {
						Eventually(doneRetiring).Should(BeClosed())
						Ω(logger.LogMessages()).Should(ContainElement("test.retire-actual-lrps.failed-to-retire"))
					})

					It("retries", func() {
						Eventually(doneRetiring).Should(BeClosed())
						Ω(fakeCellClient.StopLRPInstanceCallCount()).To(Equal(2 * lrp_bbs.RetireActualLRPRetryAttempts))
					})

					It("logs each retry", func() {
						Eventually(doneRetiring).Should(BeClosed())
						Ω(logger.LogMessages()).Should(ContainElement("test.retire-actual-lrps.retrying-failed-retire-of-actual-lrp"))
					})
				})
			})

			Context("when the cell is not present", func() {
				It("does not stop the instances", func() {
					Eventually(fakeCellClient.StopLRPInstanceCallCount).Should(Equal(0))
				})

				It("logs the error", func() {
					Eventually(logger.TestSink.LogMessages).Should(ContainElement("test.retire-actual-lrps.failed-to-retire"))
				})
			})
		})
	})

	Describe("FailActualLRP", func() {
		var (
			placementError string
			instanceGuid   string
			processGuid    string
			index          int
			createdLRP     models.ActualLRP
		)

		BeforeEach(func() {
			index = 1
			placementError = "insufficient resources"
			processGuid = "process-guid"
			instanceGuid = "instance-guid"
		})

		Context("when lrp exists", func() {
			BeforeEach(func() {
				desiredLRP := models.DesiredLRP{
					ProcessGuid: processGuid,
					Domain:      "the-domain",
					Stack:       "some-stack",
					Instances:   3,
					Action: &models.RunAction{
						Path: "true",
					},
				}

				errDesire := bbs.DesireLRP(logger, desiredLRP)
				Ω(errDesire).ShouldNot(HaveOccurred())

				createdLRPGroup, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
				Ω(err).ShouldNot(HaveOccurred())

				actualLRPKey = createdLRPGroup.Instance.ActualLRPKey
				instanceKey = models.NewActualLRPInstanceKey(instanceGuid, cellID)
			})

			Context("in unclaimed state", func() {
				BeforeEach(func() {
					err := bbs.FailActualLRP(logger, actualLRPKey, placementError)
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("sets the placement error", func() {
					failedActualLRPGroup, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(failedActualLRPGroup.Instance.PlacementError).Should(Equal(placementError))
				})

				It("updates the ModificationIndex", func() {
					failedActualLRPGroup, err := bbs.ActualLRPGroupByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(failedActualLRPGroup.Instance.ModificationTag.Index).Should(Equal(createdLRP.ModificationTag.Index + 1))
				})
			})

			Context("not in unclaimed state", func() {
				BeforeEach(func() {
					claimErr := bbs.ClaimActualLRP(logger, actualLRPKey, instanceKey)
					Ω(claimErr).ShouldNot(HaveOccurred())
				})

				It("returns an error", func() {
					err := bbs.FailActualLRP(logger, actualLRPKey, placementError)
					Ω(err).Should(HaveOccurred())
				})
			})
		})

		Context("when lrp does not exist", func() {
			It("returns an error", func() {
				actualLRPKey := models.NewActualLRPKey("non-existent-process-guid", index, "tests")
				err := bbs.FailActualLRP(logger, actualLRPKey, placementError)
				Ω(err).Should(Equal(bbserrors.ErrActualLRPCannotBeFailed))
			})
		})
	})
})
