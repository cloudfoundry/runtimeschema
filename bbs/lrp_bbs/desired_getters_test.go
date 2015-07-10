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

	Describe("DesiredLRPByProcessGuid", func() {
		BeforeEach(func() {
			createdDesiredLRPs = createDesiredLRPsInDomains(map[string]int{
				"domain-1": 1,
			})
		})

		It("returns all desired long running processes", func() {
			desiredLrp, err := lrpBBS.DesiredLRPByProcessGuid(logger, "guid-0-for-domain-1")
			Expect(err).NotTo(HaveOccurred())

			desiredLrp.ModificationTag = models.ModificationTag{}
			Expect(desiredLrp).To(Equal(createdDesiredLRPs["domain-1"][0]))
		})

		Context("when the Desired LRP does not exist", func() {
			It("returns ErrStoreResourceNotFound", func() {
				_, err := lrpBBS.DesiredLRPByProcessGuid(logger, "non-existent")
				Expect(err).To(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		})

		Context("when the process guid is empty", func() {
			It("returns an error", func() {
				_, err := lrpBBS.DesiredLRPByProcessGuid(logger, "")
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
					User: "diego",
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
