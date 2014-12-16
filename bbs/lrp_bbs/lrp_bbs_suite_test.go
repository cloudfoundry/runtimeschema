package lrp_bbs_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/lrp_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/services_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	cbfakes "github.com/cloudfoundry-incubator/runtime-schema/cb/fakes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/timeprovider/faketimeprovider"
	"github.com/cloudfoundry/storeadapter"
	"github.com/cloudfoundry/storeadapter/storerunner/etcdstorerunner"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager/lagertest"

	"testing"
	"time"
)

var etcdRunner *etcdstorerunner.ETCDClusterRunner
var etcdClient storeadapter.StoreAdapter
var bbs *lrp_bbs.LRPBBS
var timeProvider *AdvancingFakeTimeProvider
var fakeCellClient *cbfakes.FakeCellClient
var fakeAuctioneerClient *cbfakes.FakeAuctioneerClient

var logger *lagertest.TestLogger

func TestLRPBbs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Long Running Process BBS Suite")
}

var _ = BeforeSuite(func() {
	etcdRunner = etcdstorerunner.NewETCDClusterRunner(5001+config.GinkgoConfig.ParallelNode, 1)
	etcdClient = etcdRunner.Adapter()
})

var _ = AfterSuite(func() {
	etcdRunner.Stop()
})

var _ = BeforeEach(func() {
	etcdRunner.Stop()
	etcdRunner.Start()

	fakeCellClient = &cbfakes.FakeCellClient{}
	fakeAuctioneerClient = &cbfakes.FakeAuctioneerClient{}
	timeProvider = &AdvancingFakeTimeProvider{
		FakeTimeProvider: faketimeprovider.New(time.Unix(0, 1138)),
	}

	logger = lagertest.NewTestLogger("test")

	servicesBBS := services_bbs.New(etcdClient, lagertest.NewTestLogger("test"))

	bbs = lrp_bbs.New(etcdClient, timeProvider, fakeCellClient, fakeAuctioneerClient, servicesBBS, logger)
})

func registerCell(cell models.CellPresence) {
	jsonBytes, err := models.ToJSON(cell)
	Ω(err).ShouldNot(HaveOccurred())

	etcdClient.Create(storeadapter.StoreNode{
		Key:   shared.CellSchemaPath(cell.CellID),
		Value: jsonBytes,
	})
}

func registerAuctioneer(auctioneer models.AuctioneerPresence) {
	jsonBytes, err := models.ToJSON(auctioneer)
	Ω(err).ShouldNot(HaveOccurred())

	etcdClient.Create(storeadapter.StoreNode{
		Key:   shared.LockSchemaPath("auctioneer_lock"),
		Value: jsonBytes,
	})
}

func itRetriesUntilStoreComesBack(action func() error) {
	It("should keep trying until the store comes back", func() {
		etcdRunner.GoAway()

		runResult := make(chan error)
		go func() {
			err := action()
			runResult <- err
		}()

		time.Sleep(200 * time.Millisecond)

		etcdRunner.ComeBack()

		Eventually(runResult).Should(Receive(BeNil()))
	})
}

func createAndClaim(d models.DesiredLRP, cellID string, index int) (*models.ActualLRP, *models.ActualLRP, error) {
	unclaimed, err := bbs.CreateActualLRP(d, index, lagertest.NewTestLogger("test"))
	Ω(err).ShouldNot(HaveOccurred())
	unclaimed.CellID = cellID
	claimed, err := bbs.ClaimActualLRP(*unclaimed)
	Ω(err).ShouldNot(HaveOccurred())

	return unclaimed, claimed, err
}
