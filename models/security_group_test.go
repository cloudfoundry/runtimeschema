package models_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SecurityGroupRule", func() {
	var rule models.SecurityGroupRule

	rulePayload := `{
		"protocol": "tcp",
		"port_range": {
			"start": 1,
			"end": 1024
		},
		"destination": {
			"network_address": "1.2.3.4",
			"prefix_length": 16
		}
	}`

	BeforeEach(func() {
		rule = models.SecurityGroupRule{
			Protocol: models.TCPProtocol,
			PortRange: models.PortRange{
				Start: 1,
				End:   1024,
			},
			Destination: models.CIDR{
				NetworkAddress: "1.2.3.4",
				PrefixLength:   16,
			},
		}
	})

	Describe("To JSON", func() {
		It("should JSONify", func() {
			json, err := models.ToJSON(&rule)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(string(json)).Should(MatchJSON(rulePayload))
		})
	})

	Describe("Validation", func() {
		var (
			validationErr error
			protocol      string

			startPort uint
			endPort   uint

			networkAddress string
			prefixLength   uint8
		)

		BeforeEach(func() {
			protocol = "tcp"

			startPort = 1
			endPort = 65535

			networkAddress = "8.8.8.8"
			prefixLength = 16
		})

		JustBeforeEach(func() {
			rule = models.SecurityGroupRule{
				Protocol: models.ProtocolName(protocol),
				PortRange: models.PortRange{
					Start: startPort,
					End:   endPort,
				},
				Destination: models.CIDR{
					NetworkAddress: networkAddress,
					PrefixLength:   prefixLength,
				},
			}

			validationErr = rule.Validate()
		})

		Describe("protocol", func() {
			Context("when the protocol is tcp", func() {
				BeforeEach(func() {
					protocol = "tcp"
				})

				It("passes validation and does not return an error", func() {
					Ω(validationErr).ShouldNot(HaveOccurred())
				})
			})

			Context("when the protocol is udp", func() {
				BeforeEach(func() {
					protocol = "udp"
				})

				It("passes validation and does not return an error", func() {
					Ω(validationErr).ShouldNot(HaveOccurred())
				})
			})

			Context("when the protocol is icmp", func() {
				BeforeEach(func() {
					protocol = "icmp"
				})

				It("passes validation and does not return an error", func() {
					Ω(validationErr).ShouldNot(HaveOccurred())
				})
			})

			Context("when the protocol is all", func() {
				BeforeEach(func() {
					protocol = "all"
				})

				It("passes validation and does not return an error", func() {
					Ω(validationErr).ShouldNot(HaveOccurred())
				})
			})

			Context("when the protocol is invalid", func() {
				BeforeEach(func() {
					protocol = "foo"
				})

				It("returns an error", func() {
					Ω(validationErr).Should(MatchError(ContainSubstring("protocol")))
				})
			})
		})

		Describe("port range", func() {
			Context("when it is a valid port range", func() {
				BeforeEach(func() {
					startPort = 1
					endPort = 65535
				})

				It("passes validation and does not return an error", func() {
					Ω(validationErr).ShouldNot(HaveOccurred())
				})
			})

			Context("when a port is greater than 65535", func() {
				BeforeEach(func() {
					startPort = 65534
					endPort = 65536
				})

				It("returns an error", func() {
					Ω(validationErr).Should(MatchError(ContainSubstring("port_range")))
				})
			})

			Context("when port range has a start value greater than the end value", func() {
				BeforeEach(func() {
					startPort = 1024
					endPort = 1
				})

				It("returns an error", func() {
					Ω(validationErr).Should(MatchError(ContainSubstring("port_range")))
				})
			})
		})

		Describe("destination", func() {
			Context("when the destination is valid", func() {
				BeforeEach(func() {
					networkAddress = "1.2.3.4"
					prefixLength = 32
				})

				It("passes validation and does not return an error", func() {
					Ω(validationErr).ShouldNot(HaveOccurred())
				})
			})

			Context("when the destination is missing", func() {
				BeforeEach(func() {
					networkAddress = ""
					prefixLength = 0
				})

				It("returns an error", func() {
					Ω(validationErr).Should(MatchError(ContainSubstring("destination")))
				})
			})

			Context("when the subnet prefix length exceeds 32", func() {
				BeforeEach(func() {
					networkAddress = "1.2.3.4"
					prefixLength = 33
				})

				It("returns an error", func() {
					Ω(validationErr).Should(MatchError(ContainSubstring("destination")))
				})
			})
		})

		Context("when thre are multiple field validations", func() {
			BeforeEach(func() {
				protocol = "foo"
				startPort = 443
				endPort = 80
				networkAddress = ""
			})

			It("aggregates validation errors", func() {
				Ω(validationErr).Should(MatchError(ContainSubstring("protocol")))
				Ω(validationErr).Should(MatchError(ContainSubstring("port_range")))
				Ω(validationErr).Should(MatchError(ContainSubstring("destination")))
			})
		})
	})
})
