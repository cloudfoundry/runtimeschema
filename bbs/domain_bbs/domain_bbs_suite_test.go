package domain_bbs_test

import (
	"testing"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/domain_bbs"
	"github.com/cloudfoundry/storeadapter"
	"github.com/cloudfoundry/storeadapter/storerunner/etcdstorerunner"

	"github.com/onsi/ginkgo/config"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestDomainBbs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DomainBbs Suite")
}

var etcdRunner *etcdstorerunner.ETCDClusterRunner
var etcdClient storeadapter.StoreAdapter
var bbs *domain_bbs.DomainBBS
var logger *lagertest.TestLogger

var _ = BeforeSuite(func() {
	etcdRunner = etcdstorerunner.NewETCDClusterRunner(5001+config.GinkgoConfig.ParallelNode, 1)
	etcdClient = etcdRunner.RetryableAdapter(10)

	etcdRunner.Start()
})

var _ = AfterSuite(func() {
	etcdClient.Disconnect()
	etcdRunner.Stop()
})

var _ = BeforeEach(func() {
	etcdRunner.Reset()

	logger = lagertest.NewTestLogger("test")

	bbs = domain_bbs.New(etcdClient, logger)
})
