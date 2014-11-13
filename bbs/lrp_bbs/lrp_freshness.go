package lrp_bbs

import (
	"path"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
)

func (bbs *LRPBBS) BumpFreshness(freshness models.Freshness) error {
	err := freshness.Validate()
	if err != nil {
		return err
	}

	return bbs.store.SetMulti([]storeadapter.StoreNode{
		{
			Key: shared.FreshnessSchemaPath(freshness.Domain),
			TTL: uint64(freshness.TTLInSeconds),
		},
	})
}

func (bbs *LRPBBS) Freshnesses() ([]models.Freshness, error) {
	node, err := bbs.store.ListRecursively(shared.FreshnessSchemaRoot)
	if err != nil && err != storeadapter.ErrorKeyNotFound {
		return nil, err
	}

	freshnesses := make([]models.Freshness, 0, len(node.ChildNodes))

	for _, node := range node.ChildNodes {
		freshnesses = append(freshnesses, models.Freshness{
			Domain:       path.Base(node.Key),
			TTLInSeconds: int(node.TTL),
		})
	}

	return freshnesses, nil
}
