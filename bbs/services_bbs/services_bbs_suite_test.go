package services_bbs_test

import (
	"github.com/cloudfoundry-incubator/consuladapter"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"

	"testing"
)

var consulRunner *consuladapter.ClusterRunner
var consulAdapter consuladapter.Adapter

func TestServicesBbs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Services BBS Suite")
}

var _ = BeforeSuite(func() {
	consulRunner = consuladapter.NewClusterRunner(
		5001+config.GinkgoConfig.ParallelNode*consuladapter.PortOffsetLength,
		1,
		"http",
	)

	consulRunner.Start()
})

var _ = AfterSuite(func() {
	consulRunner.Stop()
})

var _ = BeforeEach(func() {
	consulRunner.Reset()
	consulAdapter = consulRunner.NewAdapter()
})
