package lrp_bbs_test

import (
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/models"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LrpFreshness", func() {
	Describe("Freshnesses", func() {
		Describe("initially", func() {
			It("is an empty set", func() {
				Ω(bbs.Freshnesses()).Should(BeEmpty())
			})
		})

		Context("when the freshness has been bumped", func() {
			BeforeEach(func() {
				err := bbs.BumpFreshness(models.Freshness{"some-domain", 1})
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("includes the fresh domain", func() {
				freshnesses, err := bbs.Freshnesses()
				Ω(err).ShouldNot(HaveOccurred())

				Ω(freshnesses).Should(HaveLen(1))

				freshness := freshnesses[0]
				Ω(freshness.Domain).Should(Equal("some-domain"))
				Ω(freshness.TTLInSeconds).Should(Equal(1))
			})

			Context("and then expires", func() {
				BeforeEach(func() {
					time.Sleep(2 * time.Second)
				})

				It("becomes empty", func() {
					Ω(bbs.Freshnesses()).Should(BeEmpty())
				})
			})
		})
	})

	Describe("BumpFreshness", func() {
		Context("when the freshness is invalid", func() {
			It("reports the validation error", func() {
				invalidFreshness := models.Freshness{"", -1}
				validationError := invalidFreshness.Validate()

				err := bbs.BumpFreshness(invalidFreshness)
				Ω(err).Should(HaveOccurred())
				Ω(err).Should(Equal(validationError))
			})
		})
	})
})
