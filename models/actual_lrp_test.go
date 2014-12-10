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

	FDescribe("IsEquivalentTo", func() {
		var lhs ActualLRP
		var rhs ActualLRP

		BeforeEach(func() {
			lhs = ActualLRP{
				ProcessGuid:  "process-guid",
				InstanceGuid: "instance-guid",
				Domain:       "domain",
				CellID:       "cell-id",
				Index:        1,
				State:        ActualLRPStateClaimed,

				Since: 1138,
				Ports: []PortMapping{{ContainerPort: 2357, HostPort: 3468}},
				Host:  "cell-host",
			}
			rhs = lhs
		})

		Context("when the actuals are equal", func() {
			It("is true", func() {
				Ω(lhs.IsEquivalentTo(rhs)).Should(BeTrue())
			})
		})

		Context("when the ProcessGuid differs", func() {
			BeforeEach(func() {
				rhs.ProcessGuid = "some-other-guid"
			})

			It("is false", func() {
				Ω(lhs.IsEquivalentTo(rhs)).Should(BeFalse())
			})
		})

		Context("when the InstanceGuid differs", func() {
			BeforeEach(func() {
				rhs.InstanceGuid = "some-other-guid"
			})

			It("is false", func() {
				Ω(lhs.IsEquivalentTo(rhs)).Should(BeFalse())
			})
		})

		Context("when the Domain differs", func() {
			BeforeEach(func() {
				rhs.Domain = "some-other-domain"
			})

			It("is false", func() {
				Ω(lhs.IsEquivalentTo(rhs)).Should(BeFalse())
			})
		})

		Context("when the CellID differs", func() {
			BeforeEach(func() {
				rhs.CellID = "some-other-cell-id"
			})

			It("is false", func() {
				Ω(lhs.IsEquivalentTo(rhs)).Should(BeFalse())
			})
		})

		Context("when the Index differs", func() {
			BeforeEach(func() {
				rhs.Index = 2
			})

			It("is false", func() {
				Ω(lhs.IsEquivalentTo(rhs)).Should(BeFalse())
			})
		})

		Context("when the State differs", func() {
			BeforeEach(func() {
				rhs.State = ActualLRPStateUnclaimed
			})

			It("is false", func() {
				Ω(lhs.IsEquivalentTo(rhs)).Should(BeFalse())
			})
		})

		Context("when the Ports differ", func() {
			BeforeEach(func() {
				rhs.Ports = nil
			})

			It("is true", func() {
				Ω(lhs.IsEquivalentTo(rhs)).Should(BeTrue())
			})
		})

		Context("when the Host differs", func() {
			BeforeEach(func() {
				rhs.Host = "some-other-host"
			})

			It("is true", func() {
				Ω(lhs.IsEquivalentTo(rhs)).Should(BeTrue())
			})
		})

		Context("when the Since differs", func() {
			BeforeEach(func() {
				rhs.Since = 3417
			})

			It("is true", func() {
				Ω(lhs.IsEquivalentTo(rhs)).Should(BeTrue())
			})
		})
	})

	Describe("Validate", func() {
		var actualLRP ActualLRP

		itValidatesCommonRequiredFields := func() {
			Context("when valid", func() {
				It("returns nil", func() {
					Ω(actualLRP.Validate()).Should(BeNil())
				})
			})

			Context("when the ProcessGuid is blank", func() {
				BeforeEach(func() {
					actualLRP.ProcessGuid = ""
				})

				It("returns a validation error", func() {
					Ω(actualLRP.Validate()).Should(ConsistOf(ErrInvalidField{"process_guid"}))
				})
			})

			Context("when the Domain is blank", func() {
				BeforeEach(func() {
					actualLRP.Domain = ""
				})

				It("returns a validation error", func() {
					Ω(actualLRP.Validate()).Should(ConsistOf(ErrInvalidField{"domain"}))
				})
			})

			Context("when the InstanceGuid is blank", func() {
				BeforeEach(func() {
					actualLRP.InstanceGuid = ""
				})

				It("returns a validation error", func() {
					Ω(actualLRP.Validate()).Should(ConsistOf(ErrInvalidField{"instance_guid"}))
				})
			})
		}

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
				actualLRP.CellID = ""
			})

			itValidatesCommonRequiredFields()

			Context("when the CellID is not blank", func() {
				BeforeEach(func() {
					actualLRP.CellID = "fake-cell-id"
				})

				It("returns a validation error", func() {
					Ω(actualLRP.Validate()).Should(ConsistOf(ErrInvalidField{"cell_id"}))
				})
			})
		})

		Context("when the State is claimed", func() {
			BeforeEach(func() {
				actualLRP.State = ActualLRPStateClaimed
			})

			itValidatesCommonRequiredFields()

			Context("when the CellID is blank", func() {
				BeforeEach(func() {
					actualLRP.CellID = ""
				})

				It("returns a validation error", func() {
					Ω(actualLRP.Validate()).Should(ConsistOf(ErrInvalidField{"cell_id"}))
				})
			})
		})

		Context("when the State is running", func() {
			BeforeEach(func() {
				actualLRP.State = ActualLRPStateRunning
			})

			itValidatesCommonRequiredFields()

			Context("when the CellID is blank", func() {
				BeforeEach(func() {
					actualLRP.CellID = ""
				})

				It("returns a validation error", func() {
					Ω(actualLRP.Validate()).Should(ConsistOf(ErrInvalidField{"cell_id"}))
				})
			})
		})
	})
})
