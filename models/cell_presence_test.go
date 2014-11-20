package models_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry-incubator/runtime-schema/models"
)

var _ = Describe("CellPresence", func() {
	var cellPresence CellPresence

	var payload string

	BeforeEach(func() {
		cellPresence = CellPresence{
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
			json, err := ToJSON(&cellPresence)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(string(json)).Should(MatchJSON(payload))
		})
	})

	Describe("NewTaskFromJSON", func() {
		It("returns a Task with correct fields", func() {
			decodedCellPresence := &CellPresence{}
			err := FromJSON([]byte(payload), decodedCellPresence)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(decodedCellPresence).Should(Equal(&cellPresence))
		})

		Context("with an invalid payload", func() {
			It("returns the error", func() {
				payload = "aliens lol"
				decodedCellPresence := &CellPresence{}
				err := FromJSON([]byte(payload), decodedCellPresence)

				Ω(err).Should(HaveOccurred())
			})
		})
	})
})
