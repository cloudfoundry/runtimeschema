package models_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

var _ = Describe("AuctioneerPresence", func() {
	var auctioneerPresence models.AuctioneerPresence

	var payload string

	BeforeEach(func() {
		auctioneerPresence = models.AuctioneerPresence{
			AuctioneerID:      "some-id",
			AuctioneerAddress: "some-address",
		}

		payload = `{
    "auctioneer_id":      "some-id",
    "auctioneer_address": "some-address"
  }`
	})

	Describe("ToJSON", func() {
		It("should JSONify", func() {
			json, err := models.ToJSON(&auctioneerPresence)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(string(json)).Should(MatchJSON(payload))
		})
	})

	Describe("FromJSON", func() {
		It("returns an AuctioneerPresence with correct fields", func() {
			decodedAuctioneerPresence := &models.AuctioneerPresence{}
			err := models.FromJSON([]byte(payload), decodedAuctioneerPresence)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(decodedAuctioneerPresence).Should(Equal(&auctioneerPresence))
		})

		Context("with an invalid payload", func() {
			It("returns the error", func() {
				payload = "aliens lol"
				decodedAuctioneerPresence := &models.AuctioneerPresence{}
				err := models.FromJSON([]byte(payload), decodedAuctioneerPresence)

				Ω(err).Should(HaveOccurred())
			})
		})
	})
})
