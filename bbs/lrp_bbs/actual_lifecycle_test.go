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
			claimErr = lrpBBS.ClaimActualLRP(logger, lrpKey, instanceKey)
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
					RootFS:      "some:rootfs",
					Instances:   index + 1,
					Action: &models.RunAction{
						Path: "true",
						User: "me",
					},
				}

				err := lrpBBS.DesireLRP(logger, desiredLRP)
				Expect(err).NotTo(HaveOccurred())

				lrpGroup, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, index)
				Expect(err).NotTo(HaveOccurred())
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
					Expect(claimErr).To(ContainElement(models.ErrInvalidField{"instance_guid"}))
				})

				It("does not modify the persisted actual LRP", func() {
					lrpGroupInBBS, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, index)
					Expect(err).NotTo(HaveOccurred())

					Expect(lrpGroupInBBS.Instance.State).To(Equal(models.ActualLRPStateUnclaimed))
				})

				It("logs the error", func() {
					Expect(logger.TestSink.LogMessages()).To(ContainElement("test.claim-actual-lrp.failed-to-marshal-actual-lrp"))
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
					Expect(claimErr).To(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
				})

				It("does not modify the persisted actual LRP", func() {
					lrpGroupInBBS, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, index)
					Expect(err).NotTo(HaveOccurred())

					Expect(lrpGroupInBBS.Instance.State).To(Equal(models.ActualLRPStateUnclaimed))
				})
			})

			Context("when the existing ActualLRP is Unclaimed", func() {
				BeforeEach(func() {
					lrpKey = createdLRP.ActualLRPKey
					instanceKey = models.NewActualLRPInstanceKey("some-instance-guid", cellID)
				})

				It("does not error", func() {
					Expect(claimErr).NotTo(HaveOccurred())
				})

				It("claims the actual LRP", func() {
					lrpGroupInBBS, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, index)
					Expect(err).NotTo(HaveOccurred())

					Expect(lrpGroupInBBS.Instance.State).To(Equal(models.ActualLRPStateClaimed))
				})

				It("updates the ModificationIndex", func() {
					lrpGroupInBBS, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, index)
					Expect(err).NotTo(HaveOccurred())

					Expect(lrpGroupInBBS.Instance.ModificationTag.Index).To(Equal(createdLRP.ModificationTag.Index + 1))
				})
			})

			Context("when the existing ActualLRP is Claimed", func() {
				var instanceGuid string

				BeforeEach(func() {
					instanceGuid = "some-instance-guid"
					err := lrpBBS.ClaimActualLRP(
						logger,
						createdLRP.ActualLRPKey,
						models.NewActualLRPInstanceKey(instanceGuid, cellID),
					)
					Expect(err).NotTo(HaveOccurred())
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
						Expect(claimErr).NotTo(HaveOccurred())
					})

					It("does not alter the state of the persisted LRP", func() {
						lrpGroupInBBS, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, index)
						Expect(err).NotTo(HaveOccurred())

						Expect(lrpGroupInBBS.Instance.State).To(Equal(models.ActualLRPStateClaimed))
					})

					It("does not update the timestamp of the persisted actual lrp", func() {
						lrpGroupInBBS, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, index)
						Expect(err).NotTo(HaveOccurred())

						Expect(lrpGroupInBBS.Instance.Since).To(Equal(previousTime))
					})
				})

				Context("with a different cell", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						instanceKey = models.NewActualLRPInstanceKey(instanceGuid, "another-cell-id")
					})

					It("returns an error", func() {
						Expect(claimErr).To(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
					})

					It("does not alter the existing LRP", func() {
						lrpGroupInBBS, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, index)
						Expect(err).NotTo(HaveOccurred())

						Expect(lrpGroupInBBS.Instance.CellID).To(Equal(cellID))
					})
				})

				Context("when the instance guid differs", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						instanceKey = models.NewActualLRPInstanceKey("another-instance-guid", cellID)
					})

					It("returns an error", func() {
						Expect(claimErr).To(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
					})

					It("does not alter the existing actual", func() {
						lrpGroupInBBS, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, index)
						Expect(err).NotTo(HaveOccurred())

						Expect(lrpGroupInBBS.Instance.InstanceGuid).To(Equal(instanceGuid))
					})
				})
			})

			Context("when the existing ActualLRP is Running", func() {
				var instanceGuid string

				BeforeEach(func() {
					instanceGuid = "some-instance-guid"
					err := lrpBBS.StartActualLRP(
						logger,
						createdLRP.ActualLRPKey,
						models.NewActualLRPInstanceKey(instanceGuid, cellID),
						models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{{ContainerPort: 1234, HostPort: 5678}}),
					)
					Expect(err).NotTo(HaveOccurred())
				})

				Context("with the same cell and instance guid", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						instanceKey = models.NewActualLRPInstanceKey(instanceGuid, cellID)
					})

					It("does not return an error", func() {
						Expect(claimErr).NotTo(HaveOccurred())
					})

					It("reverts the persisted LRP to the CLAIMED state", func() {
						lrpGroupInBBS, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, index)
						Expect(err).NotTo(HaveOccurred())

						Expect(lrpGroupInBBS.Instance.State).To(Equal(models.ActualLRPStateClaimed))
					})

					It("clears the net info", func() {
						lrpGroupInBBS, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, index)
						Expect(err).NotTo(HaveOccurred())

						Expect(lrpGroupInBBS.Instance.Address).To(BeEmpty())
						Expect(lrpGroupInBBS.Instance.Ports).To(BeEmpty())
					})
				})

				Context("with a different cell", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						instanceKey = models.NewActualLRPInstanceKey(instanceGuid, "another-cell-id")
					})

					It("returns an error", func() {
						Expect(claimErr).To(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
					})

					It("does not alter the existing LRP", func() {
						lrpGroupInBBS, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, index)
						Expect(err).NotTo(HaveOccurred())

						Expect(lrpGroupInBBS.Instance.CellID).To(Equal(cellID))
					})
				})

				Context("when the instance guid differs", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						instanceKey = models.NewActualLRPInstanceKey("another-instance-guid", cellID)
					})

					It("returns an error", func() {
						Expect(claimErr).To(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
					})

					It("does not alter the existing actual", func() {
						lrpGroupInBBS, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, index)
						Expect(err).NotTo(HaveOccurred())

						Expect(lrpGroupInBBS.Instance.InstanceGuid).To(Equal(instanceGuid))
					})
				})
			})

			Context("when there is a placement error", func() {
				BeforeEach(func() {
					lrpKey = createdLRP.ActualLRPKey
					instanceKey = models.NewActualLRPInstanceKey("some-instance-guid", cellID)

					err := lrpBBS.FailActualLRP(logger, lrpKey, "insufficient resources")
					Expect(err).NotTo(HaveOccurred())
				})

				It("should clear placement error", func() {
					createdLRP, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, index)
					Expect(err).NotTo(HaveOccurred())
					Expect(createdLRP.Instance.PlacementError).To(BeEmpty())
				})
			})
		})

		Context("when the actual LRP does not exist", func() {
			BeforeEach(func() {
				lrpKey = models.NewActualLRPKey("process-guid", 1, "domain")
				instanceKey = models.NewActualLRPInstanceKey("instance-guid", cellID)
			})

			It("cannot claim the LRP", func() {
				Expect(claimErr).To(Equal(bbserrors.ErrActualLRPCannotBeClaimed))
			})

			It("does not create an actual LRP", func() {
				_, err := etcdClient.ListRecursively(shared.ActualLRPSchemaRoot)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("StartActualLRP", func() {
		var startErr error
		var lrpKey models.ActualLRPKey
		var instanceKey models.ActualLRPInstanceKey
		var netInfo models.ActualLRPNetInfo

		JustBeforeEach(func() {
			startErr = lrpBBS.StartActualLRP(logger, lrpKey, instanceKey, netInfo)
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
					RootFS:      "some:rootfs",
					Instances:   index + 1,
					Action: &models.RunAction{
						Path: "true",
						User: "me",
					},
				}

				err := lrpBBS.DesireLRP(logger, desiredLRP)
				Expect(err).NotTo(HaveOccurred())

				lrpGroup, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, index)
				Expect(err).NotTo(HaveOccurred())
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
					Expect(startErr).To(ContainElement(models.ErrInvalidField{"instance_guid"}))
				})

				It("does not modify the persisted actual LRP", func() {
					lrpGroupInBBS, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, index)
					Expect(err).NotTo(HaveOccurred())

					Expect(lrpGroupInBBS.Instance.State).To(Equal(models.ActualLRPStateUnclaimed))
				})

				It("logs the error", func() {
					Expect(logger.TestSink.LogMessages()).To(ContainElement("test.start-actual-lrp.failed-to-marshal-actual-lrp"))
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
					Expect(startErr).To(Equal(bbserrors.ErrActualLRPCannotBeStarted))
				})

				It("does not modify the persisted actual LRP", func() {
					lrpGroupInBBS, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, index)
					Expect(err).NotTo(HaveOccurred())

					Expect(lrpGroupInBBS.Instance.State).To(Equal(models.ActualLRPStateUnclaimed))
				})
			})

			Context("when the existing ActualLRP is Unclaimed", func() {
				BeforeEach(func() {
					lrpKey = createdLRP.ActualLRPKey
					instanceKey = models.NewActualLRPInstanceKey("some-instance-guid", cellID)
					netInfo = models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{{ContainerPort: 1234, HostPort: 5678}})
				})

				It("does not error", func() {
					Expect(startErr).NotTo(HaveOccurred())
				})

				It("starts the actual LRP", func() {
					lrpGroupInBBS, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, index)
					Expect(err).NotTo(HaveOccurred())

					Expect(lrpGroupInBBS.Instance.State).To(Equal(models.ActualLRPStateRunning))
				})

				Context("when there is a placement error", func() {
					BeforeEach(func() {
						err := lrpBBS.FailActualLRP(logger, lrpKey, "found no compatible cells")
						Expect(err).NotTo(HaveOccurred())
					})

					It("should clear placement error", func() {
						createdLRP, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, index)
						Expect(err).NotTo(HaveOccurred())
						Expect(createdLRP.Instance.PlacementError).To(BeEmpty())
					})
				})
			})

			Context("when the existing ActualLRP is Claimed", func() {
				var instanceGuid string

				BeforeEach(func() {
					instanceGuid = "some-instance-guid"
					err := lrpBBS.ClaimActualLRP(
						logger,
						createdLRP.ActualLRPKey,
						models.NewActualLRPInstanceKey(instanceGuid, cellID),
					)
					Expect(err).NotTo(HaveOccurred())
				})

				Context("with the same cell and instance guid", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						instanceKey = models.NewActualLRPInstanceKey(instanceGuid, cellID)
						netInfo = models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{{ContainerPort: 1234, HostPort: 5678}})
					})

					It("does not return an error", func() {
						Expect(startErr).NotTo(HaveOccurred())
					})

					It("promotes the persisted LRP to RUNNING", func() {
						lrpGroupInBBS, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, index)
						Expect(err).NotTo(HaveOccurred())

						Expect(lrpGroupInBBS.Instance.State).To(Equal(models.ActualLRPStateRunning))
					})
				})

				Context("with a different cell", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						instanceKey = models.NewActualLRPInstanceKey(instanceGuid, "another-cell-id")
						netInfo = models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{{ContainerPort: 1234, HostPort: 5678}})
					})

					It("does not return an error", func() {
						Expect(startErr).NotTo(HaveOccurred())
					})

					It("promotes the persisted LRP to RUNNING", func() {
						lrpGroupInBBS, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, index)
						Expect(err).NotTo(HaveOccurred())

						Expect(lrpGroupInBBS.Instance.State).To(Equal(models.ActualLRPStateRunning))
					})
				})

				Context("when the instance guid differs", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						instanceKey = models.NewActualLRPInstanceKey("another-instance-guid", cellID)
						netInfo = models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{{ContainerPort: 1234, HostPort: 5678}})
					})

					It("does not return an error", func() {
						Expect(startErr).NotTo(HaveOccurred())
					})

					It("promotes the persisted LRP to RUNNING", func() {
						lrpGroupInBBS, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, index)
						Expect(err).NotTo(HaveOccurred())

						Expect(lrpGroupInBBS.Instance.State).To(Equal(models.ActualLRPStateRunning))
					})
				})

			})

			Context("when the existing ActualLRP is Running", func() {
				var instanceGuid string

				BeforeEach(func() {
					instanceGuid = "some-instance-guid"
					err := lrpBBS.StartActualLRP(
						logger,
						createdLRP.ActualLRPKey,
						models.NewActualLRPInstanceKey(instanceGuid, cellID),
						models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{{ContainerPort: 1234, HostPort: 5678}}),
					)
					Expect(err).NotTo(HaveOccurred())
				})

				Context("with the same cell and instance guid", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						instanceKey = models.NewActualLRPInstanceKey(instanceGuid, cellID)
						netInfo = models.NewActualLRPNetInfo("5.6.7.8", []models.PortMapping{{ContainerPort: 4567, HostPort: 4321}})
					})

					It("does not return an error", func() {
						Expect(startErr).NotTo(HaveOccurred())
					})

					It("does not alter the state of the persisted LRP", func() {
						lrpGroupInBBS, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, index)
						Expect(err).NotTo(HaveOccurred())

						Expect(lrpGroupInBBS.Instance.State).To(Equal(models.ActualLRPStateRunning))
					})

					It("updates the net info", func() {
						lrpGroupInBBS, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, index)
						Expect(err).NotTo(HaveOccurred())

						Expect(lrpGroupInBBS.Instance.ActualLRPNetInfo).To(Equal(netInfo))
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
							lrpGroupInBBS, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, index)
							Expect(err).NotTo(HaveOccurred())

							Expect(lrpGroupInBBS.Instance.Since).To(Equal(previousTime))
						})
					})
				})

				Context("with a different cell", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						instanceKey = models.NewActualLRPInstanceKey(instanceGuid, "another-cell-id")
					})

					It("returns an error", func() {
						Expect(startErr).To(Equal(bbserrors.ErrActualLRPCannotBeStarted))
					})

					It("does not alter the existing LRP", func() {
						lrpGroupInBBS, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, index)
						Expect(err).NotTo(HaveOccurred())

						Expect(lrpGroupInBBS.Instance.CellID).To(Equal(cellID))
					})
				})

				Context("when the instance guid differs", func() {
					BeforeEach(func() {
						lrpKey = createdLRP.ActualLRPKey
						instanceKey = models.NewActualLRPInstanceKey("another-instance-guid", cellID)
					})

					It("returns an error", func() {
						Expect(startErr).To(Equal(bbserrors.ErrActualLRPCannotBeStarted))
					})

					It("does not alter the existing actual", func() {
						lrpGroupInBBS, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, index)
						Expect(err).NotTo(HaveOccurred())

						Expect(lrpGroupInBBS.Instance.InstanceGuid).To(Equal(instanceGuid))
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
				Expect(startErr).NotTo(HaveOccurred())
			})

			It("sets the State", func() {
				lrpGroup, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, "process-guid", 1)
				Expect(err).NotTo(HaveOccurred())

				Expect(lrpGroup.Instance.State).To(Equal(models.ActualLRPStateRunning))
			})

			It("sets the ModificationTag", func() {
				lrpGroup, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, "process-guid", 1)
				Expect(err).NotTo(HaveOccurred())

				Expect(lrpGroup.Instance.ModificationTag.Epoch).NotTo(BeEmpty())
				Expect(lrpGroup.Instance.ModificationTag.Index).To(BeEquivalentTo(0))
			})
		})
	})

	Describe("RemoveActualLRP", func() {
		BeforeEach(func() {
			netInfo := models.NewActualLRPNetInfo("127.0.0.3", []models.PortMapping{{9090, 90}})
			err := lrpBBS.StartActualLRP(logger, actualLRPKey, instanceKey, netInfo)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when the LRP matches", func() {
			It("removes the LRP", func() {
				err := lrpBBS.RemoveActualLRP(logger, actualLRPKey, instanceKey)
				Expect(err).NotTo(HaveOccurred())

				_, err = etcdClient.Get(shared.ActualLRPSchemaPath(actualLRPKey.ProcessGuid, actualLRPKey.Index))
				Expect(err).To(MatchError(storeadapter.ErrorKeyNotFound))
			})
		})

		Context("when the LRP differs from the one in the store", func() {
			It("does not delete the LRP", func() {
				instanceKey.InstanceGuid = "another-instance-guid"
				err := lrpBBS.RemoveActualLRP(logger, actualLRPKey, instanceKey)
				Expect(err).To(Equal(bbserrors.ErrStoreComparisonFailed))
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
					RootFS:      "some:rootfs",
					Instances:   2,
					Action: &models.RunAction{
						Path: "true",
						User: "me",
					},
				}

				err := lrpBBS.DesireLRP(logger, desiredLRP)
				Expect(err).NotTo(HaveOccurred())
			})

			It("deletes the LRP", func() {
				lrpGroupInBBS, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, index)
				Expect(err).NotTo(HaveOccurred())

				lrpBBS.RetireActualLRPs(logger, []models.ActualLRPKey{lrpGroupInBBS.Instance.ActualLRPKey})

				_, err = lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, index)
				Expect(err).To(Equal(bbserrors.ErrStoreResourceNotFound))
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
				testHelper.SetRawActualLRP(actual)
			})

			JustBeforeEach(func() {
				lrpBBS.RetireActualLRPs(logger, []models.ActualLRPKey{actual.ActualLRPKey})
			})

			It("should remove the actual", func() {
				_, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, actual.ProcessGuid, actual.Index)
				Expect(err).To(Equal(bbserrors.ErrStoreResourceNotFound))
			})

			It("should not log a failure", func() {
				Expect(logger).NotTo(gbytes.Say("fail"))
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
					RootFS:      "some:rootfs",
					Instances:   2,
					Action: &models.RunAction{
						Path: "true",
						User: "me",
					},
				}

				errDesire := lrpBBS.DesireLRP(logger, desiredLRP)
				Expect(errDesire).NotTo(HaveOccurred())

				lrpGroups, err := lrpBBS.ActualLRPGroupsByProcessGuid(logger, desiredLRP.ProcessGuid)
				Expect(err).NotTo(HaveOccurred())

				Expect(lrpGroups).To(HaveKey(0))
				err = lrpBBS.ClaimActualLRP(
					logger,
					lrpGroups[0].Instance.ActualLRPKey,
					models.NewActualLRPInstanceKey("some-instance-guid-1", cellID),
				)
				Expect(err).NotTo(HaveOccurred())

				Expect(lrpGroups).To(HaveKey(1))
				err = lrpBBS.ClaimActualLRP(
					logger,
					lrpGroups[1].Instance.ActualLRPKey,
					models.NewActualLRPInstanceKey("some-instance-guid-2", cellID),
				)
				Expect(err).NotTo(HaveOccurred())

				blockStopInstanceChan = make(chan struct{})

				fakeCellClient.StopLRPInstanceStub = func(string, models.ActualLRPKey, models.ActualLRPInstanceKey) error {
					<-blockStopInstanceChan
					return nil
				}
			})

			JustBeforeEach(func() {
				lrpGroup1, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, 0)
				Expect(err).NotTo(HaveOccurred())
				claimedLRP1 = *lrpGroup1.Instance

				lrpGroup2, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, 1)
				Expect(err).NotTo(HaveOccurred())
				claimedLRP2 = *lrpGroup2.Instance

				doneRetiring = make(chan struct{})

				go func(lrpBBS *lrp_bbs.LRPBBS, lrp1, lrp2 models.ActualLRP, doneRetiring chan struct{}, logger lager.Logger) {
					lrpBBS.RetireActualLRPs(logger, []models.ActualLRPKey{lrp1.ActualLRPKey, lrp2.ActualLRPKey})
					close(doneRetiring)
				}(lrpBBS, claimedLRP1, claimedLRP2, doneRetiring, logger)
			})

			Context("when the cell", func() {
				Context("is present", func() {
					BeforeEach(func() {
						cellPresence = models.NewCellPresence(
							cellID,
							"cell.example.com",
							"the-zone",
							models.NewCellCapacity(128, 1024, 6),
							[]string{},
							[]string{},
						)
						testHelper.RegisterCell(cellPresence)
					})

					It("stops the LRPs in parallel", func() {
						Eventually(fakeCellClient.StopLRPInstanceCallCount).Should(Equal(2))

						addr1, key1, cnrKey1 := fakeCellClient.StopLRPInstanceArgsForCall(0)
						addr2, key2, cnrKey2 := fakeCellClient.StopLRPInstanceArgsForCall(1)

						Expect(addr1).To(Equal(cellPresence.RepAddress))
						Expect(addr2).To(Equal(cellPresence.RepAddress))

						Expect([]models.ActualLRPKey{key1, key2}).To(ConsistOf(
							claimedLRP1.ActualLRPKey,
							claimedLRP2.ActualLRPKey,
						))

						Expect([]models.ActualLRPInstanceKey{cnrKey1, cnrKey2}).To(ConsistOf(
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
							Expect(logger.LogMessages()).To(ContainElement("test.retire-actual-lrps.failed-to-retire"))
						})

						It("retries", func() {
							Eventually(doneRetiring).Should(BeClosed())
							Expect(fakeCellClient.StopLRPInstanceCallCount()).To(Equal(2 * lrp_bbs.RetireActualLRPRetryAttempts))
						})

						It("logs each retry", func() {
							Eventually(doneRetiring).Should(BeClosed())
							Expect(logger.LogMessages()).To(ContainElement("test.retire-actual-lrps.retrying-failed-retire-of-actual-lrp"))
						})
					})
				})

				Context("is not present", func() {
					It("removes the LRPs", func() {
						Eventually(doneRetiring).Should(BeClosed())

						_, err := etcdClient.Get(shared.ActualLRPSchemaPath(actualLRPKey.ProcessGuid, actualLRPKey.Index))
						Expect(err).To(Equal(storeadapter.ErrorKeyNotFound))
					})
				})

				Context("cannot be retrieved", func() {
					BeforeEach(func() {
						_, err := consulSession.SetPresence(shared.CellSchemaPath(cellID), []byte("abcd"))
						Expect(err).NotTo(HaveOccurred())
					})

					JustBeforeEach(func() {
						Eventually(doneRetiring).Should(BeClosed())
					})

					It("does not stop the instances", func() {
						Expect(fakeCellClient.StopLRPInstanceCallCount()).To(Equal(0))
					})

					It("logs the error", func() {
						Expect(logger.TestSink.LogMessages()).To(ContainElement("test.retire-actual-lrps.failed-to-retire"))
					})
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
					RootFS:      "some:rootfs",
					Instances:   3,
					Action: &models.RunAction{
						Path: "true",
						User: "me",
					},
				}

				errDesire := lrpBBS.DesireLRP(logger, desiredLRP)
				Expect(errDesire).NotTo(HaveOccurred())

				createdLRPGroup, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, index)
				Expect(err).NotTo(HaveOccurred())

				actualLRPKey = createdLRPGroup.Instance.ActualLRPKey
				instanceKey = models.NewActualLRPInstanceKey(instanceGuid, cellID)
			})

			Context("in unclaimed state", func() {
				BeforeEach(func() {
					err := lrpBBS.FailActualLRP(logger, actualLRPKey, placementError)
					Expect(err).NotTo(HaveOccurred())
				})

				It("sets the placement error", func() {
					failedActualLRPGroup, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, index)
					Expect(err).NotTo(HaveOccurred())
					Expect(failedActualLRPGroup.Instance.PlacementError).To(Equal(placementError))
				})

				It("updates the ModificationIndex", func() {
					failedActualLRPGroup, err := lrpBBS.LegacyActualLRPGroupByProcessGuidAndIndex(logger, processGuid, index)
					Expect(err).NotTo(HaveOccurred())
					Expect(failedActualLRPGroup.Instance.ModificationTag.Index).To(Equal(createdLRP.ModificationTag.Index + 1))
				})
			})

			Context("not in unclaimed state", func() {
				BeforeEach(func() {
					claimErr := lrpBBS.ClaimActualLRP(logger, actualLRPKey, instanceKey)
					Expect(claimErr).NotTo(HaveOccurred())
				})

				It("returns an error", func() {
					err := lrpBBS.FailActualLRP(logger, actualLRPKey, placementError)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("when lrp does not exist", func() {
			It("returns an error", func() {
				actualLRPKey := models.NewActualLRPKey("non-existent-process-guid", index, "tests")
				err := lrpBBS.FailActualLRP(logger, actualLRPKey, placementError)
				Expect(err).To(Equal(bbserrors.ErrActualLRPCannotBeFailed))
			})
		})
	})
})
