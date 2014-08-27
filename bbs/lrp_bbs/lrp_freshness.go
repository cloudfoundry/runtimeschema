package lrp_bbs

import (
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry/storeadapter"
)

func (bbs *LRPBBS) BumpFreshness(domain string, ttl time.Duration) error {
	return bbs.store.SetMulti([]storeadapter.StoreNode{
		{
			Key: shared.FreshnessSchemaPath(domain),
			TTL: uint64(ttl.Seconds()),
		},
	})
}

func (bbs *LRPBBS) CheckFreshness(domain string) error {
	_, err := bbs.store.Get(shared.FreshnessSchemaPath(domain))
	return err
}
