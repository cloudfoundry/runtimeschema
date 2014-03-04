package router_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/router"
	"github.com/cloudfoundry/gunk/test_server"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
	"net/http/httptest"
)

var _ = Describe("Router", func() {

	var (
		r    http.Handler
		resp *httptest.ResponseRecorder
		err  error
	)

	Describe("MakePat", func() {
		Context("when all the handlers are present", func() {
			BeforeEach(func() {
				resp = httptest.NewRecorder()
				routes := router.Routes{
					{Path: "/something", Method: "GET", Handler: "getter"},
					{Path: "/something", Method: "POST", Handler: "poster"},
					{Path: "/something", Method: "PuT", Handler: "putter"},
					{Path: "/something", Method: "DELETE", Handler: "deleter"},
				}
				r, err = routes.Router(router.Handlers{
					"getter":  test_server.Respond(http.StatusOK, "get response"),
					"poster":  test_server.Respond(http.StatusOK, "post response"),
					"putter":  test_server.Respond(http.StatusOK, "put response"),
					"deleter": test_server.Respond(http.StatusOK, "delete response"),
				})
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("makes GET handlers", func() {
				req, _ := http.NewRequest("GET", "/something", nil)

				r.ServeHTTP(resp, req)
				Ω(resp.Body.String()).Should(Equal("get response"))
			})

			It("makes POST handlers", func() {
				req, _ := http.NewRequest("POST", "/something", nil)

				r.ServeHTTP(resp, req)
				Ω(resp.Body.String()).Should(Equal("post response"))
			})

			It("makes PUT handlers", func() {
				req, _ := http.NewRequest("PUT", "/something", nil)

				r.ServeHTTP(resp, req)
				Ω(resp.Body.String()).Should(Equal("put response"))
			})

			It("makes DELETE handlers", func() {
				req, _ := http.NewRequest("DELETE", "/something", nil)

				r.ServeHTTP(resp, req)
				Ω(resp.Body.String()).Should(Equal("delete response"))
			})
		})

		Context("when a handler is missing", func() {
			It("should error", func() {
				routes := router.Routes{
					{Path: "/something", Method: "GET", Handler: "getter"},
					{Path: "/something", Method: "POST", Handler: "poster"},
				}
				r, err = routes.Router(router.Handlers{
					"getter": test_server.Respond(http.StatusOK, "get response"),
				})

				Ω(err).Should(HaveOccurred())
			})
		})

		Context("with an invalid verb", func() {
			It("should error", func() {
				routes := router.Routes{
					{Path: "/something", Method: "SMELL", Handler: "smeller"},
				}
				r, err = routes.Router(router.Handlers{
					"smeller": test_server.Respond(http.StatusOK, "smelt response"),
				})

				Ω(err).Should(HaveOccurred())
			})
		})
	})
})
