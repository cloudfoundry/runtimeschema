package bbs_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry-incubator/runtime-schema/bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/timeprovider/faketimeprovider"
)

var _ = Describe("Task BBS", func() {
	var bbs *BBS
	var runOnce *models.Task
	var timeProvider *faketimeprovider.FakeTimeProvider

	BeforeEach(func() {
		timeProvider = faketimeprovider.New(time.Unix(1238, 0))
		bbs = New(store, timeProvider)
		runOnce = &models.Task{
			Guid:      "some-guid",
			CreatedAt: time.Now().UnixNano(),
		}
	})

	Describe("GetAllPendingTasks", func() {
		BeforeEach(func() {
			err := bbs.DesireTask(runOnce)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("returns all Tasks in 'pending' state", func() {
			runOnces, err := bbs.GetAllPendingTasks()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(runOnces).Should(HaveLen(1))
			Ω(runOnces).Should(ContainElement(runOnce))
		})
	})

	Describe("GetAllClaimedTasks", func() {
		BeforeEach(func() {
			err := bbs.DesireTask(runOnce)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.ClaimTask(runOnce, "executor-ID")
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("returns all Tasks in 'claimed' state", func() {
			runOnces, err := bbs.GetAllClaimedTasks()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(runOnces).Should(HaveLen(1))
			Ω(runOnces).Should(ContainElement(runOnce))
		})
	})

	Describe("GetAllStartingTasks", func() {
		BeforeEach(func() {
			err := bbs.DesireTask(runOnce)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.ClaimTask(runOnce, "executor-ID")
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.StartTask(runOnce, "container-handle")
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("returns all Tasks in 'running' state", func() {
			runOnces, err := bbs.GetAllStartingTasks()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(runOnces).Should(HaveLen(1))
			Ω(runOnces).Should(ContainElement(runOnce))
		})
	})

	Describe("GetAllCompletedTasks", func() {
		BeforeEach(func() {
			err := bbs.DesireTask(runOnce)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.ClaimTask(runOnce, "executor-ID")
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.StartTask(runOnce, "container-handle")
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.CompleteTask(runOnce, true, "a reason", "a result")
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("returns all Tasks in 'completed' state", func() {
			runOnces, err := bbs.GetAllCompletedTasks()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(runOnces).Should(HaveLen(1))
			Ω(runOnces).Should(ContainElement(runOnce))
		})
	})
})
