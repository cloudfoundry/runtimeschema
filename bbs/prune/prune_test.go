package prune_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/prune"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/prune/fakes"

	db "github.com/cloudfoundry/storeadapter"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Prune", func() {
	var fake_bbs *fakes.FakeStoreAdapter
	var exampleTree db.StoreNode

	BeforeEach(func() {
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
		prune.Prune(fake_bbs, "/0", predicate)

		Ω(fake_bbs.ListRecursivelyCallCount()).Should(Equal(1))
		Ω(fake_bbs.ListRecursivelyArgsForCall(0)).Should(Equal("/0"))

		Ω(fake_bbs.CompareAndDeleteByIndexCallCount()).Should(Equal(1))
		Ω(fake_bbs.CompareAndDeleteByIndexArgsForCall(0)).Should(ConsistOf(exampleTree.ChildNodes[0].ChildNodes[0], exampleTree.ChildNodes[1].ChildNodes[0]))

		Ω(fake_bbs.DeleteLeavesCallCount()).Should(Equal(1))
		Ω(fake_bbs.DeleteLeavesArgsForCall(0)).Should(ConsistOf(exampleTree.ChildNodes[0].Key, exampleTree.ChildNodes[1].Key))
	})
})
