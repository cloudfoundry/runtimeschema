package prune

import (
	db "github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

var token = struct{}{}

func Prune(logger lager.Logger, store db.StoreAdapter, rootKey string, predicate func(db.StoreNode) bool) error {
	rootNode, err := store.ListRecursively(rootKey)
	if err != nil && err != db.ErrorKeyNotFound {
		return err
	}
	if err == db.ErrorKeyNotFound {
		logger.Info("no-key-found", lager.Data{"root-key": rootKey})
		return nil
	}

	p := NewPruner(rootNode, predicate)
	nodeSetsToDelete := p.FindNodesToDelete()

	dirKeySetsToDelete := [][]string{}
	leavesToDelete := []db.StoreNode{}

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

type Pruner struct {
	root             db.StoreNode
	nodeSetsToDelete [][]db.StoreNode
	predicate        func(db.StoreNode) bool
}

func NewPruner(root db.StoreNode, predicate func(db.StoreNode) bool) *Pruner {
	return &Pruner{
		root:      root,
		predicate: predicate,
	}
}

func (p *Pruner) FindNodesToDelete() [][]db.StoreNode {
	p.walk(0, p.root)
	return p.nodeSetsToDelete
}

func (p *Pruner) walk(depth int, node db.StoreNode) bool {
	if len(p.nodeSetsToDelete) < depth+1 {
		p.nodeSetsToDelete = append(p.nodeSetsToDelete, []db.StoreNode{})
	}

	if len(node.ChildNodes) == 0 {
		if node.Dir || !p.predicate(node) {
			p.markForDelete(depth, node)
			return false
		} else {
			return true
		}
	}

	empty := true
	childDepth := depth + 1
	for _, childNode := range node.ChildNodes {
		if p.walk(childDepth, childNode) {
			empty = false
		}
	}

	if empty {
		p.markForDelete(depth, node)
		return false
	}

	return true
}

func (p *Pruner) markForDelete(depth int, node db.StoreNode) {
	p.nodeSetsToDelete[depth] = append(p.nodeSetsToDelete[depth], node)
}
