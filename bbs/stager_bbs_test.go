package bbs_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/gunk/timeprovider/faketimeprovider"
	"github.com/cloudfoundry/storeadapter"

	. "github.com/cloudfoundry-incubator/runtime-schema/bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

var _ = Describe("Stager BBS", func() {
	var bbs *BBS
	var runOnce *models.Task
	var timeProvider *faketimeprovider.FakeTimeProvider

	BeforeEach(func() {
		timeProvider = faketimeprovider.New(time.Unix(1238, 0))

		bbs = New(store, timeProvider)
		runOnce = &models.Task{
			Guid:            "some-guid",
			ExecutorID:      "executor-id",
			ContainerHandle: "container-handle",
		}
	})

	itRetriesUntilStoreComesBack := func(action func() error) {
		It("should keep trying until the store comes back", func(done Done) {
			etcdRunner.GoAway()

			runResult := make(chan error)
			go func() {
				err := action()
				runResult <- err
			}()

			time.Sleep(200 * time.Millisecond)

			etcdRunner.ComeBack()

			Ω(<-runResult).ShouldNot(HaveOccurred())

			close(done)
		}, 5)
	}

	Describe("DesireTask", func() {
		Context("when the Task has a CreatedAt time", func() {
			BeforeEach(func() {
				runOnce.CreatedAt = 1234812
				err := bbs.DesireTask(runOnce)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("creates /run_once/<guid>", func() {
				node, err := store.Get("/v1/run_once/some-guid")
				Ω(err).ShouldNot(HaveOccurred())
				Ω(node.Value).Should(Equal(runOnce.ToJSON()))
			})
		})

		Context("when the Task has no CreatedAt time", func() {
			BeforeEach(func() {
				err := bbs.DesireTask(runOnce)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("creates /run_once/<guid> and sets set the CreatedAt time to now", func() {
				runOnces, err := bbs.GetAllPendingTasks()
				Ω(err).ShouldNot(HaveOccurred())

				Ω(runOnces[0].CreatedAt).Should(Equal(timeProvider.Time().UnixNano()))
			})

			It("should bump UpdatedAt", func() {
				runOnces, err := bbs.GetAllPendingTasks()
				Ω(err).ShouldNot(HaveOccurred())

				Ω(runOnces[0].UpdatedAt).Should(Equal(timeProvider.Time().UnixNano()))
			})
		})

		Context("Common cases", func() {
			BeforeEach(func() {
				runOnce.CreatedAt = 1234812
				err := bbs.DesireTask(runOnce)
				Ω(err).ShouldNot(HaveOccurred())
			})

			Context("when the Task is already pending", func() {
				It("should happily overwrite the existing Task", func() {
					err := bbs.DesireTask(runOnce)
					Ω(err).ShouldNot(HaveOccurred())
				})
			})

			Context("when the store is out of commission", func() {
				itRetriesUntilStoreComesBack(func() error {
					return bbs.DesireTask(runOnce)
				})
			})

			It("should bump UpdatedAt", func() {
				runOnces, err := bbs.GetAllPendingTasks()
				Ω(err).ShouldNot(HaveOccurred())

				Ω(runOnces[0].UpdatedAt).Should(Equal(timeProvider.Time().UnixNano()))
			})
		})
	})

	Describe("ResolvingTask", func() {
		BeforeEach(func() {
			var err error

			err = bbs.DesireTask(runOnce)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.ClaimTask(runOnce, "some-executor-id")
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.StartTask(runOnce, "some-container-handle")
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.CompleteTask(runOnce, true, "because i said so", "a result")
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("swaps /run_once/<guid>'s state to resolving", func() {
			err := bbs.ResolvingTask(runOnce)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(runOnce.State).Should(Equal(models.TaskStateResolving))

			node, err := store.Get("/v1/run_once/some-guid")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(node.Value).Should(Equal(runOnce.ToJSON()))
		})

		It("should bump UpdatedAt", func() {
			timeProvider.IncrementBySeconds(1)

			err := bbs.ResolvingTask(runOnce)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(runOnce.UpdatedAt).Should(Equal(timeProvider.Time().UnixNano()))
		})

		Context("when the Task is already resolving", func() {
			BeforeEach(func() {
				err := bbs.ResolvingTask(runOnce)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("fails", func() {
				err := bbs.ResolvingTask(runOnce)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(runOnce.State).Should(Equal(models.TaskStateResolving))

				node, err := store.Get("/v1/run_once/some-guid")
				Ω(err).ShouldNot(HaveOccurred())
				Ω(node.Value).Should(Equal(runOnce.ToJSON()))
			})
		})
	})

	Describe("ResolveTask", func() {
		BeforeEach(func() {
			err := bbs.DesireTask(runOnce)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.ClaimTask(runOnce, "some-executor-id")
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.StartTask(runOnce, "some-container-handle")
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.CompleteTask(runOnce, true, "because i said so", "a result")
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.ResolvingTask(runOnce)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("should remove /run_once/<guid>", func() {
			err := bbs.ResolveTask(runOnce)
			Ω(err).ShouldNot(HaveOccurred())

			_, err = store.Get("/v1/run_once/some-guid")
			Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))
		})

		Context("when the store is out of commission", func() {
			itRetriesUntilStoreComesBack(func() error {
				return bbs.ResolveTask(runOnce)
			})
		})
	})

	Describe("WatchForCompletedTask", func() {
		var (
			events <-chan *models.Task
			stop   chan<- bool
			errors <-chan error
		)

		BeforeEach(func() {
			events, stop, errors = bbs.WatchForCompletedTask()

			err := bbs.DesireTask(runOnce)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.ClaimTask(runOnce, "executor-ID")
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.StartTask(runOnce, "container-handle")
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("should not send any events for state transitions that we are not interested in", func() {
			Consistently(events).ShouldNot(Receive())
		})

		It("should send an event down the pipe for completed run onces", func(done Done) {
			err := bbs.CompleteTask(runOnce, true, "a reason", "a result")
			Ω(err).ShouldNot(HaveOccurred())

			Expect(<-events).To(Equal(runOnce))

			close(done)
		})

		It("should not send an event down the pipe when resolved", func(done Done) {
			err := bbs.CompleteTask(runOnce, true, "a reason", "a result")
			Ω(err).ShouldNot(HaveOccurred())

			Expect(<-events).To(Equal(runOnce))

			err = bbs.ResolvingTask(runOnce)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.ResolveTask(runOnce)
			Ω(err).ShouldNot(HaveOccurred())

			Consistently(events).ShouldNot(Receive())

			close(done)
		})

		It("closes the events and errorschannel when told to stop", func(done Done) {
			stop <- true

			err := bbs.CompleteTask(runOnce, true, "a reason", "a result")
			Ω(err).ShouldNot(HaveOccurred())

			Ω(events).Should(BeClosed())
			Ω(errors).Should(BeClosed())

			close(done)
		})
	})
})
