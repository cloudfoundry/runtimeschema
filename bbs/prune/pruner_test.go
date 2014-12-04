package prune_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/prune"
	db "github.com/cloudfoundry/storeadapter"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pruner", func() {
	var nodeSetsToDelete [][]db.StoreNode
	var expectedNodeSetsToDelete [][]db.StoreNode
	var exampleTree db.StoreNode

	JustBeforeEach(func() {
		pruner := prune.NewPruner(exampleTree, func(node db.StoreNode) bool {
			return string(node.Value) == "true"
		})

		nodeSetsToDelete = pruner.Prune()
	})

	Context("an empty tree", func() {
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
								Key:        "/0/0/0",
								Dir:        true,
								ChildNodes: []db.StoreNode{},
							},
						},
					},
					{
						Key: "/0/1",
						Dir: true,
						ChildNodes: []db.StoreNode{
							{
								Key: "/0/1/0",
								Dir: true,
							},
						},
					},
				},
			}

			expectedNodeSetsToDelete = [][]db.StoreNode{
				{exampleTree},
				exampleTree.ChildNodes,
				{exampleTree.ChildNodes[0].ChildNodes[0], exampleTree.ChildNodes[1].ChildNodes[0]},
			}
		})

		It("deletes the correct keys", func() {
			Ω(nodeSetsToDelete).Should(Equal(expectedNodeSetsToDelete))
		})
	})

	Context("a tree filled with deletables", func() {
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
								Key: "/0/0/0",
								Dir: true,
								ChildNodes: []db.StoreNode{
									{
										Key:   "/0/0/0/0",
										Value: []byte("false"),
									},
								},
							},
						},
					},
					{
						Key: "/0/1",
						Dir: true,
						ChildNodes: []db.StoreNode{
							{
								Key: "/0/1/0",
								Dir: true,
								ChildNodes: []db.StoreNode{
									{
										Key:   "/0/1/0/0",
										Value: []byte("false"),
									},
								},
							},
						},
					},
				},
			}

			expectedNodeSetsToDelete = [][]db.StoreNode{
				{exampleTree},
				exampleTree.ChildNodes,
				{exampleTree.ChildNodes[0].ChildNodes[0], exampleTree.ChildNodes[1].ChildNodes[0]},
				{exampleTree.ChildNodes[0].ChildNodes[0].ChildNodes[0], exampleTree.ChildNodes[1].ChildNodes[0].ChildNodes[0]},
			}
		})

		It("deletes the correct keys", func() {
			Ω(nodeSetsToDelete).Should(Equal(expectedNodeSetsToDelete))
		})
	})

	Context("a tree filled with keepables", func() {
		BeforeEach(func() {
			expectedNodeSetsToDelete = [][]db.StoreNode{
				{},
				{},
				{},
				{},
			}

			exampleTree = db.StoreNode{
				Key: "/0",
				Dir: true,
				ChildNodes: []db.StoreNode{
					{
						Key: "/0/0",
						Dir: true,
						ChildNodes: []db.StoreNode{
							{
								Key: "/0/0/0",
								Dir: true,
								ChildNodes: []db.StoreNode{
									{
										Key:   "/0/1/0/0",
										Value: []byte("true"),
									},
								},
							},
						},
					},
					{
						Key: "/0/1",
						Dir: true,
						ChildNodes: []db.StoreNode{
							{
								Key: "/0/1/0",
								Dir: true,
								ChildNodes: []db.StoreNode{
									{
										Key:   "/0/1/0/0",
										Value: []byte("true"),
									},
								},
							},
						},
					},
				},
			}
		})

		It("deletes the correct keys", func() {
			Ω(nodeSetsToDelete).Should(Equal(expectedNodeSetsToDelete))
		})
	})

	Context("a mixed, partially filled tree", func() {
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
								Key:        "/0/0/0",
								Dir:        true,
								ChildNodes: []db.StoreNode{},
							},
						},
					}, {
						Key: "/0/1",
						Dir: true,
						ChildNodes: []db.StoreNode{
							{
								Key: "/0/1/0",
								Dir: true,
								ChildNodes: []db.StoreNode{
									{
										Key:   "/0/1/0/0",
										Value: []byte("false"),
									},
								},
							},
						},
					},
					{
						Key: "/0/2",
						Dir: true,
						ChildNodes: []db.StoreNode{
							{
								Key: "/0/2/0",
								Dir: true,
								ChildNodes: []db.StoreNode{
									{
										Key:   "/0/2/0/0",
										Value: []byte("true"),
									},
								},
							},
						},
					},
				},
			}

			expectedNodeSetsToDelete = [][]db.StoreNode{
				{},
				{exampleTree.ChildNodes[0], exampleTree.ChildNodes[1]},
				{exampleTree.ChildNodes[0].ChildNodes[0], exampleTree.ChildNodes[1].ChildNodes[0]},
				{exampleTree.ChildNodes[1].ChildNodes[0].ChildNodes[0]},
			}
		})

		It("deletes the correct keys", func() {
			Ω(nodeSetsToDelete).Should(Equal(expectedNodeSetsToDelete))
		})
	})
})
