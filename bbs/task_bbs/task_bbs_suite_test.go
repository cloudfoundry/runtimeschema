package task_bbs_test

import (
	"database/sql"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/repositories"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/services_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/task_bbs"
	cbfakes "github.com/cloudfoundry-incubator/runtime-schema/cb/fakes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	"github.com/cloudfoundry/storeadapter/storerunner/etcdstorerunner"
	"github.com/go-gorp/gorp"
	_ "github.com/go-sql-driver/mysql"
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
var logger *lagertest.TestLogger
var servicesBBS *services_bbs.ServicesBBS
var fakeTaskClient *cbfakes.FakeTaskClient
var fakeAuctioneerClient *cbfakes.FakeAuctioneerClient
var fakeCellClient *cbfakes.FakeCellClient
var clock *fakeclock.FakeClock
var bbs *task_bbs.TaskBBS
var db *sql.DB
var dbmap *gorp.DbMap
var taskRepository repositories.TaskRepository

var dummyAction = &models.RunAction{
	Path: "cat",
	Args: []string{"/tmp/file"},
}

func TestTaskBbs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Task BBS Suite")
}

var _ = BeforeSuite(func() {
	etcdRunner = etcdstorerunner.NewETCDClusterRunner(5001+config.GinkgoConfig.ParallelNode, 1)
	etcdClient = etcdRunner.RetryableAdapter()

	etcdRunner.Start()

	var err error
	db, err = sql.Open("mysql", "root:password@tcp(127.0.0.1:3306)/bbs")
	Ω(err).ShouldNot(HaveOccurred())

	dbmap = &gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{
		Engine:   "InnoDB",
		Encoding: "UTF8",
	}}

	_, err = dbmap.Exec("drop table if exists tasks")
	Ω(err).ShouldNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	etcdClient.Disconnect()
	etcdRunner.Stop()
})

var _ = BeforeEach(func() {
	var err error
	etcdRunner.Reset()

	logger = lagertest.NewTestLogger("test")

	fakeTaskClient = new(cbfakes.FakeTaskClient)
	fakeAuctioneerClient = new(cbfakes.FakeAuctioneerClient)
	fakeCellClient = new(cbfakes.FakeCellClient)
	clock = fakeclock.NewFakeClock(time.Unix(1238, 0))
	servicesBBS = services_bbs.New(etcdClient, clock, logger)

	taskRepository, err = repositories.NewTaskRepository(dbmap)
	Ω(err).ShouldNot(HaveOccurred())

	err = dbmap.TruncateTables()
	Ω(err).ShouldNot(HaveOccurred())

	bbs = task_bbs.New(etcdClient, clock, fakeTaskClient, fakeAuctioneerClient, fakeCellClient, servicesBBS, dbmap, taskRepository)
})

func registerAuctioneer(auctioneer models.AuctioneerPresence) {
	jsonBytes, err := models.ToJSON(auctioneer)
	Ω(err).ShouldNot(HaveOccurred())

	etcdClient.Create(storeadapter.StoreNode{
		Key:   shared.LockSchemaPath("auctioneer_lock"),
		Value: jsonBytes,
	})
}

func registerCell(cell models.CellPresence) {
	jsonBytes, err := models.ToJSON(cell)
	Ω(err).ShouldNot(HaveOccurred())

	etcdClient.Create(storeadapter.StoreNode{
		Key:   shared.CellSchemaPath(cell.CellID),
		Value: jsonBytes,
	})
}
