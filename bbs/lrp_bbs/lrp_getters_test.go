package lrp_bbs_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/lrp_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LrpGetters", func() {
	var (
		desiredLrp1 models.DesiredLRP
		desiredLrp2 models.DesiredLRP
		desiredLrp3 models.DesiredLRP

		runningLrp1 models.ActualLRP
		runningLrp2 models.ActualLRP
		runningLrp3 models.ActualLRP
		newLrp      models.ActualLRP

		newLrpProcessGuid  string
		newLrpInstanceGuid string
		newLrpCellId       string
		newLrpDomain       string
		newLrpIndex        int
	)

	BeforeEach(func() {
		desiredLrp1 = models.DesiredLRP{
			Domain:      "tests",
			ProcessGuid: "guidA",
			Stack:       "stack",
			Instances:   1,
			Action: models.ExecutorAction{
				Action: models.DownloadAction{
					From: "http://example.com",
					To:   "/tmp/internet",
				},
			},
		}

		desiredLrp2 = models.DesiredLRP{
			Domain:      "tests",
			ProcessGuid: "guidB",
			Stack:       "stack",
			Instances:   1,
			Action: models.ExecutorAction{
				Action: models.DownloadAction{
					From: "http://example.com",
					To:   "/tmp/internet",
				},
			},
		}

		desiredLrp3 = models.DesiredLRP{
			Domain:      "tests",
			ProcessGuid: "guidC",
			Stack:       "stack",
			Instances:   1,
			Action: models.ExecutorAction{
				Action: models.DownloadAction{
					From: "http://example.com",
					To:   "/tmp/internet",
				},
			},
		}

		newLrpProcessGuid = "guidA"
		newLrpInstanceGuid = "some-instance-guid"
		newLrpIndex = 2
		newLrpCellId = "cell-id"
		newLrpDomain = "test"

		runningLrp1 = models.ActualLRP{
			ProcessGuid:  "guidA",
			Index:        1,
			InstanceGuid: "some-instance-guid-1",
			Domain:       "domain-a",
			State:        models.ActualLRPStateRunning,
			Since:        timeProvider.Time().UnixNano(),
			CellID:       "cell-id",
		}

		runningLrp2 = models.ActualLRP{
			ProcessGuid:  "guidB",
			Index:        2,
			InstanceGuid: "some-instance-guid-2",
			Domain:       "domain-b",
			State:        models.ActualLRPStateRunning,
			Since:        timeProvider.Time().UnixNano(),
			CellID:       "cell-id",
		}

		runningLrp3 = models.ActualLRP{
			ProcessGuid:  "guidC",
			Index:        3,
			InstanceGuid: "some-instance-guid-3",
			Domain:       "domain-b",
			State:        models.ActualLRPStateRunning,
			Since:        timeProvider.Time().UnixNano(),
			CellID:       "cell-id",
		}
	})

	Describe("GetAllDesiredLRPs", func() {
		BeforeEach(func() {
			err := bbs.DesireLRP(desiredLrp1)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.DesireLRP(desiredLrp2)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.DesireLRP(desiredLrp3)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("returns all desired long running processes", func() {
			all, err := bbs.GetAllDesiredLRPs()
			Ω(err).ShouldNot(HaveOccurred())

			Ω(all).Should(HaveLen(3))
			Ω(all).Should(ContainElement(desiredLrp1))
			Ω(all).Should(ContainElement(desiredLrp2))
			Ω(all).Should(ContainElement(desiredLrp3))
		})
	})

	Describe("GetAllDesiredLRPsByDomain", func() {
		BeforeEach(func() {
			desiredLrp1.Domain = "domain-1"
			desiredLrp2.Domain = "domain-1"
			desiredLrp3.Domain = "domain-2"

			err := bbs.DesireLRP(desiredLrp1)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.DesireLRP(desiredLrp2)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.DesireLRP(desiredLrp3)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("returns all desired long running processes for the given domain", func() {
			byDomain, err := bbs.GetAllDesiredLRPsByDomain("domain-1")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(byDomain).Should(ConsistOf([]models.DesiredLRP{desiredLrp1, desiredLrp2}))

			byDomain, err = bbs.GetAllDesiredLRPsByDomain("domain-2")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(byDomain).Should(ConsistOf([]models.DesiredLRP{desiredLrp3}))
		})

		It("blows up with an empty string domain", func() {
			_, err := bbs.GetAllDesiredLRPsByDomain("")
			Ω(err).Should(Equal(lrp_bbs.ErrNoDomain))
		})
	})

	Describe("GetDesiredLRPByProcessGuid", func() {
		BeforeEach(func() {
			err := bbs.DesireLRP(desiredLrp1)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.DesireLRP(desiredLrp2)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.DesireLRP(desiredLrp3)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("returns all desired long running processes", func() {
			desiredLrp, err := bbs.GetDesiredLRPByProcessGuid("guidA")
			Ω(err).ShouldNot(HaveOccurred())

			Ω(desiredLrp).Should(Equal(desiredLrp1))
		})
	})

	Describe("GetAllActualLRPs", func() {
		BeforeEach(func() {
			err := bbs.ReportActualLRPAsRunning(runningLrp1, "cell-id")
			Ω(err).ShouldNot(HaveOccurred())

			newLrp, err = bbs.ReportActualLRPAsStarting(newLrpProcessGuid, newLrpInstanceGuid, newLrpCellId, newLrpDomain, newLrpIndex)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.ReportActualLRPAsRunning(runningLrp2, "cell-id")
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("returns all actual long running processes", func() {
			all, err := bbs.GetAllActualLRPs()
			Ω(err).ShouldNot(HaveOccurred())

			Ω(all).Should(HaveLen(3))
			Ω(all).Should(ContainElement(runningLrp1))
			Ω(all).Should(ContainElement(newLrp))
			Ω(all).Should(ContainElement(runningLrp2))
		})
	})

	Describe("GetAllActualLRPsByCellID", func() {
		BeforeEach(func() {
			err := bbs.ReportActualLRPAsRunning(runningLrp1, "cell-id")
			Ω(err).ShouldNot(HaveOccurred())

			newLrp, err = bbs.ReportActualLRPAsStarting(newLrpProcessGuid, newLrpInstanceGuid, newLrpCellId, newLrpDomain, newLrpIndex)
			Ω(err).ShouldNot(HaveOccurred())

			_, err = bbs.ReportActualLRPAsStarting("some-other-process", "some-other-instance", "some-other-cell", "some-other-domain", 0)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.ReportActualLRPAsRunning(runningLrp2, "cell-id")
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("returns actual long running processes belongs to 'cell-id'", func() {
			actualLrpsForMainCell, err := bbs.GetAllActualLRPsByCellID("cell-id")
			Ω(err).ShouldNot(HaveOccurred())

			Ω(actualLrpsForMainCell).Should(ConsistOf(runningLrp1, newLrp, runningLrp2))

			actualLrpsForOtherCell, err := bbs.GetAllActualLRPsByCellID("some-other-cell")
			Ω(err).ShouldNot(HaveOccurred())

			Ω(actualLrpsForOtherCell).Should(HaveLen(1))
		})
	})

	Describe("GetRunningActualLRPs", func() {
		BeforeEach(func() {
			err := bbs.ReportActualLRPAsRunning(runningLrp1, "cell-id")
			Ω(err).ShouldNot(HaveOccurred())

			newLrp, err = bbs.ReportActualLRPAsStarting(newLrpProcessGuid, newLrpInstanceGuid, newLrpCellId, newLrpDomain, newLrpIndex)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.ReportActualLRPAsRunning(runningLrp2, "cell-id")
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("returns all actual long running processes", func() {
			all, err := bbs.GetRunningActualLRPs()
			Ω(err).ShouldNot(HaveOccurred())

			Ω(all).Should(HaveLen(2))
			Ω(all).Should(ContainElement(runningLrp1))
			Ω(all).Should(ContainElement(runningLrp2))
		})
	})

	Describe("GetActualLRPsByProcessGuid", func() {
		BeforeEach(func() {
			err := bbs.ReportActualLRPAsRunning(runningLrp1, "cell-id")
			Ω(err).ShouldNot(HaveOccurred())

			newLrp, err = bbs.ReportActualLRPAsStarting(newLrpProcessGuid, newLrpInstanceGuid, newLrpCellId, newLrpDomain, newLrpIndex)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.ReportActualLRPAsRunning(runningLrp2, "cell-id")
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("should fetch all LRPs for the specified guid", func() {
			lrps, err := bbs.GetActualLRPsByProcessGuid("guidA")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(lrps).Should(HaveLen(2))
			Ω(lrps).Should(ContainElement(runningLrp1))
			Ω(lrps).Should(ContainElement(newLrp))
		})
	})

	Describe("GetActualLRPsByProcessGuidAndIndex", func() {
		BeforeEach(func() {
			err := bbs.ReportActualLRPAsRunning(runningLrp1, "cell-id")
			Ω(err).ShouldNot(HaveOccurred())

			newLrp, err = bbs.ReportActualLRPAsStarting(newLrpProcessGuid, newLrpInstanceGuid, newLrpCellId, newLrpDomain, newLrpIndex)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.ReportActualLRPAsRunning(runningLrp2, "cell-id")
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("should fetch all LRPs for the specified guid", func() {
			lrps, err := bbs.GetActualLRPsByProcessGuidAndIndex("guidA", 1)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(lrps).Should(HaveLen(1))
			Ω(lrps).Should(ContainElement(runningLrp1))
		})
	})

	Describe("GetRunningActualLRPsByProcessGuid", func() {
		BeforeEach(func() {
			err := bbs.ReportActualLRPAsRunning(runningLrp1, "cell-id")
			Ω(err).ShouldNot(HaveOccurred())

			newLrp, err = bbs.ReportActualLRPAsStarting(newLrpProcessGuid, newLrpInstanceGuid, newLrpCellId, newLrpDomain, newLrpIndex)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.ReportActualLRPAsRunning(runningLrp2, "cell-id")
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("should fetch all LRPs for the specified guid", func() {
			lrps, err := bbs.GetRunningActualLRPsByProcessGuid("guidA")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(lrps).Should(HaveLen(1))
			Ω(lrps).Should(ContainElement(runningLrp1))
		})
	})

	Describe("GetAllActualLRPsByDomain", func() {
		BeforeEach(func() {
			err := bbs.ReportActualLRPAsRunning(runningLrp1, "cell-id")
			Ω(err).ShouldNot(HaveOccurred())

			newLrp, err = bbs.ReportActualLRPAsStarting(newLrpProcessGuid, newLrpInstanceGuid, newLrpCellId, newLrpDomain, newLrpIndex)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.ReportActualLRPAsRunning(runningLrp2, "cell-id")
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.ReportActualLRPAsRunning(runningLrp3, "cell-id")
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("should fetch all LRPs for the specified guid", func() {
			lrps, err := bbs.GetAllActualLRPsByDomain("domain-b")
			Ω(err).ShouldNot(HaveOccurred())

			Ω(lrps).Should(HaveLen(2))
			Ω(lrps).ShouldNot(ContainElement(runningLrp1))
			Ω(lrps).Should(ContainElement(runningLrp2))
			Ω(lrps).Should(ContainElement(runningLrp3))
		})

		Context("when there are no actual LRPs in the requested domain", func() {
			It("returns an empty list", func() {
				lrps, err := bbs.GetAllActualLRPsByDomain("bogus-domain")
				Ω(err).ShouldNot(HaveOccurred())

				Ω(lrps).ShouldNot(BeNil())
				Ω(lrps).Should(HaveLen(0))
			})
		})
	})
})
