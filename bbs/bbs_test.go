package bbs_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("BBS", func() {
	It("should compile and be able to construct and return each BBS", func() {
		bbs.NewBBS(etcdClient, consulSession, clock, logger)
		bbs.NewRepBBS(etcdClient, consulSession, clock, logger)
		bbs.NewConvergerBBS(etcdClient, consulSession, clock, logger)
		bbs.NewNsyncBBS(consulSession, clock, logger)
		bbs.NewAuctioneerBBS(etcdClient, consulSession, clock, logger)
		bbs.NewMetricsBBS(consulSession, clock, logger)
		bbs.NewRouteEmitterBBS(consulSession, clock, logger)
		bbs.NewTpsBBS(consulSession, clock, logger)
		bbs.NewVeritasBBS(etcdClient, consulSession, clock, logger)
		bbs.NewReceptorBBS(etcdClient, consulSession, clock, logger)
	})
})
