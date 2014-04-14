package bbs_test

import (
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/timeprovider/faketimeprovider"
	"github.com/cloudfoundry/storeadapter"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry-incubator/runtime-schema/bbs"
)

var _ = Context("Servistry BBS", func() {
	var bbs *BBS
	var timeProvider *faketimeprovider.FakeTimeProvider

	BeforeEach(func() {
		timeProvider = faketimeprovider.New(time.Unix(1238, 0))
		bbs = New(store, timeProvider)
	})

	Describe("GetServiceRegistrations", func() {
		expectedExecutorRegistrations := models.ServiceRegistrations{
			{Name: "executor", Id: "guid-0"},
			{Name: "executor", Id: "guid-1"},
		}

		BeforeEach(func() {
			serviceNodes := []storeadapter.StoreNode{
				{
					Key: "/v1/executor/guid-0",
				},
				{
					Key: "/v1/executor/guid-1",
				},
			}
			err := store.SetMulti(serviceNodes)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("returns the executor service registrations", func() {
			regisirations, err := bbs.GetServiceRegistrations()
			Ω(err).ShouldNot(HaveOccurred())

			executorRegistrations := regisirations.ExecutorRegistrations()
			Ω(executorRegistrations).Should(Equal(expectedExecutorRegistrations))
		})
	})
})
