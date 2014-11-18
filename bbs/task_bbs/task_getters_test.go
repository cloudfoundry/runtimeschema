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
			Action:   dummyAction,
		}
	})

	Describe("TaskByGuid", func() {
		var guid string
		var receivedTask models.Task

		BeforeEach(func() {
			err := bbs.DesireTask(task)
			Ω(err).ShouldNot(HaveOccurred())
		})

		JustBeforeEach(func() {
			receivedTask, err = bbs.TaskByGuid(guid)
		})

		Context("When there is a task with the given guid", func() {
			BeforeEach(func() {
				guid = "some-guid"
			})

			It("does not an error", func() {
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("returns the task", func() {
				Ω(receivedTask.TaskGuid).Should(Equal(guid))
			})

			It("is consistent with collection getters", func() {
				pendingTasks, err := bbs.GetAllPendingTasks()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(pendingTasks).Should(Equal([]models.Task{receivedTask}))
			})
		})

		Context("When there is no task with the given guid", func() {
			BeforeEach(func() {
				guid = "not-some-guid"
			})

			It("returns an error", func() {
				Ω(err).Should(HaveOccurred())
			})
		})
	})

	Describe("TasksByCellID", func() {
		BeforeEach(func() {
			task.CellID = "some-other-cell-id"
			err = bbs.DesireTask(task)
			Ω(err).ShouldNot(HaveOccurred())
		})
		Context("when there are no tasks for the given cell ID", func() {
			It("returns an empty list", func() {
				tasks, err := bbs.TasksByCellID("cell-id")
				Ω(err).ShouldNot(HaveOccurred())
				Ω(tasks).Should(BeEmpty())
			})
		})

		Context("when there are tasks for the given cell ID", func() {
			var task1Request = models.Task{
				TaskGuid: "some-guid-1",
				CellID:   "cell-id",
				Domain:   "tests",
				Stack:    "pancakes",
				Action:   dummyAction,
			}
			var task1 models.Task
			var task2Request = models.Task{
				TaskGuid: "some-guid-2",
				CellID:   "cell-id",
				Domain:   "tests",
				Stack:    "pancakes",
				Action:   dummyAction,
			}
			var task2 models.Task

			BeforeEach(func() {
				err = bbs.DesireTask(task1Request)
				Ω(err).ShouldNot(HaveOccurred())
				task1, err = bbs.TaskByGuid("some-guid-1")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.DesireTask(task2Request)
				Ω(err).ShouldNot(HaveOccurred())
				task2, err = bbs.TaskByGuid("some-guid-2")
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("returns only those tasks", func() {
				tasks, err := bbs.TasksByCellID("cell-id")
				Ω(err).ShouldNot(HaveOccurred())
				Ω(tasks).Should(ConsistOf(task1, task2))
			})
		})
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

			err = bbs.ClaimTask(task.TaskGuid, "cell-ID")
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

			err = bbs.ClaimTask(task.TaskGuid, "cell-ID")
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.StartTask(task.TaskGuid, "cell-ID")
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

			err = bbs.ClaimTask(task.TaskGuid, "cell-ID")
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.StartTask(task.TaskGuid, "cell-ID")
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

			err = bbs.ClaimTask(task.TaskGuid, "cell-ID")
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.StartTask(task.TaskGuid, "cell-ID")
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

	Describe("TasksByDomain", func() {
		BeforeEach(func() {
			task.TaskGuid = "guid-1"
			err = bbs.DesireTask(task)
			Ω(err).ShouldNot(HaveOccurred())

			task.TaskGuid = "guid-2"
			task.Domain = "other-domain"
			err = bbs.DesireTask(task)
			Ω(err).ShouldNot(HaveOccurred())

			task.TaskGuid = "guid-3"
			task.Domain = "tests"
			err = bbs.DesireTask(task)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("returns all Tasks in the given domain", func() {
			tasks, err := bbs.TasksByDomain("tests")

			guids := []string{}
			for _, task := range tasks {
				guids = append(guids, task.TaskGuid)
			}

			Ω(err).ShouldNot(HaveOccurred())
			Ω(guids).Should(ConsistOf([]string{"guid-1", "guid-3"}))
		})
	})
})
