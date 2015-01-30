package lrp_bbs_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/lrp_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LrpGetters", func() {
	Context("DesiredLRPs", func() {
		var (
			desiredLrp1 models.DesiredLRP
			desiredLrp2 models.DesiredLRP
			desiredLrp3 models.DesiredLRP
		)

		BeforeEach(func() {
			desiredLrp1 = models.DesiredLRP{
				Domain:      "tests",
				ProcessGuid: "guidA",
				Stack:       "stack",
				Instances:   1,
				Action: &models.DownloadAction{
					From: "http://example.com",
					To:   "/tmp/internet",
				},
			}

			desiredLrp2 = models.DesiredLRP{
				Domain:      "tests",
				ProcessGuid: "guidB",
				Stack:       "stack",
				Instances:   1,
				Action: &models.DownloadAction{
					From: "http://example.com",
					To:   "/tmp/internet",
				},
			}

			desiredLrp3 = models.DesiredLRP{
				Domain:      "tests",
				ProcessGuid: "guidC",
				Stack:       "stack",
				Instances:   1,
				Action: &models.DownloadAction{
					From: "http://example.com",
					To:   "/tmp/internet",
				},
			}
		})

		JustBeforeEach(func() {
			err := bbs.DesireLRP(logger, desiredLrp1)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.DesireLRP(logger, desiredLrp2)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.DesireLRP(logger, desiredLrp3)
			Ω(err).ShouldNot(HaveOccurred())
		})

		Describe("DesiredLRPs", func() {
			It("returns all desired long running processes", func() {
				all, err := bbs.DesiredLRPs()
				Ω(err).ShouldNot(HaveOccurred())

				Ω(all).Should(HaveLen(3))
				Ω(all).Should(ContainElement(desiredLrp1))
				Ω(all).Should(ContainElement(desiredLrp2))
				Ω(all).Should(ContainElement(desiredLrp3))
			})
		})

		Describe("DesiredLRPsByDomain", func() {
			BeforeEach(func() {
				desiredLrp1.Domain = "domain-1"
				desiredLrp2.Domain = "domain-1"
				desiredLrp3.Domain = "domain-2"
			})

			It("returns all desired long running processes for the given domain", func() {
				byDomain, err := bbs.DesiredLRPsByDomain("domain-1")
				Ω(err).ShouldNot(HaveOccurred())
				Ω(byDomain).Should(ConsistOf([]models.DesiredLRP{desiredLrp1, desiredLrp2}))

				byDomain, err = bbs.DesiredLRPsByDomain("domain-2")
				Ω(err).ShouldNot(HaveOccurred())
				Ω(byDomain).Should(ConsistOf([]models.DesiredLRP{desiredLrp3}))
			})

			It("blows up with an empty string domain", func() {
				_, err := bbs.DesiredLRPsByDomain("")
				Ω(err).Should(Equal(lrp_bbs.ErrNoDomain))
			})
		})

		Describe("DesiredLRPByProcessGuid", func() {
			It("returns all desired long running processes", func() {
				desiredLrp, err := bbs.DesiredLRPByProcessGuid("guidA")
				Ω(err).ShouldNot(HaveOccurred())

				Ω(desiredLrp).Should(Equal(desiredLrp1))
			})

			Context("when the Desired LRP does not exist", func() {
				It("returns ErrStoreResourceNotFound", func() {
					_, err := bbs.DesiredLRPByProcessGuid("non-existent")
					Ω(err).Should(Equal(bbserrors.ErrStoreResourceNotFound))
				})
			})
		})
	})

	Context("ActualLRPs", func() {
		var (
			runningLrp1 models.ActualLRP
			runningLrp2 models.ActualLRP
			runningLrp3 models.ActualLRP

			newLrp models.ActualLRP
		)

		BeforeEach(func() {
			netInfo := models.NewActualLRPNetInfo("127.0.0.1", []models.PortMapping{{8080, 80}})

			runningLrp1 = models.ActualLRP{
				ActualLRPKey:          models.NewActualLRPKey("guidA", 1, "domain-a"),
				ActualLRPContainerKey: models.NewActualLRPContainerKey("some-instance-guid-1", "cell-id"),
				ActualLRPNetInfo:      netInfo,
				State:                 models.ActualLRPStateRunning,
				Since:                 clock.Now().UnixNano(),
			}

			runningLrp2 = models.ActualLRP{
				ActualLRPKey:          models.NewActualLRPKey("guidB", 2, "domain-b"),
				ActualLRPContainerKey: models.NewActualLRPContainerKey("some-instance-guid-2", "cell-id"),
				ActualLRPNetInfo:      netInfo,
				State:                 models.ActualLRPStateRunning,
				Since:                 clock.Now().UnixNano(),
			}

			runningLrp3 = models.ActualLRP{
				ActualLRPKey:          models.NewActualLRPKey("guidC", 3, "domain-b"),
				ActualLRPContainerKey: models.NewActualLRPContainerKey("some-instance-guid-3", "cell-id"),
				ActualLRPNetInfo:      netInfo,
				State:                 models.ActualLRPStateRunning,
				Since:                 clock.Now().UnixNano(),
			}

			err := bbs.StartActualLRP(runningLrp1.ActualLRPKey, runningLrp1.ActualLRPContainerKey, runningLrp1.ActualLRPNetInfo, logger)
			Ω(err).ShouldNot(HaveOccurred())

			createAndClaim(
				models.DesiredLRP{ProcessGuid: "guidA", Domain: "test", Instances: 1},
				0,
				models.NewActualLRPContainerKey("some-instance-guid", "cell-id"),
				logger,
			)

			newLrp, err = bbs.ActualLRPByProcessGuidAndIndex("guidA", 0)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.StartActualLRP(runningLrp2.ActualLRPKey, runningLrp2.ActualLRPContainerKey, runningLrp2.ActualLRPNetInfo, logger)
			Ω(err).ShouldNot(HaveOccurred())
		})

		Describe("ActualLRPs", func() {
			It("returns all actual long running processes", func() {
				all, err := bbs.ActualLRPs()
				Ω(err).ShouldNot(HaveOccurred())

				Ω(all).Should(HaveLen(3))
				Ω(all).Should(ContainElement(runningLrp1))
				Ω(all).Should(ContainElement(newLrp))
				Ω(all).Should(ContainElement(runningLrp2))
			})
		})

		Describe("ActualLRPsByCellID", func() {
			BeforeEach(func() {
				createAndClaim(
					models.DesiredLRP{ProcessGuid: "some-other-process", Domain: "some-other-domain", Instances: 1},
					0,
					models.NewActualLRPContainerKey("some-other-instance", "some-other-cell"),
					logger,
				)
			})

			It("returns actual long running processes belongs to 'cell-id'", func() {
				actualLrpsForMainCell, err := bbs.ActualLRPsByCellID("cell-id")
				Ω(err).ShouldNot(HaveOccurred())

				Ω(actualLrpsForMainCell).Should(ConsistOf(runningLrp1, newLrp, runningLrp2))

				actualLrpsForOtherCell, err := bbs.ActualLRPsByCellID("some-other-cell")
				Ω(err).ShouldNot(HaveOccurred())

				Ω(actualLrpsForOtherCell).Should(HaveLen(1))
			})
		})

		Describe("ActualLRPsByProcessGuid", func() {
			It("should fetch all LRPs for the specified guid", func() {
				lrps, err := bbs.ActualLRPsByProcessGuid("guidA")
				Ω(err).ShouldNot(HaveOccurred())
				Ω(lrps).Should(HaveLen(2))
				Ω(lrps).Should(ContainElement(runningLrp1))
				Ω(lrps).Should(ContainElement(newLrp))
			})
		})

		Describe("ActualLRPByProcessGuidAndIndex", func() {
			It("should fetch the LRP for the specified guid", func() {
				lrp, err := bbs.ActualLRPByProcessGuidAndIndex("guidA", 1)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(lrp).Should(Equal(runningLrp1))
			})
		})

		Describe("ActualLRPsByDomain", func() {
			BeforeEach(func() {
				err := bbs.StartActualLRP(runningLrp3.ActualLRPKey, runningLrp3.ActualLRPContainerKey, runningLrp3.ActualLRPNetInfo, logger)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("should fetch all LRPs for the specified guid", func() {
				lrps, err := bbs.ActualLRPsByDomain("domain-b")
				Ω(err).ShouldNot(HaveOccurred())

				Ω(lrps).Should(HaveLen(2))
				Ω(lrps).ShouldNot(ContainElement(runningLrp1))
				Ω(lrps).Should(ContainElement(runningLrp2))
				Ω(lrps).Should(ContainElement(runningLrp3))
			})

			Context("when there are no actual LRPs in the requested domain", func() {
				It("returns an empty list", func() {
					lrps, err := bbs.ActualLRPsByDomain("bogus-domain")
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrps).ShouldNot(BeNil())
					Ω(lrps).Should(HaveLen(0))
				})
			})
		})
	})
})
