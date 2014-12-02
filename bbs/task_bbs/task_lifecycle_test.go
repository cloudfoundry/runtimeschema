package task_bbs_test

import (
	"errors"
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	. "github.com/cloudfoundry-incubator/runtime-schema/bbs/task_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/timeprovider/faketimeprovider"
	"github.com/cloudfoundry/storeadapter/fakestoreadapter"
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

	Describe("DesireTask", func() {
		Context("when the Task has a CreatedAt time", func() {
			BeforeEach(func() {
				task.CreatedAt = 1234812
				err = bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("creates /task/<guid>", func() {
				tasks, err := bbs.PendingTasks()
				Ω(err).ShouldNot(HaveOccurred())

				Ω(tasks[0].TaskGuid).Should(Equal(task.TaskGuid))
				Ω(tasks[0].CreatedAt).Should(Equal(task.CreatedAt))
				Ω(tasks[0].UpdatedAt).Should(Equal(timeProvider.Time().UnixNano()))
			})
		})

		Context("when the Task has no CreatedAt time", func() {
			BeforeEach(func() {
				err = bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("creates /task/<guid> and sets set the CreatedAt time to now", func() {
				tasks, err := bbs.PendingTasks()
				Ω(err).ShouldNot(HaveOccurred())

				Ω(tasks[0].CreatedAt).Should(Equal(timeProvider.Time().UnixNano()))
			})

			It("should bump UpdatedAt", func() {
				tasks, err := bbs.PendingTasks()
				Ω(err).ShouldNot(HaveOccurred())

				Ω(tasks[0].UpdatedAt).Should(Equal(timeProvider.Time().UnixNano()))
			})
		})

		Context("Common cases", func() {
			Context("when the Task is already pending", func() {
				It("returns an error", func() {
					err = bbs.DesireTask(task)
					Ω(err).ShouldNot(HaveOccurred())

					err = bbs.DesireTask(task)
					Ω(err).Should(HaveOccurred())
				})
			})

			Context("when the store is out of commission", func() {
				itRetriesUntilStoreComesBack(func() error {
					return bbs.DesireTask(task)
				})
			})

			It("bumps UpdatedAt", func() {
				err = bbs.DesireTask(task)

				tasks, err := bbs.PendingTasks()
				Ω(err).ShouldNot(HaveOccurred())

				Ω(tasks[0].UpdatedAt).Should(Equal(timeProvider.Time().UnixNano()))
			})
		})

		Context("with an invalid task", func() {
			var desireError error

			BeforeEach(func() {
				task.Domain = ""
				desireError = bbs.DesireTask(task)
			})

			It("returns an error", func() {
				Ω(desireError).Should(HaveOccurred())
				Ω(desireError).Should(BeAssignableToTypeOf(*new(models.ValidationError)))
			})
		})
	})

	Describe("ClaimTask", func() {
		Context("when claiming a pending Task", func() {
			BeforeEach(func() {
				err = bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("puts the Task in the claim state", func() {
				err = bbs.ClaimTask(task.TaskGuid, "cell-ID")
				Ω(err).ShouldNot(HaveOccurred())

				tasks, err := bbs.ClaimedTasks()
				Ω(err).ShouldNot(HaveOccurred())

				Ω(tasks[0].TaskGuid).Should(Equal(task.TaskGuid))
				Ω(tasks[0].State).Should(Equal(models.TaskStateClaimed))
				Ω(tasks[0].CellID).Should(Equal("cell-ID"))
			})

			It("should bump UpdatedAt", func() {
				err = bbs.ClaimTask(task.TaskGuid, "cell-ID")
				Ω(err).ShouldNot(HaveOccurred())

				tasks, err := bbs.ClaimedTasks()
				Ω(err).ShouldNot(HaveOccurred())

				Ω(tasks[0].UpdatedAt).Should(Equal(timeProvider.Time().UnixNano()))
			})

			Context("when the etcdClient is out of commission", func() {
				itRetriesUntilStoreComesBack(func() error {
					return bbs.ClaimTask(task.TaskGuid, "cell-ID")
				})
			})
		})

		Context("when claiming an already-claimed task", func() {
			It("returns an error", func() {
				err = bbs.ClaimTask(task.TaskGuid, "cell-ID")
				Ω(err).Should(HaveOccurred())
			})
		})

		Context("when claiming a Task that is not in the pending state", func() {
			BeforeEach(func() {
				err = bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.ClaimTask(task.TaskGuid, "cell-ID")
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("returns an error", func() {
				err = bbs.ClaimTask(task.TaskGuid, "cell-ID")
				Ω(err).Should(HaveOccurred())

				err = bbs.ClaimTask(task.TaskGuid, "some-other-cell-ID")
				Ω(err).Should(HaveOccurred())
			})
		})
	})

	Describe("StartTask", func() {
		Context("when starting a claimed Task", func() {
			BeforeEach(func() {
				err = bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.ClaimTask(task.TaskGuid, "cell-ID")
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("sets the state to running", func() {
				err = bbs.StartTask(task.TaskGuid, "cell-ID")
				Ω(err).ShouldNot(HaveOccurred())

				tasks, err := bbs.RunningTasks()
				Ω(err).ShouldNot(HaveOccurred())

				Ω(tasks[0].TaskGuid).Should(Equal(task.TaskGuid))
				Ω(tasks[0].State).Should(Equal(models.TaskStateRunning))
			})

			It("should bump UpdatedAt", func() {
				timeProvider.IncrementBySeconds(1)

				err = bbs.StartTask(task.TaskGuid, "cell-ID")
				Ω(err).ShouldNot(HaveOccurred())

				tasks, err := bbs.RunningTasks()
				Ω(err).ShouldNot(HaveOccurred())

				Ω(tasks[0].UpdatedAt).Should(Equal(timeProvider.Time().UnixNano()))
			})

			Context("when the store is out of commission", func() {
				itRetriesUntilStoreComesBack(func() error {
					return bbs.StartTask(task.TaskGuid, "cell-ID")
				})
			})
		})

		Context("When starting a Task that is not in the claimed state", func() {
			It("returns an error", func() {
				err = bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.StartTask(task.TaskGuid, "cell-ID")
				Ω(err).Should(HaveOccurred())
			})
		})

		Context("When starting a Task that was claimed by a different cell", func() {
			It("returns an error", func() {
				err = bbs.StartTask(task.TaskGuid, "some-other-cell-ID")
				Ω(err).Should(HaveOccurred())
			})
		})
	})

	Describe("CancelTask", func() {

		Context("when the store is reachable", func() {
			var cancelError error
			var taskAfterCancel *models.Task

			JustBeforeEach(func() {
				cancelError = bbs.CancelTask(task.TaskGuid)
				taskAfterCancel, _ = bbs.TaskByGuid(task.TaskGuid)
			})

			itMarksTaskAsCancelled := func() {
				It("does not error", func() {
					Ω(cancelError).ShouldNot(HaveOccurred())
				})

				It("marks the task as completed", func() {
					Ω(taskAfterCancel.State).Should(Equal(models.TaskStateCompleted))
				})

				It("marks the task as failed", func() {
					Ω(taskAfterCancel.Failed).Should(BeTrue())
				})

				It("sets the failure reason to cancelled", func() {
					Ω(taskAfterCancel.FailureReason).Should(Equal("task was cancelled"))
				})

				It("bumps UpdatedAt", func() {
					Ω(taskAfterCancel.UpdatedAt).Should(Equal(timeProvider.Time().UnixNano()))
				})
			}

			Context("when the task is in pending state", func() {
				BeforeEach(func() {
					err = bbs.DesireTask(task)
					Ω(err).ShouldNot(HaveOccurred())
				})

				itMarksTaskAsCancelled()
			})

			Context("when the task is in claimed state", func() {
				BeforeEach(func() {
					err = bbs.DesireTask(task)
					Ω(err).ShouldNot(HaveOccurred())
					err = bbs.ClaimTask(task.TaskGuid, "cell-id")
					Ω(err).ShouldNot(HaveOccurred())
				})

				itMarksTaskAsCancelled()
			})

			Context("when the task is in running state", func() {
				BeforeEach(func() {
					err = bbs.DesireTask(task)
					Ω(err).ShouldNot(HaveOccurred())
					err = bbs.ClaimTask(task.TaskGuid, "cell-id")
					Ω(err).ShouldNot(HaveOccurred())
					err = bbs.StartTask(task.TaskGuid, "cell-id")
					Ω(err).ShouldNot(HaveOccurred())
				})

				itMarksTaskAsCancelled()
			})

			Context("when the task is in completed state", func() {
				BeforeEach(func() {
					err = bbs.DesireTask(task)
					Ω(err).ShouldNot(HaveOccurred())
					err = bbs.ClaimTask(task.TaskGuid, "cell-id")
					Ω(err).ShouldNot(HaveOccurred())
					err = bbs.CompleteTask(task.TaskGuid, false, "", "")
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("returns an error", func() {
					Ω(cancelError).Should(HaveOccurred())
					Ω(cancelError).Should(BeAssignableToTypeOf(bbserrors.ErrTaskCannotBeCancelled))
				})
			})

			Context("when the task is in resolving state", func() {
				BeforeEach(func() {
					err = bbs.DesireTask(task)
					Ω(err).ShouldNot(HaveOccurred())
					err = bbs.ClaimTask(task.TaskGuid, "cell-id")
					Ω(err).ShouldNot(HaveOccurred())
					err = bbs.CompleteTask(task.TaskGuid, false, "", "")
					Ω(err).ShouldNot(HaveOccurred())
					err = bbs.ResolvingTask(task.TaskGuid)
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("returns an error", func() {
					Ω(cancelError).Should(HaveOccurred())
					Ω(cancelError).Should(BeAssignableToTypeOf(bbserrors.ErrTaskCannotBeCancelled))
				})
			})

			Context("when the task does not exist", func() {
				It("returns an error", func() {
					Ω(cancelError).Should(HaveOccurred())
					Ω(cancelError).Should(Equal(bbserrors.ErrTaskNotFound))
				})
			})

			Context("when the store returns some error other than key not found or timeout", func() {
				var storeError = errors.New("store error")

				BeforeEach(func() {
					fakeStoreAdapter := fakestoreadapter.New()
					fakeStoreAdapter.GetErrInjector = fakestoreadapter.NewFakeStoreAdapterErrorInjector(``, storeError)

					bbs = New(fakeStoreAdapter, timeProvider, lagertest.NewTestLogger("test"))
				})
				It("returns an error", func() {
					Ω(cancelError).Should(HaveOccurred())
					Ω(cancelError).Should(Equal(storeError))
				})
			})
		})

		Context("when the store is out of commission", func() {
			BeforeEach(func() {
				err = bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())
			})

			itRetriesUntilStoreComesBack(func() error {
				return bbs.CancelTask(task.TaskGuid)
			})
		})
	})

	Describe("CompleteTask", func() {
		Context("when completing a running Task", func() {
			BeforeEach(func() {
				err = bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.ClaimTask(task.TaskGuid, "cell-ID")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.StartTask(task.TaskGuid, "cell-ID")
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("sets the Task in the completed state", func() {
				err = bbs.CompleteTask(task.TaskGuid, true, "because i said so", "a result")
				Ω(err).ShouldNot(HaveOccurred())

				tasks, err := bbs.CompletedTasks()
				Ω(err).ShouldNot(HaveOccurred())

				Ω(tasks[0].Failed).Should(BeTrue())
				Ω(tasks[0].FailureReason).Should(Equal("because i said so"))
			})

			It("should bump UpdatedAt", func() {
				timeProvider.IncrementBySeconds(1)

				err = bbs.CompleteTask(task.TaskGuid, true, "because i said so", "a result")
				Ω(err).ShouldNot(HaveOccurred())

				tasks, err := bbs.CompletedTasks()
				Ω(err).ShouldNot(HaveOccurred())

				Ω(tasks[0].UpdatedAt).Should(Equal(timeProvider.Time().UnixNano()))
			})

			It("sets FirstCompletedAt", func() {
				timeProvider.IncrementBySeconds(1)

				err = bbs.CompleteTask(task.TaskGuid, true, "because i said so", "a result")
				Ω(err).ShouldNot(HaveOccurred())

				tasks, err := bbs.CompletedTasks()
				Ω(err).ShouldNot(HaveOccurred())

				Ω(tasks[0].FirstCompletedAt).Should(Equal(timeProvider.Time().UnixNano()))
			})

			Context("when the store is out of commission", func() {
				itRetriesUntilStoreComesBack(func() error {
					return bbs.CompleteTask(task.TaskGuid, false, "", "a result")
				})
			})
		})

		Context("When completing a Task that is not in the running/claimed state", func() {
			It("returns an error", func() {
				err = bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.CompleteTask(task.TaskGuid, true, "because i said so", "a result")
				Ω(err).Should(HaveOccurred())
			})

			It("has no error when Task is in claimed state", func() {
				err = bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.ClaimTask(task.TaskGuid, "cell-ID")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.CompleteTask(task.TaskGuid, true, "because i said so", "a result")
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("has no error when Task is in running state", func() {
				err = bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.ClaimTask(task.TaskGuid, "cell-ID")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.StartTask(task.TaskGuid, "cell-ID")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.CompleteTask(task.TaskGuid, true, "because i said so", "a result")
				Ω(err).ShouldNot(HaveOccurred())
			})
		})
	})

	Describe("ResolvingTask", func() {
		Context("when the task is complete", func() {
			BeforeEach(func() {
				err = bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.ClaimTask(task.TaskGuid, "some-cell-id")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.StartTask(task.TaskGuid, "some-cell-id")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.CompleteTask(task.TaskGuid, true, "because i said so", "a result")
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("swaps /task/<guid>'s state to resolving", func() {
				err = bbs.ResolvingTask(task.TaskGuid)
				Ω(err).ShouldNot(HaveOccurred())

				tasks, err := bbs.ResolvingTasks()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(tasks[0].TaskGuid).Should(Equal(task.TaskGuid))
				Ω(tasks[0].State).Should(Equal(models.TaskStateResolving))
			})

			It("bumps UpdatedAt", func() {
				timeProvider.IncrementBySeconds(1)

				err = bbs.ResolvingTask(task.TaskGuid)
				Ω(err).ShouldNot(HaveOccurred())

				tasks, err := bbs.ResolvingTasks()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(tasks[0].UpdatedAt).Should(Equal(timeProvider.Time().UnixNano()))
			})

			Context("when the Task is already resolving", func() {
				BeforeEach(func() {
					err = bbs.ResolvingTask(task.TaskGuid)
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("fails", func() {
					err = bbs.ResolvingTask(task.TaskGuid)
					Ω(err).Should(HaveOccurred())
				})
			})
		})

		Context("when the task is not complete", func() {
			BeforeEach(func() {
				err = bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.ClaimTask(task.TaskGuid, "some-cell-id")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.StartTask(task.TaskGuid, "some-cell-id")
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("should fail", func() {
				err := bbs.ResolvingTask(task.TaskGuid)
				Ω(err).Should(Equal(bbserrors.ErrTaskCannotBeMarkedAsResolving))
			})
		})
	})

	Describe("ResolveTask", func() {
		Context("when the task is resolving", func() {
			BeforeEach(func() {
				err = bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.ClaimTask(task.TaskGuid, "some-cell-id")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.StartTask(task.TaskGuid, "some-cell-id")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.CompleteTask(task.TaskGuid, true, "because i said so", "a result")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.ResolvingTask(task.TaskGuid)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("should remove /task/<guid>", func() {
				err = bbs.ResolveTask(task.TaskGuid)
				Ω(err).ShouldNot(HaveOccurred())

				tasks, err := bbs.Tasks()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(tasks).Should(BeEmpty())
			})

			Context("when the store is out of commission", func() {
				itRetriesUntilStoreComesBack(func() error {
					return bbs.ResolveTask(task.TaskGuid)
				})
			})
		})

		Context("when the task is not resolving", func() {
			BeforeEach(func() {
				err = bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.ClaimTask(task.TaskGuid, "some-cell-id")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.StartTask(task.TaskGuid, "some-cell-id")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.CompleteTask(task.TaskGuid, true, "because i said so", "a result")
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("should fail", func() {
				err = bbs.ResolveTask(task.TaskGuid)
				Ω(err).Should(HaveOccurred())
			})
		})
	})
})
