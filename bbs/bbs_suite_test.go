package bbs_test

import (
	"github.com/cloudfoundry-incubator/consuladapter"
	"github.com/cloudfoundry/storeadapter"
	"github.com/cloudfoundry/storeadapter/storerunner/etcdstorerunner"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/clock/fakeclock"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"

	"testing"
	"time"
)

var etcdRunner *etcdstorerunner.ETCDClusterRunner
var etcdClient storeadapter.StoreAdapter
var consulRunner consuladapter.ClusterRunner
var consulAdapter consuladapter.Adapter
var clock *fakeclock.FakeClock
var logger lager.Logger

func TestBBS(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "BBS Suite")
}

var _ = BeforeSuite(func() {
	etcdRunner = etcdstorerunner.NewETCDClusterRunner(5001+config.GinkgoConfig.ParallelNode, 1)
	etcdClient = etcdRunner.Adapter()

	consulRunner = consuladapter.NewClusterRunner(9001+config.GinkgoConfig.ParallelNode*consuladapter.PortOffsetLength, 1, "http")
	consulAdapter = consulRunner.NewAdapter()

	logger = lagertest.NewTestLogger("test")

	etcdRunner.Start()

	consulRunner.Start()
})

var _ = AfterSuite(func() {
	etcdClient.Disconnect()
	etcdRunner.Stop()

	consulRunner.Stop()
})

var _ = BeforeEach(func() {
	etcdRunner.Reset()
	consulRunner.Reset()

	clock = fakeclock.NewFakeClock(time.Unix(0, 1138))
})
