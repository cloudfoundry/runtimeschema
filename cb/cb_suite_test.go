package cb_test

import (
	"github.com/cloudfoundry-incubator/cf-http"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
	"time"
)

var cfHttpTimeout time.Duration

func TestCBRadio(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CBRadio Suite")
}

var _ = BeforeSuite(func() {
	cfHttpTimeout = 1 * time.Second
	cf_http.Initialize(cfHttpTimeout)
})
