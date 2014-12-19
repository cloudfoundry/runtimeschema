package services_bbs_test

import (
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager/lagertest"
	"github.com/tedsuo/ifrit"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/lock_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/services_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

var _ = Describe("Receptor Service Registry", func() {
	var bbs *services_bbs.ServicesBBS
	var logger *lagertest.TestLogger

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		bbs = services_bbs.New(etcdClient, logger)
	})

	Describe("AuctioneerAddress", func() {
		Context("when able to get an auctioneer presence", func() {
			var heartbeater ifrit.Process
			var auctioneerPresence models.AuctioneerPresence

			JustBeforeEach(func() {
				lockBbs := lock_bbs.New(etcdClient, logger)
				auctioneerLock, err := lockBbs.NewAuctioneerLock(auctioneerPresence, time.Second)
				Ω(err).ShouldNot(HaveOccurred())
				heartbeater = ifrit.Invoke(auctioneerLock)
			})

			AfterEach(func() {
				heartbeater.Signal(os.Interrupt)
				Eventually(heartbeater.Wait()).Should(Receive(BeNil()))
			})

			Context("when the auctionner address is present", func() {
				BeforeEach(func() {
					auctioneerPresence = models.NewAuctioneerPresence("auctioneer-id", "auctioneer.example.com")
				})

				It("returns the address", func() {
					address, err := bbs.AuctioneerAddress()
					Ω(err).ShouldNot(HaveOccurred())
					Ω(address).Should(Equal(auctioneerPresence.AuctioneerAddress))
				})
			})
		})

		Context("when unable to get any auctioneer presences", func() {
			It("returns ErrServiceUnavailable", func() {
				_, err := bbs.AuctioneerAddress()
				Ω(err).Should(Equal(bbserrors.ErrServiceUnavailable))
			})
		})
	})
})
