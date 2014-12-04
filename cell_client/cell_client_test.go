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
		const expectedStopInstancePath = "/lrp/some-process-guid/index/2/instance/some-instance-guid"
		var stopErr error
		var stopInstance = models.StopLRPInstance{
			ProcessGuid:  "some-process-guid",
			InstanceGuid: "some-instance-guid",
			Index:        2,
		}

		JustBeforeEach(func() {
			stopErr = cellClient.StopLRPInstance(fakeServer.URL(), stopInstance)
		})

		Context("when the request is successful", func() {
			BeforeEach(func() {
				fakeServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("DELETE", expectedStopInstancePath),
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
						ghttp.VerifyRequest("DELETE", expectedStopInstancePath),
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
						ghttp.VerifyRequest("DELETE", expectedStopInstancePath),
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
