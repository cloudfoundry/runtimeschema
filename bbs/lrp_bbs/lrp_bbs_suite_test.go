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
var timeProvider *faketimeprovider.FakeTimeProvider
var fakeCellClient *cbfakes.FakeCellClient

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
	timeProvider = faketimeprovider.New(time.Unix(0, 1138))

	servicesBBS := services_bbs.New(etcdClient, lagertest.NewTestLogger("test"))
	bbs = lrp_bbs.New(etcdClient, timeProvider, fakeCellClient, servicesBBS, lagertest.NewTestLogger("test"))
})

func registerCell(cell models.CellPresence) {
	jsonBytes, err := models.ToJSON(cell)
	Ω(err).ShouldNot(HaveOccurred())

	etcdClient.Create(storeadapter.StoreNode{
		Key:   shared.CellSchemaPath(cell.CellID),
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

func createAndClaim(a models.ActualLRP) (*models.ActualLRP, *models.ActualLRP, error) {
	c := a
	c.State = models.ActualLRPStateUnclaimed
	unclaimed, err := bbs.CreateActualLRP(c)
	Ω(err).ShouldNot(HaveOccurred())
	claimed, err := bbs.ClaimActualLRP(a)
	Ω(err).ShouldNot(HaveOccurred())

	return unclaimed, claimed, err
}
