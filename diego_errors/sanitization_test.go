package diego_errors_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/diego_errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SanitizeErrorMessage", func() {
	var unsanitized, sanitized string

	BeforeEach(func() {
		unsanitized = ""
	})

	JustBeforeEach(func() {
		sanitized = diego_errors.SanitizeErrorMessage(unsanitized)
	})

	Context("when the unsanitized string is INSUFFICIENT_RESOURCES_MESSAGE", func() {
		BeforeEach(func() {
			unsanitized = diego_errors.INSUFFICIENT_RESOURCES_MESSAGE
		})

		It("preserves the reason message", func() {
			Ω(sanitized).Should(Equal(unsanitized))
		})
	})

	Context("when the unsanitized string is MISSING_APP_BITS_DOWNLOAD_URI_MESSAGE", func() {
		BeforeEach(func() {
			unsanitized = diego_errors.MISSING_APP_BITS_DOWNLOAD_URI_MESSAGE
		})

		It("preserves the reason message", func() {
			Ω(sanitized).Should(Equal(unsanitized))
		})
	})

	Context("when the unsanitized string is MISSING_APP_ID_MESSAGE", func() {
		BeforeEach(func() {
			unsanitized = diego_errors.MISSING_APP_ID_MESSAGE
		})

		It("preserves the reason message", func() {
			Ω(sanitized).Should(Equal(unsanitized))
		})
	})

	Context("when the unsanitized string is MISSING_TASK_ID_MESSAGE", func() {
		BeforeEach(func() {
			unsanitized = diego_errors.MISSING_TASK_ID_MESSAGE
		})

		It("preserves the reason message", func() {
			Ω(sanitized).Should(Equal(unsanitized))
		})
	})

	Context("when the unsanitized string is NO_COMPILER_DEFINED_MESSAGE", func() {
		BeforeEach(func() {
			unsanitized = diego_errors.NO_COMPILER_DEFINED_MESSAGE
		})

		It("preserves the reason message", func() {
			Ω(sanitized).Should(Equal(unsanitized))
		})
	})

	Context("when the unsanitized string is something else", func() {
		BeforeEach(func() {
			unsanitized = "something else"
		})

		It("is unapolagetic", func() {
			Ω(sanitized).Should(Equal("staging failed"))
		})
	})
})
