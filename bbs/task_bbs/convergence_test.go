package task_bbs_test

import (
	"errors"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/dropsonde/metric_sender/fake"
	"github.com/cloudfoundry/dropsonde/metrics"
	"github.com/cloudfoundry/storeadapter"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
)

var _ = Describe("Convergence of Tasks", func() {
	var (
		sender *fake.FakeMetricSender

		task                                                                      models.Task
		kickTasksDurationInSeconds, expirePendingTaskDurationInSeconds            uint64
		kickTasksDuration, expirePendingTaskDuration, expireCompletedTaskDuration time.Duration
	)

	BeforeEach(func() {
		sender = fake.NewFakeMetricSender()
		metrics.Initialize(sender, nil)

		kickTasksDurationInSeconds = 10
		kickTasksDuration = time.Duration(kickTasksDurationInSeconds) * time.Second
		expirePendingTaskDurationInSeconds = 30
		expirePendingTaskDuration = time.Duration(expirePendingTaskDurationInSeconds) * time.Second
		expireCompletedTaskDuration = time.Hour

		task = models.Task{
			Domain:   "tests",
			TaskGuid: "some-guid",
			RootFS:   "some:rootfs",
			Action:   dummyAction,
		}
	})

	Describe("ConvergeTasks", func() {
		JustBeforeEach(func() {
			bbs.ConvergeTasks(logger, kickTasksDuration, expirePendingTaskDuration, expireCompletedTaskDuration, servicesBBS.NewCellsLoader())
		})

		It("bumps the convergence counter", func() {
			Expect(sender.GetCounter("ConvergenceTaskRuns")).To(Equal(uint64(1)))
		})

		It("reports the duration that it took to converge", func() {
			reportedDuration := sender.GetValue("ConvergenceTaskDuration")
			Expect(reportedDuration.Unit).To(Equal("nanos"))
			Expect(reportedDuration.Value).NotTo(BeZero())
		})

		Context("when a Task is malformed", func() {
			var nodeKey string

			BeforeEach(func() {
				nodeKey = path.Join(shared.TaskSchemaRoot, "some-guid")

				err := etcdClient.Create(storeadapter.StoreNode{
					Key:   nodeKey,
					Value: []byte("ß"),
				})
				Expect(err).NotTo(HaveOccurred())

				_, err = etcdClient.Get(nodeKey)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should delete it", func() {
				_, err := etcdClient.Get(nodeKey)
				Expect(err).To(Equal(storeadapter.ErrorKeyNotFound))
			})

			It("bumps the pruned counter", func() {
				Expect(sender.GetCounter("ConvergenceTasksPruned")).To(Equal(uint64(1)))
			})
		})

		Context("when Tasks are pending", func() {
			var secondTask models.Task

			BeforeEach(func() {
				err := bbs.DesireTask(logger, task)
				Expect(err).NotTo(HaveOccurred())

				secondTask = models.Task{
					Domain:                "tests",
					TaskGuid:              "some-other-guid",
					RootFS:                "some:rootfs",
					Action:                dummyAction,
					CompletionCallbackURL: &url.URL{Host: "blah"},
				}

				err = bbs.DesireTask(logger, secondTask)
				Expect(err).NotTo(HaveOccurred())
			})

			Context("when the Task has NOT been pending for too long", func() {
				BeforeEach(func() {
					clock.IncrementBySeconds(kickTasksDurationInSeconds - 1)

					auctioneerPresence := models.AuctioneerPresence{
						AuctioneerID:      "the-auctioneer-id",
						AuctioneerAddress: "the-address",
					}

					registerAuctioneer(auctioneerPresence)
				})

				It("does not request an auction for the task", func() {
					Consistently(fakeAuctioneerClient.RequestTaskAuctionsCallCount).Should(BeZero())
				})
			})

			Context("when the Tasks have been pending for longer than the kick interval", func() {
				BeforeEach(func() {
					clock.IncrementBySeconds(kickTasksDurationInSeconds + 1)
				})

				It("bumps the compare-and-swap counter", func() {
					Expect(sender.GetCounter("ConvergenceTasksKicked")).To(Equal(uint64(2)))
				})

				It("logs that it sends an auction for the pending task", func() {
					Expect(logger.TestSink.LogMessages()).To(ContainElement("test.converge-tasks.requesting-auction-for-pending-task"))
				})

				Context("when able to fetch the auctioneer address", func() {
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
						Expect(requestedTasks).To(HaveLen(2))
						Expect([]string{requestedTasks[0].TaskGuid, requestedTasks[1].TaskGuid}).To(ConsistOf(task.TaskGuid, secondTask.TaskGuid))
					})

					Context("when requesting an auction is unsuccessful", func() {
						BeforeEach(func() {
							fakeAuctioneerClient.RequestTaskAuctionsReturns(errors.New("oops"))
						})

						It("logs an error", func() {
							Expect(logger.TestSink.LogMessages()).To(ContainElement("test.converge-tasks.failed-to-request-auctions-for-pending-tasks"))
						})
					})
				})

				Context("when unable to fetch the auctioneer address", func() {
					It("logs an error", func() {
						Expect(logger.TestSink.LogMessages()).To(ContainElement("test.converge-tasks.failed-to-request-auctions-for-pending-tasks"))
					})
				})
			})

			Context("when the Task has been pending for longer than the expirePendingTasksDuration", func() {
				BeforeEach(func() {
					clock.IncrementBySeconds(expirePendingTaskDurationInSeconds + 1)
				})

				It("should mark the Task as completed & failed", func() {
					returnedTask, err := bbs.TaskByGuid(logger, task.TaskGuid)
					Expect(err).NotTo(HaveOccurred())
					Expect(returnedTask.State).To(Equal(models.TaskStateCompleted))

					Expect(returnedTask.Failed).To(Equal(true))
					Expect(returnedTask.FailureReason).To(ContainSubstring("time limit"))
				})

				It("bumps the compare-and-swap counter", func() {
					Expect(sender.GetCounter("ConvergenceTasksKicked")).To(Equal(uint64(2)))
				})

				It("logs an error", func() {
					Expect(logger.TestSink.LogMessages()).To(ContainElement("test.converge-tasks.failed-to-start-in-time"))
				})
			})
		})

		Context("when a Task is running", func() {
			BeforeEach(func() {
				err := bbs.DesireTask(logger, task)
				Expect(err).NotTo(HaveOccurred())

				_, err = bbs.StartTask(logger, task.TaskGuid, "cell-id")
				Expect(err).NotTo(HaveOccurred())
			})

			Context("when the associated cell is present", func() {
				var heartbeater ifrit.Process

				BeforeEach(func() {
					cellPresence := models.NewCellPresence("cell-id", "1.2.3.4", "the-zone", models.NewCellCapacity(128, 1024, 3))
					heartbeater = ifrit.Invoke(servicesBBS.NewCellPresence(cellPresence, 100*time.Millisecond))

					Eventually(func() map[string][]byte {
						cells, _ := consulSession.ListAcquiredValues(shared.CellSchemaRoot)
						return cells
					}, 1, 50*time.Millisecond).Should(HaveLen(1))
				})

				AfterEach(func() {
					heartbeater.Signal(os.Interrupt)
				})

				It("leaves the task running", func() {
					returnedTask, err := bbs.TaskByGuid(logger, task.TaskGuid)
					Expect(err).NotTo(HaveOccurred())
					Expect(returnedTask.State).To(Equal(models.TaskStateRunning))
				})
			})

			Context("when the associated cell is missing", func() {
				It("should mark the Task as completed & failed", func() {
					returnedTask, err := bbs.TaskByGuid(logger, task.TaskGuid)
					Expect(err).NotTo(HaveOccurred())
					Expect(returnedTask.State).To(Equal(models.TaskStateCompleted))

					Expect(returnedTask.Failed).To(Equal(true))
					Expect(returnedTask.FailureReason).To(ContainSubstring("cell"))
				})

				It("logs that the cell disappeared", func() {
					Expect(logger.TestSink.LogMessages()).To(ContainElement("test.converge-tasks.cell-disappeared"))
				})

				It("bumps the compare-and-swap counter", func() {
					Expect(sender.GetCounter("ConvergenceTasksKicked")).To(Equal(uint64(1)))
				})
			})
		})

		Describe("Completed tasks", func() {
			Context("when Tasks with a complete URL are completed", func() {
				var completeTaskError error

				BeforeEach(func() {
					task.CompletionCallbackURL = &url.URL{Host: "blah"}

					err := bbs.DesireTask(logger, task)
					Expect(err).NotTo(HaveOccurred())

					_, err = bbs.StartTask(logger, task.TaskGuid, "cell-id")
					Expect(err).NotTo(HaveOccurred())

					err = bbs.CompleteTask(logger, task.TaskGuid, "cell-id", true, "'cause I said so", "a magical result")
					Expect(err).NotTo(HaveOccurred())

					secondTask := models.Task{
						Domain:                "tests",
						TaskGuid:              "some-other-guid",
						RootFS:                "some:rootfs",
						Action:                dummyAction,
						CompletionCallbackURL: &url.URL{Host: "blah"},
					}

					err = bbs.DesireTask(logger, secondTask)
					Expect(err).NotTo(HaveOccurred())

					_, err = bbs.StartTask(logger, secondTask.TaskGuid, "cell-id")
					Expect(err).NotTo(HaveOccurred())

					err = bbs.CompleteTask(logger, secondTask.TaskGuid, "cell-id", true, "'cause I said so", "a magical result")
					Expect(err).NotTo(HaveOccurred())

					completeTaskError = nil

					fakeTaskClient.CompleteTasksStub = func(string, []models.Task) error {
						return completeTaskError
					}
				})

				Context("for longer than the convergence interval", func() {
					BeforeEach(func() {
						clock.IncrementBySeconds(expirePendingTaskDurationInSeconds + 1)
					})

					Context("when a receptor is present", func() {
						It("submits the completed tasks to the receptor in batch", func() {
							Expect(fakeTaskClient.CompleteTasksCallCount()).To(Equal(3)) // 2 initial completes + convergence

							url, completedTasks := fakeTaskClient.CompleteTasksArgsForCall(2)
							Expect(url).To(Equal(receptorURL))
							Expect(completedTasks).To(HaveLen(2))

							firstCompletedTask := completedTasks[0]
							Expect(firstCompletedTask.Failed).To(BeTrue())
							Expect(firstCompletedTask.FailureReason).To(Equal("'cause I said so"))
							Expect(firstCompletedTask.Result).To(Equal("a magical result"))

							secondCompletedTask := completedTasks[1]
							Expect(secondCompletedTask.Failed).To(BeTrue())
							Expect(secondCompletedTask.FailureReason).To(Equal("'cause I said so"))
							Expect(secondCompletedTask.Result).To(Equal("a magical result"))

							Expect([]string{firstCompletedTask.TaskGuid, secondCompletedTask.TaskGuid}).To(ConsistOf(
								[]string{"some-guid", "some-other-guid"},
							))

						})

						It("logs that it kicks the completed task", func() {
							Expect(logger.TestSink.LogMessages()).To(ContainElement("test.converge-tasks.kicking-completed-task"))
						})

						It("bumps the convergence tasks kicked counter", func() {
							Expect(sender.GetCounter("ConvergenceTasksKicked")).To(Equal(uint64(2)))
						})

						Context("when the receptor fails to complete the task", func() {
							BeforeEach(func() {
								completeTaskError = errors.New("whoops!")
							})

							It("logs that it failed to complete the task", func() {
								Expect(logger.TestSink.LogMessages()).To(ContainElement("test.converge-tasks.failed-to-complete-tasks"))
							})
						})
					})
				})
			})

			Context("when a completed task without a complete URL is present", func() {
				BeforeEach(func() {
					err := bbs.DesireTask(logger, task)
					Expect(err).NotTo(HaveOccurred())

					_, err = bbs.StartTask(logger, task.TaskGuid, "cell-id")
					Expect(err).NotTo(HaveOccurred())

					err = bbs.CompleteTask(logger, task.TaskGuid, "cell-id", true, "'cause I said so", "a magical result")
					Expect(err).NotTo(HaveOccurred())
				})

				Context("for longer than the convergence interval", func() {
					BeforeEach(func() {
						clock.IncrementBySeconds(expirePendingTaskDurationInSeconds + 1)
					})

					Context("when a receptor is present", func() {
						It("does not submit the completed task to the receptor", func() {
							Expect(fakeTaskClient.CompleteTasksCallCount()).To(BeZero())
						})
					})

					It("bumps the convergence tasks kicked counter", func() {
						Expect(sender.GetCounter("ConvergenceTasksKicked")).To(Equal(uint64(1)))
					})
				})

				Context("when the task has been completed for longer than the time-to-resolve interval", func() {
					BeforeEach(func() {
						clock.IncrementBySeconds(uint64(expireCompletedTaskDuration.Seconds()) + 1)
					})

					It("should delete the task", func() {
						_, err := bbs.TaskByGuid(logger, task.TaskGuid)
						Expect(err).To(Equal(bbserrors.ErrStoreResourceNotFound))
					})

					It("logs that it failed to start resolving the task in time", func() {
						Expect(logger.TestSink.LogMessages()).To(ContainElement("test.converge-tasks.failed-to-start-resolving-in-time"))
					})
				})

				Context("when the task has been completed for less than the convergence interval", func() {
					var previousTime int64

					BeforeEach(func() {
						previousTime = clock.Now().UnixNano()
						clock.IncrementBySeconds(1)
					})

					It("should NOT kick the Task", func() {
						returnedTask, err := bbs.TaskByGuid(logger, task.TaskGuid)
						Expect(err).NotTo(HaveOccurred())
						Expect(returnedTask.State).To(Equal(models.TaskStateCompleted))
						Expect(returnedTask.UpdatedAt).To(Equal(previousTime))
					})
				})
			})
		})

		Context("when a Task is resolving", func() {
			BeforeEach(func() {
				task.CompletionCallbackURL = &url.URL{Host: "blah"}

				err := bbs.DesireTask(logger, task)
				Expect(err).NotTo(HaveOccurred())

				_, err = bbs.StartTask(logger, task.TaskGuid, "cell-id")
				Expect(err).NotTo(HaveOccurred())

				err = bbs.CompleteTask(logger, task.TaskGuid, "cell-id", true, "'cause I said so", "a result")
				Expect(err).NotTo(HaveOccurred())

				err = bbs.ResolvingTask(logger, task.TaskGuid)
				Expect(err).NotTo(HaveOccurred())
			})

			Context("when the task is in resolving state for less than the convergence interval", func() {
				var previousTime int64

				BeforeEach(func() {
					previousTime = clock.Now().UnixNano()
					clock.IncrementBySeconds(1)
				})

				It("should do nothing", func() {
					Expect(fakeTaskClient.CompleteTasksCallCount()).To(Equal(1))

					returnedTask, err := bbs.TaskByGuid(logger, task.TaskGuid)
					Expect(err).NotTo(HaveOccurred())
					Expect(returnedTask.State).To(Equal(models.TaskStateResolving))
					Expect(returnedTask.UpdatedAt).To(Equal(previousTime))
				})
			})

			Context("when the task has been resolving for longer than a convergence interval", func() {
				BeforeEach(func() {
					clock.IncrementBySeconds(expirePendingTaskDurationInSeconds)
				})

				It("should put the Task back into the completed state", func() {
					returnedTask, err := bbs.TaskByGuid(logger, task.TaskGuid)
					Expect(err).NotTo(HaveOccurred())
					Expect(returnedTask.State).To(Equal(models.TaskStateCompleted))
					Expect(returnedTask.UpdatedAt).To(Equal(clock.Now().UnixNano()))
				})

				It("logs that it is demoting task from resolving to completed", func() {
					Expect(logger.TestSink.LogMessages()).To(ContainElement("test.converge-tasks.demoting-resolving-to-completed"))
				})

				Context("when a receptor is present", func() {
					It("submits the completed task to the receptor", func() {
						Expect(fakeTaskClient.CompleteTasksCallCount()).To(Equal(2))

						url, completedTasks := fakeTaskClient.CompleteTasksArgsForCall(1)
						Expect(url).To(Equal(receptorURL))
						Expect(completedTasks).To(HaveLen(1))
						Expect(completedTasks[0].TaskGuid).To(Equal(task.TaskGuid))
					})
				})

				It("bumps the compare-and-swap counter", func() {
					Expect(sender.GetCounter("ConvergenceTasksKicked")).To(Equal(uint64(1)))
				})
			})

			Context("when the resolving task has been completed for longer than the time-to-resolve interval", func() {
				BeforeEach(func() {
					clock.IncrementBySeconds(uint64(expireCompletedTaskDuration.Seconds()) + 1)
				})

				It("should delete the task", func() {
					_, err := bbs.TaskByGuid(logger, task.TaskGuid)
					Expect(err).To(Equal(bbserrors.ErrStoreResourceNotFound))
				})

				It("logs that has failed to resolve task in time", func() {
					Expect(logger.TestSink.LogMessages()).To(ContainElement("test.converge-tasks.failed-to-resolve-in-time"))
				})
			})
		})
	})
})
