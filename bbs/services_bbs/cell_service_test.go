package services_bbs_test

import (
	"os"
	"time"

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

var _ = Describe("Cell Service Registry", func() {
	const retryInterval = time.Second
	var (
		clock *fakeclock.FakeClock

		bbs                *services_bbs.ServicesBBS
		presence1          ifrit.Process
		presence2          ifrit.Process
		firstCellPresence  models.CellPresence
		secondCellPresence models.CellPresence
	)

	BeforeEach(func() {
		clock = fakeclock.NewFakeClock(time.Now())
		bbs = services_bbs.New(consulSession, clock, lagertest.NewTestLogger("test"))

		firstCellPresence = models.NewCellPresence("first-rep", "1.2.3.4", "the-zone", models.NewCellCapacity(128, 1024, 3), []string{}, []string{})
		secondCellPresence = models.NewCellPresence("second-rep", "4.5.6.7", "the-zone", models.NewCellCapacity(128, 1024, 3), []string{}, []string{})

		presence1 = nil
		presence2 = nil
	})

	AfterEach(func() {
		if presence1 != nil {
			presence1.Signal(os.Interrupt)
			Eventually(presence1.Wait()).Should(Receive(BeNil()))
		}

		if presence2 != nil {
			presence2.Signal(os.Interrupt)
			Eventually(presence2.Wait()).Should(Receive(BeNil()))
		}
	})

	Describe("MaintainCellPresence", func() {
		BeforeEach(func() {
			presence1 = ifrit.Invoke(bbs.NewCellPresence(firstCellPresence, retryInterval))
		})

		It("should put /cell/CELL_ID in the store", func() {
			expectedJSON, err := models.ToJSON(firstCellPresence)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() []byte {
				value, _ := consulSession.GetAcquiredValue(shared.CellSchemaPath(firstCellPresence.CellID))
				return value
			}, time.Second).Should(MatchJSON(expectedJSON))
		})
	})

	Describe("CellById", func() {
		Context("when the cell exists", func() {
			BeforeEach(func() {
				presence1 = ifrit.Invoke(bbs.NewCellPresence(firstCellPresence, retryInterval))
			})

			It("returns the correct CellPresence", func() {
				Eventually(func() (models.CellPresence, error) {
					return bbs.CellById(firstCellPresence.CellID)
				}).Should(Equal(firstCellPresence))
			})
		})

		Context("when the cell does not exist", func() {
			It("returns ErrStoreResourceNotFound", func() {
				_, err := bbs.CellById(firstCellPresence.CellID)
				Expect(err).To(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		})
	})

	Describe("Cells", func() {
		Context("when there are available Cells", func() {
			BeforeEach(func() {
				presence1 = ifrit.Invoke(bbs.NewCellPresence(firstCellPresence, retryInterval))
				presence2 = ifrit.Invoke(bbs.NewCellPresence(secondCellPresence, retryInterval))
			})

			It("should get from /v1/cell/", func() {
				Eventually(func() ([]models.CellPresence, error) {
					return bbs.Cells()
				}).Should(ConsistOf(firstCellPresence, secondCellPresence))
			})

			Context("when there is unparsable JSON in there...", func() {
				BeforeEach(func() {
					err := consulSession.AcquireLock(shared.CellSchemaPath("blah"), []byte("ß"))
					Expect(err).NotTo(HaveOccurred())

					Eventually(func() map[string][]byte {
						cells, err := consulSession.ListAcquiredValues(shared.CellSchemaRoot)
						Expect(err).NotTo(HaveOccurred())
						return cells
					}, 1, 50*time.Millisecond).Should(HaveLen(3))
				})

				It("should ignore the unparsable JSON and move on", func() {
					cellPresences, err := bbs.Cells()
					Expect(err).NotTo(HaveOccurred())
					Expect(cellPresences).To(HaveLen(2))
					Expect(cellPresences).To(ContainElement(firstCellPresence))
					Expect(cellPresences).To(ContainElement(secondCellPresence))
				})
			})
		})

		Context("when there are none", func() {
			It("should return empty", func() {
				reps, err := bbs.Cells()
				Expect(err).NotTo(HaveOccurred())
				Expect(reps).To(BeEmpty())
			})
		})
	})

	Describe("CellEvents", func() {
		var receivedEvents <-chan services_bbs.CellEvent
		var otherSession *consuladapter.Session

		setPresences := func() {
			presence1 = ifrit.Invoke(bbs.NewCellPresence(firstCellPresence, retryInterval))

			Eventually(func() ([]models.CellPresence, error) {
				return bbs.Cells()
			}).Should(HaveLen(1))
		}

		BeforeEach(func() {
			otherSession = consulRunner.NewSession("other-session")
			otherbbs := services_bbs.New(otherSession, clock, lagertest.NewTestLogger("test"))
			receivedEvents = otherbbs.CellEvents()
		})

		Context("when the store is up", func() {
			Context("when cells are present and then one disappears", func() {
				BeforeEach(func() {
					otherSession = consulRunner.NewSession("other-session")
					otherbbs := services_bbs.New(otherSession, clock, lagertest.NewTestLogger("test"))
					receivedEvents = otherbbs.CellEvents()

					setPresences()
					ginkgomon.Interrupt(presence1)

					Eventually(func() ([]models.CellPresence, error) {
						return bbs.Cells()
					}).Should(HaveLen(0))
				})

				AfterEach(func() {
					otherSession.Destroy()
				})

				It("receives a CellDisappeared event", func() {
					Eventually(receivedEvents).Should(Receive(Equal(
						services_bbs.CellDisappearedEvent{IDs: []string{firstCellPresence.CellID}},
					)))
				})
			})
		})

		Context("when the store is down", func() {
			BeforeEach(func() {
				otherSession = consulRunner.NewSession("other-session")
				otherbbs := services_bbs.New(otherSession, clock, lagertest.NewTestLogger("test"))
				receivedEvents = otherbbs.CellEvents()

				consulRunner.Stop()
			})

			It("attaches when the store is back", func() {
				consulRunner.Start()
				consulRunner.WaitUntilReady()

				By("setting presences")
				newSession, err := consulSession.Recreate()
				Expect(err).NotTo(HaveOccurred())
				bbs = services_bbs.New(newSession, clock, lagertest.NewTestLogger("test"))

				setPresences()

				Eventually(func() ([]models.CellPresence, error) {
					return bbs.Cells()
				}).Should(HaveLen(1))

				time.Sleep(2 * time.Second) //wait for a watch fail cycle

				By("stopping the presence")
				ginkgomon.Interrupt(presence1)

				Eventually(func() ([]models.CellPresence, error) {
					return bbs.Cells()
				}).Should(HaveLen(0))

				Eventually(receivedEvents).Should(Receive(Equal(
					services_bbs.CellDisappearedEvent{IDs: []string{firstCellPresence.CellID}},
				)))
			})
		})
	})
})
