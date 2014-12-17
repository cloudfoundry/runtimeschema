package services_bbs_test

import (
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager/lagertest"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	. "github.com/cloudfoundry-incubator/runtime-schema/bbs/services_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

var _ = Describe("Receptor Service Registry", func() {
	var (
		bbs                    *ServicesBBS
		interval               = time.Second
		heartbeat1             ifrit.Process
		heartbeat2             ifrit.Process
		firstReceptorPresence  models.ReceptorPresence
		secondReceptorPresence models.ReceptorPresence
	)

	BeforeEach(func() {
		bbs = New(etcdClient, lagertest.NewTestLogger("test"))

		firstReceptorPresence = models.NewReceptorPresence("first-receptor", "first-receptor-url")
		secondReceptorPresence = models.NewReceptorPresence("second-receptor", "second-receptor-url")

		interval = 1 * time.Second

		heartbeat1 = ifrit.Invoke(bbs.NewReceptorHeartbeat(firstReceptorPresence, interval))
		heartbeat2 = ifrit.Invoke(bbs.NewReceptorHeartbeat(secondReceptorPresence, interval))
	})

	AfterEach(func() {
		heartbeat1.Signal(os.Interrupt)
		heartbeat2.Signal(os.Interrupt)
		Eventually(heartbeat1.Wait()).Should(Receive(BeNil()))
		Eventually(heartbeat2.Wait()).Should(Receive(BeNil()))
	})

	Describe("MaintainReceptorPresence", func() {
		It("should put /receptor/RECEPTOR_ID in the store with a TTL", func() {
			node, err := etcdClient.Get("/v1/receptor/" + firstReceptorPresence.ReceptorID)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(node.TTL).ShouldNot(BeZero())

			expectedJSON, err := models.ToJSON(firstReceptorPresence)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(node.Value).Should(MatchJSON(expectedJSON))
		})
	})

	Describe("Receptor", func() {
		Context("when a receptor exists", func() {
			It("returns a ReceptorPresence", func() {
				receptorPresence, err := bbs.Receptor()
				Ω(err).ShouldNot(HaveOccurred())
				Ω([]string{"first-receptor", "second-receptor"}).Should(ContainElement(receptorPresence.ReceptorID))
			})
		})

		Context("when a receptor does not exist", func() {
			BeforeEach(func() {
				ginkgomon.Interrupt(heartbeat1)
				ginkgomon.Interrupt(heartbeat2)
			})

			It("returns ErrServiceUnavailable", func() {
				_, err := bbs.Receptor()
				Ω(err).Should(Equal(bbserrors.ErrServiceUnavailable))
			})
		})
	})
})
