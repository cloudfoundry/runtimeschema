package models_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Freshness", func() {
	Describe("Validate", func() {
		var freshness models.Freshness

		It("passes a freshness with a domain and a positive TTL", func() {
			freshness = models.Freshness{
				Domain:       "my-fun-domain",
				TTLInSeconds: 1000,
			}

			err := freshness.Validate()
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("passes a freshness with a domain and a zero TTL", func() {
			freshness = models.Freshness{
				Domain: "my-fun-domain",
			}

			err := freshness.Validate()
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("checks the domain is not empty", func() {
			freshness = models.Freshness{
				TTLInSeconds: 1000,
			}

			err := freshness.Validate()
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(ContainSubstring("domain"))
		})

		It("checks the TTL is not negative", func() {
			freshness = models.Freshness{
				Domain:       "legit-domain",
				TTLInSeconds: -1000,
			}

			err := freshness.Validate()
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(ContainSubstring("ttl_in_seconds"))

		})
	})
})
