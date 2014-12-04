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
    "state": "RUNNING",
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
			State:  ActualLRPStateRunning,
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
			actualLrp := NewActualLRP(
				"processGuid",
				"instanceGuid",
				"cellID",
				"domain",
				1,
				ActualLRPStateClaimed,
			)
			Ω(actualLrp.ProcessGuid).Should(Equal("processGuid"))
			Ω(actualLrp.InstanceGuid).Should(Equal("instanceGuid"))
			Ω(actualLrp.CellID).Should(Equal("cellID"))
			Ω(actualLrp.Domain).Should(Equal("domain"))
			Ω(actualLrp.Index).Should(Equal(1))
			Ω(actualLrp.State).Should(Equal(ActualLRPStateClaimed))
		})
	})

	Describe("Validate", func() {
		var actualLRP ActualLRP

		BeforeEach(func() {
			actualLRP = ActualLRP{
				Domain:       "fake-domain",
				ProcessGuid:  "fake-process-guid",
				InstanceGuid: "fake-instance-guid",
				CellID:       "fake-cell-id",
				Index:        2,
			}
		})

		Context("when the State is unclaimed", func() {
			BeforeEach(func() {
				actualLRP.State = ActualLRPStateUnclaimed
			})

			Context("when the InstanceGuid and CellID are blank", func() {
				BeforeEach(func() {
					actualLRP.InstanceGuid = ""
					actualLRP.CellID = ""
				})

				It("returns nil", func() {
					Ω(actualLRP.Validate()).Should(BeNil())
				})
			})

			Context("when the InstanceGuid is not blank", func() {
				BeforeEach(func() {
					actualLRP.InstanceGuid = "fake-instance-guid"
					actualLRP.CellID = ""
				})

				It("returns a validation error", func() {
					Ω(actualLRP.Validate()).Should(ConsistOf(ErrInvalidField{"instance_guid"}))
				})
			})

			Context("when the CellID is not blank", func() {
				BeforeEach(func() {
					actualLRP.InstanceGuid = ""
					actualLRP.CellID = "fake-cell-id"
				})

				It("returns a validation error", func() {
					Ω(actualLRP.Validate()).Should(ConsistOf(ErrInvalidField{"cell_id"}))
				})
			})
		})

		Context("when the State is not unclaimed", func() {
			BeforeEach(func() {
				actualLRP.State = ActualLRPStateRunning
			})

			Context("when the InstanceGuid is blank", func() {
				BeforeEach(func() {
					actualLRP.InstanceGuid = ""
				})

				It("returns a validation error", func() {
					Ω(actualLRP.Validate()).Should(ConsistOf(ErrInvalidField{"instance_guid"}))
				})
			})

			Context("when the CellID is blank", func() {
				BeforeEach(func() {
					actualLRP.CellID = ""
				})

				It("returns a validation error", func() {
					Ω(actualLRP.Validate()).Should(ConsistOf(ErrInvalidField{"cell_id"}))
				})
			})

			Context("when the InstanceGuid and CellID are not blank", func() {
				BeforeEach(func() {
					actualLRP.InstanceGuid = "fake-instance-guid"
					actualLRP.CellID = "fake-cell-id"
				})

				It("returns nil", func() {
					Ω(actualLRP.Validate()).Should(BeNil())
				})
			})
		})
	})
})
