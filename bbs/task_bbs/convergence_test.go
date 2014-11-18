package task_bbs_test

import (
	"os"
	"path"
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/services_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	. "github.com/cloudfoundry-incubator/runtime-schema/bbs/task_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/dropsonde/metric_sender/fake"
	"github.com/cloudfoundry/dropsonde/metrics"
	"github.com/cloudfoundry/gunk/timeprovider/faketimeprovider"
	"github.com/cloudfoundry/storeadapter"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager/lagertest"
	"github.com/tedsuo/ifrit"
)

var _ = Describe("Convergence of Tasks", func() {
	var (
		sender *fake.FakeMetricSender

		bbs                              *TaskBBS
		task                             models.Task
		timeToClaimInSeconds             uint64
		convergenceIntervalInSeconds     uint64
		timeToClaim, convergenceInterval time.Duration
		timeToResolveInterval            time.Duration
		timeProvider                     *faketimeprovider.FakeTimeProvider
		err                              error
		servicesBBS                      *services_bbs.ServicesBBS
	)

	BeforeEach(func() {
		sender = fake.NewFakeMetricSender()
		metrics.Initialize(sender)

		err = nil

		timeToClaimInSeconds = 30
		timeToClaim = time.Duration(timeToClaimInSeconds) * time.Second
		convergenceIntervalInSeconds = 10
		convergenceInterval = time.Duration(convergenceIntervalInSeconds) * time.Second
		timeToResolveInterval = time.Hour

		timeProvider = faketimeprovider.New(time.Unix(1238, 0))

		task = models.Task{
			Domain:   "tests",
			TaskGuid: "some-guid",
			Stack:    "pancakes",
			Action:   dummyAction,
		}

		logger := lagertest.NewTestLogger("test")

		bbs = New(etcdClient, timeProvider, logger)

		servicesBBS = services_bbs.New(etcdClient, logger)
	})

	Describe("ConvergeTask", func() {
		var desiredEvents <-chan models.Task
		var completedEvents <-chan models.Task

		JustBeforeEach(func() {
			desiredEvents, _, _ = bbs.WatchForDesiredTask()
			completedEvents, _, _ = bbs.WatchForCompletedTask()

			bbs.ConvergeTask(timeToClaim, convergenceInterval, timeToResolveInterval)
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
				_, err = etcdClient.Get(nodeKey)
				Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))
			})

			It("bumps the pruned counter", func() {
				Ω(sender.GetCounter("ConvergenceTasksPruned")).Should(Equal(uint64(1)))
			})
		})

		Context("when a Task is pending", func() {
			BeforeEach(func() {
				err = bbs.DesireTask(task)
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
					Ω(noticedOnce.UpdatedAt).Should(Equal(timeProvider.Time().UnixNano()))
				})

				It("bumps the compare-and-swap counter", func() {
					Ω(sender.GetCounter("ConvergenceTasksKicked")).Should(Equal(uint64(1)))
				})
			})

			Context("when the Task has been pending for longer than the timeToClaim", func() {
				BeforeEach(func() {
					timeProvider.IncrementBySeconds(timeToClaimInSeconds + 1)
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

		Context("when a Task is claimed", func() {
			var heartbeat ifrit.Process

			BeforeEach(func() {
				err = bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.ClaimTask(task.TaskGuid, "cell-id")
				Ω(err).ShouldNot(HaveOccurred())

				cellPresence := models.CellPresence{
					CellID: "cell-id",
				}
				heartbeat = ifrit.Envoke(servicesBBS.NewCellHeartbeat(cellPresence, time.Minute))
			})

			AfterEach(func() {
				heartbeat.Signal(os.Interrupt)
				Eventually(heartbeat.Wait()).Should(Receive(BeNil()))
			})

			It("should do nothing", func() {
				Consistently(desiredEvents).ShouldNot(Receive())
				Consistently(completedEvents).ShouldNot(Receive())
			})

			Context("when the associated cell is missing", func() {
				BeforeEach(func() {
					heartbeat.Signal(os.Interrupt)
					Eventually(heartbeat.Wait()).Should(Receive(BeNil()))

					timeProvider.IncrementBySeconds(1)
				})

				It("should mark the Task as completed & failed", func() {
					Consistently(desiredEvents).ShouldNot(Receive())

					var noticedOnce models.Task
					Eventually(completedEvents).Should(Receive(&noticedOnce))

					Ω(noticedOnce.Failed).Should(Equal(true))
					Ω(noticedOnce.FailureReason).Should(ContainSubstring("cell"))
					Ω(noticedOnce.UpdatedAt).Should(Equal(timeProvider.Time().UnixNano()))
				})

				It("bumps the compare-and-swap counter", func() {
					Ω(sender.GetCounter("ConvergenceTasksKicked")).Should(Equal(uint64(1)))
				})
			})
		})

		Context("when a Task is running", func() {
			var heartbeater ifrit.Process

			BeforeEach(func() {
				err = bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.ClaimTask(task.TaskGuid, "cell-id")
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
					Ω(noticedOnce.UpdatedAt).Should(Equal(timeProvider.Time().UnixNano()))
				})

				It("bumps the compare-and-swap counter", func() {
					Ω(sender.GetCounter("ConvergenceTasksKicked")).Should(Equal(uint64(1)))
				})
			})
		})

		Context("when a Task is completed", func() {
			BeforeEach(func() {
				err = bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.ClaimTask(task.TaskGuid, "cell-id")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.StartTask(task.TaskGuid, "cell-id")
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.CompleteTask(task.TaskGuid, true, "'cause I said so", "a magical result")
				Ω(err).ShouldNot(HaveOccurred())
			})

			Context("when the task has been completed for > the convergence interval", func() {
				BeforeEach(func() {
					timeProvider.IncrementBySeconds(convergenceIntervalInSeconds + 1)
				})

				It("should kick the Task", func() {
					Consistently(desiredEvents).ShouldNot(Receive())

					var noticedOnce models.Task
					Eventually(completedEvents).Should(Receive(&noticedOnce))

					Ω(noticedOnce.Failed).Should(Equal(true))
					Ω(noticedOnce.FailureReason).Should(Equal("'cause I said so"))
					Ω(noticedOnce.Result).Should(Equal("a magical result"))
					Ω(noticedOnce.UpdatedAt).Should(Equal(timeProvider.Time().UnixNano()))
				})

				It("bumps the compare-and-swap counter", func() {
					Ω(sender.GetCounter("ConvergenceTasksKicked")).Should(Equal(uint64(1)))
				})
			})

			Context("when the task has been completed for > the time to resolve interval", func() {
				BeforeEach(func() {
					timeProvider.IncrementBySeconds(uint64(timeToResolveInterval.Seconds()) + 1)
				})

				It("should delete the task", func() {
					_, err := bbs.TaskByGuid(task.TaskGuid)
					Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))
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

		Context("when a Task is resolving", func() {
			BeforeEach(func() {
				err = bbs.DesireTask(task)
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.ClaimTask(task.TaskGuid, "cell-id")
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
					Ω(noticedOnce.UpdatedAt).Should(Equal(timeProvider.Time().UnixNano()))
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
				Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))
			})
		})
	})
})
