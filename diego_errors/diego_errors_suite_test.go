package diego_errors_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestDiegoErrors(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DiegoErrors Suite")
}
