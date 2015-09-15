package bbs

import (
	"time"

	"github.com/cloudfoundry-incubator/consuladapter"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/lrp_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/services_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/cb"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/ifrit"
)

//Bulletin Board System/Store

//go:generate counterfeiter -o fake_bbs/fake_receptor_bbs.go . ReceptorBBS
type ReceptorBBS interface {
	// cells
	Cells() ([]models.CellPresence, error)
}

//go:generate counterfeiter -o fake_bbs/fake_rep_bbs.go . RepBBS
type RepBBS interface {
	//services
	NewCellPresence(cellPresence models.CellPresence, retryInterval time.Duration) ifrit.Runner
}

//go:generate counterfeiter -o fake_bbs/fake_converger_bbs.go . ConvergerBBS
type ConvergerBBS interface {
	//cells
	CellEvents() <-chan services_bbs.CellEvent
}

const ConvergerBBSWorkPoolSize = 50

//go:generate counterfeiter -o fake_bbs/fake_auctioneer_bbs.go . AuctioneerBBS
type AuctioneerBBS interface {
	//services
	Cells() ([]models.CellPresence, error)
}

type VeritasBBS interface {
	//services
	Cells() ([]models.CellPresence, error)
	AuctioneerAddress() (string, error)
}

func NewReceptorBBS(store storeadapter.StoreAdapter, consul *consuladapter.Session, clock clock.Clock, logger lager.Logger) ReceptorBBS {
	return NewBBS(store, consul, clock, logger)
}

func NewRepBBS(store storeadapter.StoreAdapter, consul *consuladapter.Session, clock clock.Clock, logger lager.Logger) RepBBS {
	return NewBBS(store, consul, clock, logger)
}

func NewConvergerBBS(store storeadapter.StoreAdapter, consul *consuladapter.Session, clock clock.Clock, logger lager.Logger) ConvergerBBS {
	return NewBBS(store, consul, clock, logger)
}

func NewAuctioneerBBS(store storeadapter.StoreAdapter, consul *consuladapter.Session, clock clock.Clock, logger lager.Logger) AuctioneerBBS {
	return NewBBS(store, consul, clock, logger)
}

func NewVeritasBBS(store storeadapter.StoreAdapter, consul *consuladapter.Session, clock clock.Clock, logger lager.Logger) VeritasBBS {
	return NewBBS(store, consul, clock, logger)
}

func NewBBS(store storeadapter.StoreAdapter, consul *consuladapter.Session, clock clock.Clock, logger lager.Logger) *BBS {
	services := services_bbs.New(consul, clock, logger.Session("services-bbs"))
	auctioneerClient := cb.NewAuctioneerClient()
	cellClient := cb.NewCellClient()

	return &BBS{
		LRPBBS:      lrp_bbs.New(store, clock, cellClient, auctioneerClient, services),
		ServicesBBS: services,
	}
}

type BBS struct {
	*lrp_bbs.LRPBBS
	*services_bbs.ServicesBBS
}
