package flags_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/cc_messages/flags"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Lifecycles", func() {
	Describe("Set", func() {
		var lifecycles flags.LifecycleMap
		BeforeEach(func() {
			lifecycles = flags.LifecycleMap{}
		})
		It("adds the mapping", func() {
			err := lifecycles.Set("foo:bar/baz")
			Ω(err).ShouldNot(HaveOccurred())

			Ω(lifecycles["foo"]).Should(Equal("bar/baz"))
		})

		It("errors when the value is not of the form 'lifecycle:path'", func() {
			err := lifecycles.Set("blork")
			Ω(err).Should(Equal(flags.ErrLifecycleFormatInvalid))
		})

		It("errors when the value has an empty lifecycle", func() {
			err := lifecycles.Set(":mindy")
			Ω(err).Should(Equal(flags.ErrLifecycleNameEmpty))
		})

		It("errors when the value is not of the form 'lifecycle:path'", func() {
			err := lifecycles.Set("blork:")
			Ω(err).Should(Equal(flags.ErrLifecyclePathEmpty))
		})
	})
})
