package task_bbs_test

import (
	"errors"
	"net/url"
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	. "github.com/cloudfoundry-incubator/runtime-schema/bbs/task_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter/fakestoreadapter"
	"github.com/hashicorp/consul/consul/structs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"
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
						Ω(errDesire).ShouldNot(HaveOccurred())
					})

					It("persists the task", func() {
						persistedTask, err := bbs.TaskByGuid(taskGuid)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(persistedTask.Domain).Should(Equal(domain))
						Ω(persistedTask.RootFS).Should(Equal(rootfs))
						Ω(persistedTask.Action).Should(Equal(dummyAction))
					})

					It("honours the CreatedAt time", func() {
						persistedTask, err := bbs.TaskByGuid(taskGuid)
						Ω(err).ShouldNot(HaveOccurred())
						Ω(persistedTask.CreatedAt).Should(Equal(createdAtTime))
					})

					It("sets the UpdatedAt time", func() {
						persistedTask, err := bbs.TaskByGuid(taskGuid)
						Ω(err).ShouldNot(HaveOccurred())
						Ω(persistedTask.UpdatedAt).Should(Equal(clock.Now().UnixNano()))
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
							Ω(fakeAuctioneerClient.RequestTaskAuctionsCallCount()).Should(Equal(1))

							requestAddress, requestedTasks := fakeAuctioneerClient.RequestTaskAuctionsArgsForCall(0)
							Ω(requestAddress).Should(Equal(auctioneerPresence.AuctioneerAddress))
							Ω(requestedTasks).Should(HaveLen(1))
							Ω(requestedTasks[0].TaskGuid).Should(Equal(taskGuid))
						})

						Context("when requesting a task auction succeeds", func() {
							BeforeEach(func() {
								fakeAuctioneerClient.RequestTaskAuctionsReturns(nil)
							})

							It("does not return an error", func() {
								Ω(errDesire).ShouldNot(HaveOccurred())
							})
						})

						Context("when requesting a task auction fails", func() {
							BeforeEach(func() {
								fakeAuctioneerClient.RequestTaskAuctionsReturns(errors.New("oops"))
							})

							It("does not return an error", func() {
								// The creation succeeded, we can ignore the auction request error (converger will eventually do it)
								Ω(errDesire).ShouldNot(HaveOccurred())
							})
						})
					})

					Context("when unable to fetch the Auctioneer address", func() {
						It("does not request an auction", func() {
							Consistently(fakeAuctioneerClient.RequestTaskAuctionsCallCount).Should(BeZero())
						})

						It("does not return an error", func() {
							// The creation succeeded, we can ignore the auction request error (converger will eventually do it)
							Ω(errDesire).ShouldNot(HaveOccurred())
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
						Ω(errDesire).ShouldNot(HaveOccurred())
					})

					It("persists the task", func() {
						persistedTask, err := bbs.TaskByGuid(taskGuid)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(persistedTask.Domain).Should(Equal(domain))
						Ω(persistedTask.RootFS).Should(Equal(rootfs))
						Ω(persistedTask.Action).Should(Equal(dummyAction))
					})

					It("provides a CreatedAt time", func() {
						persistedTask, err := bbs.TaskByGuid(taskGuid)
						Ω(err).ShouldNot(HaveOccurred())
						Ω(persistedTask.CreatedAt).Should(Equal(clock.Now().UnixNano()))
					})

					It("sets the UpdatedAt time", func() {
						persistedTask, err := bbs.TaskByGuid(taskGuid)
						Ω(err).ShouldNot(HaveOccurred())
						Ω(persistedTask.UpdatedAt).Should(Equal(clock.Now().UnixNano()))
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
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("does not persist another", func() {
					Consistently(
						func() ([]models.Task, error) {
							return bbs.Tasks(logger)
						}).Should(HaveLen(1))
				})

				It("does not request an auction", func() {
					Consistently(fakeAuctioneerClient.RequestTaskAuctionsCallCount).Should(BeZero())
				})

				It("returns an error", func() {
					Ω(errDesire).Should(Equal(bbserrors.ErrStoreResourceExists))
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
					return bbs.Tasks(logger)
				}).Should(BeEmpty())
			})

			It("does not request an auction", func() {
				Consistently(fakeAuctioneerClient.RequestTaskAuctionsCallCount).Should(BeZero())
			})

			It("returns an error", func() {
				Ω(errDesire).Should(ContainElement(models.ErrInvalidField{"domain"}))
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
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("sets the state to running", func() {
				started, err := bbs.StartTask(logger, task.TaskGuid, "cell-ID")
				Ω(err).ShouldNot(HaveOccurred())
				Ω(started).Should(BeTrue())

				tasks, err := bbs.RunningTasks(logger)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(tasks[0].TaskGuid).Should(Equal(task.TaskGuid))
				Ω(tasks[0].State).Should(Equal(models.TaskStateRunning))
			})

			It("should bump UpdatedAt", func() {
				clock.IncrementBySeconds(1)

				started, err := bbs.StartTask(logger, task.TaskGuid, "cell-ID")
				Ω(err).ShouldNot(HaveOccurred())
				Ω(started).Should(BeTrue())

				tasks, err := bbs.RunningTasks(logger)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(tasks[0].UpdatedAt).Should(Equal(clock.Now().UnixNano()))
			})
		})

		Context("When starting a Task that is already started", func() {
			var changed bool

			BeforeEach(func() {
				err := bbs.DesireTask(logger, task)
				Ω(err).ShouldNot(HaveOccurred())

				_, err = bbs.StartTask(logger, task.TaskGuid, "cell-ID")
				Ω(err).ShouldNot(HaveOccurred())
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
					Ω(startErr).ShouldNot(HaveOccurred())
				})

				It("returns false", func() {
					Ω(changed).Should(BeFalse())
				})

				It("does not change the Task in the store", func() {
					task, err := bbs.TaskByGuid(task.TaskGuid)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(task.UpdatedAt).Should(Equal(previousTime))
				})
			})

			Context("on another cell", func() {
				It("returns an error", func() {
					changed, err := bbs.StartTask(logger, task.TaskGuid, "another-cell-ID")
					Ω(err).Should(HaveOccurred())
					Ω(changed).Should(BeFalse())
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
					Ω(taskAfterCancel.UpdatedAt).Should(Equal(clock.Now().UnixNano()))
				})
			}

			Context("when the task is in pending state", func() {
				BeforeEach(func() {
					err := bbs.DesireTask(logger, task)
					Ω(err).ShouldNot(HaveOccurred())
				})

				itMarksTaskAsCancelled()

				It("does not cancel the task", func() {
					Ω(fakeCellClient.CancelTaskCallCount()).Should(Equal(0))
				})
			})

			Context("when the task is in running state", func() {
				var cellID string

				BeforeEach(func() {
					cellID = "cell-ID"

					err := bbs.DesireTask(logger, task)
					Ω(err).ShouldNot(HaveOccurred())

					_, err = bbs.StartTask(logger, task.TaskGuid, cellID)
					Ω(err).ShouldNot(HaveOccurred())
				})

				itMarksTaskAsCancelled()

				Context("when the cell is present", func() {
					var cellPresence models.CellPresence

					BeforeEach(func() {
						cellPresence = models.NewCellPresence(cellID, "cell.example.com", "the-zone", models.NewCellCapacity(128, 1024, 6))
						registerCell(cellPresence)
					})

					It("cancels the task", func() {
						Ω(fakeCellClient.CancelTaskCallCount()).Should(Equal(1))

						addr, taskGuid := fakeCellClient.CancelTaskArgsForCall(0)
						Ω(addr).Should(Equal(cellPresence.RepAddress))
						Ω(taskGuid).Should(Equal(task.TaskGuid))
					})
				})

				Context("when the cell is not present", func() {
					It("does not cancel the task", func() {
						Ω(fakeCellClient.CancelTaskCallCount()).Should(Equal(0))
					})

					It("logs the error", func() {
						Eventually(logger.TestSink.LogMessages).Should(ContainElement("test.cancel-task.failed-to-cancel-immediately"))
					})
				})
			})

			Context("when the task is in completed state", func() {
				BeforeEach(func() {
					err := bbs.DesireTask(logger, task)
					Ω(err).ShouldNot(HaveOccurred())

					_, err = bbs.StartTask(logger, task.TaskGuid, "cell-ID")
					Ω(err).ShouldNot(HaveOccurred())

					err = bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", false, "", "")
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("returns an error", func() {
					Ω(cancelError).Should(HaveOccurred())
					Ω(cancelError).Should(Equal(bbserrors.NewTaskStateTransitionError(models.TaskStateCompleted, models.TaskStateCompleted)))
				})
			})

			Context("when the task is in resolving state", func() {
				BeforeEach(func() {
					err := bbs.DesireTask(logger, task)
					Ω(err).ShouldNot(HaveOccurred())

					_, err = bbs.StartTask(logger, task.TaskGuid, "cell-ID")
					Ω(err).ShouldNot(HaveOccurred())

					err = bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", false, "", "")
					Ω(err).ShouldNot(HaveOccurred())

					err = bbs.ResolvingTask(logger, task.TaskGuid)
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("returns an error", func() {
					Ω(cancelError).Should(HaveOccurred())
					Ω(cancelError).Should(Equal(bbserrors.NewTaskStateTransitionError(models.TaskStateResolving, models.TaskStateCompleted)))
				})
			})

			Context("when the task does not exist", func() {
				It("returns an error", func() {
					Ω(cancelError).Should(HaveOccurred())
					Ω(cancelError).Should(Equal(bbserrors.ErrStoreResourceNotFound))
				})
			})

			Context("when the store returns some error other than key not found or timeout", func() {
				var storeError = errors.New("store error")

				BeforeEach(func() {
					fakeStoreAdapter := fakestoreadapter.New()
					fakeStoreAdapter.GetErrInjector = fakestoreadapter.NewFakeStoreAdapterErrorInjector(``, storeError)

					bbs = New(fakeStoreAdapter, consulAdapter, clock, fakeTaskClient, fakeAuctioneerClient, fakeCellClient, servicesBBS)
				})

				It("returns an error", func() {
					Ω(cancelError).Should(HaveOccurred())
					Ω(cancelError).Should(Equal(storeError))
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
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("returns an error", func() {
				err := bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", true, "another failure reason", "")
				Ω(err).Should(HaveOccurred())
			})
		})

		Context("when completing a running Task", func() {
			JustBeforeEach(func() {
				err := bbs.DesireTask(logger, task)
				Ω(err).ShouldNot(HaveOccurred())

				_, err = bbs.StartTask(logger, task.TaskGuid, "cell-ID")
				Ω(err).ShouldNot(HaveOccurred())
			})

			Context("when the cell id is not the same", func() {
				It("returns an error", func() {
					err := bbs.CompleteTask(logger, task.TaskGuid, "another-cell-ID", true, "because i said so", "a result")
					Ω(err).Should(Equal(bbserrors.ErrTaskRunningOnDifferentCell))
				})
			})

			Context("when the cell id is the same", func() {
				It("sets the Task in the completed state", func() {
					err := bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", true, "because i said so", "a result")
					Ω(err).ShouldNot(HaveOccurred())

					tasks, err := bbs.CompletedTasks(logger)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(tasks[0].Failed).Should(BeTrue())
					Ω(tasks[0].FailureReason).Should(Equal("because i said so"))
				})

				It("should bump UpdatedAt", func() {
					clock.IncrementBySeconds(1)

					err := bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", true, "because i said so", "a result")
					Ω(err).ShouldNot(HaveOccurred())

					tasks, err := bbs.CompletedTasks(logger)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(tasks[0].UpdatedAt).Should(Equal(clock.Now().UnixNano()))
				})

				It("sets FirstCompletedAt", func() {
					clock.IncrementBySeconds(1)

					err := bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", true, "because i said so", "a result")
					Ω(err).ShouldNot(HaveOccurred())

					tasks, err := bbs.CompletedTasks(logger)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(tasks[0].FirstCompletedAt).Should(Equal(clock.Now().UnixNano()))
				})

				Context("when a receptor is present", func() {
					var receptorPresence ifrit.Process

					BeforeEach(func() {
						presence := models.ReceptorPresence{
							ReceptorID:  "some-receptor",
							ReceptorURL: "some-receptor-url",
						}

						heartbeat := servicesBBS.NewReceptorHeartbeat(presence, structs.SessionTTLMin, 100*time.Millisecond)

						receptorPresence = ifrit.Invoke(heartbeat)
					})

					AfterEach(func() {
						ginkgomon.Interrupt(receptorPresence)
					})

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
								Ω(err).ShouldNot(HaveOccurred())

								Ω(fakeTaskClient.CompleteTasksCallCount()).Should(Equal(1))
								receptorURL, completedTasks := fakeTaskClient.CompleteTasksArgsForCall(0)
								Ω(receptorURL).Should(Equal("some-receptor-url"))
								Ω(completedTasks).Should(HaveLen(1))
								Ω(completedTasks[0].TaskGuid).Should(Equal(task.TaskGuid))
								Ω(completedTasks[0].Failed).Should(BeTrue())
								Ω(completedTasks[0].FailureReason).Should(Equal("because"))
								Ω(completedTasks[0].Result).Should(Equal("a result"))
							})
						})

						Context("but the task has no complete URL", func() {
							BeforeEach(func() {
								task.CompletionCallbackURL = nil
							})

							It("does not complete the task via the receptor", func() {
								err := bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", true, "because", "a result")
								Ω(err).ShouldNot(HaveOccurred())

								Ω(fakeTaskClient.CompleteTasksCallCount()).Should(BeZero())
							})
						})
					})

					Context("and completing fails", func() {
						BeforeEach(func() {
							fakeTaskClient.CompleteTasksReturns(errors.New("welp"))
						})

						It("swallows the error, as we'll retry again eventually (via convergence)", func() {
							err := bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", true, "because", "a result")
							Ω(err).ShouldNot(HaveOccurred())
						})
					})
				})

			})

			Context("when no receptors are present", func() {
				It("swallows the error, as we'll retry again eventually (via convergence)", func() {
					err := bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", true, "because", "a result")
					Ω(err).ShouldNot(HaveOccurred())
				})
			})
		})

		Context("When completing a Task that is already completed", func() {
			BeforeEach(func() {
				err := bbs.DesireTask(logger, task)
				Ω(err).ShouldNot(HaveOccurred())

				_, err = bbs.StartTask(logger, task.TaskGuid, "cell-ID")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", true, "some failure reason", "")
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("returns an error", func() {
				err := bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", true, "another failure reason", "")
				Ω(err).Should(HaveOccurred())
			})
		})

		Context("When completing a Task that is already completed", func() {
			BeforeEach(func() {
				err := bbs.DesireTask(logger, task)
				Ω(err).ShouldNot(HaveOccurred())

				_, err = bbs.StartTask(logger, task.TaskGuid, "cell-ID")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", true, "some failure reason", "")
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("returns an error", func() {
				err := bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", true, "another failure reason", "")
				Ω(err).Should(HaveOccurred())
			})
		})

		Context("When completing a Task that is resolving", func() {
			BeforeEach(func() {
				err := bbs.DesireTask(logger, task)
				Ω(err).ShouldNot(HaveOccurred())

				_, err = bbs.StartTask(logger, task.TaskGuid, "cell-ID")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", false, "", "result")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.ResolvingTask(logger, task.TaskGuid)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("returns an error", func() {
				err := bbs.CompleteTask(logger, task.TaskGuid, "cell-ID", false, "", "another result")
				Ω(err).Should(HaveOccurred())
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
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("sets the Task in the completed state", func() {
					err := bbs.FailTask(logger, task.TaskGuid, "because i said so")
					Ω(err).ShouldNot(HaveOccurred())

					tasks, err := bbs.CompletedTasks(logger)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(tasks[0].Failed).Should(BeTrue())
					Ω(tasks[0].FailureReason).Should(Equal("because i said so"))
				})

				It("should bump UpdatedAt", func() {
					clock.IncrementBySeconds(1)

					err := bbs.FailTask(logger, task.TaskGuid, "because i said so")
					Ω(err).ShouldNot(HaveOccurred())

					tasks, err := bbs.CompletedTasks(logger)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(tasks[0].UpdatedAt).Should(Equal(clock.Now().UnixNano()))
				})

				It("sets FirstCompletedAt", func() {
					clock.IncrementBySeconds(1)

					err := bbs.FailTask(logger, task.TaskGuid, "because i said so")
					Ω(err).ShouldNot(HaveOccurred())

					tasks, err := bbs.CompletedTasks(logger)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(tasks[0].FirstCompletedAt).Should(Equal(clock.Now().UnixNano()))
				})

				Context("when a receptor is present", func() {
					var receptorPresence ifrit.Process

					BeforeEach(func() {
						presence := models.ReceptorPresence{
							ReceptorID:  "some-receptor",
							ReceptorURL: "some-receptor-url",
						}

						heartbeat := servicesBBS.NewReceptorHeartbeat(presence, structs.SessionTTLMin, 100*time.Millisecond)

						receptorPresence = ifrit.Invoke(heartbeat)
					})

					AfterEach(func() {
						ginkgomon.Interrupt(receptorPresence)
					})

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
								Ω(err).ShouldNot(HaveOccurred())

								Ω(fakeTaskClient.CompleteTasksCallCount()).Should(Equal(1))
								receptorURL, completedTasks := fakeTaskClient.CompleteTasksArgsForCall(0)
								Ω(receptorURL).Should(Equal("some-receptor-url"))
								Ω(completedTasks).Should(HaveLen(1))
								Ω(completedTasks[0].TaskGuid).Should(Equal(task.TaskGuid))
								Ω(completedTasks[0].Failed).Should(BeTrue())
								Ω(completedTasks[0].FailureReason).Should(Equal("because"))
								Ω(completedTasks[0].Result).Should(BeEmpty())
							})
						})

						Context("but the task has no complete URL", func() {
							BeforeEach(func() {
								task.CompletionCallbackURL = nil
							})

							It("does not complete the task via the receptor", func() {
								err := bbs.FailTask(logger, task.TaskGuid, "because")
								Ω(err).ShouldNot(HaveOccurred())

								Ω(fakeTaskClient.CompleteTasksCallCount()).Should(BeZero())
							})
						})
					})

					Context("and failing fails", func() {
						BeforeEach(func() {
							fakeTaskClient.CompleteTasksReturns(errors.New("welp"))
						})

						It("swallows the error, as we'll retry again eventually (via convergence)", func() {
							err := bbs.FailTask(logger, task.TaskGuid, "because")
							Ω(err).ShouldNot(HaveOccurred())
						})
					})
				})

				Context("when no receptors are present", func() {
					It("swallows the error, as we'll retry again eventually (via convergence)", func() {
						err := bbs.FailTask(logger, task.TaskGuid, "because")
						Ω(err).ShouldNot(HaveOccurred())
					})
				})
			})

			Context("when the task is completed", func() {
				JustBeforeEach(func() {
					err := bbs.DesireTask(logger, task)
					Ω(err).ShouldNot(HaveOccurred())

					_, err = bbs.StartTask(logger, task.TaskGuid, "some-cell-id")
					Ω(err).ShouldNot(HaveOccurred())

					err = bbs.CompleteTask(logger, task.TaskGuid, "some-cell-id", true, "because", "some result")
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("fails", func() {
					err := bbs.FailTask(logger, task.TaskGuid, "because")
					Ω(err).Should(HaveOccurred())
				})
			})

			Context("when the task is resolving", func() {
				JustBeforeEach(func() {
					err := bbs.DesireTask(logger, task)
					Ω(err).ShouldNot(HaveOccurred())

					_, err = bbs.StartTask(logger, task.TaskGuid, "some-cell-id")
					Ω(err).ShouldNot(HaveOccurred())

					err = bbs.CompleteTask(logger, task.TaskGuid, "some-cell-id", true, "because", "some result")
					Ω(err).ShouldNot(HaveOccurred())

					err = bbs.ResolvingTask(logger, task.TaskGuid)
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("fails", func() {
					err := bbs.FailTask(logger, task.TaskGuid, "because")
					Ω(err).Should(HaveOccurred())
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
				Ω(err).ShouldNot(HaveOccurred())

				_, err = bbs.StartTask(logger, task.TaskGuid, "some-cell-id")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.CompleteTask(logger, task.TaskGuid, "some-cell-id", true, "because i said so", "a result")
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("swaps /task/<guid>'s state to resolving", func() {
				err := bbs.ResolvingTask(logger, task.TaskGuid)
				Ω(err).ShouldNot(HaveOccurred())

				tasks, err := bbs.ResolvingTasks(logger)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(tasks[0].TaskGuid).Should(Equal(task.TaskGuid))
				Ω(tasks[0].State).Should(Equal(models.TaskStateResolving))
			})

			It("bumps UpdatedAt", func() {
				clock.IncrementBySeconds(1)

				err := bbs.ResolvingTask(logger, task.TaskGuid)
				Ω(err).ShouldNot(HaveOccurred())

				tasks, err := bbs.ResolvingTasks(logger)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(tasks[0].UpdatedAt).Should(Equal(clock.Now().UnixNano()))
			})

			Context("when the Task is already resolving", func() {
				BeforeEach(func() {
					err := bbs.ResolvingTask(logger, task.TaskGuid)
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("fails", func() {
					err := bbs.ResolvingTask(logger, task.TaskGuid)
					Ω(err).Should(HaveOccurred())
				})
			})
		})

		Context("when the task is not complete", func() {
			BeforeEach(func() {
				err := bbs.DesireTask(logger, task)
				Ω(err).ShouldNot(HaveOccurred())

				_, err = bbs.StartTask(logger, task.TaskGuid, "some-cell-id")
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("should fail", func() {
				err := bbs.ResolvingTask(logger, task.TaskGuid)
				Ω(err).Should(Equal(bbserrors.NewTaskStateTransitionError(models.TaskStateRunning, models.TaskStateResolving)))
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
				Ω(err).ShouldNot(HaveOccurred())

				_, err = bbs.StartTask(logger, task.TaskGuid, "some-cell-id")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.CompleteTask(logger, task.TaskGuid, "some-cell-id", true, "because i said so", "a result")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.ResolvingTask(logger, task.TaskGuid)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("should remove /task/<guid>", func() {
				err := bbs.ResolveTask(logger, task.TaskGuid)
				Ω(err).ShouldNot(HaveOccurred())

				tasks, err := bbs.Tasks(logger)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(tasks).Should(BeEmpty())
			})
		})

		Context("when the task is not resolving", func() {
			BeforeEach(func() {
				err := bbs.DesireTask(logger, task)
				Ω(err).ShouldNot(HaveOccurred())

				_, err = bbs.StartTask(logger, task.TaskGuid, "some-cell-id")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.CompleteTask(logger, task.TaskGuid, "some-cell-id", true, "because i said so", "a result")
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("should fail", func() {
				err := bbs.ResolveTask(logger, task.TaskGuid)
				Ω(err).Should(HaveOccurred())
			})
		})
	})
})
