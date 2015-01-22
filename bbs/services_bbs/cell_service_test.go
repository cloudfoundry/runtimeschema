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
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/timeprovider/faketimeprovider"
	"github.com/cloudfoundry/storeadapter"
)

var _ = Describe("Cell Service Registry", func() {
	const interval = time.Second
	var (
		timeProvider *faketimeprovider.FakeTimeProvider

		bbs                *ServicesBBS
		heartbeat1         ifrit.Process
		heartbeat2         ifrit.Process
		firstCellPresence  models.CellPresence
		secondCellPresence models.CellPresence
	)

	BeforeEach(func() {
		timeProvider = faketimeprovider.New(time.Now())
		bbs = New(etcdClient, timeProvider, lagertest.NewTestLogger("test"))

		firstCellPresence = models.NewCellPresence("first-rep", "lucid64", "1.2.3.4", "the-zone")
		secondCellPresence = models.NewCellPresence("second-rep", ".Net", "4.5.6.7", "the-zone")

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
			heartbeat1 = ifrit.Invoke(bbs.NewCellHeartbeat(firstCellPresence, interval))
		})

		It("should put /cell/CELL_ID in the store with a TTL", func() {
			node, err := etcdClient.Get("/v1/cell/" + firstCellPresence.CellID)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(node.TTL).ShouldNot(BeZero())

			expectedJSON, err := models.ToJSON(firstCellPresence)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(node.Value).Should(MatchJSON(expectedJSON))
		})
	})

	Describe("CellById", func() {
		Context("when the cell exists", func() {
			BeforeEach(func() {
				heartbeat1 = ifrit.Invoke(bbs.NewCellHeartbeat(firstCellPresence, interval))
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
				heartbeat1 = ifrit.Invoke(bbs.NewCellHeartbeat(firstCellPresence, interval))
				heartbeat2 = ifrit.Invoke(bbs.NewCellHeartbeat(secondCellPresence, interval))
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
					etcdClient.Create(storeadapter.StoreNode{
						Key:   shared.CellSchemaPath("blah"),
						Value: []byte("ß"),
					})
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
			var receivedEvents <-chan CellEvent

			BeforeEach(func() {
				eventChan := make(chan CellEvent, 1)
				receivedEvents = eventChan

				go func() {
					defer GinkgoRecover()

					event, err := bbs.WaitForCellEvent()
					Ω(err).ShouldNot(HaveOccurred())

					eventChan <- event
				}()

				time.Sleep(100 * time.Millisecond) //give the watcher a chance to connect
			})

			Context("when a cell presence appears", func() {
				BeforeEach(func() {
					heartbeat1 = ifrit.Invoke(bbs.NewCellHeartbeat(firstCellPresence, interval))
				})

				It("receives a CellAppeared event", func() {
					Eventually(receivedEvents).Should(Receive(Equal(CellAppearedEvent{
						Presence: firstCellPresence,
					})))
				})

				Describe("watching again", func() {
					var receivedEvents <-chan CellEvent

					BeforeEach(func() {
						eventChan := make(chan CellEvent, 1)
						receivedEvents = eventChan

						go func() {
							defer GinkgoRecover()

							event, err := bbs.WaitForCellEvent()
							Ω(err).ShouldNot(HaveOccurred())

							eventChan <- event
						}()

						time.Sleep(100 * time.Millisecond) //give the watcher a chance to connect
					})

					Context("when the cell then disappears", func() {
						BeforeEach(func() {
							ginkgomon.Interrupt(heartbeat1)
						})

						It("receives a CellDisappeared event", func() {
							Eventually(receivedEvents).Should(Receive(Equal(CellDisappearedEvent{
								Presence: firstCellPresence,
							})))
						})
					})
				})
			})
		})

		Context("when the store is down", func() {
			BeforeEach(func() {
				etcdRunner.Stop()
			})

			It("returns an error", func() {
				_, err := bbs.WaitForCellEvent()
				Ω(err).Should(HaveOccurred())
			})
		})
	})
})
