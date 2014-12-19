package services_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

func (bbs *ServicesBBS) AuctioneerAddress() (string, error) {
	node, err := bbs.store.Get(shared.LockSchemaPath("auctioneer_lock"))
	if err != nil {
		return "", bbserrors.ErrServiceUnavailable
	}

	auctioneerPresence := models.AuctioneerPresence{}
	err = models.FromJSON(node.Value, &auctioneerPresence)
	if err != nil {
		return "", err
	}

	return auctioneerPresence.AuctioneerAddress, nil
}
