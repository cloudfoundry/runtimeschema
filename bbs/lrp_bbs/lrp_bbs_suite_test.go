package lrp_bbs_test

import (
	"github.com/cloudfoundry-incubator/consuladapter"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/domain_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/lrp_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/lrp_bbs/internal/test_helpers"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/services_bbs"
	cbfakes "github.com/cloudfoundry-incubator/runtime-schema/cb/fakes"
	"github.com/cloudfoundry/storeadapter"
	"github.com/cloudfoundry/storeadapter/storerunner/etcdstorerunner"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/clock/fakeclock"
	"github.com/pivotal-golang/lager/lagertest"

	"testing"
	"time"
)

var etcdRunner *etcdstorerunner.ETCDClusterRunner
var etcdClient storeadapter.StoreAdapter
var consulSession *consuladapter.Session
var consulRunner *consuladapter.ClusterRunner
var testHelper *test_helpers.TestHelper
var lrpBBS *lrp_bbs.LRPBBS
var domainBBS *domain_bbs.DomainBBS
var clock *AdvancingFakeClock
var fakeCellClient *cbfakes.FakeCellClient
var fakeAuctioneerClient *cbfakes.FakeAuctioneerClient
var servicesBBS *services_bbs.ServicesBBS

var logger *lagertest.TestLogger

func TestLRPBbs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Long Running Process BBS Suite")
}

var _ = BeforeSuite(func() {
	etcdRunner = etcdstorerunner.NewETCDClusterRunner(5001+config.GinkgoConfig.ParallelNode, 1, nil)

	consulRunner = consuladapter.NewClusterRunner(9001+config.GinkgoConfig.ParallelNode*consuladapter.PortOffsetLength, 1, "http")

	etcdRunner.Start()
	consulRunner.Start()
	consulRunner.WaitUntilReady()
})

var _ = AfterSuite(func() {
	etcdRunner.Stop()
	consulRunner.Stop()
})

var _ = AfterEach(func() {
	etcdClient.Disconnect()
})

var _ = BeforeEach(func() {
	etcdRunner.Reset()
	etcdClient = etcdRunner.RetryableAdapter(bbs.ConvergerBBSWorkPoolSize, nil)

	consulRunner.Reset()
	consulSession = consulRunner.NewSession("a-session")

	testHelper = test_helpers.NewTestHelper(etcdClient, consulSession)

	fakeCellClient = &cbfakes.FakeCellClient{}
	fakeAuctioneerClient = &cbfakes.FakeAuctioneerClient{}
	clock = &AdvancingFakeClock{
		FakeClock: fakeclock.NewFakeClock(time.Unix(0, 1138)),
	}

	logger = lagertest.NewTestLogger("test")

	servicesBBS = services_bbs.New(consulSession, clock, lagertest.NewTestLogger("test"))

	lrpBBS = lrp_bbs.New(
		etcdClient,
		clock,
		fakeCellClient,
		fakeAuctioneerClient,
		servicesBBS,
	)

	domainBBS = domain_bbs.New(etcdClient, lagertest.NewTestLogger("test"))
})
