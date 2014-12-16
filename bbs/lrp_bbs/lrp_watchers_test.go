package lrp_bbs_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LrpWatchers", func() {
	Describe("WatchForDesiredLRPChanges", func() {
		var (
			events <-chan models.DesiredLRPChange
			stop   chan<- bool
			errors <-chan error
			lrp    models.DesiredLRP
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
			events, stop, errors = bbs.WatchForDesiredLRPChanges()
		})

		AfterEach(func() {
			stop <- true
		})

		It("sends an event down the pipe for creates", func() {
			err := bbs.DesireLRP(lrp)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(events).Should(Receive(Equal(models.DesiredLRPChange{
				Before: nil,
				After:  &lrp,
			})))
		})

		It("sends an event down the pipe for updates", func() {
			err := bbs.DesireLRP(lrp)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(events).Should(Receive())

			changedLRP := newLRP()
			changedLRP.Instances++

			err = bbs.ChangeDesiredLRP(models.DesiredLRPChange{
				Before: &lrp,
				After:  &changedLRP,
			})
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(events).Should(Receive(Equal(models.DesiredLRPChange{
				Before: &lrp,
				After:  &changedLRP,
			})))
		})

		It("sends an event down the pipe for deletes", func() {
			err := bbs.DesireLRP(lrp)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(events).Should(Receive())

			err = etcdClient.Delete(shared.DesiredLRPSchemaPath(lrp))
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(events).Should(Receive(Equal(models.DesiredLRPChange{
				Before: &lrp,
				After:  nil,
			})))
		})
	})

	Describe("WatchForActualLRPChanges", func() {
		var (
			events <-chan models.ActualLRPChange
			stop   chan<- bool
			errors <-chan error

			lrpProcessGuid string
			lrpDomain      string
			desiredLRP     models.DesiredLRP

			lrpIndex int

			lrpCellId string

			logger *lagertest.TestLogger
		)

		BeforeEach(func() {
			events, stop, errors = bbs.WatchForActualLRPChanges()

			lrpProcessGuid = "some-process-guid"
			lrpDomain = "lrp-domain"
			desiredLRP = models.DesiredLRP{
				ProcessGuid: "some-process-guid",
				Domain:      "domain",
				Instances:   1,
			}

			lrpIndex = 0

			lrpCellId = "cell-id"

			logger = lagertest.NewTestLogger("test")
		})

		AfterEach(func() {
			stop <- true
		})

		It("sends an event down the pipe for creates", func() {
			_, err := bbs.CreateActualLRP(desiredLRP, lrpIndex, logger)
			Ω(err).ShouldNot(HaveOccurred())

			var change models.ActualLRPChange
			Eventually(events).Should(Receive(&change))

			before := change.Before
			after := change.After

			Ω(before).Should(BeNil())

			Ω(after.ProcessGuid).Should(Equal(lrpProcessGuid))
			Ω(after.Index).Should(Equal(lrpIndex))
			Ω(after.State).Should(Equal(models.ActualLRPStateUnclaimed))
		})

		It("sends an event down the pipe for updates", func() {
			lrp, err := bbs.CreateActualLRP(desiredLRP, lrpIndex, logger)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(events).Should(Receive())

			lrp.CellID = lrpCellId
			_, err = bbs.ClaimActualLRP(*lrp)
			Ω(err).ShouldNot(HaveOccurred())

			var change models.ActualLRPChange
			Eventually(events).Should(Receive(&change))

			before := change.Before
			after := change.After

			Ω(before.ProcessGuid).Should(Equal(after.ProcessGuid))
			Ω(before.InstanceGuid).Should(Equal(after.InstanceGuid))
			Ω(before.Index).Should(Equal(after.Index))

			Ω(before.State).Should(Equal(models.ActualLRPStateUnclaimed))
			Ω(after.State).Should(Equal(models.ActualLRPStateClaimed))

			Ω(after.CellID).Should(Equal(lrpCellId))
		})

		It("sends an event down the pipe for delete", func() {
			lrp, err := bbs.CreateActualLRP(desiredLRP, lrpIndex, logger)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(events).Should(Receive())

			err = bbs.RemoveActualLRP(*lrp)
			Ω(err).ShouldNot(HaveOccurred())

			var change models.ActualLRPChange
			Eventually(events).Should(Receive(&change))

			before := change.Before
			after := change.After

			Ω(after).Should(BeNil())

			Ω(before.ProcessGuid).Should(Equal(lrpProcessGuid))
			Ω(before.Index).Should(Equal(lrpIndex))
			Ω(before.State).Should(Equal(models.ActualLRPStateUnclaimed))
		})
	})
})
