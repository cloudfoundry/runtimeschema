package lrp_bbs_test

import (
	. "github.com/cloudfoundry-incubator/runtime-schema/bbs/lrp_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Reconcile", func() {
	var (
		numDesired int
		actual     models.ActualLRPsByIndex
	)

	Context("When the actual state matches the desired state", func() {
		BeforeEach(func() {
			numDesired = 3

			actual = models.ActualLRPsByIndex{
				0: {ActualLRPKey: models.NewActualLRPKey("a", 0, "some-domain")},
				1: {ActualLRPKey: models.NewActualLRPKey("a", 1, "some-domain")},
				2: {ActualLRPKey: models.NewActualLRPKey("a", 2, "some-domain")},
			}
		})

		It("returns an empty result", func() {
			result := Reconcile(numDesired, actual)
			Ω(result.IndicesToStart).Should(BeEmpty())
			Ω(result.IndicesToStop).Should(BeEmpty())
		})
	})

	Context("When the number of desired instances is less than the actual number of instances", func() {
		BeforeEach(func() {
			numDesired = 1
			actual = models.ActualLRPsByIndex{
				0: {ActualLRPKey: models.NewActualLRPKey("a", 0, "some-domain")},
				1: {ActualLRPKey: models.NewActualLRPKey("a", 1, "some-domain")},
				2: {ActualLRPKey: models.NewActualLRPKey("a", 2, "some-domain")},
			}
		})

		It("instructs the caller to stop the extra guids", func() {
			result := Reconcile(numDesired, actual)
			Ω(result.IndicesToStart).Should(BeEmpty())
			Ω(result.IndicesToStop).Should(ConsistOf([]int{1, 2}))
		})
	})

	Context("When the number of desired instances is greater than the actual number of instances", func() {
		BeforeEach(func() {
			numDesired = 5
			actual = models.ActualLRPsByIndex{
				0: {ActualLRPKey: models.NewActualLRPKey("a", 0, "some-domain")},
				1: {ActualLRPKey: models.NewActualLRPKey("a", 1, "some-domain")},
				2: {ActualLRPKey: models.NewActualLRPKey("a", 2, "some-domain")},
			}
		})

		It("instructs the caller to start the missing indices", func() {
			result := Reconcile(numDesired, actual)
			Ω(result.IndicesToStart).Should(Equal([]int{3, 4}))
			Ω(result.IndicesToStop).Should(BeEmpty())
		})
	})

	Context("When the indices are not contiguous", func() {
		BeforeEach(func() {
			numDesired = 4
			actual = models.ActualLRPsByIndex{
				0: {ActualLRPKey: models.NewActualLRPKey("a", 0, "some-domain")},
				1: {ActualLRPKey: models.NewActualLRPKey("a", 1, "some-domain")},
				2: {ActualLRPKey: models.NewActualLRPKey("a", 2, "some-domain")},
				4: {ActualLRPKey: models.NewActualLRPKey("a", 4, "some-domain")},
			}
		})

		It("instructs the caller to start the missing indices and to stop any extra indices", func() {
			result := Reconcile(numDesired, actual)
			Ω(result.IndicesToStart).Should(Equal([]int{3}))
			Ω(result.IndicesToStop).Should(Equal([]int{4}))
		})
	})

	Describe("Result", func() {
		Context("when empty", func() {
			It("should say so", func() {
				result := Result{}
				Ω(result.Empty()).Should(BeTrue())

				result = Result{
					IndicesToStart: []int{},
					IndicesToStop:  []int{},
				}
				Ω(result.Empty()).Should(BeTrue())

				result = Result{
					IndicesToStart: []int{1},
					IndicesToStop:  []int{},
				}
				Ω(result.Empty()).Should(BeFalse())

				result = Result{
					IndicesToStart: []int{},
					IndicesToStop:  []int{1},
				}
				Ω(result.Empty()).Should(BeFalse())
			})
		})
	})
})
