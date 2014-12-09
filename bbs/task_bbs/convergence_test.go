package task_bbs_test

import (
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

			bbs.ConvergeTask(timeToStart, convergenceInterval, timeToResolveInterval)
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

			Context("when the Task has *not* been pending for too long", func() {
				BeforeEach(func() {
					timeProvider.IncrementBySeconds(convergenceIntervalInSeconds - 1)
				})

				It("should not kick the Task", func() {
					Consistently(desiredEvents).ShouldNot(Receive())
				})
			})

			Context("when the Task has been pending for longer than the convergence interval", func() {
				BeforeEach(func() {
					timeProvider.IncrementBySeconds(convergenceIntervalInSeconds + 1)
				})

				It("should kick the Task", func() {
					var noticedOnce models.Task
					Eventually(desiredEvents).Should(Receive(&noticedOnce))

					Ω(noticedOnce.TaskGuid).Should(Equal(task.TaskGuid))
					Ω(noticedOnce.State).Should(Equal(models.TaskStatePending))
					Ω(noticedOnce.UpdatedAt).Should(Equal(timeProvider.Now().UnixNano()))
				})

				It("bumps the compare-and-swap counter", func() {
					Ω(sender.GetCounter("ConvergenceTasksKicked")).Should(Equal(uint64(1)))
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
			})
		})

		Context("when a Task is running", func() {
			var heartbeater ifrit.Process

			BeforeEach(func() {
				err := bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.StartTask(task.TaskGuid, "cell-id")
				Ω(err).ShouldNot(HaveOccurred())

				heartbeater = ifrit.Envoke(servicesBBS.NewCellHeartbeat(models.CellPresence{
					CellID: "cell-id",
				}, time.Minute))
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

				It("bumps the compare-and-swap counter", func() {
					Ω(sender.GetCounter("ConvergenceTasksKicked")).Should(Equal(uint64(1)))
				})
			})
		})

		Describe("Completed tasks", func() {
			Context("when a Task with a complete URL is completed", func() {
				BeforeEach(func() {
					task.CompletionCallbackURL = &url.URL{Host: "blah"}

					err := bbs.DesireTask(task)
					Ω(err).ShouldNot(HaveOccurred())

					err = bbs.StartTask(task.TaskGuid, "cell-id")
					Ω(err).ShouldNot(HaveOccurred())

					err = bbs.CompleteTask(task.TaskGuid, true, "'cause I said so", "a magical result")
					Ω(err).ShouldNot(HaveOccurred())
				})

				Context("for > the convergence interval", func() {
					BeforeEach(func() {
						timeProvider.IncrementBySeconds(convergenceIntervalInSeconds + 1)
					})

					Context("when a receptor is present", func() {
						var receptorPresence ifrit.Process

						BeforeEach(func() {
							presence := models.ReceptorPresence{
								ReceptorID:  "some-receptor",
								ReceptorURL: "some-receptor-url",
							}

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
							Ω(completedTask.Failed).Should(BeTrue())
							Ω(completedTask.FailureReason).Should(Equal("'cause I said so"))
							Ω(completedTask.Result).Should(Equal("a magical result"))
						})

						It("bumps the convergence tasks kicked counter", func() {
							Ω(sender.GetCounter("ConvergenceTasksKicked")).Should(Equal(uint64(1)))
						})
					})

					Context("when a receptor is not present", func() {
						It("does not submit a completed task to anything", func() {
							Ω(fakeTaskClient.CompleteTaskCallCount()).Should(BeZero())
						})

						It("bumps the convergence tasks kicked counter anyway", func() {
							Ω(sender.GetCounter("ConvergenceTasksKicked")).Should(Equal(uint64(1)))
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

				Context("for > the convergence interval", func() {
					BeforeEach(func() {
						timeProvider.IncrementBySeconds(convergenceIntervalInSeconds + 1)
					})

					Context("when a receptor is present", func() {
						var receptorPresence ifrit.Process

						BeforeEach(func() {
							presence := models.ReceptorPresence{
								ReceptorID:  "some-receptor",
								ReceptorURL: "some-receptor-url",
							}

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

				Context("when the task has been completed for > the time to resolve interval", func() {
					BeforeEach(func() {
						timeProvider.IncrementBySeconds(uint64(timeToResolveInterval.Seconds()) + 1)
					})

					It("should delete the task", func() {
						_, err := bbs.TaskByGuid(task.TaskGuid)
						Ω(err).Should(Equal(bbserrors.ErrStoreResourceNotFound))
					})
				})

				Context("when the task has been completed for < the convergence interval", func() {
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

			Context("when the run once has been resolving for > 30 seconds", func() {
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

				It("bumps the compare-and-swap counter", func() {
					Ω(sender.GetCounter("ConvergenceTasksKicked")).Should(Equal(uint64(1)))
				})
			})
		})

		Context("when the resolving task has been completed for > the time to resolve interval", func() {
			BeforeEach(func() {
				timeProvider.IncrementBySeconds(uint64(timeToResolveInterval.Seconds()) + 1)
			})

			It("should delete the task", func() {
				_, err := bbs.TaskByGuid(task.TaskGuid)
				Ω(err).Should(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		})
	})
})
