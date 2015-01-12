package lrp_bbs_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/domain_bbs"
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
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"

	"testing"
	"time"
)

var etcdRunner *etcdstorerunner.ETCDClusterRunner
var etcdClient storeadapter.StoreAdapter
var bbs *lrp_bbs.LRPBBS
var domainBBS *domain_bbs.DomainBBS
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

	servicesBBS := services_bbs.New(etcdClient, timeProvider, lagertest.NewTestLogger("test"))

	bbs = lrp_bbs.New(etcdClient, timeProvider, fakeCellClient, fakeAuctioneerClient, servicesBBS)

	domainBBS = domain_bbs.New(etcdClient, lagertest.NewTestLogger("test"))
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

func createAndClaim(d models.DesiredLRP, index int, containerKey models.ActualLRPContainerKey, logger lager.Logger) {
	err := bbs.CreateActualLRP(d, index, logger)
	Ω(err).ShouldNot(HaveOccurred())

	unclaimed, err := bbs.ActualLRPByProcessGuidAndIndex(d.ProcessGuid, index)
	Ω(err).ShouldNot(HaveOccurred())

	err = bbs.ClaimActualLRP(unclaimed.ActualLRPKey, containerKey, logger)
	Ω(err).ShouldNot(HaveOccurred())
}

func createRawActualLRP(lrp models.ActualLRP) {
	value, err := models.ToJSON(lrp)
	Ω(err).ShouldNot(HaveOccurred())

	err = shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return etcdClient.Create(storeadapter.StoreNode{
			Key:   shared.ActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index),
			Value: value,
		})
	})

	Ω(err).ShouldNot(HaveOccurred())
}

func createRawDesiredLRP(d models.DesiredLRP) {
	value, err := models.ToJSON(d)
	Ω(err).ShouldNot(HaveOccurred())

	err = shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return etcdClient.Create(storeadapter.StoreNode{
			Key:   shared.DesiredLRPSchemaPath(d),
			Value: value,
		})
	})

	Ω(err).ShouldNot(HaveOccurred())
}

func createRawDomain(domain string) {
	err := shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return etcdClient.Create(storeadapter.StoreNode{
			Key:   shared.DomainSchemaPath(domain),
			Value: []byte(domain),
		})
	})

	Ω(err).ShouldNot(HaveOccurred())
}

func getActualLRP(lrpKey models.ActualLRPKey) models.ActualLRP {
	actualLRP, err := bbs.ActualLRPByProcessGuidAndIndex(lrpKey.ProcessGuid, lrpKey.Index)
	Ω(err).ShouldNot(HaveOccurred())
	return actualLRP
}

func defaultNetInfo() models.ActualLRPNetInfo {
	return models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{{ContainerPort: 1234, HostPort: 5678}})
}
