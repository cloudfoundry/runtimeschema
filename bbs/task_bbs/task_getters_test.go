package task_bbs_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Task BBS", func() {
	var task models.Task

	BeforeEach(func() {
		task = models.Task{
			Domain:   "tests",
			TaskGuid: "some-guid",
			RootFS:   "some:rootfs",
			Action:   dummyAction,
		}
	})

	Describe("TaskByGuid", func() {
		var guid string
		var receivedTask models.Task
		var lookupErr error

		BeforeEach(func() {
			err := bbs.DesireTask(logger, task)
			Ω(err).ShouldNot(HaveOccurred())
		})

		JustBeforeEach(func() {
			receivedTask, lookupErr = bbs.TaskByGuid(guid)
		})

		Context("When there is a task with the given guid", func() {
			BeforeEach(func() {
				guid = "some-guid"
			})

			It("does not an error", func() {
				Ω(lookupErr).ShouldNot(HaveOccurred())
			})

			It("returns the task", func() {
				Ω(receivedTask.TaskGuid).Should(Equal(guid))
			})

			It("is consistent with collection getters", func() {
				pendingTasks, err := bbs.PendingTasks(logger)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(pendingTasks).Should(Equal([]models.Task{receivedTask}))
			})
		})

		Context("When there is no task with the given guid", func() {
			BeforeEach(func() {
				guid = "not-some-guid"
			})

			It("returns an error", func() {
				Ω(lookupErr).Should(HaveOccurred())
			})
		})
	})

	Describe("TasksByCellID", func() {
		BeforeEach(func() {
			task.CellID = "some-other-cell-id"
			err := bbs.DesireTask(logger, task)
			Ω(err).ShouldNot(HaveOccurred())
		})
		Context("when there are no tasks for the given cell ID", func() {
			It("returns an empty list", func() {
				tasks, err := bbs.TasksByCellID(logger, "cell-id")
				Ω(err).ShouldNot(HaveOccurred())
				Ω(tasks).Should(BeEmpty())
			})
		})

		Context("when there are tasks for the given cell ID", func() {
			var task1Request = models.Task{
				TaskGuid: "some-guid-1",
				CellID:   "cell-id",
				Domain:   "tests",
				RootFS:   "some:rootfs",
				Action:   dummyAction,
			}
			var task1 models.Task
			var task2Request = models.Task{
				TaskGuid: "some-guid-2",
				CellID:   "cell-id",
				Domain:   "tests",
				RootFS:   "some:rootfs",
				Action:   dummyAction,
			}
			var task2 models.Task

			BeforeEach(func() {
				err := bbs.DesireTask(logger, task1Request)
				Ω(err).ShouldNot(HaveOccurred())

				task1, err = bbs.TaskByGuid("some-guid-1")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.DesireTask(logger, task2Request)
				Ω(err).ShouldNot(HaveOccurred())
				task2, err = bbs.TaskByGuid("some-guid-2")
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("returns only those tasks", func() {
				tasks, err := bbs.TasksByCellID(logger, "cell-id")
				Ω(err).ShouldNot(HaveOccurred())
				Ω(tasks).Should(ConsistOf(task1, task2))
			})
		})
	})

	Describe("PendingTasks", func() {
		BeforeEach(func() {
			err := bbs.DesireTask(logger, task)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("returns all Tasks in 'pending' state", func() {
			tasks, err := bbs.PendingTasks(logger)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(tasks).Should(HaveLen(1))
			Ω(tasks[0].TaskGuid).Should(Equal(task.TaskGuid))
		})
	})

	Describe("RunningTasks", func() {
		BeforeEach(func() {
			err := bbs.DesireTask(logger, task)
			Ω(err).ShouldNot(HaveOccurred())

			_, err = bbs.StartTask(logger, task.TaskGuid, "cell-ID")
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("returns all Tasks in 'running' state", func() {
			tasks, err := bbs.RunningTasks(logger)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(tasks).Should(HaveLen(1))
			Ω(tasks[0].TaskGuid).Should(Equal(task.TaskGuid))
		})
	})

	Describe("CompletedTasks", func() {
		BeforeEach(func() {
			err := bbs.DesireTask(logger, task)
			Ω(err).ShouldNot(HaveOccurred())

			_, err = bbs.StartTask(logger, task.TaskGuid, "cell-ID")
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", true, "a reason", "a result")
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("returns all Tasks in 'completed' state", func() {
			tasks, err := bbs.CompletedTasks(logger)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(tasks).Should(HaveLen(1))
			Ω(tasks[0].TaskGuid).Should(Equal(task.TaskGuid))
		})
	})

	Describe("ResolvingTasks", func() {
		BeforeEach(func() {
			err := bbs.DesireTask(logger, task)
			Ω(err).ShouldNot(HaveOccurred())

			_, err = bbs.StartTask(logger, task.TaskGuid, "cell-ID")
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", true, "a reason", "a result")
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.ResolvingTask(logger, task.TaskGuid)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("returns all Tasks in 'completed' state", func() {
			tasks, err := bbs.ResolvingTasks(logger)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(tasks).Should(HaveLen(1))
			Ω(tasks[0].TaskGuid).Should(Equal(task.TaskGuid))
		})
	})

	Describe("TasksByDomain", func() {
		BeforeEach(func() {
			task.TaskGuid = "guid-1"
			err := bbs.DesireTask(logger, task)
			Ω(err).ShouldNot(HaveOccurred())

			task.TaskGuid = "guid-2"
			task.Domain = "other-domain"
			err = bbs.DesireTask(logger, task)
			Ω(err).ShouldNot(HaveOccurred())

			task.TaskGuid = "guid-3"
			task.Domain = "tests"
			err = bbs.DesireTask(logger, task)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("returns all Tasks in the given domain", func() {
			tasks, err := bbs.TasksByDomain(logger, "tests")

			guids := []string{}
			for _, task := range tasks {
				guids = append(guids, task.TaskGuid)
			}

			Ω(err).ShouldNot(HaveOccurred())
			Ω(guids).Should(ConsistOf([]string{"guid-1", "guid-3"}))
		})
	})
})
