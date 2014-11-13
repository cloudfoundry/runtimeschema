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
				Action: models.ExecutorAction{
					Action: models.DownloadAction{
						From: "http://example.com",
						To:   "/tmp/internet",
					},
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
			events                                         <-chan models.ActualLRPChange
			stop                                           chan<- bool
			errors                                         <-chan error
			lrp                                            models.ActualLRP
			lrpProcessGuid, lrpInstanceGuid, lrpCellId string
			lrpDomain                                      string
			lrpIndex                                       int
		)

		BeforeEach(func() {
			lrpProcessGuid = "some-process-guid"
			lrpInstanceGuid = "some-instance-guid"
			lrpCellId = "cell-id"
			lrpDomain = "lrp-domain"
			lrpIndex = 0

			events, stop, errors = bbs.WatchForActualLRPChanges()

			var err error
			lrp, err = bbs.ReportActualLRPAsStarting(lrpProcessGuid, lrpInstanceGuid, lrpCellId, lrpDomain, lrpIndex)
			Ω(err).ShouldNot(HaveOccurred())
		})

		AfterEach(func() {
			stop <- true
		})

		It("sends an event down the pipe for creates", func() {
			Eventually(events).Should(Receive(Equal(models.ActualLRPChange{
				Before: nil,
				After:  &lrp,
			})))
		})

		It("sends an event down the pipe for updates", func() {
			Eventually(events).Should(Receive())

			changedLRP := lrp
			changedLRP.State = models.ActualLRPStateRunning
			changedLRP.CellID = "cell-id"

			err := bbs.ReportActualLRPAsRunning(changedLRP, "cell-id")
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(events).Should(Receive(Equal(models.ActualLRPChange{
				Before: &lrp,
				After:  &changedLRP,
			})))
		})

		It("sends an event down the pipe for delete", func() {
			Eventually(events).Should(Receive())

			err := bbs.RemoveActualLRP(lrp)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(events).Should(Receive(Equal(models.ActualLRPChange{
				Before: &lrp,
				After:  nil,
			})))
		})
	})
})
