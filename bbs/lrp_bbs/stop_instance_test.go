package lrp_bbs_test

import (
	. "github.com/cloudfoundry-incubator/runtime-schema/bbs/lrp_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("StopInstance", func() {
	var bbs *LRPBBS
	var stopInstance models.StopLRPInstance

	BeforeEach(func() {
		bbs = New(etcdClient)
		stopInstance = models.StopLRPInstance{
			InstanceGuid: "some-instance-guid",
		}
	})

	Describe("RequestStopLRPInstance", func() {
		It("creates /v1/stop-instance/<instance-guid>", func() {
			err := bbs.RequestStopLRPInstance(stopInstance)
			Ω(err).ShouldNot(HaveOccurred())

			node, err := etcdClient.Get("/v1/stop-instance/some-instance-guid")
			Ω(err).ShouldNot(HaveOccurred())

			Ω(node.Value).Should(Equal(stopInstance.ToJSON()))
		})

		Context("when the key already exists", func() {
			It("sets it again", func() {
				err := bbs.RequestStopLRPInstance(stopInstance)
				Ω(err).ShouldNot(HaveOccurred())

				err = bbs.RequestStopLRPInstance(stopInstance)
				Ω(err).ShouldNot(HaveOccurred())
			})
		})

		Context("when the store is out of commission", func() {
			itRetriesUntilStoreComesBack(func() error {
				return bbs.RequestStopLRPInstance(stopInstance)
			})
		})
	})

	Describe("RemoveStopLRPInstance", func() {
		It("removes the key if it exists", func() {
			err := bbs.RequestStopLRPInstance(stopInstance)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.RemoveStopLRPInstance(stopInstance)
			Ω(err).ShouldNot(HaveOccurred())

			_, err = etcdClient.Get("/v1/stop-instance/some-instance-guid")
			Ω(err).Should(MatchError(storeadapter.ErrorKeyNotFound))
		})

		It("does not error if the key does not exist", func() {
			err := bbs.RemoveStopLRPInstance(stopInstance)
			Ω(err).ShouldNot(HaveOccurred())
		})

		Context("when the store is out of commission", func() {
			BeforeEach(func() {
				err := bbs.RequestStopLRPInstance(stopInstance)
				Ω(err).ShouldNot(HaveOccurred())
			})

			itRetriesUntilStoreComesBack(func() error {
				return bbs.RemoveStopLRPInstance(stopInstance)
			})
		})
	})

	Describe("WatchForStopLRPInstance", func() {
		var (
			events       <-chan models.StopLRPInstance
			stop         chan<- bool
			errors       <-chan error
			stopped      bool
			stopInstance models.StopLRPInstance
		)

		BeforeEach(func() {
			events, stop, errors = bbs.WatchForStopLRPInstance()
		})

		AfterEach(func() {
			if !stopped {
				stop <- true
			}
		})

		It("sends an event down the pipe for creates", func() {
			err := bbs.RequestStopLRPInstance(stopInstance)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(events).Should(Receive(Equal(stopInstance)))
		})

		It("sends an event down the pipe for updates", func() {
			err := bbs.RequestStopLRPInstance(stopInstance)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(events).Should(Receive(Equal(stopInstance)))

			err = bbs.RequestStopLRPInstance(stopInstance)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(events).Should(Receive(Equal(stopInstance)))
		})

		It("does not send an event down the pipe for deletes", func() {
			err := bbs.RequestStopLRPInstance(stopInstance)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(events).Should(Receive(Equal(stopInstance)))

			err = bbs.RemoveStopLRPInstance(stopInstance)
			Ω(err).ShouldNot(HaveOccurred())

			Consistently(events).ShouldNot(Receive())
		})

		It("closes the events and errors channel when told to stop", func() {
			stop <- true
			stopped = true

			err := bbs.RequestStopLRPInstance(stopInstance)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(events).Should(BeClosed())
			Ω(errors).Should(BeClosed())
		})
	})

})
