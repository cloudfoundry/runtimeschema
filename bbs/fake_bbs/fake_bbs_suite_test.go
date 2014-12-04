package fake_bbs_test

import (
	_ "github.com/cloudfoundry-incubator/runtime-schema/bbs/fake_bbs"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestFakeBbs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Fake BBS Suite")
}
