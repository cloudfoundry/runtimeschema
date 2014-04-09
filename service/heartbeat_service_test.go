package service_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/service"
	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/storeadapter"
	"github.com/cloudfoundry/storeadapter/storerunner/etcdstorerunner"
	. "github.com/onsi/ginkgo"
	ginkoconfig "github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("HeartbeatService", func() {
	var etcdRunner = etcdstorerunner.NewETCDClusterRunner(5001+ginkoconfig.GinkgoConfig.ParallelNode, 1)
	var interruptService service.Service
	var logger *gosteno.Logger
	var heart service.Service
	var store storeadapter.StoreAdapter
	var interval = 1 * time.Second
	var serviceName = "example_service"
	var serviceId = "service-id"
	var serviceValue = "service-value"
	var expectedKey = "/v1/example_service/service-id"

	BeforeEach(func() {
		logger = gosteno.NewLogger("test")
		store = etcdRunner.Adapter()
		heart = service.NewHeartbeatService(store, logger, interval, serviceName, serviceId, serviceValue)
	})

	Context("when etcd is available", func() {

		BeforeEach(func() {
			etcdRunner.Start()
			interruptService = service.NewInterrupterService(logger)
			interruptService.Start(func() {
				etcdRunner.Stop()
			})
		})

		AfterEach(func() {
			interruptService.Stop()
			etcdRunner.Stop()
		})

		It("starting should not error", func() {
			err := heart.Start(func() {
				Fail("Premature interuption should not have occured")
			})
			defer heart.Stop()

			Ω(err).ShouldNot(HaveOccurred())
		})

		Context("while the heartbeat service is running", func() {

			BeforeEach(func() {
				err := heart.Start(func() {
					Fail("Premature interuption should not have occured")
				})
				Ω(err).ShouldNot(HaveOccurred())
				time.Sleep(10 * time.Millisecond)
			})

			AfterEach(func() {
				err := heart.Stop()
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("should put /key/value in the store with a TTL", func() {
				node, err := store.Get(expectedKey)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(node).Should(Equal(storeadapter.StoreNode{
					Key:   expectedKey,
					Value: []byte(serviceValue),
					TTL:   uint64(interval.Seconds()),
				}))
			})

			It("fails to start the heartbeat service a second time", func() {
				err := heart.Start(func() {
					Fail("Premature interuption should not have occured")
				})
				Ω(err).Should(HaveOccurred())
			})

			It("can be stopped and then started again", func() {
				err := heart.Stop()
				Ω(err).ShouldNot(HaveOccurred())

				err = heart.Start(func() {
					Fail("Premature interuption should not have occured")
				})
				Ω(err).ShouldNot(HaveOccurred())
			})
		})
	})

})
