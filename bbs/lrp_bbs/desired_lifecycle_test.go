package lrp_bbs_test

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DesiredLRP Lifecycle", func() {
	const cellID = "some-cell-id"

	var lrp models.DesiredLRP

	BeforeEach(func() {
		rawMessage := json.RawMessage([]byte(`{"port":8080,"hosts":["route-1","route-2"]}`))
		lrp = models.DesiredLRP{
			Domain:      "tests",
			ProcessGuid: "some-process-guid",
			Instances:   5,
			RootFS:      "some:rootfs",
			MemoryMB:    1024,
			DiskMB:      512,
			Routes: map[string]*json.RawMessage{
				"router": &rawMessage,
			},
			Action: &models.DownloadAction{
				From: "http://example.com",
				To:   "/tmp/internet",
				User: "diego",
			},
		}
	})

	Describe("RemoveDesiredLRPByProcessGuid", func() {
		Context("when the desired LRP exists", func() {
			BeforeEach(func() {
				err := lrpBBS.DesireLRP(logger, lrp)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should delete it", func() {
				err := lrpBBS.RemoveDesiredLRPByProcessGuid(logger, lrp.ProcessGuid)
				Expect(err).NotTo(HaveOccurred())

				_, err = etcdClient.Get("/v1/desired/some-process-guid")
				Expect(err).To(MatchError(storeadapter.ErrorKeyNotFound))
			})

			Context("when there are running instances on a present cell", func() {
				cellPresence := models.NewCellPresence("the-cell-id", "cell.example.com", "az1", models.NewCellCapacity(128, 1024, 6), []string{}, []string{})

				BeforeEach(func() {
					testHelper.RegisterCell(cellPresence)

					for i := 0; i < lrp.Instances; i++ {
						err := lrpBBS.LegacyClaimActualLRP(
							logger,
							models.NewActualLRPKey(lrp.ProcessGuid, i, lrp.Domain),
							models.NewActualLRPInstanceKey(fmt.Sprintf("some-instance-guid-%d", i), cellPresence.CellID),
						)
						Expect(err).NotTo(HaveOccurred())
					}
				})

				It("stops all actual lrps for the desired lrp", func() {
					originalStopCallCount := fakeCellClient.StopLRPInstanceCallCount()

					err := lrpBBS.RemoveDesiredLRPByProcessGuid(logger, lrp.ProcessGuid)
					Expect(err).NotTo(HaveOccurred())

					Expect(fakeCellClient.StopLRPInstanceCallCount()).To(Equal(originalStopCallCount + (lrp.Instances)))

					stoppedActuals := make([]int, lrp.Instances)
					for i := 0; i < lrp.Instances; i++ {
						addr, key, _ := fakeCellClient.StopLRPInstanceArgsForCall(originalStopCallCount + i)
						Expect(addr).To(Equal(cellPresence.RepAddress))

						stoppedActuals[i] = key.Index
					}

					Expect(stoppedActuals).To(ConsistOf([]int{0, 1, 2, 3, 4}))
				})
			})
		})

		Context("when the desired LRP does not exist", func() {
			It("returns an ErrorKeyNotFound", func() {
				err := lrpBBS.RemoveDesiredLRPByProcessGuid(logger, "monkey")
				Expect(err).To(MatchError(bbserrors.ErrStoreResourceNotFound))
			})
		})
	})

	Describe("Updating DesireLRP", func() {
		var update models.DesiredLRPUpdate
		var desiredLRP models.DesiredLRP

		BeforeEach(func() {
			err := lrpBBS.DesireLRP(logger, lrp)
			Expect(err).NotTo(HaveOccurred())

			desiredLRP, err = lrpBBS.LegacyDesiredLRPByProcessGuid(logger, lrp.ProcessGuid)
			Expect(err).NotTo(HaveOccurred())

			update = models.DesiredLRPUpdate{}
		})

		Context("When the updates are valid", func() {
			BeforeEach(func() {
				annotation := "new-annotation"
				instances := 16

				rawMessage := json.RawMessage([]byte(`{"port":8080,"hosts":["new-route-1","new-route-2"]}`))
				update.Routes = map[string]*json.RawMessage{
					"router": &rawMessage,
				}
				update.Annotation = &annotation
				update.Instances = &instances
			})

			It("updates an existing DesireLRP", func() {
				err := lrpBBS.UpdateDesiredLRP(logger, lrp.ProcessGuid, update)
				Expect(err).NotTo(HaveOccurred())

				updated, err := lrpBBS.LegacyDesiredLRPByProcessGuid(logger, lrp.ProcessGuid)
				Expect(err).NotTo(HaveOccurred())

				Expect(updated.Routes).To(HaveKey("router"))
				json, err := update.Routes["router"].MarshalJSON()
				Expect(err).NotTo(HaveOccurred())
				updatedJson, err := updated.Routes["router"].MarshalJSON()
				Expect(err).NotTo(HaveOccurred())
				Expect(updatedJson).To(MatchJSON(string(json)))
				Expect(updated.Annotation).To(Equal(*update.Annotation))
				Expect(updated.Instances).To(Equal(*update.Instances))
				Expect(updated.ModificationTag.Epoch).To(Equal(desiredLRP.ModificationTag.Epoch))
				Expect(updated.ModificationTag.Index).To(Equal(desiredLRP.ModificationTag.Index + 1))
			})

			Context("when the instances are increased", func() {
				BeforeEach(func() {
					instances := 6
					update.Instances = &instances
				})

				Context("when an auctioneer is present", func() {
					BeforeEach(func() {
						auctioneerPresence := models.NewAuctioneerPresence("auctioneer-id", "example.com")
						testHelper.RegisterAuctioneer(auctioneerPresence)
					})

					It("emits start auction requests", func() {
						originalAuctionCallCount := fakeAuctioneerClient.RequestLRPAuctionsCallCount()

						err := lrpBBS.UpdateDesiredLRP(logger, lrp.ProcessGuid, update)
						Expect(err).NotTo(HaveOccurred())

						Consistently(fakeAuctioneerClient.RequestLRPAuctionsCallCount).Should(Equal(originalAuctionCallCount + 1))

						updated, err := lrpBBS.LegacyDesiredLRPByProcessGuid(logger, lrp.ProcessGuid)
						Expect(err).NotTo(HaveOccurred())

						_, startAuctions := fakeAuctioneerClient.RequestLRPAuctionsArgsForCall(originalAuctionCallCount)
						Expect(startAuctions).To(HaveLen(1))
						Expect(startAuctions[0].DesiredLRP).To(Equal(updated))
						Expect(startAuctions[0].Indices).To(HaveLen(1))
						Expect(startAuctions[0].Indices).To(ContainElement(uint(5)))
					})
				})
			})

			Context("when the instances are decreased", func() {
				BeforeEach(func() {
					instances := 2
					update.Instances = &instances
				})

				Context("when the cell is present", func() {
					cellPresence := models.NewCellPresence("the-cell-id", "cell.example.com", "az1", models.NewCellCapacity(128, 1024, 6), []string{}, []string{})

					BeforeEach(func() {
						testHelper.RegisterCell(cellPresence)

						for i := 0; i < lrp.Instances; i++ {
							err := lrpBBS.LegacyClaimActualLRP(
								logger,
								models.NewActualLRPKey(lrp.ProcessGuid, i, lrp.Domain),
								models.NewActualLRPInstanceKey(fmt.Sprintf("some-instance-guid-%d", i), cellPresence.CellID),
							)
							Expect(err).NotTo(HaveOccurred())
						}
					})

					It("stops the instances at the removed indices", func() {
						originalStopCallCount := fakeCellClient.StopLRPInstanceCallCount()

						err := lrpBBS.UpdateDesiredLRP(logger, lrp.ProcessGuid, update)
						Expect(err).NotTo(HaveOccurred())

						Expect(fakeCellClient.StopLRPInstanceCallCount()).To(Equal(originalStopCallCount + (lrp.Instances - *update.Instances)))

						stoppedActuals := make([]int, lrp.Instances-*update.Instances)
						for i := 0; i < (lrp.Instances - *update.Instances); i++ {
							addr, key, _ := fakeCellClient.StopLRPInstanceArgsForCall(originalStopCallCount + i)
							Expect(addr).To(Equal(cellPresence.RepAddress))

							stoppedActuals[i] = key.Index
						}

						Expect(stoppedActuals).To(ConsistOf([]int{2, 3, 4}))
					})
				})
			})
		})

		Context("When the updates are invalid", func() {
			It("instances cannot be less than zero", func() {
				instances := -1

				update := models.DesiredLRPUpdate{
					Instances: &instances,
				}

				desiredBeforeUpdate, err := lrpBBS.LegacyDesiredLRPByProcessGuid(logger, lrp.ProcessGuid)
				Expect(err).NotTo(HaveOccurred())

				err = lrpBBS.UpdateDesiredLRP(logger, lrp.ProcessGuid, update)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("instances"))

				desiredAfterUpdate, err := lrpBBS.LegacyDesiredLRPByProcessGuid(logger, lrp.ProcessGuid)
				Expect(err).NotTo(HaveOccurred())

				Expect(desiredAfterUpdate).To(Equal(desiredBeforeUpdate))
			})
		})

		Context("When the LRP does not exist", func() {
			It("returns an ErrorKeyNotFound", func() {
				instances := 0

				err := lrpBBS.UpdateDesiredLRP(logger, "garbage-guid", models.DesiredLRPUpdate{
					Instances: &instances,
				})
				Expect(err).To(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		})
	})
})
