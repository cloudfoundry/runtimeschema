package bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
)

//Bulletin Board System/Store

type ExecutorBBS interface {
	WatchForDesiredRunOnce() (<-chan models.RunOnce, chan<- bool, <-chan error) //filter out delete...

	ClaimRunOnce(models.RunOnce) error
	StartRunOnce(models.RunOnce) error
	CompletedRunOnce(models.RunOnce) error

	ConvergeRunOnce() //should be executed periodically
}

type StagerBBS interface {
	WatchForCompletedRunOnce() (<-chan models.RunOnce, chan<- bool, <-chan error) //filter out delete...

	DesireRunOnce(models.RunOnce) error
	ResolveRunOnce(models.RunOnce) error
}

func New(store storeadapter.StoreAdapter) *BBS {
	return &BBS{
		ExecutorBBS: &executorBBS{store: store},
		StagerBBS:   &stagerBBS{store: store},
		store:       store,
	}
}

type BBS struct {
	ExecutorBBS
	StagerBBS
	store storeadapter.StoreAdapter
}
