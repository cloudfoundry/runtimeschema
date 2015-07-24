package task_bbs_test

import (
	"errors"
	"net/url"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	. "github.com/cloudfoundry-incubator/runtime-schema/bbs/task_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter/fakestoreadapter"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Task BBS", func() {
	var task models.Task

	Describe("DesireTask", func() {
		var errDesire error

		JustBeforeEach(func() {
			errDesire = bbs.DesireTask(logger, task)
		})

		Context("when given a valid task", func() {
			Context("when a task is not already present at the desired key", func() {
				Context("when given a task with a CreatedAt time", func() {
					var taskGuid string
					var domain string
					var rootfs string
					var createdAtTime int64

					BeforeEach(func() {
						taskGuid = "some-guid"
						domain = "tests"
						rootfs = "some:rootfs"
						createdAtTime = 1234812

						task = models.Task{
							Domain:    domain,
							TaskGuid:  taskGuid,
							RootFS:    rootfs,
							Action:    dummyAction,
							CreatedAt: createdAtTime,
						}
					})

					It("does not error", func() {
						Expect(errDesire).NotTo(HaveOccurred())
					})

					It("persists the task", func() {
						persistedTask, err := bbs.OldTaskByGuid(logger, taskGuid)
						Expect(err).NotTo(HaveOccurred())

						Expect(persistedTask.Domain).To(Equal(domain))
						Expect(persistedTask.RootFS).To(Equal(rootfs))
						Expect(persistedTask.Action).To(Equal(dummyAction))
					})

					It("honours the CreatedAt time", func() {
						persistedTask, err := bbs.OldTaskByGuid(logger, taskGuid)
						Expect(err).NotTo(HaveOccurred())
						Expect(persistedTask.CreatedAt).To(Equal(createdAtTime))
					})

					It("sets the UpdatedAt time", func() {
						persistedTask, err := bbs.OldTaskByGuid(logger, taskGuid)
						Expect(err).NotTo(HaveOccurred())
						Expect(persistedTask.UpdatedAt).To(Equal(clock.Now().UnixNano()))
					})

					Context("when able to fetch the Auctioneer address", func() {
						var auctioneerPresence models.AuctioneerPresence

						BeforeEach(func() {
							auctioneerPresence = models.AuctioneerPresence{
								AuctioneerID:      "the-auctioneer-id",
								AuctioneerAddress: "the-address",
							}

							registerAuctioneer(auctioneerPresence)
						})

						It("requests an auction", func() {
							Expect(fakeAuctioneerClient.RequestTaskAuctionsCallCount()).To(Equal(1))

							requestAddress, requestedTasks := fakeAuctioneerClient.RequestTaskAuctionsArgsForCall(0)
							Expect(requestAddress).To(Equal(auctioneerPresence.AuctioneerAddress))
							Expect(requestedTasks).To(HaveLen(1))
							Expect(requestedTasks[0].TaskGuid).To(Equal(taskGuid))
						})

						Context("when requesting a task auction succeeds", func() {
							BeforeEach(func() {
								fakeAuctioneerClient.RequestTaskAuctionsReturns(nil)
							})

							It("does not return an error", func() {
								Expect(errDesire).NotTo(HaveOccurred())
							})
						})

						Context("when requesting a task auction fails", func() {
							BeforeEach(func() {
								fakeAuctioneerClient.RequestTaskAuctionsReturns(errors.New("oops"))
							})

							It("does not return an error", func() {
								// The creation succeeded, we can ignore the auction request error (converger will eventually do it)
								Expect(errDesire).NotTo(HaveOccurred())
							})
						})
					})

					Context("when unable to fetch the Auctioneer address", func() {
						It("does not request an auction", func() {
							Consistently(fakeAuctioneerClient.RequestTaskAuctionsCallCount).Should(BeZero())
						})

						It("does not return an error", func() {
							// The creation succeeded, we can ignore the auction request error (converger will eventually do it)
							Expect(errDesire).NotTo(HaveOccurred())
						})
					})
				})

				Context("when given a task without a CreatedAt time", func() {
					var taskGuid string
					var domain string
					var rootfs string

					BeforeEach(func() {
						taskGuid = "some-guid"
						domain = "tests"
						rootfs = "some:rootfs"
						task = models.Task{
							Domain:   domain,
							TaskGuid: taskGuid,
							RootFS:   rootfs,
							Action:   dummyAction,
						}
					})

					It("does not error", func() {
						Expect(errDesire).NotTo(HaveOccurred())
					})

					It("persists the task", func() {
						persistedTask, err := bbs.OldTaskByGuid(logger, taskGuid)
						Expect(err).NotTo(HaveOccurred())

						Expect(persistedTask.Domain).To(Equal(domain))
						Expect(persistedTask.RootFS).To(Equal(rootfs))
						Expect(persistedTask.Action).To(Equal(dummyAction))
					})

					It("provides a CreatedAt time", func() {
						persistedTask, err := bbs.OldTaskByGuid(logger, taskGuid)
						Expect(err).NotTo(HaveOccurred())
						Expect(persistedTask.CreatedAt).To(Equal(clock.Now().UnixNano()))
					})

					It("sets the UpdatedAt time", func() {
						persistedTask, err := bbs.OldTaskByGuid(logger, taskGuid)
						Expect(err).NotTo(HaveOccurred())
						Expect(persistedTask.UpdatedAt).To(Equal(clock.Now().UnixNano()))
					})
				})
			})

			Context("when a task is already present at the desired key", func() {
				BeforeEach(func() {
					task = models.Task{
						Domain:   "tests",
						TaskGuid: "some-guid",
						RootFS:   "some:rootfs",
						Action:   dummyAction,
					}

					err := bbs.DesireTask(logger, task)
					Expect(err).NotTo(HaveOccurred())
				})

				It("does not persist another", func() {
					Consistently(
						func() ([]models.Task, error) {
							return bbs.OldTasks(logger)
						}).Should(HaveLen(1))
				})

				It("does not request an auction", func() {
					Consistently(fakeAuctioneerClient.RequestTaskAuctionsCallCount).Should(BeZero())
				})

				It("returns an error", func() {
					Expect(errDesire).To(Equal(bbserrors.ErrStoreResourceExists))
				})
			})
		})

		Context("when given an invalid task", func() {
			BeforeEach(func() {
				task = models.Task{
					TaskGuid: "some-guid",
					RootFS:   "some:rootfs",
					Action:   dummyAction,
					// missing Domain
				}
			})

			It("does not persist a task", func() {
				Consistently(func() ([]models.Task, error) {
					return bbs.OldTasks(logger)
				}).Should(BeEmpty())
			})

			It("does not request an auction", func() {
				Consistently(fakeAuctioneerClient.RequestTaskAuctionsCallCount).Should(BeZero())
			})

			It("returns an error", func() {
				Expect(errDesire).To(ContainElement(models.ErrInvalidField{"domain"}))
			})
		})
	})

	Describe("StartTask", func() {
		BeforeEach(func() {
			task = models.Task{
				TaskGuid:  "some-guid",
				Domain:    "tests",
				RootFS:    "some:rootfs",
				Action:    dummyAction,
				CreatedAt: 1234812,
			}
		})

		Context("when starting a pending Task", func() {
			BeforeEach(func() {
				err := bbs.DesireTask(logger, task)
				Expect(err).NotTo(HaveOccurred())
			})

			It("sets the state to running", func() {
				started, err := bbs.StartTask(logger, task.TaskGuid, "cell-ID")
				Expect(err).NotTo(HaveOccurred())
				Expect(started).To(BeTrue())

				tasks, err := bbs.RunningTasks(logger)
				Expect(err).NotTo(HaveOccurred())

				Expect(tasks[0].TaskGuid).To(Equal(task.TaskGuid))
				Expect(tasks[0].State).To(Equal(models.TaskStateRunning))
			})

			It("should bump UpdatedAt", func() {
				clock.IncrementBySeconds(1)

				started, err := bbs.StartTask(logger, task.TaskGuid, "cell-ID")
				Expect(err).NotTo(HaveOccurred())
				Expect(started).To(BeTrue())

				tasks, err := bbs.RunningTasks(logger)
				Expect(err).NotTo(HaveOccurred())

				Expect(tasks[0].UpdatedAt).To(Equal(clock.Now().UnixNano()))
			})
		})

		Context("When starting a Task that is already started", func() {
			var changed bool

			BeforeEach(func() {
				err := bbs.DesireTask(logger, task)
				Expect(err).NotTo(HaveOccurred())

				_, err = bbs.StartTask(logger, task.TaskGuid, "cell-ID")
				Expect(err).NotTo(HaveOccurred())
			})

			Context("on the same cell", func() {
				var startErr error
				var previousTime int64

				BeforeEach(func() {
					previousTime = clock.Now().UnixNano()
					clock.IncrementBySeconds(1)

					changed, startErr = bbs.StartTask(logger, task.TaskGuid, "cell-ID")
				})

				It("does not return an error", func() {
					Expect(startErr).NotTo(HaveOccurred())
				})

				It("returns false", func() {
					Expect(changed).To(BeFalse())
				})

				It("does not change the Task in the store", func() {
					task, err := bbs.OldTaskByGuid(logger, task.TaskGuid)
					Expect(err).NotTo(HaveOccurred())

					Expect(task.UpdatedAt).To(Equal(previousTime))
				})
			})

			Context("on another cell", func() {
				It("returns an error", func() {
					changed, err := bbs.StartTask(logger, task.TaskGuid, "another-cell-ID")
					Expect(err).To(HaveOccurred())
					Expect(changed).To(BeFalse())
				})
			})
		})
	})

	Describe("CancelTask", func() {
		BeforeEach(func() {
			task = models.Task{
				TaskGuid:  "some-guid",
				Domain:    "tests",
				RootFS:    "some:rootfs",
				Action:    dummyAction,
				CreatedAt: 1234812,
			}
		})

		Context("when the store is reachable", func() {
			var cancelError error
			var taskAfterCancel models.Task

			JustBeforeEach(func() {
				cancelError = bbs.CancelTask(logger, task.TaskGuid)
				taskAfterCancel, _ = bbs.OldTaskByGuid(logger, task.TaskGuid)
			})

			itMarksTaskAsCancelled := func() {
				It("does not error", func() {
					Expect(cancelError).NotTo(HaveOccurred())
				})

				It("marks the task as completed", func() {
					Expect(taskAfterCancel.State).To(Equal(models.TaskStateCompleted))
				})

				It("marks the task as failed", func() {
					Expect(taskAfterCancel.Failed).To(BeTrue())
				})

				It("sets the failure reason to cancelled", func() {
					Expect(taskAfterCancel.FailureReason).To(Equal("task was cancelled"))
				})

				It("bumps UpdatedAt", func() {
					Expect(taskAfterCancel.UpdatedAt).To(Equal(clock.Now().UnixNano()))
				})
			}

			Context("when the task is in pending state", func() {
				BeforeEach(func() {
					err := bbs.DesireTask(logger, task)
					Expect(err).NotTo(HaveOccurred())
				})

				itMarksTaskAsCancelled()

				It("does not cancel the task", func() {
					Expect(fakeCellClient.CancelTaskCallCount()).To(Equal(0))
				})
			})

			Context("when the task is in running state", func() {
				var cellID string

				BeforeEach(func() {
					cellID = "cell-ID"

					err := bbs.DesireTask(logger, task)
					Expect(err).NotTo(HaveOccurred())

					_, err = bbs.StartTask(logger, task.TaskGuid, cellID)
					Expect(err).NotTo(HaveOccurred())
				})

				itMarksTaskAsCancelled()

				Context("when the cell is present", func() {
					var cellPresence models.CellPresence

					BeforeEach(func() {
						cellPresence = models.NewCellPresence(cellID, "cell.example.com", "the-zone", models.NewCellCapacity(128, 1024, 6), []string{}, []string{})
						registerCell(cellPresence)
					})

					It("cancels the task", func() {
						Expect(fakeCellClient.CancelTaskCallCount()).To(Equal(1))

						addr, taskGuid := fakeCellClient.CancelTaskArgsForCall(0)
						Expect(addr).To(Equal(cellPresence.RepAddress))
						Expect(taskGuid).To(Equal(task.TaskGuid))
					})
				})

				Context("when the cell is not present", func() {
					It("does not cancel the task", func() {
						Expect(fakeCellClient.CancelTaskCallCount()).To(Equal(0))
					})

					It("logs the error", func() {
						Eventually(logger.TestSink.LogMessages).Should(ContainElement("test.cancel-task.failed-getting-cell-info"))
					})
				})
			})

			Context("when the task is in completed state", func() {
				BeforeEach(func() {
					err := bbs.DesireTask(logger, task)
					Expect(err).NotTo(HaveOccurred())

					_, err = bbs.StartTask(logger, task.TaskGuid, "cell-ID")
					Expect(err).NotTo(HaveOccurred())

					err = bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", false, "", "")
					Expect(err).NotTo(HaveOccurred())
				})

				It("returns an error", func() {
					Expect(cancelError).To(HaveOccurred())
					Expect(cancelError).To(Equal(bbserrors.NewTaskStateTransitionError(models.TaskStateCompleted, models.TaskStateCompleted)))
				})
			})

			Context("when the task is in resolving state", func() {
				BeforeEach(func() {
					err := bbs.DesireTask(logger, task)
					Expect(err).NotTo(HaveOccurred())

					_, err = bbs.StartTask(logger, task.TaskGuid, "cell-ID")
					Expect(err).NotTo(HaveOccurred())

					err = bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", false, "", "")
					Expect(err).NotTo(HaveOccurred())

					err = bbs.ResolvingTask(logger, task.TaskGuid)
					Expect(err).NotTo(HaveOccurred())
				})

				It("returns an error", func() {
					Expect(cancelError).To(HaveOccurred())
					Expect(cancelError).To(Equal(bbserrors.NewTaskStateTransitionError(models.TaskStateResolving, models.TaskStateCompleted)))
				})
			})

			Context("when the task does not exist", func() {
				It("returns an error", func() {
					Expect(cancelError).To(HaveOccurred())
					Expect(cancelError).To(Equal(bbserrors.ErrStoreResourceNotFound))
				})
			})

			Context("when the store returns some error other than key not found or timeout", func() {
				var storeError = errors.New("store error")

				BeforeEach(func() {
					fakeStoreAdapter := fakestoreadapter.New()
					fakeStoreAdapter.GetErrInjector = fakestoreadapter.NewFakeStoreAdapterErrorInjector(``, storeError)

					bbs = New(fakeStoreAdapter, consulSession, clock, fakeTaskClient, fakeAuctioneerClient, fakeCellClient,
						servicesBBS, receptorURL)
				})

				It("returns an error", func() {
					Expect(cancelError).To(HaveOccurred())
					Expect(cancelError).To(Equal(storeError))
				})
			})
		})
	})

	Describe("CompleteTask", func() {
		BeforeEach(func() {
			task = models.Task{
				TaskGuid:  "some-guid",
				Domain:    "tests",
				RootFS:    "some:rootfs",
				Action:    dummyAction,
				CreatedAt: 1234812,
			}
		})

		Context("when completing a pending Task", func() {
			JustBeforeEach(func() {
				err := bbs.DesireTask(logger, task)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns an error", func() {
				err := bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", true, "another failure reason", "")
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when completing a running Task", func() {
			JustBeforeEach(func() {
				err := bbs.DesireTask(logger, task)
				Expect(err).NotTo(HaveOccurred())

				_, err = bbs.StartTask(logger, task.TaskGuid, "cell-ID")
				Expect(err).NotTo(HaveOccurred())
			})

			Context("when the cell id is not the same", func() {
				It("returns an error", func() {
					err := bbs.CompleteTask(logger, task.TaskGuid, "another-cell-ID", true, "because i said so", "a result")
					Expect(err).To(Equal(bbserrors.ErrTaskRunningOnDifferentCell))
				})
			})

			Context("when the cell id is the same", func() {
				It("sets the Task in the completed state", func() {
					err := bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", true, "because i said so", "a result")
					Expect(err).NotTo(HaveOccurred())

					tasks, err := bbs.CompletedTasks(logger)
					Expect(err).NotTo(HaveOccurred())

					Expect(tasks[0].Failed).To(BeTrue())
					Expect(tasks[0].FailureReason).To(Equal("because i said so"))
				})

				It("should bump UpdatedAt", func() {
					clock.IncrementBySeconds(1)

					err := bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", true, "because i said so", "a result")
					Expect(err).NotTo(HaveOccurred())

					tasks, err := bbs.CompletedTasks(logger)
					Expect(err).NotTo(HaveOccurred())

					Expect(tasks[0].UpdatedAt).To(Equal(clock.Now().UnixNano()))
				})

				It("sets FirstCompletedAt", func() {
					clock.IncrementBySeconds(1)

					err := bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", true, "because i said so", "a result")
					Expect(err).NotTo(HaveOccurred())

					tasks, err := bbs.CompletedTasks(logger)
					Expect(err).NotTo(HaveOccurred())

					Expect(tasks[0].FirstCompletedAt).To(Equal(clock.Now().UnixNano()))
				})

				Context("when a receptor is present", func() {
					Context("and completing succeeds", func() {
						BeforeEach(func() {
							fakeTaskClient.CompleteTasksReturns(nil)
						})

						Context("and the task has a complete URL", func() {
							BeforeEach(func() {
								task.CompletionCallbackURL = &url.URL{Host: "bogus"}
							})

							It("completes the task using its address", func() {
								err := bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", true, "because", "a result")
								Expect(err).NotTo(HaveOccurred())

								Expect(fakeTaskClient.CompleteTasksCallCount()).To(Equal(1))
								url, completedTasks := fakeTaskClient.CompleteTasksArgsForCall(0)
								Expect(url).To(Equal(receptorURL))
								Expect(completedTasks).To(HaveLen(1))
								Expect(completedTasks[0].TaskGuid).To(Equal(task.TaskGuid))
								Expect(completedTasks[0].Failed).To(BeTrue())
								Expect(completedTasks[0].FailureReason).To(Equal("because"))
								Expect(completedTasks[0].Result).To(Equal("a result"))
							})
						})

						Context("but the task has no complete URL", func() {
							BeforeEach(func() {
								task.CompletionCallbackURL = nil
							})

							It("does not complete the task via the receptor", func() {
								err := bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", true, "because", "a result")
								Expect(err).NotTo(HaveOccurred())

								Expect(fakeTaskClient.CompleteTasksCallCount()).To(BeZero())
							})
						})
					})

					Context("and completing fails", func() {
						BeforeEach(func() {
							fakeTaskClient.CompleteTasksReturns(errors.New("welp"))
						})

						It("swallows the error, as we'll retry again eventually (via convergence)", func() {
							err := bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", true, "because", "a result")
							Expect(err).NotTo(HaveOccurred())
						})
					})
				})

			})

			Context("when no receptors are present", func() {
				It("swallows the error, as we'll retry again eventually (via convergence)", func() {
					err := bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", true, "because", "a result")
					Expect(err).NotTo(HaveOccurred())
				})
			})
		})

		Context("When completing a Task that is already completed", func() {
			BeforeEach(func() {
				err := bbs.DesireTask(logger, task)
				Expect(err).NotTo(HaveOccurred())

				_, err = bbs.StartTask(logger, task.TaskGuid, "cell-ID")
				Expect(err).NotTo(HaveOccurred())

				err = bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", true, "some failure reason", "")
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns an error", func() {
				err := bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", true, "another failure reason", "")
				Expect(err).To(HaveOccurred())
			})
		})

		Context("When completing a Task that is already completed", func() {
			BeforeEach(func() {
				err := bbs.DesireTask(logger, task)
				Expect(err).NotTo(HaveOccurred())

				_, err = bbs.StartTask(logger, task.TaskGuid, "cell-ID")
				Expect(err).NotTo(HaveOccurred())

				err = bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", true, "some failure reason", "")
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns an error", func() {
				err := bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", true, "another failure reason", "")
				Expect(err).To(HaveOccurred())
			})
		})

		Context("When completing a Task that is resolving", func() {
			BeforeEach(func() {
				err := bbs.DesireTask(logger, task)
				Expect(err).NotTo(HaveOccurred())

				_, err = bbs.StartTask(logger, task.TaskGuid, "cell-ID")
				Expect(err).NotTo(HaveOccurred())

				err = bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", false, "", "result")
				Expect(err).NotTo(HaveOccurred())

				err = bbs.ResolvingTask(logger, task.TaskGuid)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns an error", func() {
				err := bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", false, "", "another result")
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("FailTask", func() {
		BeforeEach(func() {
			task = models.Task{
				TaskGuid:  "some-guid",
				Domain:    "tests",
				RootFS:    "some:rootfs",
				Action:    dummyAction,
				CreatedAt: 1234812,
			}
		})

		Context("when failing a Task", func() {
			Context("when the task is pending", func() {
				JustBeforeEach(func() {
					err := bbs.DesireTask(logger, task)
					Expect(err).NotTo(HaveOccurred())
				})

				It("sets the Task in the completed state", func() {
					err := bbs.FailTask(logger, task.TaskGuid, "because i said so")
					Expect(err).NotTo(HaveOccurred())

					tasks, err := bbs.CompletedTasks(logger)
					Expect(err).NotTo(HaveOccurred())

					Expect(tasks[0].Failed).To(BeTrue())
					Expect(tasks[0].FailureReason).To(Equal("because i said so"))
				})

				It("should bump UpdatedAt", func() {
					clock.IncrementBySeconds(1)

					err := bbs.FailTask(logger, task.TaskGuid, "because i said so")
					Expect(err).NotTo(HaveOccurred())

					tasks, err := bbs.CompletedTasks(logger)
					Expect(err).NotTo(HaveOccurred())

					Expect(tasks[0].UpdatedAt).To(Equal(clock.Now().UnixNano()))
				})

				It("sets FirstCompletedAt", func() {
					clock.IncrementBySeconds(1)

					err := bbs.FailTask(logger, task.TaskGuid, "because i said so")
					Expect(err).NotTo(HaveOccurred())

					tasks, err := bbs.CompletedTasks(logger)
					Expect(err).NotTo(HaveOccurred())

					Expect(tasks[0].FirstCompletedAt).To(Equal(clock.Now().UnixNano()))
				})

				Context("when a receptor is present", func() {
					Context("and failing succeeds", func() {
						BeforeEach(func() {
							fakeTaskClient.CompleteTasksReturns(nil)
						})

						Context("and the task has a complete URL", func() {
							BeforeEach(func() {
								task.CompletionCallbackURL = &url.URL{Host: "bogus"}
							})

							It("completes the task using its address", func() {
								err := bbs.FailTask(logger, task.TaskGuid, "because")
								Expect(err).NotTo(HaveOccurred())

								Expect(fakeTaskClient.CompleteTasksCallCount()).To(Equal(1))
								url, completedTasks := fakeTaskClient.CompleteTasksArgsForCall(0)
								Expect(url).To(Equal(receptorURL))
								Expect(completedTasks).To(HaveLen(1))
								Expect(completedTasks[0].TaskGuid).To(Equal(task.TaskGuid))
								Expect(completedTasks[0].Failed).To(BeTrue())
								Expect(completedTasks[0].FailureReason).To(Equal("because"))
								Expect(completedTasks[0].Result).To(BeEmpty())
							})
						})

						Context("but the task has no complete URL", func() {
							BeforeEach(func() {
								task.CompletionCallbackURL = nil
							})

							It("does not complete the task via the receptor", func() {
								err := bbs.FailTask(logger, task.TaskGuid, "because")
								Expect(err).NotTo(HaveOccurred())

								Expect(fakeTaskClient.CompleteTasksCallCount()).To(BeZero())
							})
						})
					})

					Context("and failing fails", func() {
						BeforeEach(func() {
							fakeTaskClient.CompleteTasksReturns(errors.New("welp"))
						})

						It("swallows the error, as we'll retry again eventually (via convergence)", func() {
							err := bbs.FailTask(logger, task.TaskGuid, "because")
							Expect(err).NotTo(HaveOccurred())
						})
					})
				})

				Context("when no receptors are present", func() {
					It("swallows the error, as we'll retry again eventually (via convergence)", func() {
						err := bbs.FailTask(logger, task.TaskGuid, "because")
						Expect(err).NotTo(HaveOccurred())
					})
				})
			})

			Context("when the task is completed", func() {
				JustBeforeEach(func() {
					err := bbs.DesireTask(logger, task)
					Expect(err).NotTo(HaveOccurred())

					_, err = bbs.StartTask(logger, task.TaskGuid, "some-cell-id")
					Expect(err).NotTo(HaveOccurred())

					err = bbs.CompleteTask(logger, task.TaskGuid, "some-cell-id", true, "because", "some result")
					Expect(err).NotTo(HaveOccurred())
				})

				It("fails", func() {
					err := bbs.FailTask(logger, task.TaskGuid, "because")
					Expect(err).To(HaveOccurred())
				})
			})

			Context("when the task is resolving", func() {
				JustBeforeEach(func() {
					err := bbs.DesireTask(logger, task)
					Expect(err).NotTo(HaveOccurred())

					_, err = bbs.StartTask(logger, task.TaskGuid, "some-cell-id")
					Expect(err).NotTo(HaveOccurred())

					err = bbs.CompleteTask(logger, task.TaskGuid, "some-cell-id", true, "because", "some result")
					Expect(err).NotTo(HaveOccurred())

					err = bbs.ResolvingTask(logger, task.TaskGuid)
					Expect(err).NotTo(HaveOccurred())
				})

				It("fails", func() {
					err := bbs.FailTask(logger, task.TaskGuid, "because")
					Expect(err).To(HaveOccurred())
				})
			})
		})
	})

	Describe("ResolvingTask", func() {
		BeforeEach(func() {
			task = models.Task{
				TaskGuid:  "some-guid",
				Domain:    "tests",
				RootFS:    "some:rootfs",
				Action:    dummyAction,
				CreatedAt: 1234812,
			}
		})

		Context("when the task is complete", func() {
			BeforeEach(func() {
				err := bbs.DesireTask(logger, task)
				Expect(err).NotTo(HaveOccurred())

				_, err = bbs.StartTask(logger, task.TaskGuid, "some-cell-id")
				Expect(err).NotTo(HaveOccurred())

				err = bbs.CompleteTask(logger, task.TaskGuid, "some-cell-id", true, "because i said so", "a result")
				Expect(err).NotTo(HaveOccurred())
			})

			It("swaps /task/<guid>'s state to resolving", func() {
				err := bbs.ResolvingTask(logger, task.TaskGuid)
				Expect(err).NotTo(HaveOccurred())

				tasks, err := bbs.ResolvingTasks(logger)
				Expect(err).NotTo(HaveOccurred())
				Expect(tasks[0].TaskGuid).To(Equal(task.TaskGuid))
				Expect(tasks[0].State).To(Equal(models.TaskStateResolving))
			})

			It("bumps UpdatedAt", func() {
				clock.IncrementBySeconds(1)

				err := bbs.ResolvingTask(logger, task.TaskGuid)
				Expect(err).NotTo(HaveOccurred())

				tasks, err := bbs.ResolvingTasks(logger)
				Expect(err).NotTo(HaveOccurred())
				Expect(tasks[0].UpdatedAt).To(Equal(clock.Now().UnixNano()))
			})

			Context("when the Task is already resolving", func() {
				BeforeEach(func() {
					err := bbs.ResolvingTask(logger, task.TaskGuid)
					Expect(err).NotTo(HaveOccurred())
				})

				It("fails", func() {
					err := bbs.ResolvingTask(logger, task.TaskGuid)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("when the task is not complete", func() {
			BeforeEach(func() {
				err := bbs.DesireTask(logger, task)
				Expect(err).NotTo(HaveOccurred())

				_, err = bbs.StartTask(logger, task.TaskGuid, "some-cell-id")
				Expect(err).NotTo(HaveOccurred())
			})

			It("should fail", func() {
				err := bbs.ResolvingTask(logger, task.TaskGuid)
				Expect(err).To(Equal(bbserrors.NewTaskStateTransitionError(models.TaskStateRunning, models.TaskStateResolving)))
			})
		})
	})

	Describe("ResolveTask", func() {
		BeforeEach(func() {
			task = models.Task{
				TaskGuid:  "some-guid",
				Domain:    "tests",
				RootFS:    "some:rootfs",
				Action:    dummyAction,
				CreatedAt: 1234812,
			}
		})

		Context("when the task is resolving", func() {
			BeforeEach(func() {
				err := bbs.DesireTask(logger, task)
				Expect(err).NotTo(HaveOccurred())

				_, err = bbs.StartTask(logger, task.TaskGuid, "some-cell-id")
				Expect(err).NotTo(HaveOccurred())

				err = bbs.CompleteTask(logger, task.TaskGuid, "some-cell-id", true, "because i said so", "a result")
				Expect(err).NotTo(HaveOccurred())

				err = bbs.ResolvingTask(logger, task.TaskGuid)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should remove /task/<guid>", func() {
				err := bbs.ResolveTask(logger, task.TaskGuid)
				Expect(err).NotTo(HaveOccurred())

				tasks, err := bbs.OldTasks(logger)
				Expect(err).NotTo(HaveOccurred())
				Expect(tasks).To(BeEmpty())
			})
		})

		Context("when the task is not resolving", func() {
			BeforeEach(func() {
				err := bbs.DesireTask(logger, task)
				Expect(err).NotTo(HaveOccurred())

				_, err = bbs.StartTask(logger, task.TaskGuid, "some-cell-id")
				Expect(err).NotTo(HaveOccurred())

				err = bbs.CompleteTask(logger, task.TaskGuid, "some-cell-id", true, "because i said so", "a result")
				Expect(err).NotTo(HaveOccurred())
			})

			It("should fail", func() {
				err := bbs.ResolveTask(logger, task.TaskGuid)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
