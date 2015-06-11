package task_bbs_test

import (
	"github.com/cloudfoundry-incubator/consuladapter"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/services_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/task_bbs"
	cbfakes "github.com/cloudfoundry-incubator/runtime-schema/cb/fakes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
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

const receptorURL = "http://some-receptor-url"

var etcdRunner *etcdstorerunner.ETCDClusterRunner
var etcdClient storeadapter.StoreAdapter
var consulRunner *consuladapter.ClusterRunner
var consulSession *consuladapter.Session
var logger *lagertest.TestLogger
var servicesBBS *services_bbs.ServicesBBS
var fakeTaskClient *cbfakes.FakeTaskClient
var fakeAuctioneerClient *cbfakes.FakeAuctioneerClient
var fakeCellClient *cbfakes.FakeCellClient
var clock *fakeclock.FakeClock
var bbs *task_bbs.TaskBBS

var dummyAction = &models.RunAction{
	Path: "cat",
	Args: []string{"/tmp/file"},
}

func TestTaskBbs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Task BBS Suite")
}

var _ = BeforeSuite(func() {
	etcdRunner = etcdstorerunner.NewETCDClusterRunner(5001+config.GinkgoConfig.ParallelNode, 1, nil)
	etcdClient = etcdRunner.RetryableAdapter(10, nil)
	etcdRunner.Start()

	consulRunner = consuladapter.NewClusterRunner(
		9001+config.GinkgoConfig.ParallelNode*consuladapter.PortOffsetLength,
		1,
		"http",
	)

	consulRunner.Start()
	consulRunner.WaitUntilReady()
})

var _ = AfterSuite(func() {
	etcdClient.Disconnect()
	etcdRunner.Stop()

	consulRunner.Stop()
})

var _ = BeforeEach(func() {
	etcdRunner.Reset()

	consulRunner.Reset()
	consulSession = consulRunner.NewSession("a-session")

	logger = lagertest.NewTestLogger("test")

	fakeTaskClient = new(cbfakes.FakeTaskClient)
	fakeAuctioneerClient = new(cbfakes.FakeAuctioneerClient)
	fakeCellClient = new(cbfakes.FakeCellClient)
	clock = fakeclock.NewFakeClock(time.Unix(1238, 0))
	servicesBBS = services_bbs.New(consulSession, clock, logger)
	bbs = task_bbs.New(etcdClient, consulSession, clock, fakeTaskClient, fakeAuctioneerClient, fakeCellClient,
		servicesBBS, receptorURL)
})

func registerAuctioneer(auctioneer models.AuctioneerPresence) {
	jsonBytes, err := models.ToJSON(auctioneer)
	Expect(err).NotTo(HaveOccurred())

	err = consulSession.AcquireLock(shared.LockSchemaPath("auctioneer_lock"), jsonBytes)
	Expect(err).NotTo(HaveOccurred())
}

func registerCell(cell models.CellPresence) {
	jsonBytes, err := models.ToJSON(cell)
	Expect(err).NotTo(HaveOccurred())

	_, err = consulSession.SetPresence(shared.CellSchemaPath(cell.CellID), jsonBytes)
	Expect(err).NotTo(HaveOccurred())
}
