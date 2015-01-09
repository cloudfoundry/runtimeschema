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
			createsAndUpdates <-chan models.DesiredLRP
			deletes           <-chan models.DesiredLRP
			errors            <-chan error
			lrp               models.DesiredLRP
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
			createsAndUpdates, deletes, errors = bbs.WatchForDesiredLRPChanges(logger)
		})

		It("sends an event down the pipe for creates", func() {
			err := bbs.DesireLRP(logger, lrp)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(createsAndUpdates).Should(Receive(Equal(lrp)))
		})

		It("sends an event down the pipe for updates", func() {
			err := bbs.DesireLRP(logger, lrp)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(createsAndUpdates).Should(Receive())

			changedLRP := newLRP()
			changedLRP.Instances++

			err = bbs.DesireLRP(logger, changedLRP)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(createsAndUpdates).Should(Receive(Equal(changedLRP)))
		})

		It("sends an event down the pipe for deletes", func() {
			err := bbs.DesireLRP(logger, lrp)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(createsAndUpdates).Should(Receive())

			err = etcdClient.Delete(shared.DesiredLRPSchemaPath(lrp))
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(deletes).Should(Receive(Equal(lrp)))
		})
	})

	Describe("WatchForActualLRPChanges", func() {
		var (
			createsAndUpdates <-chan models.ActualLRP
			deletes           <-chan models.ActualLRP
			errors            <-chan error

			lrpProcessGuid string
			desiredLRP     models.DesiredLRP

			lrpIndex int

			lrpCellId string
		)

		BeforeEach(func() {
			createsAndUpdates, deletes, errors = bbs.WatchForActualLRPChanges(logger)

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
			Eventually(createsAndUpdates).Should(Receive(&actualLRP))

			Ω(actualLRP.ProcessGuid).Should(Equal(lrpProcessGuid))
			Ω(actualLRP.Index).Should(Equal(lrpIndex))
			Ω(actualLRP.State).Should(Equal(models.ActualLRPStateUnclaimed))
		})

		It("sends an event down the pipe for updates", func() {
			err := bbs.CreateActualLRP(desiredLRP, lrpIndex, logger)
			Ω(err).ShouldNot(HaveOccurred())

			lrp, err := bbs.ActualLRPByProcessGuidAndIndex(lrpProcessGuid, lrpIndex)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(createsAndUpdates).Should(Receive())

			containerKey := models.NewActualLRPContainerKey("instance-guid", lrpCellId)
			err = bbs.ClaimActualLRP(lrp.ActualLRPKey, containerKey, logger)
			Ω(err).ShouldNot(HaveOccurred())

			var actualLRP models.ActualLRP
			Eventually(createsAndUpdates).Should(Receive(&actualLRP))

			Ω(actualLRP.ProcessGuid).Should(Equal(lrpProcessGuid))
			Ω(actualLRP.Index).Should(Equal(lrpIndex))
			Ω(actualLRP.State).Should(Equal(models.ActualLRPStateClaimed))
			Ω(actualLRP.ActualLRPContainerKey).Should(Equal(containerKey))
		})

		It("sends an event down the pipe for delete", func() {
			err := bbs.CreateActualLRP(desiredLRP, lrpIndex, logger)
			Ω(err).ShouldNot(HaveOccurred())

			lrp, err := bbs.ActualLRPByProcessGuidAndIndex(lrpProcessGuid, lrpIndex)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(createsAndUpdates).Should(Receive())

			err = bbs.RemoveActualLRP(lrp.ActualLRPKey, lrp.ActualLRPContainerKey, logger)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(deletes).Should(Receive(Equal(lrp)))
		})
	})
})
