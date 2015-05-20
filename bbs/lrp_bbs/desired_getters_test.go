package lrp_bbs_test

import (
	"fmt"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Desired LRP Getters", func() {
	var createdDesiredLRPs map[string][]models.DesiredLRP

	Describe("DesiredLRPs", func() {
		Context("with existing desired lrps", func() {
			BeforeEach(func() {
				createdDesiredLRPs = createDesiredLRPsInDomains(map[string]int{
					"domain-1": 3,
				})
			})

			It("returns all desired long running processes", func() {
				all, err := lrpBBS.DesiredLRPs()
				Expect(err).NotTo(HaveOccurred())

				all = clearModificationTags(all)

				Expect(all).To(ConsistOf(createdDesiredLRPs["domain-1"]))
			})
		})

		Context("when the desired root node exists with no desired lrps", func() {
			BeforeEach(func() {
				testHelper.CreateValidDesiredLRP("some-guid")
				err := lrpBBS.RemoveDesiredLRPByProcessGuid(logger, "some-guid")
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns an empty list of desired long running processes", func() {
				all, err := lrpBBS.DesiredLRPs()
				Expect(err).NotTo(HaveOccurred())
				Expect(all).To(BeEmpty())
			})
		})

		Context("with invalid data", func() {
			BeforeEach(func() {
				testHelper.CreateMalformedDesiredLRP("some-guid")
				testHelper.CreateValidDesiredLRP("another-guid")
			})

			It("errors", func() {
				_, err := lrpBBS.DesiredLRPs()
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("DesiredLRPsByDomain", func() {
		Context("with existing data", func() {
			BeforeEach(func() {
				createdDesiredLRPs = createDesiredLRPsInDomains(map[string]int{
					"domain-1": 2,
					"domain-2": 1,
				})
			})

			It("returns an error when the domain is empty", func() {
				_, err := lrpBBS.DesiredLRPsByDomain("")
				Expect(err).To(Equal(bbserrors.ErrNoDomain))
			})

			It("returns all desired long running processes for the given domain", func() {
				byDomain, err := lrpBBS.DesiredLRPsByDomain("domain-1")
				Expect(err).NotTo(HaveOccurred())

				byDomain = clearModificationTags(byDomain)
				Expect(byDomain).To(ConsistOf(createdDesiredLRPs["domain-1"]))

				byDomain, err = lrpBBS.DesiredLRPsByDomain("domain-2")
				byDomain = clearModificationTags(byDomain)

				Expect(err).NotTo(HaveOccurred())
				Expect(byDomain).To(ConsistOf(createdDesiredLRPs["domain-2"]))
			})
		})

		Context("when the desired root node exists with no desired lrps", func() {
			BeforeEach(func() {
				testHelper.CreateValidDesiredLRP("some-guid")
				err := lrpBBS.RemoveDesiredLRPByProcessGuid(logger, "some-guid")
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns an empty list of desired long running processes", func() {
				all, err := lrpBBS.DesiredLRPsByDomain("foobar")
				Expect(err).NotTo(HaveOccurred())
				Expect(all).To(BeEmpty())
			})
		})

		Context("with invalid data", func() {
			BeforeEach(func() {
				testHelper.CreateMalformedDesiredLRP("some-guid")
				testHelper.CreateValidDesiredLRP("another-guid")
			})

			It("errors", func() {
				_, err := lrpBBS.DesiredLRPs()
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("DesiredLRPByProcessGuid", func() {
		BeforeEach(func() {
			createdDesiredLRPs = createDesiredLRPsInDomains(map[string]int{
				"domain-1": 1,
			})
		})

		It("returns all desired long running processes", func() {
			desiredLrp, err := lrpBBS.DesiredLRPByProcessGuid("guid-0-for-domain-1")
			Expect(err).NotTo(HaveOccurred())

			desiredLrp.ModificationTag = models.ModificationTag{}
			Expect(desiredLrp).To(Equal(createdDesiredLRPs["domain-1"][0]))
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

func createDesiredLRPsInDomains(domainCounts map[string]int) map[string][]models.DesiredLRP {
	createdDesiredLRPs := map[string][]models.DesiredLRP{}

	for domain, count := range domainCounts {
		createdDesiredLRPs[domain] = []models.DesiredLRP{}

		for i := 0; i < count; i++ {
			desiredLRP := models.DesiredLRP{
				Domain:      domain,
				ProcessGuid: fmt.Sprintf("guid-%d-for-%s", i, domain),
				RootFS:      "some:rootfs",
				Instances:   1,
				Action: &models.DownloadAction{
					From: "http://example.com",
					To:   "/tmp/internet",
				},
			}
			err := lrpBBS.DesireLRP(logger, desiredLRP)
			Expect(err).NotTo(HaveOccurred())

			createdDesiredLRPs[domain] = append(createdDesiredLRPs[domain], desiredLRP)
		}
	}

	return createdDesiredLRPs
}

func clearModificationTags(lrps []models.DesiredLRP) []models.DesiredLRP {
	result := []models.DesiredLRP{}
	for _, lrp := range lrps {
		lrp.ModificationTag = models.ModificationTag{}
		result = append(result, lrp)
	}
	return result
}
