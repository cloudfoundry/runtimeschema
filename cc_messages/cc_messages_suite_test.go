package cc_messages_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestStagingMessages(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CC Messages Suite")
}
