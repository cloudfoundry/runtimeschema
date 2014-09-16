package lrp_bbs_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LrpFreshness", func() {
	Describe("initially", func() {
		Describe("GetAllFreshness", func() {
			It("is an empty set", func() {
				Ω(bbs.GetAllFreshness()).Should(BeEmpty())
			})
		})
	})

	Context("when the freshness has been bumped", func() {
		BeforeEach(func() {
			err := bbs.BumpFreshness("some-domain", 1*time.Second)
			Ω(err).ShouldNot(HaveOccurred())
		})

		Describe("GetAllFreshness", func() {
			It("includes the fresh domain", func() {
				Ω(bbs.GetAllFreshness()).Should(ConsistOf([]string{"some-domain"}))
			})
		})

		Context("and then expires", func() {
			BeforeEach(func() {
				time.Sleep(2 * time.Second)
			})

			Describe("GetAllFreshness", func() {
				It("becomes empty", func() {
					Ω(bbs.GetAllFreshness()).Should(BeEmpty())
				})
			})
		})
	})
})
