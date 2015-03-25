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
	"github.com/hashicorp/consul/consul/structs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"
)

var _ = Describe("Convergence of Tasks", func() {
	var (
		sender *fake.FakeMetricSender

		task                             models.Task
		timeToStartInSeconds             uint64
		convergenceIntervalInSeconds     uint64
		timeToStart, convergenceInterval time.Duration
		timeToResolveInterval            time.Duration
	)

	BeforeEach(func() {
		sender = fake.NewFakeMetricSender()
		metrics.Initialize(sender)

		timeToStartInSeconds = 30
		timeToStart = time.Duration(timeToStartInSeconds) * time.Second
		convergenceIntervalInSeconds = 10
		convergenceInterval = time.Duration(convergenceIntervalInSeconds) * time.Second
		timeToResolveInterval = time.Hour

		task = models.Task{
			Domain:   "tests",
			TaskGuid: "some-guid",
			RootFS:   "some:rootfs",
			Action:   dummyAction,
		}
	})

	Describe("ConvergeTasks", func() {
		JustBeforeEach(func() {
			bbs.ConvergeTasks(logger, timeToStart, convergenceInterval, timeToResolveInterval)
		})

		It("bumps the convergence counter", func() {
			Ω(sender.GetCounter("ConvergenceTaskRuns")).Should(Equal(uint64(1)))
		})

		It("reports the duration that it took to converge", func() {
			reportedDuration := sender.GetValue("ConvergenceTaskDuration")
			Ω(reportedDuration.Unit).Should(Equal("nanos"))
			Ω(reportedDuration.Value).ShouldNot(BeZero())
		})

		Context("when a Task is malformed", func() {
			var nodeKey string

			BeforeEach(func() {
				nodeKey = path.Join(shared.TaskSchemaRoot, "some-guid")

				err := etcdClient.Create(storeadapter.StoreNode{
					Key:   nodeKey,
					Value: []byte("ß"),
				})
				Ω(err).ShouldNot(HaveOccurred())

				_, err = etcdClient.Get(nodeKey)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("should delete it", func() {
				_, err := etcdClient.Get(nodeKey)
				Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))
			})

			It("bumps the pruned counter", func() {
				Ω(sender.GetCounter("ConvergenceTasksPruned")).Should(Equal(uint64(1)))
			})
		})

		Context("when Tasks are pending", func() {
			var secondTask models.Task

			BeforeEach(func() {
				err := bbs.DesireTask(logger, task)
				Ω(err).ShouldNot(HaveOccurred())

				secondTask = models.Task{
					Domain:                "tests",
					TaskGuid:              "some-other-guid",
					RootFS:                "some:rootfs",
					Action:                dummyAction,
					CompletionCallbackURL: &url.URL{Host: "blah"},
				}

				err = bbs.DesireTask(logger, secondTask)
				Ω(err).ShouldNot(HaveOccurred())
			})

			Context("when the Task has NOT been pending for too long", func() {
				BeforeEach(func() {
					clock.IncrementBySeconds(convergenceIntervalInSeconds - 1)

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

			Context("when the Tasks have been pending for longer than the convergence interval", func() {
				BeforeEach(func() {
					clock.IncrementBySeconds(convergenceIntervalInSeconds + 1)
				})

				It("bumps the compare-and-swap counter", func() {
					Ω(sender.GetCounter("ConvergenceTasksKicked")).Should(Equal(uint64(2)))
				})

				It("logs that it sends an auction for the pending task", func() {
					Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-tasks.requesting-auction-for-pending-task"))
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
						Ω(fakeAuctioneerClient.RequestTaskAuctionsCallCount()).Should(Equal(1))

						requestAddress, requestedTasks := fakeAuctioneerClient.RequestTaskAuctionsArgsForCall(0)
						Ω(requestAddress).Should(Equal(auctioneerPresence.AuctioneerAddress))
						Ω(requestedTasks).Should(HaveLen(2))
						Ω([]string{requestedTasks[0].TaskGuid, requestedTasks[1].TaskGuid}).Should(ConsistOf(task.TaskGuid, secondTask.TaskGuid))
					})

					Context("when requesting an auction is unsuccessful", func() {
						BeforeEach(func() {
							fakeAuctioneerClient.RequestTaskAuctionsReturns(errors.New("oops"))
						})

						It("logs an error", func() {
							Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-tasks.failed-to-request-auctions-for-pending-tasks"))
						})
					})
				})

				Context("when unable to fetch the auctioneer address", func() {
					It("logs an error", func() {
						Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-tasks.failed-to-request-auctions-for-pending-tasks"))
					})
				})
			})

			Context("when the Task has been pending for longer than the timeToStart", func() {
				BeforeEach(func() {
					clock.IncrementBySeconds(timeToStartInSeconds + 1)
				})

				It("should mark the Task as completed & failed", func() {
					returnedTask, err := bbs.TaskByGuid(task.TaskGuid)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(returnedTask.State).Should(Equal(models.TaskStateCompleted))

					Ω(returnedTask.Failed).Should(Equal(true))
					Ω(returnedTask.FailureReason).Should(ContainSubstring("time limit"))
				})

				It("bumps the compare-and-swap counter", func() {
					Ω(sender.GetCounter("ConvergenceTasksKicked")).Should(Equal(uint64(2)))
				})

				It("logs an error", func() {
					Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-tasks.failed-to-start-in-time"))
				})
			})
		})

		Context("when a Task is running", func() {
			BeforeEach(func() {
				err := bbs.DesireTask(logger, task)
				Ω(err).ShouldNot(HaveOccurred())

				_, err = bbs.StartTask(logger, task.TaskGuid, "cell-id")
				Ω(err).ShouldNot(HaveOccurred())
			})

			Context("when the associated cell is present", func() {
				var heartbeater ifrit.Process

				BeforeEach(func() {
					cellPresence := models.NewCellPresence("cell-id", "1.2.3.4", "the-zone", models.NewCellCapacity(128, 1024, 3))
					heartbeater = ifrit.Invoke(servicesBBS.NewCellHeartbeat(cellPresence, time.Minute, 100*time.Millisecond))
				})

				AfterEach(func() {
					heartbeater.Signal(os.Interrupt)
				})

				It("leaves the task running", func() {
					returnedTask, err := bbs.TaskByGuid(task.TaskGuid)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(returnedTask.State).Should(Equal(models.TaskStateRunning))
				})
			})

			Context("when the associated cell is missing", func() {
				It("should mark the Task as completed & failed", func() {
					returnedTask, err := bbs.TaskByGuid(task.TaskGuid)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(returnedTask.State).Should(Equal(models.TaskStateCompleted))

					Ω(returnedTask.Failed).Should(Equal(true))
					Ω(returnedTask.FailureReason).Should(ContainSubstring("cell"))
				})

				It("logs that the cell disappeared", func() {
					Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-tasks.cell-disappeared"))
				})

				It("bumps the compare-and-swap counter", func() {
					Ω(sender.GetCounter("ConvergenceTasksKicked")).Should(Equal(uint64(1)))
				})
			})
		})

		Describe("Completed tasks", func() {
			Context("when Tasks with a complete URL are completed", func() {
				var completeTaskError error

				BeforeEach(func() {
					task.CompletionCallbackURL = &url.URL{Host: "blah"}

					err := bbs.DesireTask(logger, task)
					Ω(err).ShouldNot(HaveOccurred())

					_, err = bbs.StartTask(logger, task.TaskGuid, "cell-id")
					Ω(err).ShouldNot(HaveOccurred())

					err = bbs.CompleteTask(logger, task.TaskGuid, "cell-id", true, "'cause I said so", "a magical result")
					Ω(err).ShouldNot(HaveOccurred())

					secondTask := models.Task{
						Domain:                "tests",
						TaskGuid:              "some-other-guid",
						RootFS:                "some:rootfs",
						Action:                dummyAction,
						CompletionCallbackURL: &url.URL{Host: "blah"},
					}

					err = bbs.DesireTask(logger, secondTask)
					Ω(err).ShouldNot(HaveOccurred())

					_, err = bbs.StartTask(logger, secondTask.TaskGuid, "cell-id")
					Ω(err).ShouldNot(HaveOccurred())

					err = bbs.CompleteTask(logger, secondTask.TaskGuid, "cell-id", true, "'cause I said so", "a magical result")
					Ω(err).ShouldNot(HaveOccurred())

					completeTaskError = nil

					fakeTaskClient.CompleteTasksStub = func(string, []models.Task) error {
						return completeTaskError
					}
				})

				Context("for longer than the convergence interval", func() {
					BeforeEach(func() {
						clock.IncrementBySeconds(convergenceIntervalInSeconds + 1)
					})

					Context("when a receptor is present", func() {
						var receptorPresence ifrit.Process

						BeforeEach(func() {
							presence := models.NewReceptorPresence("some-receptor", "some-receptor-url")

							heartbeat := servicesBBS.NewReceptorHeartbeat(presence, structs.SessionTTLMin, 100*time.Millisecond)

							receptorPresence = ifrit.Invoke(heartbeat)
						})

						AfterEach(func() {
							ginkgomon.Interrupt(receptorPresence)
						})

						It("submits the completed tasks to the receptor in batch", func() {
							Ω(fakeTaskClient.CompleteTasksCallCount()).Should(Equal(1))

							receptorURL, completedTasks := fakeTaskClient.CompleteTasksArgsForCall(0)
							Ω(receptorURL).Should(Equal("some-receptor-url"))
							Ω(completedTasks).Should(HaveLen(2))

							firstCompletedTask := completedTasks[0]
							Ω(firstCompletedTask.Failed).Should(BeTrue())
							Ω(firstCompletedTask.FailureReason).Should(Equal("'cause I said so"))
							Ω(firstCompletedTask.Result).Should(Equal("a magical result"))

							secondCompletedTask := completedTasks[1]
							Ω(secondCompletedTask.Failed).Should(BeTrue())
							Ω(secondCompletedTask.FailureReason).Should(Equal("'cause I said so"))
							Ω(secondCompletedTask.Result).Should(Equal("a magical result"))

							Ω([]string{firstCompletedTask.TaskGuid, secondCompletedTask.TaskGuid}).Should(ConsistOf(
								[]string{"some-guid", "some-other-guid"},
							))
						})

						It("logs that it kicks the completed task", func() {
							Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-tasks.kicking-completed-task"))
						})

						It("bumps the convergence tasks kicked counter", func() {
							Ω(sender.GetCounter("ConvergenceTasksKicked")).Should(Equal(uint64(2)))
						})

						Context("when the receptor fails to complete the task", func() {
							BeforeEach(func() {
								completeTaskError = errors.New("whoops!")
							})

							It("logs that it failed to complete the task", func() {
								Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-tasks.failed-to-complete-tasks"))
							})
						})
					})

					Context("when a receptor is not present", func() {
						It("does not submit a completed task to anything", func() {
							Ω(fakeTaskClient.CompleteTasksCallCount()).Should(BeZero())
						})

						It("bumps the convergence tasks kicked counter anyway", func() {
							Ω(sender.GetCounter("ConvergenceTasksKicked")).Should(Equal(uint64(2)))
						})

						It("logs that it failed to find a receptor", func() {
							Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-tasks.failed-to-find-receptor"))
						})
					})
				})
			})

			Context("when a completed task without a complete URL is present", func() {
				BeforeEach(func() {
					err := bbs.DesireTask(logger, task)
					Ω(err).ShouldNot(HaveOccurred())

					_, err = bbs.StartTask(logger, task.TaskGuid, "cell-id")
					Ω(err).ShouldNot(HaveOccurred())

					err = bbs.CompleteTask(logger, task.TaskGuid, "cell-id", true, "'cause I said so", "a magical result")
					Ω(err).ShouldNot(HaveOccurred())
				})

				Context("for longer than the convergence interval", func() {
					BeforeEach(func() {
						clock.IncrementBySeconds(convergenceIntervalInSeconds + 1)
					})

					Context("when a receptor is present", func() {
						var receptorPresence ifrit.Process

						BeforeEach(func() {
							presence := models.NewReceptorPresence("some-receptor", "some-receptor-url")

							heartbeat := servicesBBS.NewReceptorHeartbeat(presence, structs.SessionTTLMin, 100*time.Millisecond)

							receptorPresence = ifrit.Invoke(heartbeat)
						})

						AfterEach(func() {
							ginkgomon.Interrupt(receptorPresence)
						})

						It("does not submit the completed task to the receptor", func() {
							Ω(fakeTaskClient.CompleteTasksCallCount()).Should(BeZero())
						})
					})

					It("bumps the convergence tasks kicked counter", func() {
						Ω(sender.GetCounter("ConvergenceTasksKicked")).Should(Equal(uint64(1)))
					})
				})

				Context("when the task has been completed for longer than the time-to-resolve interval", func() {
					BeforeEach(func() {
						clock.IncrementBySeconds(uint64(timeToResolveInterval.Seconds()) + 1)
					})

					It("should delete the task", func() {
						_, err := bbs.TaskByGuid(task.TaskGuid)
						Ω(err).Should(Equal(bbserrors.ErrStoreResourceNotFound))
					})

					It("logs that it failed to start resolving the task in time", func() {
						Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-tasks.failed-to-start-resolving-in-time"))
					})
				})

				Context("when the task has been completed for less than the convergence interval", func() {
					var previousTime int64

					BeforeEach(func() {
						previousTime = clock.Now().UnixNano()
						clock.IncrementBySeconds(1)
					})

					It("should NOT kick the Task", func() {
						returnedTask, err := bbs.TaskByGuid(task.TaskGuid)
						Ω(err).ShouldNot(HaveOccurred())
						Ω(returnedTask.State).Should(Equal(models.TaskStateCompleted))
						Ω(returnedTask.UpdatedAt).Should(Equal(previousTime))
					})
				})
			})
		})

		Context("when a Task is resolving", func() {
			BeforeEach(func() {
				task.CompletionCallbackURL = &url.URL{Host: "blah"}

				err := bbs.DesireTask(logger, task)
				Ω(err).ShouldNot(HaveOccurred())

				_, err = bbs.StartTask(logger, task.TaskGuid, "cell-id")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.CompleteTask(logger, task.TaskGuid, "cell-id", true, "'cause I said so", "a result")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.ResolvingTask(logger, task.TaskGuid)
				Ω(err).ShouldNot(HaveOccurred())
			})

			Context("when the task is in resolving state for less than the convergence interval", func() {
				var previousTime int64

				BeforeEach(func() {
					previousTime = clock.Now().UnixNano()
					clock.IncrementBySeconds(1)
				})

				It("should do nothing", func() {
					returnedTask, err := bbs.TaskByGuid(task.TaskGuid)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(returnedTask.State).Should(Equal(models.TaskStateResolving))
					Ω(returnedTask.UpdatedAt).Should(Equal(previousTime))
				})
			})

			Context("when the task has been resolving for longer than a convergence interval", func() {
				BeforeEach(func() {
					clock.IncrementBySeconds(convergenceIntervalInSeconds)
				})

				It("should put the Task back into the completed state", func() {
					returnedTask, err := bbs.TaskByGuid(task.TaskGuid)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(returnedTask.State).Should(Equal(models.TaskStateCompleted))
					Ω(returnedTask.UpdatedAt).Should(Equal(clock.Now().UnixNano()))
				})

				It("logs that it is demoting task from resolving to completed", func() {
					Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-tasks.demoting-resolving-to-completed"))
				})

				Context("when a receptor is present", func() {
					var receptorPresence ifrit.Process

					BeforeEach(func() {
						presence := models.NewReceptorPresence("some-receptor", "some-receptor-url")

						heartbeat := servicesBBS.NewReceptorHeartbeat(presence, structs.SessionTTLMin, 100*time.Millisecond)

						receptorPresence = ifrit.Invoke(heartbeat)
					})

					AfterEach(func() {
						ginkgomon.Interrupt(receptorPresence)
					})

					It("submits the completed task to the receptor", func() {
						Ω(fakeTaskClient.CompleteTasksCallCount()).Should(Equal(1))

						receptorURL, completedTasks := fakeTaskClient.CompleteTasksArgsForCall(0)
						Ω(receptorURL).Should(Equal("some-receptor-url"))
						Ω(completedTasks).Should(HaveLen(1))
						Ω(completedTasks[0].TaskGuid).Should(Equal(task.TaskGuid))
					})

					Context("Tasks are completed after they are demoted", func() {
						BeforeEach(func() {
							fakeTaskClient.CompleteTasksStub = func(string, []models.Task) error {
								returnedTask, err := bbs.TaskByGuid(task.TaskGuid)
								Ω(err).ShouldNot(HaveOccurred())
								Ω(returnedTask.State).Should(Equal(models.TaskStateCompleted))
								return nil
							}
						})

						It("is completed after demoted", func() {
							Ω(fakeTaskClient.CompleteTasksCallCount()).Should(Equal(1))
						})
					})
				})

				It("bumps the compare-and-swap counter", func() {
					Ω(sender.GetCounter("ConvergenceTasksKicked")).Should(Equal(uint64(1)))
				})
			})

			Context("when the resolving task has been completed for longer than the time-to-resolve interval", func() {
				BeforeEach(func() {
					clock.IncrementBySeconds(uint64(timeToResolveInterval.Seconds()) + 1)
				})

				It("should delete the task", func() {
					_, err := bbs.TaskByGuid(task.TaskGuid)
					Ω(err).Should(Equal(bbserrors.ErrStoreResourceNotFound))
				})

				It("logs that has failed to resolve task in time", func() {
					Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-tasks.failed-to-resolve-in-time"))
				})
			})
		})
	})
})
