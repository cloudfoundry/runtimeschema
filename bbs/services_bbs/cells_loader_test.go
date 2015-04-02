package services_bbs_test

import (
	"os"
	"time"

	"github.com/pivotal-golang/lager/lagertest"
	"github.com/tedsuo/ifrit"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/services_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/clock/fakeclock"
)

var _ = Describe("CellsLoader", func() {
	Describe("Cells", func() {

		const ttl = 10 * time.Second
		const retryInterval = time.Second
		var (
			clock *fakeclock.FakeClock

			bbs                *services_bbs.ServicesBBS
			heartbeat1         ifrit.Process
			heartbeat2         ifrit.Process
			firstCellPresence  models.CellPresence
			secondCellPresence models.CellPresence
			logger             *lagertest.TestLogger
		)

		BeforeEach(func() {
			logger = lagertest.NewTestLogger("test")
			clock = fakeclock.NewFakeClock(time.Now())
			bbs = services_bbs.New(consulAdapter, clock, logger)

			firstCellPresence = models.NewCellPresence("first-rep", "1.2.3.4", "the-zone", models.NewCellCapacity(128, 1024, 3))
			secondCellPresence = models.NewCellPresence("second-rep", "4.5.6.7", "the-zone", models.NewCellCapacity(128, 1024, 3))

			heartbeat1 = nil
			heartbeat2 = nil
		})

		AfterEach(func() {
			if heartbeat1 != nil {
				heartbeat1.Signal(os.Interrupt)
				Eventually(heartbeat1.Wait()).Should(Receive(BeNil()))
			}

			if heartbeat2 != nil {
				heartbeat2.Signal(os.Interrupt)
				Eventually(heartbeat2.Wait()).Should(Receive(BeNil()))
			}
		})

		Context("when there is a single cell", func() {
			var cellsLoader *services_bbs.CellsLoader
			var cells models.CellSet
			var err error

			BeforeEach(func() {
				cellsLoader = services_bbs.NewCellsLoader(logger, consulAdapter, clock)
				heartbeat1 = ifrit.Invoke(bbs.NewCellHeartbeat(firstCellPresence, ttl, retryInterval))
				cells, err = cellsLoader.Cells()
			})

			It("returns only one cell", func() {
				Ω(err).ShouldNot(HaveOccurred())
				Ω(cells).Should(HaveLen(1))
				Ω(cells).Should(HaveKey("first-rep"))
			})

			Context("when one more cell is added", func() {

				BeforeEach(func() {
					heartbeat2 = ifrit.Invoke(bbs.NewCellHeartbeat(secondCellPresence, ttl, retryInterval))
				})

				It("returns only one cell", func() {
					cells2, err2 := cellsLoader.Cells()
					Ω(err2).ShouldNot(HaveOccurred())
					Ω(cells2).Should(Equal(cells))
				})

				Context("when a new loader is created", func() {
					It("returns two cells", func() {
						newCellsLoader := services_bbs.NewCellsLoader(logger, consulAdapter, clock)
						cells, err = newCellsLoader.Cells()
						Ω(err).ShouldNot(HaveOccurred())
						Ω(cells).Should(HaveLen(2))
						Ω(cells).Should(HaveKey("first-rep"))
						Ω(cells).Should(HaveKey("second-rep"))
					})
				})
			})
		})
	})
})
