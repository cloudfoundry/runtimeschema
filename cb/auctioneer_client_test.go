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

var _ = Describe("AuctioneerClient", func() {
	var fakeServer *ghttp.Server
	var auctioneerClient cb.AuctioneerClient

	BeforeEach(func() {
		fakeServer = ghttp.NewServer()
		auctioneerClient = cb.NewAuctioneerClient()
	})

	AfterEach(func() {
		fakeServer.Close()
	})

	Describe("RequestLRPStartAuctions", func() {
		const auctioneerAddr = "auctioneer.example.com"
		var lrpStarts = []models.LRPStartRequest{{
			DesiredLRP: models.DesiredLRP{},
			Indices:    []uint{2},
		}}
		var err error

		JustBeforeEach(func() {
			err = auctioneerClient.RequestLRPAuctions(fakeServer.URL(), lrpStarts)
		})

		Context("when the request is successful", func() {
			BeforeEach(func() {
				fakeServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/lrps"),
						ghttp.VerifyJSONRepresenting(lrpStarts),
						ghttp.RespondWith(http.StatusAccepted, ""),
					),
				)
			})

			It("makes the request and does not return an error", func() {
				Ω(err).ShouldNot(HaveOccurred())
				Ω(fakeServer.ReceivedRequests()).Should(HaveLen(1))
			})
		})

		Context("when the request returns 400", func() {
			BeforeEach(func() {
				fakeServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/lrps"),
						ghttp.VerifyJSONRepresenting(lrpStarts),
						ghttp.RespondWith(http.StatusBadRequest, ""),
					),
				)
			})

			It("makes the request and returns an error", func() {
				Ω(err).Should(HaveOccurred())
				Ω(err.Error()).Should(ContainSubstring("http error: status code 400"))
				Ω(fakeServer.ReceivedRequests()).Should(HaveLen(1))
			})
		})

		Context("when the connection fails", func() {
			BeforeEach(func() {
				fakeServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/lrps"),
						ghttp.VerifyJSONRepresenting(lrpStarts),
						func(w http.ResponseWriter, r *http.Request) {
							fakeServer.CloseClientConnections()
						},
					),
				)
			})

			It("makes the request and returns an error", func() {
				Ω(err).Should(HaveOccurred())
				Ω(err.Error()).Should(ContainSubstring("EOF"))
				Ω(fakeServer.ReceivedRequests()).Should(HaveLen(1))
			})
		})

		Context("when the connection times out", func() {
			BeforeEach(func() {
				fakeServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/lrps"),
						ghttp.VerifyJSONRepresenting(lrpStarts),
						func(w http.ResponseWriter, r *http.Request) {
							time.Sleep(cfHttpTimeout + 100*time.Millisecond)
						},
					),
				)
			})

			It("makes the request and returns an error", func() {
				Ω(err).Should(HaveOccurred())
				Ω(err.Error()).Should(ContainSubstring("use of closed network connection"))
				Ω(fakeServer.ReceivedRequests()).Should(HaveLen(1))
			})
		})
	})

	Describe("RequestTaskAuction", func() {
		const auctioneerAddr = "auctioneer.example.com"
		var task = models.Task{
			TaskGuid: "the-task-guid",
		}
		var err error

		JustBeforeEach(func() {
			err = auctioneerClient.RequestTaskAuctions(fakeServer.URL(), []models.Task{task})
		})

		Context("when the request is successful", func() {
			BeforeEach(func() {
				fakeServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/tasks"),
						ghttp.VerifyJSONRepresenting([]models.Task{task}),
						ghttp.RespondWith(http.StatusAccepted, ""),
					),
				)
			})

			It("makes the request and does not return an error", func() {
				Ω(err).ShouldNot(HaveOccurred())
				Ω(fakeServer.ReceivedRequests()).Should(HaveLen(1))
			})
		})

		Context("when the request returns 400", func() {
			BeforeEach(func() {
				fakeServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/tasks"),
						ghttp.VerifyJSONRepresenting([]models.Task{task}),
						ghttp.RespondWith(http.StatusBadRequest, ""),
					),
				)
			})

			It("makes the request and returns an error", func() {
				Ω(err).Should(HaveOccurred())
				Ω(err.Error()).Should(ContainSubstring("http error: status code 400"))
				Ω(fakeServer.ReceivedRequests()).Should(HaveLen(1))
			})
		})

		Context("when the connection fails", func() {
			BeforeEach(func() {
				fakeServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/tasks"),
						ghttp.VerifyJSONRepresenting([]models.Task{task}),
						func(w http.ResponseWriter, r *http.Request) {
							fakeServer.CloseClientConnections()
						},
					),
				)
			})

			It("makes the request and returns an error", func() {
				Ω(err).Should(HaveOccurred())
				Ω(err.Error()).Should(ContainSubstring("EOF"))
				Ω(fakeServer.ReceivedRequests()).Should(HaveLen(1))
			})
		})

		Context("when the connection times out", func() {
			BeforeEach(func() {
				fakeServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/tasks"),
						ghttp.VerifyJSONRepresenting([]models.Task{task}),
						func(w http.ResponseWriter, r *http.Request) {
							time.Sleep(cfHttpTimeout + 100*time.Millisecond)
						},
					),
				)
			})

			It("makes the request and returns an error", func() {
				Ω(err).Should(HaveOccurred())
				Ω(err.Error()).Should(ContainSubstring("use of closed network connection"))
				Ω(fakeServer.ReceivedRequests()).Should(HaveLen(1))
			})
		})
	})
})
