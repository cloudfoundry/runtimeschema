package lrp_bbs_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/lrp_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Desired LRP Getters", func() {
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
