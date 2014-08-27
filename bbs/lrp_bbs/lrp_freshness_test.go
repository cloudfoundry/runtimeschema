package lrp_bbs_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = FDescribe("LrpFreshness", func() {
	Describe("CheckFreshness", func() {
		Describe("initially", func() {
			It("returns false", func() {
				err := bbs.CheckFreshness("some-domain")
				Ω(err).Should(HaveOccurred())
			})
		})

		Context("when the freshness has been bumped", func() {
			BeforeEach(func() {
				err := bbs.BumpFreshness("some-domain", 1*time.Second)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("returns true", func() {
				err := bbs.CheckFreshness("some-domain")
				Ω(err).ShouldNot(HaveOccurred())
			})

			Context("and then expires", func() {
				BeforeEach(func() {
					time.Sleep(2 * time.Second)
				})

				It("eventually becomes false", func() {
					err := bbs.CheckFreshness("some-domain")
					Ω(err).Should(HaveOccurred())
				})
			})
		})
	})
})
