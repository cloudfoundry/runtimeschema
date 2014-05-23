package lrp_bbs_test

import (
	. "github.com/cloudfoundry-incubator/runtime-schema/bbs/lrp_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LrpWatchers", func() {
	var bbs *LongRunningProcessBBS

	BeforeEach(func() {
		bbs = New(etcdClient)
	})

	Describe("WatchForDesiredLRPChanges", func() {
		var (
			events  <-chan models.DesiredLRPChange
			stop    chan<- bool
			errors  <-chan error
			stopped bool
		)

		lrp := models.DesiredLRP{
			ProcessGuid: "some-process-guid",
			Instances:   5,
			Stack:       "some-stack",
			MemoryMB:    1024,
			DiskMB:      512,
			Routes:      []string{"route-1", "route-2"},
		}

		BeforeEach(func() {
			events, stop, errors = bbs.WatchForDesiredLRPChanges()
		})

		AfterEach(func() {
			if !stopped {
				stop <- true
			}
		})

		It("sends an event down the pipe for creates", func() {
			err := bbs.DesireLongRunningProcess(lrp)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(events).Should(Receive(Equal(models.DesiredLRPChange{
				Before: nil,
				After:  &lrp,
			})))
		})

		It("sends an event down the pipe for updates", func() {
			err := bbs.DesireLongRunningProcess(lrp)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(events).Should(Receive())

			changedLRP := lrp
			changedLRP.Instances++

			err = bbs.DesireLongRunningProcess(changedLRP)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(events).Should(Receive(Equal(models.DesiredLRPChange{
				Before: &lrp,
				After:  &changedLRP,
			})))
		})

		It("sends an event down the pipe for deletes", func() {
			err := bbs.DesireLongRunningProcess(lrp)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(events).Should(Receive())

			err = etcdClient.Delete(shared.DesiredLRPSchemaPath(lrp))
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(events).Should(Receive(Equal(models.DesiredLRPChange{
				Before: &lrp,
				After:  nil,
			})))
		})

		It("closes the events and errors channel when told to stop", func() {
			stop <- true
			stopped = true

			err := bbs.DesireLongRunningProcess(lrp)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(events).Should(BeClosed())
			Ω(errors).Should(BeClosed())
		})
	})

	Describe("WatchForActualLongRunningProcesses", func() {
		var (
			events  <-chan models.LRP
			stop    chan<- bool
			errors  <-chan error
			stopped bool
		)

		lrp := models.LRP{ProcessGuid: "some-process-guid", State: models.LRPStateRunning}

		BeforeEach(func() {
			events, stop, errors = bbs.WatchForActualLongRunningProcesses()
		})

		AfterEach(func() {
			if !stopped {
				stop <- true
			}
		})

		It("sends an event down the pipe for creates", func() {
			err := bbs.ReportActualLongRunningProcessAsRunning(lrp)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(events).Should(Receive(Equal(lrp)))
		})

		It("sends an event down the pipe for updates", func() {
			err := bbs.ReportActualLongRunningProcessAsRunning(lrp)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(events).Should(Receive(Equal(lrp)))

			err = bbs.ReportActualLongRunningProcessAsRunning(lrp)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(events).Should(Receive(Equal(lrp)))
		})

		It("closes the events and errors channel when told to stop", func() {
			stop <- true
			stopped = true

			err := bbs.ReportActualLongRunningProcessAsRunning(lrp)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(events).Should(BeClosed())
			Ω(errors).Should(BeClosed())
		})
	})

})
