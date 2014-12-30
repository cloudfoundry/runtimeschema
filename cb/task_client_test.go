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

var _ = Describe("TaskClient", func() {
	var fakeServer *ghttp.Server
	var client cb.TaskClient

	BeforeEach(func() {
		fakeServer = ghttp.NewServer()
		client = cb.NewTaskClient()
	})

	AfterEach(func() {
		fakeServer.Close()
	})

	Describe("CompleteTask", func() {
		var completeErr error

		var task = models.Task{
			TaskGuid:      "some-guid",
			CellID:        "some-cell-id",
			Failed:        true,
			FailureReason: "because",
		}

		JustBeforeEach(func() {
			completeErr = client.CompleteTasks(fakeServer.URL(), []models.Task{task})
		})

		Context("when the request is successful", func() {
			BeforeEach(func() {
				fakeServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("PUT", "/internal/tasks/complete"),
						ghttp.VerifyJSONRepresenting([]models.Task{task}),
						ghttp.RespondWith(http.StatusAccepted, ""),
					),
				)
			})

			It("makes the request and does not return an error", func() {
				Ω(completeErr).ShouldNot(HaveOccurred())
				Ω(fakeServer.ReceivedRequests()).Should(HaveLen(1))
			})
		})

		Context("when the request returns 500", func() {
			BeforeEach(func() {
				fakeServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("PUT", "/internal/tasks/complete"),
						ghttp.VerifyJSONRepresenting([]models.Task{task}),
						ghttp.RespondWith(http.StatusInternalServerError, ""),
					),
				)
			})

			It("makes the request and returns an error", func() {
				Ω(completeErr).Should(HaveOccurred())
				Ω(fakeServer.ReceivedRequests()).Should(HaveLen(1))
			})
		})

		Context("when the connection fails", func() {
			BeforeEach(func() {
				fakeServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("PUT", "/internal/tasks/complete"),
						ghttp.VerifyJSONRepresenting([]models.Task{task}),
						func(w http.ResponseWriter, r *http.Request) {
							fakeServer.CloseClientConnections()
						},
					),
				)
			})

			It("makes the request and returns an error", func() {
				Ω(completeErr).Should(HaveOccurred())
				Ω(fakeServer.ReceivedRequests()).Should(HaveLen(1))
			})
		})

		Context("when the connection times out", func() {
			BeforeEach(func() {
				fakeServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("PUT", "/internal/tasks/complete"),
						ghttp.VerifyJSONRepresenting([]models.Task{task}),
						func(w http.ResponseWriter, r *http.Request) {
							time.Sleep(cfHttpTimeout + 100*time.Millisecond)
						},
					),
				)
			})

			It("makes the request and returns an error", func() {
				Ω(completeErr).Should(HaveOccurred())
				Ω(completeErr.Error()).Should(ContainSubstring("use of closed network connection"))
				Ω(fakeServer.ReceivedRequests()).Should(HaveLen(1))
			})
		})
	})
})
