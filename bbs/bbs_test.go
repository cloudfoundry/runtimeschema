package bbs_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("BBS", func() {
	It("should compile and be able to construct and return each BBS", func() {
		bbs.NewBBS(etcdClient, clock, logger)
		bbs.NewRepBBS(etcdClient, clock, logger)
		bbs.NewConvergerBBS(etcdClient, clock, logger)
		bbs.NewNsyncBBS(etcdClient, clock, logger)
		bbs.NewAuctioneerBBS(etcdClient, clock, logger)
		bbs.NewMetricsBBS(etcdClient, clock, logger)
		bbs.NewRouteEmitterBBS(etcdClient, clock, logger)
		bbs.NewVeritasBBS(etcdClient, clock, logger)
	})
})
