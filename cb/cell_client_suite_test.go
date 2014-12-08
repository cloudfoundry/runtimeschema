package cb_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCBRadio(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CBRadio Suite")
}
