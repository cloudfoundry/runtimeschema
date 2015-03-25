package heartbeater_test

import (
	"fmt"
	"log"
	"net/http/httputil"
	"net/url"

	"github.com/cloudfoundry-incubator/consuladapter"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/http_server"

	"testing"
)

var (
	proxyRunner  ifrit.Runner
	proxyAddress string

	consulStartingPort int
	consulRunner       consuladapter.ClusterRunner
)

const (
	defaultScheme     = "http"
	defaultDatacenter = "dc"
)

func TestHeartbeater(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Heartbeater Suite")
}

var _ = BeforeSuite(func() {
	consulStartingPort = 5001 + config.GinkgoConfig.ParallelNode*consuladapter.PortOffsetLength
	consulRunner = consuladapter.NewClusterRunner(consulStartingPort, 1, defaultScheme)

	proxyAddress = fmt.Sprintf("127.0.0.1:%d", 6001+config.GinkgoConfig.ParallelNode)

	consulRunner.Start()
})

var _ = BeforeEach(func() {
	proxyRunner = newConsulProxy(proxyAddress, consulStartingPort+consuladapter.PortOffsetHTTP)
	consulRunner.Reset()
})

var _ = AfterSuite(func() {
	consulRunner.Stop()
})

func newConsulProxy(proxyAddress string, consulHTTPPort int) ifrit.Runner {
	consulURL := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("127.0.0.1:%d", consulHTTPPort),
	}

	proxyHandler := httputil.NewSingleHostReverseProxy(consulURL)
	proxyHandler.ErrorLog = log.New(GinkgoWriter, "consul-proxy", log.LstdFlags)

	return http_server.New(proxyAddress, proxyHandler)
}
