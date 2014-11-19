package models_test

import (
	. "github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LRPStopAuction", func() {
	var stopAuctionPayload string
	var stopAuction LRPStopAuction

	BeforeEach(func() {
		stopAuctionPayload = `{
    "process_guid":"some-guid",
    "index": 2,
    "updated_at": 1138,
    "state": 1
  }`

		stopAuction = LRPStopAuction{
			ProcessGuid: "some-guid",
			Index:       2,
			State:       LRPStopAuctionStatePending,
			UpdatedAt:   1138,
		}
	})
	Describe("ToJSON", func() {
		It("should JSONify", func() {
			json, err := ToJSON(stopAuction)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(string(json)).Should(MatchJSON(stopAuctionPayload))
		})
	})

	Describe("NewLRPStopAuctionFromJSON", func() {
		It("returns a LRP with correct fields", func() {
			decodedStopAuction := &LRPStopAuction{}
			err := FromJSON([]byte(stopAuctionPayload), decodedStopAuction)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(decodedStopAuction).Should(Equal(&stopAuction))
		})

		Context("with an invalid payload", func() {
			It("returns the error", func() {
				stopAuctionPayload = "aliens lol"
				decodedStopAuction := &LRPStopAuction{}
				err := FromJSON([]byte(stopAuctionPayload), decodedStopAuction)

				Ω(err).Should(HaveOccurred())
			})
		})

		for field, payload := range map[string]string{
			"process_guid": `{"index": 0}`,
		} {
			json := payload
			missingField := field

			Context("when the json is missing a "+missingField, func() {
				It("returns an error indicating so", func() {
					decodedStopAuction := &LRPStopAuction{}
					err := FromJSON([]byte(json), decodedStopAuction)
					Ω(err).Should(HaveOccurred())
					Ω(err.Error()).Should(Equal("Invalid field: " + missingField))
				})
			})
		}
	})
})
