package services_bbs_test

import (
	"os"
	"time"

	"github.com/hashicorp/consul/consul/structs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager/lagertest"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"

	"github.com/cloudfoundry-incubator/consuladapter"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/services_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/pivotal-golang/clock/fakeclock"
)

var _ = Describe("Receptor Service Registry", func() {
	var (
		clock *fakeclock.FakeClock

		bbs                    *services_bbs.ServicesBBS
		heartbeat1             ifrit.Process
		heartbeat2             ifrit.Process
		firstReceptorPresence  models.ReceptorPresence
		secondReceptorPresence models.ReceptorPresence
	)

	BeforeEach(func() {
		clock = fakeclock.NewFakeClock(time.Now())
		bbs = services_bbs.New(consulAdapter, clock, lagertest.NewTestLogger("test"))

		firstReceptorPresence = models.NewReceptorPresence("first-receptor", "first-receptor-url")
		secondReceptorPresence = models.NewReceptorPresence("second-receptor", "second-receptor-url")

		heartbeat1 = ifrit.Invoke(bbs.NewReceptorHeartbeat(firstReceptorPresence, structs.SessionTTLMin, 100*time.Millisecond))
		heartbeat2 = ifrit.Invoke(bbs.NewReceptorHeartbeat(secondReceptorPresence, structs.SessionTTLMin, 100*time.Millisecond))
	})

	AfterEach(func() {
		heartbeat1.Signal(os.Interrupt)
		heartbeat2.Signal(os.Interrupt)
		Eventually(heartbeat1.Wait()).Should(Receive(BeNil()))
		Eventually(heartbeat2.Wait()).Should(Receive(BeNil()))
	})

	Describe("MaintainReceptorPresence", func() {
		It("should put the receptor's presence in the store", func() {
			value, err := consulAdapter.GetValue(shared.ReceptorSchemaPath(firstReceptorPresence.ReceptorID))
			Ω(err).ShouldNot(HaveOccurred())

			expectedJSON, err := models.ToJSON(firstReceptorPresence)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(value).Should(MatchJSON(expectedJSON))
		})

		Context("when heartbeating stops", func() {
			BeforeEach(func() {
				ginkgomon.Interrupt(heartbeat1)
			})

			It("should eventually expire the data from the store", func() {
				key := shared.ReceptorSchemaPath(firstReceptorPresence.ReceptorID)
				Eventually(func() error {
					_, err := consulAdapter.GetValue(key)
					return err
				}, structs.SessionTTLMin+time.Second).Should(Equal(consuladapter.NewKeyNotFoundError(key)))
			})
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
