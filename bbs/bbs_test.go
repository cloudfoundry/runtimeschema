package bbs_test

import (
	. "github.com/cloudfoundry-incubator/runtime-schema/bbs"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("BBS", func() {
	It("should compile and be able to construct and return each BBS", func() {
		NewBBS(etcdClient, clock, logger)
		NewRepBBS(etcdClient, clock, logger)
		NewConvergerBBS(etcdClient, clock, logger)
		NewNsyncBBS(etcdClient, clock, logger)
		NewAuctioneerBBS(etcdClient, clock, logger)
		NewMetricsBBS(etcdClient, clock, logger)
		NewRouteEmitterBBS(etcdClient, clock, logger)
		NewVeritasBBS(etcdClient, clock, logger)
	})
})
