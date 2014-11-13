package services_bbs_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/cloudfoundry-incubator/runtime-schema/bbs/services_bbs"
)

var _ = Context("Getting Generic Services", func() {
	var bbs *ServicesBBS

	BeforeEach(func() {
		bbs = New(etcdClient, lagertest.NewTestLogger("test"))
	})

	Describe("GetServiceRegistrations", func() {
		var registrations models.ServiceRegistrations
		var registrationsErr error

		JustBeforeEach(func() {
			registrations, registrationsErr = bbs.GetServiceRegistrations()
		})

		Context("when etcd returns sucessfully", func() {
			BeforeEach(func() {
				serviceNodes := []storeadapter.StoreNode{
					{
						Key:   "/v1/cell/guid-0",
						Value: []byte("{}"),
					},
					{
						Key:   "/v1/cell/guid-1",
						Value: []byte("{}"),
					},
					{
						Key:   "/v1/file_server/guid-0",
						Value: []byte("http://example.com/file-server"),
					},
				}
				err := etcdClient.SetMulti(serviceNodes)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("does not return an error", func() {
				Ω(registrationsErr).ShouldNot(HaveOccurred())
			})

			It("returns the cell service registrations", func() {
				cellRegistrations := registrations.FilterByName(models.CellServiceName)
				Ω(cellRegistrations).Should(HaveLen(2))
				Ω(cellRegistrations).Should(ContainElement(models.ServiceRegistration{
					Name: models.CellServiceName, Id: "guid-0",
				}))
				Ω(cellRegistrations).Should(ContainElement(models.ServiceRegistration{
					Name: models.CellServiceName, Id: "guid-1",
				}))
			})
		})

		Context("when etcd comes up empty", func() {
			It("does not return an error", func() {
				Ω(registrationsErr).ShouldNot(HaveOccurred())
			})

			It("returns empty registrations", func() {
				Ω(registrations).Should(BeEmpty())
			})
		})
	})
})
