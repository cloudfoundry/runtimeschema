package lrp_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/services_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/cb"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/clock"
)

type LRPBBS struct {
	store            storeadapter.StoreAdapter
	clock            clock.Clock
	cellClient       cb.CellClient
	auctioneerClient cb.AuctioneerClient
	services         *services_bbs.ServicesBBS
}

func New(
	store storeadapter.StoreAdapter,
	clock clock.Clock,
	cellClient cb.CellClient,
	auctioneerClient cb.AuctioneerClient,
	services *services_bbs.ServicesBBS,
) *LRPBBS {
	return &LRPBBS{
		store:            store,
		clock:            clock,
		cellClient:       cellClient,
		auctioneerClient: auctioneerClient,
		services:         services,
	}
}
