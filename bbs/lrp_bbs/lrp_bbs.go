package lrp_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/services_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/cb"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/timeprovider"
	"github.com/cloudfoundry/storeadapter"
)

type LRPBBS struct {
	store             storeadapter.StoreAdapter
	timeProvider      timeprovider.TimeProvider
	cellClient        cb.CellClient
	auctioneerClient  cb.AuctioneerClient
	services          *services_bbs.ServicesBBS
	restartCalculator models.RestartCalculator
}

func New(
	store storeadapter.StoreAdapter,
	timeProvider timeprovider.TimeProvider,
	cellClient cb.CellClient,
	auctioneerClient cb.AuctioneerClient,
	services *services_bbs.ServicesBBS,
	restartCalculator models.RestartCalculator,
) *LRPBBS {
	return &LRPBBS{
		store:             store,
		timeProvider:      timeProvider,
		cellClient:        cellClient,
		auctioneerClient:  auctioneerClient,
		services:          services,
		restartCalculator: restartCalculator,
	}
}
