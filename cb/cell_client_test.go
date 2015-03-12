package cb_test

import (
	"net/http"
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/cb"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("CellClient", func() {
	var fakeServer *ghttp.Server
	var cellClient cb.CellClient

	BeforeEach(func() {
		fakeServer = ghttp.NewServer()
		cellClient = cb.NewCellClient()
	})

	AfterEach(func() {
		fakeServer.Close()
	})

	Describe("StopLRPInstance", func() {
		const cellAddr = "cell.example.com"
		var stopErr error
		var actualLRP = models.ActualLRP{
			ActualLRPKey:         models.NewActualLRPKey("some-process-guid", 2, "test-domain"),
			ActualLRPInstanceKey: models.NewActualLRPInstanceKey("some-instance-guid", "some-cell-id"),
		}

		JustBeforeEach(func() {
			stopErr = cellClient.StopLRPInstance(fakeServer.URL(), actualLRP.ActualLRPKey, actualLRP.ActualLRPInstanceKey)
		})

		Context("when the request is successful", func() {
			BeforeEach(func() {
				fakeServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/lrps/some-process-guid/instances/some-instance-guid/stop"),
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
						ghttp.VerifyRequest("POST", "/lrps/some-process-guid/instances/some-instance-guid/stop"),
						ghttp.RespondWith(http.StatusInternalServerError, ""),
					),
				)
			})

			It("makes the request and returns an error", func() {
				Ω(stopErr).Should(HaveOccurred())
				Ω(stopErr.Error()).Should(ContainSubstring("http error: status code 500"))
				Ω(fakeServer.ReceivedRequests()).Should(HaveLen(1))
			})
		})

		Context("when the connection fails", func() {
			BeforeEach(func() {
				fakeServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/lrps/some-process-guid/instances/some-instance-guid/stop"),
						func(w http.ResponseWriter, r *http.Request) {
							fakeServer.CloseClientConnections()
						},
					),
				)
			})

			It("makes the request and returns an error", func() {
				Ω(stopErr).Should(HaveOccurred())
				Ω(stopErr.Error()).Should(ContainSubstring("EOF"))
				Ω(fakeServer.ReceivedRequests()).Should(HaveLen(1))
			})
		})

		Context("when the connection times out", func() {
			BeforeEach(func() {
				fakeServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/lrps/some-process-guid/instances/some-instance-guid/stop"),
						func(w http.ResponseWriter, r *http.Request) {
							time.Sleep(cfHttpTimeout + 100*time.Millisecond)
						},
					),
				)
			})

			It("makes the request and returns an error", func() {
				Ω(stopErr).Should(HaveOccurred())
				Ω(stopErr.Error()).Should(ContainSubstring("use of closed network connection"))
				Ω(fakeServer.ReceivedRequests()).Should(HaveLen(1))
			})
		})
	})

	Describe("CancelTask", func() {
		const cellAddr = "cell.example.com"
		var cancelErr error
		var taskGuid = "some-task-guid"

		JustBeforeEach(func() {
			cancelErr = cellClient.CancelTask(fakeServer.URL(), taskGuid)
		})

		Context("when the request is successful", func() {
			BeforeEach(func() {
				fakeServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/tasks/some-task-guid/cancel"),
						ghttp.RespondWith(http.StatusAccepted, ""),
					),
				)
			})

			It("makes the request and does not return an error", func() {
				Ω(cancelErr).ShouldNot(HaveOccurred())
				Ω(fakeServer.ReceivedRequests()).Should(HaveLen(1))
			})
		})

		Context("when the request returns 500", func() {
			BeforeEach(func() {
				fakeServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/tasks/some-task-guid/cancel"),
						ghttp.RespondWith(http.StatusInternalServerError, ""),
					),
				)
			})

			It("makes the request and returns an error", func() {
				Ω(cancelErr).Should(HaveOccurred())
				Ω(cancelErr.Error()).Should(ContainSubstring("http error: status code 500"))
				Ω(fakeServer.ReceivedRequests()).Should(HaveLen(1))
			})
		})

		Context("when the connection fails", func() {
			BeforeEach(func() {
				fakeServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/tasks/some-task-guid/cancel"),
						func(w http.ResponseWriter, r *http.Request) {
							fakeServer.CloseClientConnections()
						},
					),
				)
			})

			It("makes the request and returns an error", func() {
				Ω(cancelErr).Should(HaveOccurred())
				Ω(cancelErr.Error()).Should(ContainSubstring("EOF"))
				Ω(fakeServer.ReceivedRequests()).Should(HaveLen(1))
			})
		})

		Context("when the connection times out", func() {
			BeforeEach(func() {
				fakeServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/tasks/some-task-guid/cancel"),
						func(w http.ResponseWriter, r *http.Request) {
							time.Sleep(cfHttpTimeout + 100*time.Millisecond)
						},
					),
				)
			})

			It("makes the request and returns an error", func() {
				Ω(cancelErr).Should(HaveOccurred())
				Ω(cancelErr.Error()).Should(ContainSubstring("use of closed network connection"))
				Ω(fakeServer.ReceivedRequests()).Should(HaveLen(1))
			})
		})
	})
})
