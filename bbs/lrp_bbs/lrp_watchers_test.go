package lrp_bbs_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LrpWatchers", func() {
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
			return models.DesiredLRP{
				Domain:      "tests",
				ProcessGuid: "some-process-guid",
				Instances:   5,
				Stack:       "some-stack",
				MemoryMB:    1024,
				DiskMB:      512,
				Routes:      []string{"route-1", "route-2"},
				Action: &models.DownloadAction{
					From: "http://example.com",
					To:   "/tmp/internet",
				},
			}
		}

		BeforeEach(func() {
			lrp = newLRP()

			creates = make(chan models.DesiredLRP)
			changes = make(chan models.DesiredLRPChange)
			deletes = make(chan models.DesiredLRP)

			stop, errors = bbs.WatchForDesiredLRPChanges(logger,
				func(created models.DesiredLRP) { creates <- created },
				func(changed models.DesiredLRPChange) { changes <- changed },
				func(deleted models.DesiredLRP) { deletes <- deleted },
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
			creates chan models.ActualLRP
			changes chan models.ActualLRPChange
			deletes chan models.ActualLRP
			stop    chan<- bool
			errors  <-chan error

			lrpProcessGuid string
			desiredLRP     models.DesiredLRP

			lrpIndex int

			lrpCellId string
		)

		BeforeEach(func() {
			creates = make(chan models.ActualLRP)
			changes = make(chan models.ActualLRPChange)
			deletes = make(chan models.ActualLRP)

			stop, errors = bbs.WatchForActualLRPChanges(logger,
				func(created models.ActualLRP) { creates <- created },
				func(changed models.ActualLRPChange) { changes <- changed },
				func(deleted models.ActualLRP) { deletes <- deleted },
			)

			lrpProcessGuid = "some-process-guid"
			desiredLRP = models.DesiredLRP{
				ProcessGuid: lrpProcessGuid,
				Domain:      "lrp-domain",
				Instances:   1,
			}

			lrpIndex = 0

			lrpCellId = "cell-id"
		})

		It("sends an event down the pipe for creates", func() {
			err := bbs.CreateActualLRP(desiredLRP, lrpIndex, logger)
			Ω(err).ShouldNot(HaveOccurred())

			var actualLRP models.ActualLRP
			Eventually(creates).Should(Receive(&actualLRP))

			Ω(actualLRP.ProcessGuid).Should(Equal(lrpProcessGuid))
			Ω(actualLRP.Index).Should(Equal(lrpIndex))
			Ω(actualLRP.State).Should(Equal(models.ActualLRPStateUnclaimed))
		})

		It("sends an event down the pipe for updates", func() {
			err := bbs.CreateActualLRP(desiredLRP, lrpIndex, logger)
			Ω(err).ShouldNot(HaveOccurred())

			lrp, err := bbs.ActualLRPByProcessGuidAndIndex(lrpProcessGuid, lrpIndex)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(creates).Should(Receive())

			containerKey := models.NewActualLRPContainerKey("instance-guid", lrpCellId)
			err = bbs.ClaimActualLRP(lrp.ActualLRPKey, containerKey, logger)
			Ω(err).ShouldNot(HaveOccurred())

			var actualLRPChange models.ActualLRPChange
			Eventually(changes).Should(Receive(&actualLRPChange))

			before, after := actualLRPChange.Before, actualLRPChange.After
			Ω(before).Should(Equal(lrp))
			Ω(after.ProcessGuid).Should(Equal(lrpProcessGuid))
			Ω(after.Index).Should(Equal(lrpIndex))
			Ω(after.State).Should(Equal(models.ActualLRPStateClaimed))
			Ω(after.ActualLRPContainerKey).Should(Equal(containerKey))
		})

		It("sends an event down the pipe for delete", func() {
			err := bbs.CreateActualLRP(desiredLRP, lrpIndex, logger)
			Ω(err).ShouldNot(HaveOccurred())

			lrp, err := bbs.ActualLRPByProcessGuidAndIndex(lrpProcessGuid, lrpIndex)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(creates).Should(Receive())

			err = bbs.RemoveActualLRP(lrp.ActualLRPKey, lrp.ActualLRPContainerKey, logger)
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
})
