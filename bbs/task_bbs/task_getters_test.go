package task_bbs_test

import (
	"time"

	. "github.com/cloudfoundry-incubator/runtime-schema/bbs/task_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/timeprovider/faketimeprovider"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("Task BBS", func() {
	var bbs *TaskBBS
	var task models.Task
	var timeProvider *faketimeprovider.FakeTimeProvider
	var err error

	BeforeEach(func() {
		err = nil
		timeProvider = faketimeprovider.New(time.Unix(1238, 0))

		bbs = New(etcdClient, timeProvider, lagertest.NewTestLogger("test"))
		task = models.Task{
			Domain:   "tests",
			TaskGuid: "some-guid",
			Stack:    "pancakes",
			Actions:  dummyActions,
		}
	})

	Describe("GetAllPendingTasks", func() {
		BeforeEach(func() {
			err = bbs.DesireTask(task)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("returns all Tasks in 'pending' state", func() {
			tasks, err := bbs.GetAllPendingTasks()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(tasks).Should(HaveLen(1))
			Ω(tasks[0].TaskGuid).Should(Equal(task.TaskGuid))
		})
	})

	Describe("GetAllClaimedTasks", func() {
		BeforeEach(func() {
			err = bbs.DesireTask(task)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.ClaimTask(task.TaskGuid, "executor-ID")
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("returns all Tasks in 'claimed' state", func() {
			tasks, err := bbs.GetAllClaimedTasks()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(tasks).Should(HaveLen(1))
			Ω(tasks[0].TaskGuid).Should(Equal(task.TaskGuid))
		})
	})

	Describe("GetAllRunningTasks", func() {
		BeforeEach(func() {
			err = bbs.DesireTask(task)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.ClaimTask(task.TaskGuid, "executor-ID")
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.StartTask(task.TaskGuid, "executor-ID", "container-handle")
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("returns all Tasks in 'running' state", func() {
			tasks, err := bbs.GetAllRunningTasks()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(tasks).Should(HaveLen(1))
			Ω(tasks[0].TaskGuid).Should(Equal(task.TaskGuid))
		})
	})

	Describe("GetAllCompletedTasks", func() {
		BeforeEach(func() {
			err = bbs.DesireTask(task)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.ClaimTask(task.TaskGuid, "executor-ID")
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.StartTask(task.TaskGuid, "executor-ID", "container-handle")
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.CompleteTask(task.TaskGuid, true, "a reason", "a result")
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("returns all Tasks in 'completed' state", func() {
			tasks, err := bbs.GetAllCompletedTasks()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(tasks).Should(HaveLen(1))
			Ω(tasks[0].TaskGuid).Should(Equal(task.TaskGuid))
		})
	})

	Describe("GetAllResolvingTasks", func() {
		BeforeEach(func() {
			err = bbs.DesireTask(task)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.ClaimTask(task.TaskGuid, "executor-ID")
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.StartTask(task.TaskGuid, "executor-ID", "container-handle")
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.CompleteTask(task.TaskGuid, true, "a reason", "a result")
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.ResolvingTask(task.TaskGuid)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("returns all Tasks in 'completed' state", func() {
			tasks, err := bbs.GetAllResolvingTasks()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(tasks).Should(HaveLen(1))
			Ω(tasks[0].TaskGuid).Should(Equal(task.TaskGuid))
		})
	})
})
