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
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/services_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/pivotal-golang/clock/fakeclock"
)

var _ = Describe("Cell Service Registry", func() {
	const ttl = 10 * time.Second
	const retryInterval = time.Second
	var (
		clock *fakeclock.FakeClock

		bbs                *services_bbs.ServicesBBS
		heartbeat1         ifrit.Process
		heartbeat2         ifrit.Process
		firstCellPresence  models.CellPresence
		secondCellPresence models.CellPresence
	)

	BeforeEach(func() {
		clock = fakeclock.NewFakeClock(time.Now())
		bbs = services_bbs.New(consulAdapter, clock, lagertest.NewTestLogger("test"))

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

	Describe("MaintainCellPresence", func() {
		BeforeEach(func() {
			heartbeat1 = ifrit.Invoke(bbs.NewCellHeartbeat(firstCellPresence, ttl, retryInterval))
		})

		It("should put /cell/CELL_ID in the store with a TTL", func() {
			expectedJSON, err := models.ToJSON(firstCellPresence)
			Ω(err).ShouldNot(HaveOccurred())

			Consistently(func() ([]byte, error) {
				return consulAdapter.GetValue(shared.CellSchemaPath(firstCellPresence.CellID))
			}, ttl+time.Second).Should(MatchJSON(expectedJSON))
		})
	})

	Describe("CellById", func() {
		Context("when the cell exists", func() {
			BeforeEach(func() {
				heartbeat1 = ifrit.Invoke(bbs.NewCellHeartbeat(firstCellPresence, ttl, retryInterval))
			})

			It("returns the correct CellPresence", func() {
				cellPresence, err := bbs.CellById(firstCellPresence.CellID)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(cellPresence).Should(Equal(firstCellPresence))
			})
		})

		Context("when the cell does not exist", func() {
			It("returns ErrStoreResourceNotFound", func() {
				_, err := bbs.CellById(firstCellPresence.CellID)
				Ω(err).Should(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		})
	})

	Describe("Cells", func() {
		Context("when there are available Cells", func() {
			BeforeEach(func() {
				heartbeat1 = ifrit.Invoke(bbs.NewCellHeartbeat(firstCellPresence, ttl, retryInterval))
				heartbeat2 = ifrit.Invoke(bbs.NewCellHeartbeat(secondCellPresence, ttl, retryInterval))
			})

			It("should get from /v1/cell/", func() {
				cellPresences, err := bbs.Cells()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(cellPresences).Should(HaveLen(2))
				Ω(cellPresences).Should(ContainElement(firstCellPresence))
				Ω(cellPresences).Should(ContainElement(secondCellPresence))
			})

			Context("when there is unparsable JSON in there...", func() {
				BeforeEach(func() {
					err := consulAdapter.SetValue(shared.CellSchemaPath("blah"), []byte("ß"))
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("should ignore the unparsable JSON and move on", func() {
					cellPresences, err := bbs.Cells()
					Ω(err).ShouldNot(HaveOccurred())
					Ω(cellPresences).Should(HaveLen(2))
					Ω(cellPresences).Should(ContainElement(firstCellPresence))
					Ω(cellPresences).Should(ContainElement(secondCellPresence))
				})
			})
		})

		Context("when there are none", func() {
			It("should return empty", func() {
				reps, err := bbs.Cells()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(reps).Should(BeEmpty())
			})
		})
	})

	Describe("WaitForCellEvent", func() {
		Context("when the store is around", func() {
			var receivedEvents <-chan services_bbs.CellEvent

			BeforeEach(func() {
				eventChan := make(chan services_bbs.CellEvent)
				receivedEvents = eventChan

				go func() {
					defer GinkgoRecover()

					for {
						event, err := bbs.WaitForCellEvent()
						if err == nil {
							eventChan <- event
							break
						}

						time.Sleep(50 * time.Millisecond)
					}
				}()
			})

			Context("when cells are present and then one disappears", func() {
				BeforeEach(func() {
					expectedPresence1, err := models.ToJSON(firstCellPresence)
					Ω(err).ShouldNot(HaveOccurred())

					heartbeat1 = ifrit.Invoke(bbs.NewCellHeartbeat(firstCellPresence, ttl, retryInterval))
					Eventually(func() ([]byte, error) {
						return consulAdapter.GetValue(shared.CellSchemaPath(firstCellPresence.CellID))
					}).Should(Equal(expectedPresence1))

					expectedPresence2, err := models.ToJSON(secondCellPresence)
					Ω(err).ShouldNot(HaveOccurred())

					heartbeat2 = ifrit.Invoke(bbs.NewCellHeartbeat(secondCellPresence, ttl, retryInterval))
					Eventually(func() ([]byte, error) {
						return consulAdapter.GetValue(shared.CellSchemaPath(secondCellPresence.CellID))
					}).Should(Equal(expectedPresence2))

					time.Sleep(200 * time.Millisecond) // give the watcher time to register the cells

					ginkgomon.Interrupt(heartbeat1)
				})

				AfterEach(func() {
					ginkgomon.Interrupt(heartbeat2)
				})

				It("receives a CellDisappeared event", func() {
					Eventually(receivedEvents, 20*time.Second).Should(Receive(Equal(services_bbs.CellDisappearedEvent{
						IDs: []string{firstCellPresence.CellID},
					})))
				})
			})
		})

		Context("when the store is down", func() {
			BeforeEach(func() {
				consulRunner.Reset()
			})

			It("returns an error", func() {
				_, err := bbs.WaitForCellEvent()
				Ω(err).Should(HaveOccurred())
			})
		})
	})
})
