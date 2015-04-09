package bbs_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("BBS", func() {
	It("should compile and be able to construct and return each BBS", func() {
		receptorTaskHandlerURL := "http://example.com"
		bbs.NewBBS(etcdClient, consulSession, receptorTaskHandlerURL, clock, logger)
		bbs.NewRepBBS(etcdClient, consulSession, receptorTaskHandlerURL, clock, logger)
		bbs.NewConvergerBBS(etcdClient, consulSession, receptorTaskHandlerURL, clock, logger)
		bbs.NewNsyncBBS(consulSession, clock, logger)
		bbs.NewAuctioneerBBS(etcdClient, consulSession, receptorTaskHandlerURL, clock, logger)
		bbs.NewMetricsBBS(etcdClient, consulSession, clock, logger)
		bbs.NewRouteEmitterBBS(consulSession, clock, logger)
		bbs.NewTpsBBS(consulSession, clock, logger)
		bbs.NewVeritasBBS(etcdClient, consulSession, clock, logger)
	})
})
