package cell_client_test

import (
	"net/http"

	"github.com/cloudfoundry-incubator/runtime-schema/cell_client"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("CellClient", func() {
	var fakeServer *ghttp.Server
	var cellClient cell_client.Client

	BeforeEach(func() {
		fakeServer = ghttp.NewServer()
		cellClient = cell_client.New()
	})

	AfterEach(func() {
		fakeServer.Close()
	})

	Describe("StopLRPInstance", func() {
		const cellAddr = "cell.example.com"
		var stopErr error
		var actualLRP = models.ActualLRP{
			ProcessGuid:  "some-process-guid",
			InstanceGuid: "some-instance-guid",
			Index:        2,
			CellID:       "some-cell-id",
		}

		JustBeforeEach(func() {
			stopErr = cellClient.StopLRPInstance(fakeServer.URL(), actualLRP)
		})

		Context("when the request is successful", func() {
			BeforeEach(func() {
				fakeServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/lrps/stop"),
						ghttp.VerifyJSONRepresenting(actualLRP),
						ghttp.RespondWith(http.StatusAccepted, ""),
					),
				)
			})

			It("makes the request and does not return an error", func() {
				Ω(stopErr).ShouldNot(HaveOccurred())
				Ω(fakeServer.ReceivedRequests()).Should(HaveLen(1))
			})
		})

		Context("when the request returns 500", func() {
			BeforeEach(func() {
				fakeServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/lrps/stop"),
						ghttp.VerifyJSONRepresenting(actualLRP),
						ghttp.RespondWith(http.StatusInternalServerError, ""),
					),
				)
			})

			It("makes the request and returns an error", func() {
				Ω(stopErr).Should(HaveOccurred())
				Ω(fakeServer.ReceivedRequests()).Should(HaveLen(1))
			})
		})

		Context("when the connection fails", func() {
			BeforeEach(func() {
				fakeServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/lrps/stop"),
						ghttp.VerifyJSONRepresenting(actualLRP),
						func(w http.ResponseWriter, r *http.Request) {
							fakeServer.CloseClientConnections()
						},
					),
				)
			})

			It("makes the request and returns an error", func() {
				Ω(stopErr).Should(HaveOccurred())
				Ω(fakeServer.ReceivedRequests()).Should(HaveLen(1))
			})
		})
	})
})
