package prune_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/prune"
	"github.com/cloudfoundry/storeadapter"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pruner", func() {
	var nodeSetsToDelete prune.NodesByDepth
	var expectedNodeSetsToDelete prune.NodesByDepth
	var exampleTree storeadapter.StoreNode

	JustBeforeEach(func() {
		pruner := prune.NewPruner(exampleTree, func(node storeadapter.StoreNode) bool {
			return string(node.Value) == "true"
		})

		nodeSetsToDelete = pruner.FindNodesToDelete()
	})

	Context("an empty tree", func() {
		BeforeEach(func() {
			exampleTree = storeadapter.StoreNode{
				Key: "/0",
				Dir: true,
				ChildNodes: []storeadapter.StoreNode{
					{
						Key: "/0/0",
						Dir: true,
						ChildNodes: []storeadapter.StoreNode{
							{
								Key:        "/0/0/0",
								Dir:        true,
								ChildNodes: []storeadapter.StoreNode{},
							},
						},
					},
					{
						Key: "/0/1",
						Dir: true,
						ChildNodes: []storeadapter.StoreNode{
							{
								Key: "/0/1/0",
								Dir: true,
							},
						},
					},
				},
			}

			expectedNodeSetsToDelete = prune.NodesByDepth{
				0: {exampleTree},
				1: exampleTree.ChildNodes,
				2: {exampleTree.ChildNodes[0].ChildNodes[0], exampleTree.ChildNodes[1].ChildNodes[0]},
			}
		})

		It("deletes the correct keys", func() {
			Expect(nodeSetsToDelete).To(Equal(expectedNodeSetsToDelete))
		})
	})

	Context("a tree filled with deletables", func() {
		BeforeEach(func() {
			exampleTree = storeadapter.StoreNode{
				Key: "/0",
				Dir: true,
				ChildNodes: []storeadapter.StoreNode{
					{
						Key: "/0/0",
						Dir: true,
						ChildNodes: []storeadapter.StoreNode{
							{
								Key: "/0/0/0",
								Dir: true,
								ChildNodes: []storeadapter.StoreNode{
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
						ChildNodes: []storeadapter.StoreNode{
							{
								Key: "/0/1/0",
								Dir: true,
								ChildNodes: []storeadapter.StoreNode{
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

			expectedNodeSetsToDelete = prune.NodesByDepth{
				0: {exampleTree},
				1: exampleTree.ChildNodes,
				2: {exampleTree.ChildNodes[0].ChildNodes[0], exampleTree.ChildNodes[1].ChildNodes[0]},
				3: {exampleTree.ChildNodes[0].ChildNodes[0].ChildNodes[0], exampleTree.ChildNodes[1].ChildNodes[0].ChildNodes[0]},
			}
		})

		It("deletes the correct keys", func() {
			Expect(nodeSetsToDelete).To(Equal(expectedNodeSetsToDelete))
		})
	})

	Context("a tree filled with keepables", func() {
		BeforeEach(func() {
			expectedNodeSetsToDelete = prune.NodesByDepth{
				0: {},
				1: {},
				2: {},
				3: {},
			}

			exampleTree = storeadapter.StoreNode{
				Key: "/0",
				Dir: true,
				ChildNodes: []storeadapter.StoreNode{
					{
						Key: "/0/0",
						Dir: true,
						ChildNodes: []storeadapter.StoreNode{
							{
								Key: "/0/0/0",
								Dir: true,
								ChildNodes: []storeadapter.StoreNode{
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
						ChildNodes: []storeadapter.StoreNode{
							{
								Key: "/0/1/0",
								Dir: true,
								ChildNodes: []storeadapter.StoreNode{
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
			Expect(nodeSetsToDelete).To(Equal(expectedNodeSetsToDelete))
		})
	})

	Context("a mixed, partially filled tree", func() {
		BeforeEach(func() {
			exampleTree = storeadapter.StoreNode{
				Key: "/0",
				Dir: true,
				ChildNodes: []storeadapter.StoreNode{
					{
						Key: "/0/0",
						Dir: true,
						ChildNodes: []storeadapter.StoreNode{
							{
								Key:        "/0/0/0",
								Dir:        true,
								ChildNodes: []storeadapter.StoreNode{},
							},
						},
					}, {
						Key: "/0/1",
						Dir: true,
						ChildNodes: []storeadapter.StoreNode{
							{
								Key: "/0/1/0",
								Dir: true,
								ChildNodes: []storeadapter.StoreNode{
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
						ChildNodes: []storeadapter.StoreNode{
							{
								Key: "/0/2/0",
								Dir: true,
								ChildNodes: []storeadapter.StoreNode{
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

			expectedNodeSetsToDelete = prune.NodesByDepth{
				0: {},
				1: {exampleTree.ChildNodes[0], exampleTree.ChildNodes[1]},
				2: {exampleTree.ChildNodes[0].ChildNodes[0], exampleTree.ChildNodes[1].ChildNodes[0]},
				3: {exampleTree.ChildNodes[1].ChildNodes[0].ChildNodes[0]},
			}
		})

		It("deletes the correct keys", func() {
			Expect(nodeSetsToDelete).To(Equal(expectedNodeSetsToDelete))
		})
	})
})
