package bbs_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("BBS", func() {
	It("should compile and be able to construct and return each BBS", func() {
		bbs.NewBBS(etcdClient, consulAdapter, clock, logger)
		bbs.NewRepBBS(etcdClient, consulAdapter, clock, logger)
		bbs.NewConvergerBBS(etcdClient, consulAdapter, clock, logger)
		bbs.NewNsyncBBS(consulAdapter, clock, logger)
		bbs.NewAuctioneerBBS(etcdClient, consulAdapter, clock, logger)
		bbs.NewMetricsBBS(etcdClient, consulAdapter, clock, logger)
		bbs.NewRouteEmitterBBS(consulAdapter, clock, logger)
		bbs.NewVeritasBBS(etcdClient, clock, logger)
	})
})
