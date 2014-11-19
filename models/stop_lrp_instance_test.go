package models_test

import (
	. "github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("StopLrpInstance", func() {
	var stopInstancePayload string
	var stopInstance StopLRPInstance

	BeforeEach(func() {
		stopInstancePayload = `{
		"process_guid":"some-process-guid",
    "instance_guid":"some-instance-guid",
    "index":1234
  }`

		stopInstance = StopLRPInstance{
			ProcessGuid:  "some-process-guid",
			InstanceGuid: "some-instance-guid",
			Index:        1234,
		}
	})
	Describe("ToJSON", func() {
		It("should JSONify", func() {
			json, err := ToJSON(stopInstance)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(string(json)).Should(MatchJSON(stopInstancePayload))
		})
	})

	Describe("NewStopLRPInstanceFromJSON", func() {
		It("returns a LRP with correct fields", func() {
			decodedStopInstance := &StopLRPInstance{}
			err := FromJSON([]byte(stopInstancePayload), decodedStopInstance)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(decodedStopInstance).Should(Equal(&stopInstance))
		})

		Context("with an invalid payload", func() {
			It("returns the error", func() {
				stopInstancePayload = "aliens lol"
				decodedStopInstance := &StopLRPInstance{}
				err := FromJSON([]byte(stopInstancePayload), decodedStopInstance)

				Ω(err).Should(HaveOccurred())
			})
		})

		for field, payload := range map[string]string{
			"process_guid":  `{"instance_guid": "instance_guid", "cell_id": "cell_id"}`,
			"instance_guid": `{"process_guid": "process-guid", "cell_id": "cell_id"}`,
		} {
			json := payload
			missingField := field

			Context("when the json is missing a "+missingField, func() {
				It("returns an error indicating so", func() {
					decodedStopInstance := &StopLRPInstance{}
					err := FromJSON([]byte(json), decodedStopInstance)
					Ω(err).Should(HaveOccurred())
					Ω(err.Error()).Should(Equal("Invalid field: " + missingField))
				})
			})
		}
	})
})
