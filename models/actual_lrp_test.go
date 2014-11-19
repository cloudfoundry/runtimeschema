package models_test

import (
	"encoding/json"

	. "github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ActualLRP", func() {
	var lrp ActualLRP

	lrpPayload := `{
    "process_guid":"some-guid",
    "instance_guid":"some-instance-guid",
    "host": "1.2.3.4",
    "ports": [
      { "container_port": 8080 },
      { "container_port": 8081, "host_port": 1234 }
    ],
    "index": 2,
    "state": 0,
    "since": 1138,
    "cell_id":"some-cell-id",
    "domain":"some-domain"
  }`

	BeforeEach(func() {
		lrp = ActualLRP{
			ProcessGuid:  "some-guid",
			InstanceGuid: "some-instance-guid",
			Host:         "1.2.3.4",
			Ports: []PortMapping{
				{ContainerPort: 8080},
				{ContainerPort: 8081, HostPort: 1234},
			},
			Index:  2,
			Since:  1138,
			CellID: "some-cell-id",
			Domain: "some-domain",
		}
	})

	Describe("To JSON", func() {
		It("should JSONify", func() {
			marshalled, err := json.Marshal(&lrp)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(string(marshalled)).Should(MatchJSON(lrpPayload))
		})
	})

	Describe("FromJSON", func() {
		It("returns a LRP with correct fields", func() {
			aLRP := &ActualLRP{}
			err := FromJSON([]byte(lrpPayload), aLRP)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(aLRP).Should(Equal(&lrp))
		})

		Context("with an invalid payload", func() {
			It("returns the error", func() {
				aLRP := &ActualLRP{}
				err := FromJSON([]byte("something lol"), aLRP)
				Ω(err).Should(HaveOccurred())
			})
		})

		for field, payload := range map[string]string{
			"process_guid":  `{"instance_guid": "instance_guid", "cell_id": "cell_id", "domain": "domain"}`,
			"instance_guid": `{"process_guid": "process-guid", "cell_id": "cell_id", "domain": "domain"}`,
			"cell_id":       `{"process_guid": "process-guid", "instance_guid": "instance_guid", "domain": "domain"}`,
			"domain":        `{"process_guid": "process-guid", "cell_id": "cell_id", "instance_guid": "instance_guid"}`,
		} {
			missingField := field
			jsonPayload := payload

			Context("when the json is missing a "+missingField, func() {
				It("returns an error indicating so", func() {
					aLRP := &ActualLRP{}
					err := FromJSON([]byte(jsonPayload), aLRP)
					Ω(err.Error()).Should(Equal("Invalid field: " + missingField))
				})
			})
		}
	})

	Describe("NewActualLRP", func() {
		It("returns a LRP with correct fields", func() {
			actualLrp, err := NewActualLRP(
				"processGuid",
				"instanceGuid",
				"cellID",
				"domain",
				0,
			)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(actualLrp.ProcessGuid).Should(Equal("processGuid"))
			Ω(actualLrp.InstanceGuid).Should(Equal("instanceGuid"))
			Ω(actualLrp.CellID).Should(Equal("cellID"))
			Ω(actualLrp.Domain).Should(Equal("domain"))
			Ω(actualLrp.Index).Should(BeZero())
		})

		Context("When given a blank process guid", func() {
			It("returns a validation error", func() {
				_, err := NewActualLRP("", "instanceGuid", "cellID", "domain", 0)
				Ω(err).Should(HaveOccurred())
				Ω(err).Should(BeAssignableToTypeOf(ValidationError{}))
			})
		})
	})
})
