package lrp_bbs_test

import (
	"encoding/json"

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
				RootFS:      "some:rootfs",
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

			desiredLRP, err := bbs.DesiredLRPByProcessGuid(lrp.ProcessGuid)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(creates).Should(Receive(Equal(desiredLRP)))
		})

		It("sends an event down the pipe for updates", func() {
			err := bbs.DesireLRP(logger, lrp)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(creates).Should(Receive())

			desiredBeforeUpdate, err := bbs.DesiredLRPByProcessGuid(lrp.ProcessGuid)
			Ω(err).ShouldNot(HaveOccurred())

			lrp.Instances++
			err = bbs.UpdateDesiredLRP(logger, lrp.ProcessGuid, models.DesiredLRPUpdate{
				Instances: &lrp.Instances,
			})
			Ω(err).ShouldNot(HaveOccurred())

			desiredAfterUpdate, err := bbs.DesiredLRPByProcessGuid(lrp.ProcessGuid)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(changes).Should(Receive(Equal(models.DesiredLRPChange{
				Before: desiredBeforeUpdate,
				After:  desiredAfterUpdate,
			})))
		})

		It("sends an event down the pipe for deletes", func() {
			err := bbs.DesireLRP(logger, lrp)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(creates).Should(Receive())

			desired, err := bbs.DesiredLRPByProcessGuid(lrp.ProcessGuid)
			Ω(err).ShouldNot(HaveOccurred())

			err = etcdClient.Delete(shared.DesiredLRPSchemaPath(desired))
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(deletes).Should(Receive(Equal(desired)))
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
		const (
			lrpProcessGuid = "some-process-guid"
			lrpDomain      = "lrp-domain"
			lrpIndex       = 0
			lrpCellId      = "cell-id"
		)

		var (
			creates           chan models.ActualLRP
			createsEvacuating chan bool
			changes           chan models.ActualLRPChange
			changesEvacuating chan bool
			deletes           chan models.ActualLRP
			deletesEvacuating chan bool
			stop              chan<- bool
			errors            <-chan error

			actualLRP models.ActualLRP
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

			actualLRP = models.ActualLRP{
				ActualLRPKey: models.NewActualLRPKey(lrpProcessGuid, lrpIndex, lrpDomain),
				State:        models.ActualLRPStateUnclaimed,
				Since:        clock.Now().UnixNano(),
			}
		})

		It("sends an event down the pipe for create", func() {
			setRawActualLRP(actualLRP)
			Eventually(creates).Should(Receive(Equal(actualLRP)))
			Eventually(createsEvacuating).Should(Receive(Equal(false)))
		})

		It("sends an event down the pipe for updates", func() {
			setRawActualLRP(actualLRP)
			Eventually(creates).Should(Receive())
			Eventually(createsEvacuating).Should(Receive())

			updatedLRP := actualLRP
			updatedLRP.ActualLRPInstanceKey = models.NewActualLRPInstanceKey("instance-guid", lrpCellId)
			updatedLRP.State = models.ActualLRPStateClaimed
			setRawActualLRP(updatedLRP)

			var actualLRPChange models.ActualLRPChange
			Eventually(changes).Should(Receive(&actualLRPChange))
			Eventually(changesEvacuating).Should(Receive(Equal(false)))

			before, after := actualLRPChange.Before, actualLRPChange.After
			Ω(before).Should(Equal(actualLRP))
			Ω(after).Should(Equal(updatedLRP))
		})

		It("sends an event down the pipe for delete", func() {
			setRawActualLRP(actualLRP)
			Eventually(creates).Should(Receive())
			Eventually(createsEvacuating).Should(Receive())

			deleteActualLRP(actualLRP.ActualLRPKey)

			Eventually(deletes).Should(Receive(Equal(actualLRP)))
			Eventually(deletesEvacuating).Should(Receive(Equal(false)))
		})

		It("ignores delete events for directories", func() {
			setRawActualLRP(actualLRP)
			Eventually(creates).Should(Receive())
			Eventually(createsEvacuating).Should(Receive())

			deleteActualLRP(actualLRP.ActualLRPKey)

			Eventually(deletes).Should(Receive(Equal(actualLRP)))
			Eventually(deletesEvacuating).Should(Receive(Equal(false)))

			bbs.ConvergeLRPs(logger, servicesBBS.NewCellsLoader())

			Consistently(logger).ShouldNot(Say("failed-to-unmarshal"))
		})

		Context("when the evacuating key changes", func() {
			It("indicates passes the correct evacuating flag on event callbacks", func() {
				key := models.ActualLRPKey{
					Domain:      lrpDomain,
					ProcessGuid: lrpProcessGuid,
					Index:       lrpIndex,
				}
				instanceKey := models.NewActualLRPInstanceKey("instance-guid", "cell-id")
				netInfo := models.ActualLRPNetInfo{Address: "1.1.1.1"}
				evacuatedLRP := models.ActualLRP{
					ActualLRPKey:         key,
					ActualLRPInstanceKey: instanceKey,
					ActualLRPNetInfo:     netInfo,
					State:                models.ActualLRPStateRunning,
					Since:                clock.Now().UnixNano(),
				}

				setRawEvacuatingActualLRP(evacuatedLRP, 0)

				Eventually(creates).Should(Receive(Equal(evacuatedLRP)))
				Eventually(createsEvacuating).Should(Receive(Equal(true)))

				updatedLRP := evacuatedLRP
				updatedLRP.ActualLRPNetInfo = models.ActualLRPNetInfo{Address: "2.2.2.2"}
				setRawEvacuatingActualLRP(updatedLRP, 0)

				var actualLRPChange models.ActualLRPChange
				Eventually(changes).Should(Receive(&actualLRPChange))
				Eventually(changesEvacuating).Should(Receive(Equal(true)))

				Ω(actualLRPChange.Before).Should(Equal(evacuatedLRP))
				Ω(actualLRPChange.After).Should(Equal(updatedLRP))

				deleteEvacuatingActualLRP(key)

				Eventually(deletes).Should(Receive(Equal(updatedLRP)))
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
