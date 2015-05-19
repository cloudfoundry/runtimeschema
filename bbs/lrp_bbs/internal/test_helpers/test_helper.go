package test_helpers

import (
	"github.com/cloudfoundry-incubator/consuladapter"
	"github.com/cloudfoundry/storeadapter"
)

func NewTestHelper(etcdClient storeadapter.StoreAdapter, consulSession *consuladapter.Session) *TestHelper {
	return &TestHelper{etcdClient: etcdClient, consulSession: consulSession}
}

type TestHelper struct {
	etcdClient    storeadapter.StoreAdapter
	consulSession *consuladapter.Session
}
