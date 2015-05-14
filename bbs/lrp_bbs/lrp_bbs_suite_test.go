package lrp_bbs_test

import (
	"encoding/json"

	"github.com/cloudfoundry-incubator/consuladapter"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/domain_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/lrp_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/services_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	cbfakes "github.com/cloudfoundry-incubator/runtime-schema/cb/fakes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
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
var consulSession *consuladapter.Session
var consulRunner *consuladapter.ClusterRunner
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
	etcdRunner = etcdstorerunner.NewETCDClusterRunner(5001+config.GinkgoConfig.ParallelNode, 1)
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
	etcdClient = etcdRunner.RetryableAdapter(bbs.ConvergerBBSWorkPoolSize)

	consulRunner.Reset()
	consulSession = consulRunner.NewSession("a-session")

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

func registerCell(cell models.CellPresence) {
	jsonBytes, err := models.ToJSON(cell)
	Expect(err).NotTo(HaveOccurred())

	err = consulSession.AcquireLock(shared.CellSchemaPath(cell.CellID), jsonBytes)
	Expect(err).NotTo(HaveOccurred())
}

func registerAuctioneer(auctioneer models.AuctioneerPresence) {
	jsonBytes, err := models.ToJSON(auctioneer)
	Expect(err).NotTo(HaveOccurred())

	err = consulSession.AcquireLock(shared.LockSchemaPath("auctioneer_lock"), jsonBytes)
	Expect(err).NotTo(HaveOccurred())
}

func claimDesireLRPByIndex(d models.DesiredLRP, index int, instanceKey models.ActualLRPInstanceKey, logger lager.Logger) {
	unclaimedLRPGroup, err := lrpBBS.ActualLRPGroupByProcessGuidAndIndex(d.ProcessGuid, index)
	Expect(err).NotTo(HaveOccurred())

	err = lrpBBS.ClaimActualLRP(logger, unclaimedLRPGroup.Instance.ActualLRPKey, instanceKey)
	Expect(err).NotTo(HaveOccurred())
}

func setRawActualLRP(lrp models.ActualLRP) {
	value, err := json.Marshal(lrp) // do NOT use models.ToJSON; don't want validations
	Expect(err).NotTo(HaveOccurred())

	err = etcdClient.SetMulti([]storeadapter.StoreNode{
		{
			Key:   shared.ActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index),
			Value: value,
		},
	})

	Expect(err).NotTo(HaveOccurred())
}

func setRawEvacuatingActualLRP(lrp models.ActualLRP, ttlInSeconds uint64) {
	value, err := json.Marshal(lrp) // do NOT use models.ToJSON; don't want validations
	Expect(err).NotTo(HaveOccurred())

	err = etcdClient.SetMulti([]storeadapter.StoreNode{
		{
			Key:   shared.EvacuatingActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index),
			Value: value,
			TTL:   ttlInSeconds,
		},
	})

	Expect(err).NotTo(HaveOccurred())
}

func setRawDesiredLRP(d models.DesiredLRP) {
	value, err := json.Marshal(d) // do NOT use models.ToJSON; don't want validations
	Expect(err).NotTo(HaveOccurred())

	err = etcdClient.SetMulti([]storeadapter.StoreNode{
		{
			Key:   shared.DesiredLRPSchemaPath(d),
			Value: value,
		},
	})

	Expect(err).NotTo(HaveOccurred())
}

func deleteActualLRP(key models.ActualLRPKey) {
	err := etcdClient.Delete(shared.ActualLRPSchemaPath(key.ProcessGuid, key.Index))
	Expect(err).NotTo(HaveOccurred())
}

func deleteEvacuatingActualLRP(key models.ActualLRPKey) {
	err := etcdClient.Delete(shared.EvacuatingActualLRPSchemaPath(key.ProcessGuid, key.Index))
	Expect(err).NotTo(HaveOccurred())
}

func createRawDomain(domain string) {
	err := domainBBS.UpsertDomain(domain, 0)
	Expect(err).NotTo(HaveOccurred())
}

func getInstanceActualLRP(lrpKey models.ActualLRPKey) (models.ActualLRP, error) {
	node, err := etcdClient.Get(shared.ActualLRPSchemaPath(lrpKey.ProcessGuid, lrpKey.Index))
	if err == storeadapter.ErrorKeyNotFound {
		return models.ActualLRP{}, bbserrors.ErrStoreResourceNotFound
	}
	Expect(err).NotTo(HaveOccurred())

	var lrp models.ActualLRP
	err = models.FromJSON(node.Value, &lrp)
	Expect(err).NotTo(HaveOccurred())

	return lrp, nil
}

func getEvacuatingActualLRP(lrpKey models.ActualLRPKey) (models.ActualLRP, uint64, error) {
	node, err := etcdClient.Get(shared.EvacuatingActualLRPSchemaPath(lrpKey.ProcessGuid, lrpKey.Index))
	if err == storeadapter.ErrorKeyNotFound {
		return models.ActualLRP{}, 0, bbserrors.ErrStoreResourceNotFound
	}
	Expect(err).NotTo(HaveOccurred())

	var lrp models.ActualLRP
	err = models.FromJSON(node.Value, &lrp)
	Expect(err).NotTo(HaveOccurred())

	return lrp, node.TTL, nil
}

func defaultNetInfo() models.ActualLRPNetInfo {
	return models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{{ContainerPort: 1234, HostPort: 5678}})
}
