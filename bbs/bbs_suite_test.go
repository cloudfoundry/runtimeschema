package bbs_test

import (
	"github.com/cloudfoundry-incubator/consuladapter"
	"github.com/cloudfoundry/storeadapter"
	"github.com/cloudfoundry/storeadapter/storerunner/etcdstorerunner"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/clock/fakeclock"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"

	"testing"
)

var etcdRunner *etcdstorerunner.ETCDClusterRunner
var etcdClient storeadapter.StoreAdapter
var consulRunner *consuladapter.ClusterRunner
var consulSession *consuladapter.Session
var clock *fakeclock.FakeClock
var logger lager.Logger

func TestBBS(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "BBS Suite")
}

var _ = BeforeSuite(func() {
	logger = lagertest.NewTestLogger("test")
})
