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

func (bbs *LRPBBS) GetAllFreshness() ([]string, error) {
	node, err := bbs.store.ListRecursively(shared.FreshnessSchemaRoot)
	if err != nil && err != storeadapter.ErrorKeyNotFound {
		return nil, err
	}

	var domains []string
	for _, node := range node.ChildNodes {
		domains = append(domains, path.Base(node.Key))
	}

	return domains, nil
}
