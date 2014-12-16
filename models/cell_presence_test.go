package models_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

var _ = Describe("CellPresence", func() {
	var cellPresence models.CellPresence

	var payload string

	BeforeEach(func() {
		cellPresence = models.CellPresence{
			CellID:     "some-id",
			Stack:      "some-stack",
			RepAddress: "some-address",
		}

		payload = `{
    "cell_id":"some-id",
    "stack": "some-stack",
    "rep_address": "some-address"
  }`
	})

	Describe("ToJSON", func() {
		It("should JSONify", func() {
			json, err := models.ToJSON(&cellPresence)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(string(json)).Should(MatchJSON(payload))
		})
	})

	Describe("FromJSON", func() {
		It("returns a CellPresence with correct fields", func() {
			decodedCellPresence := &models.CellPresence{}
			err := models.FromJSON([]byte(payload), decodedCellPresence)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(decodedCellPresence).Should(Equal(&cellPresence))
		})

		Context("with an invalid payload", func() {
			It("returns the error", func() {
				payload = "aliens lol"
				decodedCellPresence := &models.CellPresence{}
				err := models.FromJSON([]byte(payload), decodedCellPresence)

				Ω(err).Should(HaveOccurred())
			})
		})
	})
})
