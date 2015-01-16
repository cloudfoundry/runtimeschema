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
		"destination": "1.2.3.4/16",
		"port_range": {
			"start": 1,
			"end": 1024
		}
	}`

	BeforeEach(func() {
		rule = models.SecurityGroupRule{
			Protocol:    models.TCPProtocol,
			Destination: "1.2.3.4/16",
			PortRange: &models.PortRange{
				Start: 1,
				End:   1024,
			},
		}
	})

	Describe("To JSON", func() {
		It("should JSONify a rule", func() {
			json, err := models.ToJSON(&rule)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(string(json)).Should(MatchJSON(rulePayload))
		})

		It("should JSONify icmp info", func() {
			icmpRule := models.SecurityGroupRule{
				Protocol:    models.ICMPProtocol,
				Destination: "1.2.3.4/16",
				IcmpInfo:    &models.ICMPInfo{},
			}

			json, err := models.ToJSON(&icmpRule)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(string(json)).Should(MatchJSON(`{"protocol": "icmp", "destination": "1.2.3.4/16", "icmp_info": {"type":0,"code":0} }`))
		})
	})

	Describe("Validation", func() {
		var (
			validationErr error

			protocol    string
			destination string

			portRange *models.PortRange

			icmpInfo *models.ICMPInfo
		)

		BeforeEach(func() {
			protocol = "tcp"
			destination = "8.8.8.8/16"

			portRange = &models.PortRange{1, 65535}
			icmpInfo = nil
		})

		JustBeforeEach(func() {
			rule = models.SecurityGroupRule{
				Protocol:    models.ProtocolName(protocol),
				Destination: destination,
				PortRange:   portRange,
				IcmpInfo:    icmpInfo,
			}

			validationErr = rule.Validate()
		})

		itExpectsAPortRange := func() {
			Describe("port range", func() {
				Context("when it is a valid port range", func() {
					BeforeEach(func() {
						portRange = &models.PortRange{1, 65535}
					})

					It("passes validation and does not return an error", func() {
						Ω(validationErr).ShouldNot(HaveOccurred())
					})
				})

				Context("when a port is greater than 65535", func() {
					BeforeEach(func() {
						portRange = &models.PortRange{65534, 65536}
					})

					It("returns an error", func() {
						Ω(validationErr).Should(MatchError(ContainSubstring("port_range")))
					})
				})

				Context("when port range has a start value greater than the end value", func() {
					BeforeEach(func() {
						portRange = &models.PortRange{1024, 1}
					})

					It("returns an error", func() {
						Ω(validationErr).Should(MatchError(ContainSubstring("port_range")))
					})
				})
			})
		}

		itExpectsADestination := func() {
			Describe("destination", func() {
				Context("when the destination is valid", func() {
					BeforeEach(func() {
						destination = "1.2.3.4/32"
					})

					It("passes validation and does not return an error", func() {
						Ω(validationErr).ShouldNot(HaveOccurred())
					})
				})

				Context("when the destination is invalid", func() {
					BeforeEach(func() {
						destination = "garbage/32"
					})

					It("returns an error", func() {
						Ω(validationErr).Should(MatchError(ContainSubstring("destination")))
					})
				})
			})
		}

		itFailsWithICMPInfo := func() {
			Context("when ICMP info is provided", func() {
				BeforeEach(func() {
					icmpInfo = &models.ICMPInfo{}
				})
				It("fails", func() {
					Ω(validationErr).Should(HaveOccurred())
				})
			})
		}

		Describe("protocol", func() {
			Context("when the protocol is tcp", func() {
				BeforeEach(func() {
					protocol = "tcp"
				})

				It("passes validation and does not return an error", func() {
					Ω(validationErr).ShouldNot(HaveOccurred())
				})

				itFailsWithICMPInfo()
				itExpectsAPortRange()
				itExpectsADestination()
			})

			Context("when the protocol is udp", func() {
				BeforeEach(func() {
					protocol = "udp"
				})

				It("passes validation and does not return an error", func() {
					Ω(validationErr).ShouldNot(HaveOccurred())
				})

				itFailsWithICMPInfo()
				itExpectsAPortRange()
				itExpectsADestination()
			})

			Context("when the protocol is icmp", func() {
				BeforeEach(func() {
					protocol = "icmp"
					portRange = nil
					icmpInfo = &models.ICMPInfo{}
				})

				It("passes validation and does not return an error", func() {
					Ω(validationErr).ShouldNot(HaveOccurred())
				})
				itExpectsADestination()

				Context("when no ICMPInfo is provided", func() {
					BeforeEach(func() {
						icmpInfo = nil
					})

					It("fails", func() {
						Ω(validationErr).Should(HaveOccurred())
					})
				})

				Context("when Port range is provided", func() {
					BeforeEach(func() {
						portRange = &models.PortRange{1, 65535}
					})

					It("fails", func() {
						Ω(validationErr).Should(HaveOccurred())
					})
				})
			})

			Context("when the protocol is all", func() {
				BeforeEach(func() {
					protocol = "all"
					portRange = nil
				})

				It("passes validation and does not return an error", func() {
					Ω(validationErr).ShouldNot(HaveOccurred())
				})

				itFailsWithICMPInfo()
				itExpectsADestination()

				Context("when Port range is provided", func() {
					BeforeEach(func() {
						portRange = &models.PortRange{1, 65535}
					})

					It("fails", func() {
						Ω(validationErr).Should(HaveOccurred())
					})
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

		Context("when thre are multiple field validations", func() {
			BeforeEach(func() {
				protocol = "tcp"
				destination = "garbage"
				portRange = &models.PortRange{443, 80}
			})

			It("aggregates validation errors", func() {
				Ω(validationErr).Should(MatchError(ContainSubstring("port_range")))
				Ω(validationErr).Should(MatchError(ContainSubstring("destination")))
			})
		})
	})
})
