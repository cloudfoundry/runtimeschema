package lrp_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
)

type LongRunningProcessBBS struct {
	store storeadapter.StoreAdapter
}

func New(store storeadapter.StoreAdapter) *LongRunningProcessBBS {
	return &LongRunningProcessBBS{
		store: store,
	}
}

func (bbs *LongRunningProcessBBS) DesireLRP(lrp models.DesiredLRP) error {
	return shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return bbs.store.SetMulti([]storeadapter.StoreNode{
			{
				Key:   shared.DesiredLRPSchemaPath(lrp),
				Value: lrp.ToJSON(),
			},
		})
	})
}

func (bbs *LongRunningProcessBBS) RemoveActualLRP(lrp models.LRP) error {
	return shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return bbs.store.Delete(shared.ActualLRPSchemaPath(lrp))
	})
}

func (bbs *LongRunningProcessBBS) ReportActualLRPAsStarting(lrp models.LRP) error {
	lrp.State = models.LRPStateStarting
	return shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return bbs.store.SetMulti([]storeadapter.StoreNode{
			{
				Key:   shared.ActualLRPSchemaPath(lrp),
				Value: lrp.ToJSON(),
			},
		})
	})
}

func (bbs *LongRunningProcessBBS) ReportActualLRPAsRunning(lrp models.LRP) error {
	lrp.State = models.LRPStateRunning
	return shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return bbs.store.SetMulti([]storeadapter.StoreNode{
			{
				Key:   shared.ActualLRPSchemaPath(lrp),
				Value: lrp.ToJSON(),
			},
		})
	})
}
