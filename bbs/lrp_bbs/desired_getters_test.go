package lrp_bbs_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Desired LRP Getters", func() {
	var (
		desiredLrp1 models.DesiredLRP
		desiredLrp2 models.DesiredLRP
		desiredLrp3 models.DesiredLRP
		desiredLRPs map[string]*models.DesiredLRP
	)

	BeforeEach(func() {
		desiredLrp1 = models.DesiredLRP{
			Domain:      "tests",
			ProcessGuid: "guidA",
			RootFS:      "some:rootfs",
			Instances:   1,
			Action: &models.DownloadAction{
				From: "http://example.com",
				To:   "/tmp/internet",
			},
		}

		desiredLrp2 = models.DesiredLRP{
			Domain:      "tests",
			ProcessGuid: "guidB",
			RootFS:      "some:rootfs",
			Instances:   1,
			Action: &models.DownloadAction{
				From: "http://example.com",
				To:   "/tmp/internet",
			},
		}

		desiredLrp3 = models.DesiredLRP{
			Domain:      "tests",
			ProcessGuid: "guidC",
			RootFS:      "some:rootfs",
			Instances:   1,
			Action: &models.DownloadAction{
				From: "http://example.com",
				To:   "/tmp/internet",
			},
		}

		desiredLRPs = map[string]*models.DesiredLRP{
			"guidA": &desiredLrp1,
			"guidB": &desiredLrp2,
			"guidC": &desiredLrp3,
		}
	})

	JustBeforeEach(func() {
		err := lrpBBS.DesireLRP(logger, desiredLrp1)
		Expect(err).NotTo(HaveOccurred())

		err = lrpBBS.DesireLRP(logger, desiredLrp2)
		Expect(err).NotTo(HaveOccurred())

		err = lrpBBS.DesireLRP(logger, desiredLrp3)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("DesiredLRPs", func() {
		It("returns all desired long running processes", func() {
			all, err := lrpBBS.DesiredLRPs()
			Expect(err).NotTo(HaveOccurred())

			all = clearModificationTags(all)

			Expect(all).To(HaveLen(3))
			Expect(all).To(ContainElement(desiredLrp1))
			Expect(all).To(ContainElement(desiredLrp2))
			Expect(all).To(ContainElement(desiredLrp3))
		})
	})

	Describe("DesiredLRPsByDomain", func() {
		BeforeEach(func() {
			desiredLrp1.Domain = "domain-1"
			desiredLrp2.Domain = "domain-1"
			desiredLrp3.Domain = "domain-2"
		})

		It("returns all desired long running processes for the given domain", func() {
			byDomain, err := lrpBBS.DesiredLRPsByDomain("domain-1")
			Expect(err).NotTo(HaveOccurred())

			byDomain = clearModificationTags(byDomain)
			Expect(byDomain).To(ConsistOf([]models.DesiredLRP{desiredLrp1, desiredLrp2}))

			byDomain, err = lrpBBS.DesiredLRPsByDomain("domain-2")
			byDomain = clearModificationTags(byDomain)

			Expect(err).NotTo(HaveOccurred())
			Expect(byDomain).To(ConsistOf([]models.DesiredLRP{desiredLrp3}))
		})

		It("returns an error when the domain is empty", func() {
			_, err := lrpBBS.DesiredLRPsByDomain("")
			Expect(err).To(Equal(bbserrors.ErrNoDomain))
		})
	})

	Describe("DesiredLRPByProcessGuid", func() {
		It("returns all desired long running processes", func() {
			desiredLrp, err := lrpBBS.DesiredLRPByProcessGuid("guidA")
			Expect(err).NotTo(HaveOccurred())

			desiredLrp.ModificationTag = models.ModificationTag{}
			Expect(desiredLrp).To(Equal(desiredLrp1))
		})

		Context("when the Desired LRP does not exist", func() {
			It("returns ErrStoreResourceNotFound", func() {
				_, err := lrpBBS.DesiredLRPByProcessGuid("non-existent")
				Expect(err).To(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		})

		Context("when the process guid is empty", func() {
			It("returns an error", func() {
				_, err := lrpBBS.DesiredLRPByProcessGuid("")
				Expect(err).To(Equal(bbserrors.ErrNoProcessGuid))
			})
		})
	})
})

func clearModificationTags(lrps []models.DesiredLRP) []models.DesiredLRP {
	result := []models.DesiredLRP{}
	for _, lrp := range lrps {
		lrp.ModificationTag = models.ModificationTag{}
		result = append(result, lrp)
	}
	return result
}
