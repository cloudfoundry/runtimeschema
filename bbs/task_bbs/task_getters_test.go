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
			Expect(err).NotTo(HaveOccurred())
		})

		JustBeforeEach(func() {
			receivedTask, lookupErr = bbs.TaskByGuid(logger, guid)
		})

		Context("When there is a task with the given guid", func() {
			BeforeEach(func() {
				guid = "some-guid"
			})

			It("does not an error", func() {
				Expect(lookupErr).NotTo(HaveOccurred())
			})

			It("returns the task", func() {
				Expect(receivedTask.TaskGuid).To(Equal(guid))
			})

			It("is consistent with collection getters", func() {
				pendingTasks, err := bbs.PendingTasks(logger)
				Expect(err).NotTo(HaveOccurred())
				Expect(pendingTasks).To(Equal([]models.Task{receivedTask}))
			})
		})

		Context("When there is no task with the given guid", func() {
			BeforeEach(func() {
				guid = "not-some-guid"
			})

			It("returns an error", func() {
				Expect(lookupErr).To(HaveOccurred())
			})
		})
	})

	Describe("TasksByCellID", func() {
		BeforeEach(func() {
			task.CellID = "some-other-cell-id"
			err := bbs.DesireTask(logger, task)
			Expect(err).NotTo(HaveOccurred())
		})
		Context("when there are no tasks for the given cell ID", func() {
			It("returns an empty list", func() {
				tasks, err := bbs.TasksByCellID(logger, "cell-id")
				Expect(err).NotTo(HaveOccurred())
				Expect(tasks).To(BeEmpty())
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
				Expect(err).NotTo(HaveOccurred())

				task1, err = bbs.TaskByGuid(logger, "some-guid-1")
				Expect(err).NotTo(HaveOccurred())

				err = bbs.DesireTask(logger, task2Request)
				Expect(err).NotTo(HaveOccurred())
				task2, err = bbs.TaskByGuid(logger, "some-guid-2")
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns only those tasks", func() {
				tasks, err := bbs.TasksByCellID(logger, "cell-id")
				Expect(err).NotTo(HaveOccurred())
				Expect(tasks).To(ConsistOf(task1, task2))
			})
		})
	})

	Describe("PendingTasks", func() {
		BeforeEach(func() {
			err := bbs.DesireTask(logger, task)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns all Tasks in 'pending' state", func() {
			tasks, err := bbs.PendingTasks(logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(tasks).To(HaveLen(1))
			Expect(tasks[0].TaskGuid).To(Equal(task.TaskGuid))
		})
	})

	Describe("RunningTasks", func() {
		BeforeEach(func() {
			err := bbs.DesireTask(logger, task)
			Expect(err).NotTo(HaveOccurred())

			_, err = bbs.StartTask(logger, task.TaskGuid, "cell-ID")
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns all Tasks in 'running' state", func() {
			tasks, err := bbs.RunningTasks(logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(tasks).To(HaveLen(1))
			Expect(tasks[0].TaskGuid).To(Equal(task.TaskGuid))
		})
	})

	Describe("CompletedTasks", func() {
		BeforeEach(func() {
			err := bbs.DesireTask(logger, task)
			Expect(err).NotTo(HaveOccurred())

			_, err = bbs.StartTask(logger, task.TaskGuid, "cell-ID")
			Expect(err).NotTo(HaveOccurred())

			err = bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", true, "a reason", "a result")
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns all Tasks in 'completed' state", func() {
			tasks, err := bbs.CompletedTasks(logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(tasks).To(HaveLen(1))
			Expect(tasks[0].TaskGuid).To(Equal(task.TaskGuid))
		})
	})

	Describe("FailedTasks", func() {
		BeforeEach(func() {
			err := bbs.DesireTask(logger, task)
			Expect(err).NotTo(HaveOccurred())

			_, err = bbs.StartTask(logger, task.TaskGuid, "cell-ID")
			Expect(err).NotTo(HaveOccurred())

			err = bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", true, "a reason", "a result")
			Expect(err).NotTo(HaveOccurred())

			otherTask := models.Task{
				Domain:   "tests",
				TaskGuid: "some-other-guid",
				RootFS:   "some:rootfs",
				Action:   dummyAction,
			}

			err = bbs.DesireTask(logger, otherTask)
			Expect(err).NotTo(HaveOccurred())

			_, err = bbs.StartTask(logger, otherTask.TaskGuid, "cell-ID")
			Expect(err).NotTo(HaveOccurred())

			err = bbs.CompleteTask(logger, otherTask.TaskGuid, "cell-ID", false, "", "a result")
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns all Tasks in 'completed' state that have failed", func() {
			tasks, err := bbs.FailedTasks(logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(tasks).To(HaveLen(1))
			Expect(tasks[0].TaskGuid).To(Equal(task.TaskGuid))
		})
	})

	Describe("ResolvingTasks", func() {
		BeforeEach(func() {
			err := bbs.DesireTask(logger, task)
			Expect(err).NotTo(HaveOccurred())

			_, err = bbs.StartTask(logger, task.TaskGuid, "cell-ID")
			Expect(err).NotTo(HaveOccurred())

			err = bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", true, "a reason", "a result")
			Expect(err).NotTo(HaveOccurred())

			err = bbs.ResolvingTask(logger, task.TaskGuid)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns all Tasks in 'completed' state", func() {
			tasks, err := bbs.ResolvingTasks(logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(tasks).To(HaveLen(1))
			Expect(tasks[0].TaskGuid).To(Equal(task.TaskGuid))
		})
	})

	Describe("TasksByDomain", func() {
		BeforeEach(func() {
			task.TaskGuid = "guid-1"
			err := bbs.DesireTask(logger, task)
			Expect(err).NotTo(HaveOccurred())

			task.TaskGuid = "guid-2"
			task.Domain = "other-domain"
			err = bbs.DesireTask(logger, task)
			Expect(err).NotTo(HaveOccurred())

			task.TaskGuid = "guid-3"
			task.Domain = "tests"
			err = bbs.DesireTask(logger, task)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns all Tasks in the given domain", func() {
			tasks, err := bbs.TasksByDomain(logger, "tests")

			guids := []string{}
			for _, task := range tasks {
				guids = append(guids, task.TaskGuid)
			}

			Expect(err).NotTo(HaveOccurred())
			Expect(guids).To(ConsistOf([]string{"guid-1", "guid-3"}))
		})
	})
})
