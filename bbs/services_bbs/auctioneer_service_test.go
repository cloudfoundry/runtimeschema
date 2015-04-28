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
	"github.com/pivotal-golang/clock/fakeclock"
)

var _ = Describe("Receptor Service Registry", func() {
	var clock *fakeclock.FakeClock
	var bbs *services_bbs.ServicesBBS
	var logger *lagertest.TestLogger

	BeforeEach(func() {
		clock = fakeclock.NewFakeClock(time.Now())
		logger = lagertest.NewTestLogger("test")
		bbs = services_bbs.New(consulSession, clock, logger)
	})

	Describe("AuctioneerAddress", func() {
		Context("when able to get an auctioneer presence", func() {
			var heartbeater ifrit.Process
			var auctioneerPresence models.AuctioneerPresence

			JustBeforeEach(func() {
				lockBbs := lock_bbs.New(consulSession, clock, logger)
				auctioneerLock, err := lockBbs.NewAuctioneerLock(auctioneerPresence, 100*time.Millisecond)
				Expect(err).NotTo(HaveOccurred())
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
					Expect(err).NotTo(HaveOccurred())
					Expect(address).To(Equal(auctioneerPresence.AuctioneerAddress))
				})
			})
		})

		Context("when unable to get any auctioneer presences", func() {
			It("returns ErrServiceUnavailable", func() {
				_, err := bbs.AuctioneerAddress()
				Expect(err).To(Equal(bbserrors.ErrServiceUnavailable))
			})
		})
	})
})
