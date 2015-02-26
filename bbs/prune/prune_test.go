package prune_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/prune"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/prune/fakes"
	"github.com/pivotal-golang/lager/lagertest"

	db "github.com/cloudfoundry/storeadapter"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Prune", func() {
	var fake_bbs *fakes.FakeStoreAdapter
	var exampleTree db.StoreNode
	var logger *lagertest.TestLogger

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		exampleTree = db.StoreNode{
			Key: "/0",
			Dir: true,
			ChildNodes: []db.StoreNode{
				{
					Key: "/0/0",
					Dir: true,
					ChildNodes: []db.StoreNode{
						{
							Key:   "/0/0/0",
							Value: []byte("a-node"),
						},
					},
				},
				{
					Key: "/0/1",
					Dir: true,
					ChildNodes: []db.StoreNode{
						{
							Key:   "/0/1/0",
							Value: []byte("b-node"),
						},
					},
				},
			},
		}

		fake_bbs = new(fakes.FakeStoreAdapter)

		fake_bbs.ListRecursivelyReturns(exampleTree, nil)
	})

	It("compares-and-deletes files by index and deletes empty dirs", func() {
		predicate := func(db.StoreNode) bool { return false }
		prune.Prune(logger, fake_bbs, "/0", predicate)

		Ω(fake_bbs.ListRecursivelyCallCount()).Should(Equal(1))
		Ω(fake_bbs.ListRecursivelyArgsForCall(0)).Should(Equal("/0"))

		Ω(fake_bbs.CompareAndDeleteByIndexCallCount()).Should(Equal(1))
		Ω(fake_bbs.CompareAndDeleteByIndexArgsForCall(0)).Should(ConsistOf(exampleTree.ChildNodes[0].ChildNodes[0], exampleTree.ChildNodes[1].ChildNodes[0]))

		Ω(fake_bbs.DeleteLeavesCallCount()).Should(Equal(1))
		Ω(fake_bbs.DeleteLeavesArgsForCall(0)).Should(ConsistOf(exampleTree.ChildNodes[0].Key, exampleTree.ChildNodes[1].Key))
	})

	Context("store node doesn't exist", func() {

		BeforeEach(func() {
			fake_bbs.ListRecursivelyReturns(db.StoreNode{}, db.ErrorKeyNotFound)
		})

		It("logs and returns without calling predicate function if the store node doesn't exist", func() {
			predicate := func(db.StoreNode) bool {
				Ω(false).Should(BeTrue(), "This method shouldn't have been called")
				return false
			}
			prune.Prune(logger, fake_bbs, "/not-present", predicate)
			Ω(logger).Should(gbytes.Say("test.no-key-found"))
		})
	})

})
