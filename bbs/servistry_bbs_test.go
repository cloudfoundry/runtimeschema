package bbs_test

import (
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/timeprovider/faketimeprovider"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry-incubator/runtime-schema/bbs"
)

var _ = Describe("Servistry BBS", func() {
	var bbs *BBS
	var timeProvider *faketimeprovider.FakeTimeProvider
	BeforeEach(func() {
		timeProvider = faketimeprovider.New(time.Unix(1238, 0))

		bbs = New(store, timeProvider)
	})

	Describe("RegisterCC", func() {
		var expectedRegistration = models.CCRegistrationMessage{
			Host: "1.2.3.4",
			Port: 8080,
		}
		var ttl = 120 * time.Second

		BeforeEach(func() {
			err := bbs.RegisterCC(expectedRegistration, ttl)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("creates /cloud_controller/<host:ip>", func() {
			node, err := store.Get("/v1/cloud_controller/1.2.3.4:8080")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(node.Value).Should(Equal([]byte("http://1.2.3.4:8080")))
		})
	})
})
