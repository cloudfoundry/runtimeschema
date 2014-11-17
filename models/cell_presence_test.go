package models_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry-incubator/runtime-schema/models"
)

var _ = Describe("CellPresence", func() {
	var cellPresence CellPresence

	const payload = `{
    "cell_id":"some-id",
    "stack": "some-stack"
  }`

	BeforeEach(func() {
		cellPresence = CellPresence{
			CellID: "some-id",
			Stack:  "some-stack",
		}
	})

	Describe("ToJSON", func() {
		It("should JSONify", func() {
			json := cellPresence.ToJSON()
			Ω(string(json)).Should(MatchJSON(payload))
		})
	})

	Describe("NewTaskFromJSON", func() {
		It("returns a Task with correct fields", func() {
			decodedCellPresence, err := NewCellPresenceFromJSON([]byte(payload))
			Ω(err).ShouldNot(HaveOccurred())

			Ω(decodedCellPresence).Should(Equal(cellPresence))
		})

		Context("with an invalid payload", func() {
			It("returns the error", func() {
				decodedCellPresence, err := NewCellPresenceFromJSON([]byte("aliens lol"))
				Ω(err).Should(HaveOccurred())

				Ω(decodedCellPresence).Should(BeZero())
			})
		})
	})
})
