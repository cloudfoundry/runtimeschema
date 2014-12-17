package task_bbs_test

import (
	"errors"
	"net/url"
	"os"
	"path"
	"sync"
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
			Stack:    "pancakes",
			Action:   dummyAction,
		}
	})

	Describe("ConvergeTask", func() {
		var desiredEvents <-chan models.Task
		var completedEvents <-chan models.Task

		JustBeforeEach(func() {
			desiredEvents, _, _ = bbs.WatchForDesiredTask()
			completedEvents, _, _ = bbs.WatchForCompletedTask()

			bbs.ConvergeTasks(timeToStart, convergenceInterval, timeToResolveInterval)
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

		Context("when a Task is pending", func() {
			BeforeEach(func() {
				err := bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())
			})

			Context("when the Task has NOT been pending for too long", func() {
				BeforeEach(func() {
					timeProvider.IncrementBySeconds(convergenceIntervalInSeconds - 1)

					auctioneerPresence := models.AuctioneerPresence{
						AuctioneerID:      "the-auctioneer-id",
						AuctioneerAddress: "the-address",
					}

					registerAuctioneer(auctioneerPresence)
				})

				It("does not request an auction for the task", func() {
					Consistently(fakeAuctioneerClient.RequestTaskAuctionCallCount).Should(BeZero())
				})
			})

			Context("when the Task has been pending for longer than the convergence interval", func() {
				BeforeEach(func() {
					timeProvider.IncrementBySeconds(convergenceIntervalInSeconds + 1)
				})

				It("bumps the compare-and-swap counter", func() {
					Ω(sender.GetCounter("ConvergenceTasksKicked")).Should(Equal(uint64(1)))
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
						Ω(fakeAuctioneerClient.RequestTaskAuctionCallCount()).Should(Equal(1))

						requestAddress, requestedTask := fakeAuctioneerClient.RequestTaskAuctionArgsForCall(0)
						Ω(requestAddress).Should(Equal(auctioneerPresence.AuctioneerAddress))
						Ω(requestedTask.TaskGuid).Should(Equal(task.TaskGuid))

					})

					Context("when requesting an auction is unsuccessful", func() {
						BeforeEach(func() {
							fakeAuctioneerClient.RequestTaskAuctionReturns(errors.New("oops"))
						})

						It("logs an error", func() {
							Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-tasks.failed-to-request-auction-for-pending-task"))
						})
					})
				})

				Context("when unable to fetch the auctioneer address", func() {
					It("logs an error", func() {
						Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-tasks.failed-to-request-auction-for-pending-task"))
					})
				})
			})

			Context("when the Task has been pending for longer than the timeToStart", func() {
				BeforeEach(func() {
					timeProvider.IncrementBySeconds(timeToStartInSeconds + 1)
				})

				It("should mark the Task as completed & failed", func() {
					Consistently(desiredEvents).ShouldNot(Receive())

					var noticedOnce models.Task
					Eventually(completedEvents).Should(Receive(&noticedOnce))

					Ω(noticedOnce.Failed).Should(Equal(true))
					Ω(noticedOnce.FailureReason).Should(ContainSubstring("time limit"))
				})

				It("bumps the compare-and-swap counter", func() {
					Ω(sender.GetCounter("ConvergenceTasksKicked")).Should(Equal(uint64(1)))
				})

				It("logs an error", func() {
					Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-tasks.failed-to-start-in-time"))
				})
			})
		})

		Context("when a Task is running", func() {
			var heartbeater ifrit.Process

			BeforeEach(func() {
				err := bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.StartTask(task.TaskGuid, "cell-id")
				Ω(err).ShouldNot(HaveOccurred())

				cellPresence := models.NewCellPresence("cell-id", "stack", "1.2.3.4")
				heartbeater = ifrit.Envoke(servicesBBS.NewCellHeartbeat(cellPresence, time.Minute))
			})

			AfterEach(func() {
				heartbeater.Signal(os.Interrupt)
			})

			It("should do nothing", func() {
				Consistently(desiredEvents).ShouldNot(Receive())
				Consistently(completedEvents).ShouldNot(Receive())
			})

			Context("when the associated cell is missing", func() {
				BeforeEach(func() {
					heartbeater.Signal(os.Interrupt)

					timeProvider.IncrementBySeconds(1)
				})

				It("should mark the Task as completed & failed", func() {
					Consistently(desiredEvents).ShouldNot(Receive())

					var noticedOnce models.Task
					Eventually(completedEvents).Should(Receive(&noticedOnce))

					Ω(noticedOnce.Failed).Should(Equal(true))
					Ω(noticedOnce.FailureReason).Should(ContainSubstring("cell"))
					Ω(noticedOnce.UpdatedAt).Should(Equal(timeProvider.Now().UnixNano()))
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

					err := bbs.DesireTask(task)
					Ω(err).ShouldNot(HaveOccurred())

					err = bbs.StartTask(task.TaskGuid, "cell-id")
					Ω(err).ShouldNot(HaveOccurred())

					err = bbs.CompleteTask(task.TaskGuid, true, "'cause I said so", "a magical result")
					Ω(err).ShouldNot(HaveOccurred())

					secondTask := models.Task{
						Domain:                "tests",
						TaskGuid:              "some-other-guid",
						Stack:                 "pancakes",
						Action:                dummyAction,
						CompletionCallbackURL: &url.URL{Host: "blah"},
					}

					err = bbs.DesireTask(secondTask)
					Ω(err).ShouldNot(HaveOccurred())

					err = bbs.StartTask(secondTask.TaskGuid, "cell-id")
					Ω(err).ShouldNot(HaveOccurred())

					err = bbs.CompleteTask(secondTask.TaskGuid, true, "'cause I said so", "a magical result")
					Ω(err).ShouldNot(HaveOccurred())

					completeTaskError = nil

					wg := new(sync.WaitGroup)
					wg.Add(2)

					fakeTaskClient.CompleteTaskStub = func(string, *models.Task) error {
						wg.Done()
						wg.Wait()
						return completeTaskError
					}
				})

				Context("for longer than the convergence interval", func() {
					BeforeEach(func() {
						timeProvider.IncrementBySeconds(convergenceIntervalInSeconds + 1)
					})

					Context("when a receptor is present", func() {
						var receptorPresence ifrit.Process

						BeforeEach(func() {
							presence := models.NewReceptorPresence("some-receptor", "some-receptor-url")

							heartbeat := servicesBBS.NewReceptorHeartbeat(presence, 1*time.Second)

							receptorPresence = ifrit.Invoke(heartbeat)
						})

						AfterEach(func() {
							ginkgomon.Interrupt(receptorPresence)
						})

						It("submits the completed tasks to the receptor in parallel", func() {
							Ω(fakeTaskClient.CompleteTaskCallCount()).Should(Equal(2))

							receptorURL, firstCompletedTask := fakeTaskClient.CompleteTaskArgsForCall(0)
							Ω(receptorURL).Should(Equal("some-receptor-url"))

							Ω(firstCompletedTask.Failed).Should(BeTrue())
							Ω(firstCompletedTask.FailureReason).Should(Equal("'cause I said so"))
							Ω(firstCompletedTask.Result).Should(Equal("a magical result"))

							receptorURL, secondCompletedTask := fakeTaskClient.CompleteTaskArgsForCall(1)
							Ω(receptorURL).Should(Equal("some-receptor-url"))

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
								Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-tasks.failed-to-complete"))
							})
						})
					})

					Context("when a receptor is not present", func() {
						It("does not submit a completed task to anything", func() {
							Ω(fakeTaskClient.CompleteTaskCallCount()).Should(BeZero())
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
					err := bbs.DesireTask(task)
					Ω(err).ShouldNot(HaveOccurred())

					err = bbs.StartTask(task.TaskGuid, "cell-id")
					Ω(err).ShouldNot(HaveOccurred())

					err = bbs.CompleteTask(task.TaskGuid, true, "'cause I said so", "a magical result")
					Ω(err).ShouldNot(HaveOccurred())
				})

				Context("for longer than the convergence interval", func() {
					BeforeEach(func() {
						timeProvider.IncrementBySeconds(convergenceIntervalInSeconds + 1)
					})

					Context("when a receptor is present", func() {
						var receptorPresence ifrit.Process

						BeforeEach(func() {
							presence := models.NewReceptorPresence("some-receptor", "some-receptor-url")

							heartbeat := servicesBBS.NewReceptorHeartbeat(presence, 1*time.Second)

							receptorPresence = ifrit.Invoke(heartbeat)
						})

						AfterEach(func() {
							ginkgomon.Interrupt(receptorPresence)
						})

						It("does not submit the completed task to the receptor", func() {
							Ω(fakeTaskClient.CompleteTaskCallCount()).Should(BeZero())
						})
					})

					It("bumps the convergence tasks kicked counter", func() {
						Ω(sender.GetCounter("ConvergenceTasksKicked")).Should(Equal(uint64(1)))
					})
				})

				Context("when the task has been completed for longer than the time-to-resolve interval", func() {
					BeforeEach(func() {
						timeProvider.IncrementBySeconds(uint64(timeToResolveInterval.Seconds()) + 1)
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
					BeforeEach(func() {
						timeProvider.IncrementBySeconds(1)
					})

					It("should NOT kick the Task", func() {
						Consistently(desiredEvents).ShouldNot(Receive())
						Consistently(completedEvents).ShouldNot(Receive())
					})
				})
			})
		})

		Context("when a Task is resolving", func() {
			BeforeEach(func() {
				task.CompletionCallbackURL = &url.URL{Host: "blah"}

				err := bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.StartTask(task.TaskGuid, "cell-id")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.CompleteTask(task.TaskGuid, true, "'cause I said so", "a result")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.ResolvingTask(task.TaskGuid)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("should do nothing", func() {
				Consistently(desiredEvents).ShouldNot(Receive())
				Consistently(completedEvents).ShouldNot(Receive())
			})

			Context("when the run once has been resolving for longer than a convergence interval", func() {
				BeforeEach(func() {
					timeProvider.IncrementBySeconds(convergenceIntervalInSeconds)
				})

				It("should put the Task back into the completed state", func() {
					var noticedOnce models.Task
					Eventually(completedEvents).Should(Receive(&noticedOnce))

					Ω(noticedOnce.TaskGuid).Should(Equal(task.TaskGuid))
					Ω(noticedOnce.State).Should(Equal(models.TaskStateCompleted))
					Ω(noticedOnce.UpdatedAt).Should(Equal(timeProvider.Now().UnixNano()))
				})

				It("logs that it is demoting task from resolving to completed", func() {
					Ω(logger.TestSink.LogMessages()).Should(ContainElement("test.converge-tasks.demoting-resolving-to-completed"))
				})

				Context("when a receptor is present", func() {
					var receptorPresence ifrit.Process

					BeforeEach(func() {
						presence := models.NewReceptorPresence("some-receptor", "some-receptor-url")

						heartbeat := servicesBBS.NewReceptorHeartbeat(presence, 1*time.Second)

						receptorPresence = ifrit.Invoke(heartbeat)
					})

					AfterEach(func() {
						ginkgomon.Interrupt(receptorPresence)
					})

					It("submits the completed task to the receptor", func() {
						Ω(fakeTaskClient.CompleteTaskCallCount()).Should(Equal(1))

						receptorURL, completedTask := fakeTaskClient.CompleteTaskArgsForCall(0)
						Ω(receptorURL).Should(Equal("some-receptor-url"))
						Ω(completedTask.TaskGuid).Should(Equal(task.TaskGuid))
					})
				})

				It("bumps the compare-and-swap counter", func() {
					Ω(sender.GetCounter("ConvergenceTasksKicked")).Should(Equal(uint64(1)))
				})
			})

			Context("when the resolving task has been completed for longer than the time-to-resolve interval", func() {
				BeforeEach(func() {
					timeProvider.IncrementBySeconds(uint64(timeToResolveInterval.Seconds()) + 1)
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
