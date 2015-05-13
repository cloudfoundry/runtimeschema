package prune

import (
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

var token = struct{}{}

func Prune(
	logger lager.Logger,
	store storeadapter.StoreAdapter,
	rootKey string,
	predicate func(storeadapter.StoreNode) bool,
) error {
	rootNode, err := store.ListRecursively(rootKey)
	if err != nil && err != storeadapter.ErrorKeyNotFound {
		return err
	}
	if err == storeadapter.ErrorKeyNotFound {
		logger.Info("no-key-found", lager.Data{"root-key": rootKey})
		return nil
	}

	p := NewPruner(rootNode, predicate)
	nodeSetsToDelete := p.FindNodesToDelete()

	dirKeySetsToDelete := [][]string{}
	leavesToDelete := []storeadapter.StoreNode{}

	for _, nodes := range nodeSetsToDelete {
		dirKeysToDelete := []string{}
		for _, node := range nodes {
			if node.Dir {
				dirKeysToDelete = append(dirKeysToDelete, node.Key)
			} else {
				leavesToDelete = append(leavesToDelete, node)
			}
		}
		dirKeySetsToDelete = append(dirKeySetsToDelete, dirKeysToDelete)
	}

	store.CompareAndDeleteByIndex(leavesToDelete...)

	// note: we don't want to delete the root node, so do not delete the 0 index key set
	for i := len(dirKeySetsToDelete) - 1; i > 0; i-- {
		if len(dirKeySetsToDelete[i]) > 0 {
			store.DeleteLeaves(dirKeySetsToDelete[i]...)
		}
	}

	return nil
}

type NodesByDepth map[int][]storeadapter.StoreNode

type Pruner struct {
	root             storeadapter.StoreNode
	nodeSetsToDelete NodesByDepth
	predicate        func(storeadapter.StoreNode) bool
}

func NewPruner(root storeadapter.StoreNode, predicate func(storeadapter.StoreNode) bool) *Pruner {
	return &Pruner{
		root:             root,
		predicate:        predicate,
		nodeSetsToDelete: NodesByDepth{},
	}
}

func (p *Pruner) FindNodesToDelete() NodesByDepth {
	p.walk(0, p.root)
	return p.nodeSetsToDelete
}

func (p *Pruner) walk(depth int, node storeadapter.StoreNode) bool {
	if _, hasDepth := p.nodeSetsToDelete[depth]; !hasDepth {
		p.nodeSetsToDelete[depth] = []storeadapter.StoreNode{}
	}

	if !node.Dir {
		if p.predicate(node) {
			return true
		} else {
			p.markForDelete(depth, node)
			return false
		}
	}

	willBeEmpty := true
	childDepth := depth + 1
	for _, childNode := range node.ChildNodes {
		if p.walk(childDepth, childNode) {
			willBeEmpty = false
		}
	}

	if willBeEmpty {
		p.markForDelete(depth, node)
		return false
	}

	return true
}

func (p *Pruner) markForDelete(depth int, node storeadapter.StoreNode) {
	p.nodeSetsToDelete[depth] = append(p.nodeSetsToDelete[depth], node)
}
