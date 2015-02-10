package lrp_bbs_test

import (
	"encoding/json"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
)

var _ = Describe("Watchers", func() {
	Describe("WatchForDesiredLRPChanges", func() {
		var (
			creates chan models.DesiredLRP
			changes chan models.DesiredLRPChange
			deletes chan models.DesiredLRP
			stop    chan<- bool
			errors  <-chan error
			lrp     models.DesiredLRP
		)

		newLRP := func() models.DesiredLRP {
			rawMessage := json.RawMessage([]byte(`{"port":8080,"hosts":["route-1","route-2"]}`))
			return models.DesiredLRP{
				Domain:      "tests",
				ProcessGuid: "some-process-guid",
				Instances:   5,
				Stack:       "some-stack",
				MemoryMB:    1024,
				DiskMB:      512,
				Routes: map[string]*json.RawMessage{
					"router": &rawMessage,
				},
				Action: &models.DownloadAction{
					From: "http://example.com",
					To:   "/tmp/internet",
				},
			}
		}

		BeforeEach(func() {
			lrp = newLRP()

			createsCh := make(chan models.DesiredLRP)
			creates = createsCh
			changesCh := make(chan models.DesiredLRPChange)
			changes = changesCh
			deletesCh := make(chan models.DesiredLRP)
			deletes = deletesCh

			stop, errors = bbs.WatchForDesiredLRPChanges(logger,
				func(created models.DesiredLRP) { createsCh <- created },
				func(changed models.DesiredLRPChange) { changesCh <- changed },
				func(deleted models.DesiredLRP) { deletesCh <- deleted },
			)
		})

		It("sends an event down the pipe for creates", func() {
			err := bbs.DesireLRP(logger, lrp)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(creates).Should(Receive(Equal(lrp)))
		})

		It("sends an event down the pipe for updates", func() {
			err := bbs.DesireLRP(logger, lrp)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(creates).Should(Receive())

			changedLRP := newLRP()
			changedLRP.Instances++

			err = bbs.UpdateDesiredLRP(logger, lrp.ProcessGuid, models.DesiredLRPUpdate{
				Instances: &changedLRP.Instances,
			})
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(changes).Should(Receive(Equal(models.DesiredLRPChange{
				Before: lrp,
				After:  changedLRP,
			})))
		})

		It("sends an event down the pipe for deletes", func() {
			err := bbs.DesireLRP(logger, lrp)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(creates).Should(Receive())

			err = etcdClient.Delete(shared.DesiredLRPSchemaPath(lrp))
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(deletes).Should(Receive(Equal(lrp)))
		})

		Context("when the caller closes the stop channel", func() {
			It("closes the error channel without error", func() {
				close(stop)
				Consistently(errors).ShouldNot(Receive())
				Eventually(errors).Should(BeClosed())
			})
		})
	})

	Describe("WatchForActualLRPChanges", func() {
		var (
			creates           chan models.ActualLRP
			createsEvacuating chan bool
			changes           chan models.ActualLRPChange
			changesEvacuating chan bool
			deletes           chan models.ActualLRP
			deletesEvacuating chan bool
			stop              chan<- bool
			errors            <-chan error

			lrpProcessGuid string
			desiredLRP     models.DesiredLRP

			lrpIndex int

			lrpCellId string
		)

		BeforeEach(func() {
			createsCh := make(chan models.ActualLRP)
			creates = createsCh
			createsEvacuatingCh := make(chan bool)
			createsEvacuating = createsEvacuatingCh

			changesCh := make(chan models.ActualLRPChange)
			changes = changesCh
			changesEvacuatingCh := make(chan bool)
			changesEvacuating = changesEvacuatingCh

			deletesCh := make(chan models.ActualLRP)
			deletes = deletesCh
			deletesEvacuatingCh := make(chan bool)
			deletesEvacuating = deletesEvacuatingCh

			stop, errors = bbs.WatchForActualLRPChanges(logger,
				func(created models.ActualLRP, evacuating bool) {
					createsCh <- created
					createsEvacuatingCh <- evacuating
				},
				func(changed models.ActualLRPChange, evacuating bool) {
					changesCh <- changed
					changesEvacuatingCh <- evacuating
				},
				func(deleted models.ActualLRP, evacuating bool) {
					deletesCh <- deleted
					deletesEvacuatingCh <- evacuating
				},
			)

			lrpProcessGuid = "some-process-guid"
			desiredLRP = models.DesiredLRP{
				ProcessGuid: lrpProcessGuid,
				Domain:      "lrp-domain",
				Stack:       "some-stack",
				Instances:   1,
				Action:      &models.RunAction{Path: "/bin/true"},
			}

			lrpIndex = 0
			lrpCellId = "cell-id"
		})

		It("sends an event down the pipe for create", func() {
			err := bbs.CreateActualLRP(logger, desiredLRP, lrpIndex)
			Ω(err).ShouldNot(HaveOccurred())

			var actualLRP models.ActualLRP
			Eventually(creates).Should(Receive(&actualLRP))
			Eventually(createsEvacuating).Should(Receive(Equal(false)))

			Ω(actualLRP.ProcessGuid).Should(Equal(lrpProcessGuid))
			Ω(actualLRP.Index).Should(Equal(lrpIndex))
			Ω(actualLRP.State).Should(Equal(models.ActualLRPStateUnclaimed))
		})

		It("sends an event down the pipe for updates", func() {
			err := bbs.CreateActualLRP(logger, desiredLRP, lrpIndex)
			Ω(err).ShouldNot(HaveOccurred())

			lrp, err := bbs.ActualLRPByProcessGuidAndIndex(lrpProcessGuid, lrpIndex)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(creates).Should(Receive())
			Eventually(createsEvacuating).Should(Receive())

			containerKey := models.NewActualLRPContainerKey("instance-guid", lrpCellId)
			err = bbs.ClaimActualLRP(logger, lrp.ActualLRPKey, containerKey)
			Ω(err).ShouldNot(HaveOccurred())

			var actualLRPChange models.ActualLRPChange
			Eventually(changes).Should(Receive(&actualLRPChange))
			Eventually(changesEvacuating).Should(Receive(Equal(false)))

			before, after := actualLRPChange.Before, actualLRPChange.After
			Ω(before).Should(Equal(lrp))
			Ω(after.ProcessGuid).Should(Equal(lrpProcessGuid))
			Ω(after.Index).Should(Equal(lrpIndex))
			Ω(after.State).Should(Equal(models.ActualLRPStateClaimed))
			Ω(after.ActualLRPContainerKey).Should(Equal(containerKey))
		})

		It("sends an event down the pipe for delete", func() {
			err := bbs.CreateActualLRP(logger, desiredLRP, lrpIndex)
			Ω(err).ShouldNot(HaveOccurred())

			lrp, err := bbs.ActualLRPByProcessGuidAndIndex(lrpProcessGuid, lrpIndex)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(creates).Should(Receive())
			Eventually(createsEvacuating).Should(Receive())

			err = bbs.RemoveActualLRP(logger, lrp.ActualLRPKey, lrp.ActualLRPContainerKey)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(deletes).Should(Receive(Equal(lrp)))
			Eventually(deletesEvacuating).Should(Receive(Equal(false)))
		})

		It("ignores delete events for directories", func() {
			err := bbs.CreateActualLRP(logger, desiredLRP, lrpIndex)
			Ω(err).ShouldNot(HaveOccurred())

			lrp, err := bbs.ActualLRPByProcessGuidAndIndex(lrpProcessGuid, lrpIndex)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(creates).Should(Receive())
			Eventually(createsEvacuating).Should(Receive())

			err = bbs.RemoveActualLRP(logger, lrp.ActualLRPKey, lrp.ActualLRPContainerKey)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(deletes).Should(Receive(Equal(lrp)))
			Eventually(deletesEvacuating).Should(Receive(Equal(false)))

			bbs.ConvergeLRPs(logger)

			Consistently(logger).ShouldNot(Say("failed-to-unmarshal"))
		})

		Context("when an actual LRP begins evacuation", func() {
			var key models.ActualLRPKey

			BeforeEach(func() {
				err := bbs.DesireLRP(logger, desiredLRP)
				Ω(err).ShouldNot(HaveOccurred())

				key = models.ActualLRPKey{
					Domain:      desiredLRP.Domain,
					ProcessGuid: lrpProcessGuid,
					Index:       lrpIndex,
				}
			})

			It("indicates passes the correct evacuating flag on event callbacks", func() {
				lrp, err := bbs.ActualLRPByProcessGuidAndIndex(lrpProcessGuid, lrpIndex)
				Ω(err).ShouldNot(HaveOccurred())

				Eventually(creates).Should(Receive(Equal(lrp)))
				Eventually(createsEvacuating).Should(Receive(Equal(false)))

				By("evacuating")
				containerKey := models.NewActualLRPContainerKey("instance-guid", "cell-id")
				netInfo := models.ActualLRPNetInfo{Address: "1.1.1.1"}
				err = bbs.EvacuateRunningActualLRP(logger, key, containerKey, netInfo, 0)
				Ω(err).ShouldNot(HaveOccurred())

				lrpGroup, err := bbs.ActualLRPGroupByProcessGuidAndIndex(lrpProcessGuid, lrpIndex)
				Ω(err).ShouldNot(HaveOccurred())

				evacuatedLRP := *lrpGroup.Evacuating

				Eventually(creates).Should(Receive(Equal(evacuatedLRP)))
				Eventually(createsEvacuating).Should(Receive(Equal(true)))

				By("restarting on new cell")
				lrp, err = bbs.ActualLRPByProcessGuidAndIndex(lrpProcessGuid, lrpIndex)
				Ω(err).ShouldNot(HaveOccurred())

				newContainerKey := models.NewActualLRPContainerKey("instance-guid-2", "cell-id-2")
				netInfo = models.ActualLRPNetInfo{Address: "2.2.2.2"}
				err = bbs.StartActualLRP(logger, key, newContainerKey, netInfo)
				Ω(err).ShouldNot(HaveOccurred())

				var actualLRPChange models.ActualLRPChange
				Eventually(changes).Should(Receive(&actualLRPChange))
				Eventually(changesEvacuating).Should(Receive(Equal(false)))

				Ω(actualLRPChange.Before).Should(Equal(lrp))
				lrp = actualLRPChange.After

				By("evacuating the new cell")
				err = bbs.EvacuateRunningActualLRP(logger, key, newContainerKey, netInfo, 0)
				Ω(err).Should(Equal(bbserrors.ErrServiceUnavailable))

				Eventually(changes).Should(Receive(&actualLRPChange))
				Eventually(changesEvacuating).Should(Receive(Equal(true)))
				Ω(actualLRPChange.Before).Should(Equal(evacuatedLRP))
				evacuatedLRP = actualLRPChange.After

				Eventually(changes).Should(Receive(&actualLRPChange))
				Eventually(changesEvacuating).Should(Receive(Equal(false)))
				Ω(actualLRPChange.Before).Should(Equal(evacuatedLRP))

				By("completion")
				err = bbs.RemoveEvacuatingActualLRP(logger, key, newContainerKey)
				Ω(err).ShouldNot(HaveOccurred())

				Eventually(deletes).Should(Receive(Equal(evacuatedLRP)))
				Eventually(deletesEvacuating).Should(Receive(Equal(true)))
			})
		})

		Context("when the caller closes the stop channel", func() {
			It("closes the error channel without error", func() {
				close(stop)
				Consistently(errors).ShouldNot(Receive())
				Eventually(errors).Should(BeClosed())
			})
		})
	})
})
