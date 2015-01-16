package models_test

import (
	"encoding/json"
	"fmt"
	"strings"

	. "github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DesiredLRP", func() {
	var lrp DesiredLRP

	lrpPayload := `{
	  "process_guid": "some-guid",
	  "domain": "some-domain",
	  "root_fs": "docker:///docker.com/docker",
	  "instances": 1,
	  "stack": "some-stack",
		"annotation": "some-annotation",
		"start_timeout": 0,
	  "env":[
	    {
	      "name": "ENV_VAR_NAME",
	      "value": "some environment variable value"
	    }
	  ],
		"setup": {
			"download": {
				"from": "http://example.com",
				"to": "/tmp/internet",
				"cache_key": ""
			}
		},
		"action": {
			"run": {
				"path": "ls",
				"args": null,
				"env": null,
				"resource_limits":{}
			}
		},
		"monitor": {
			"run": {
				"path": "reboot",
				"args": null,
				"env": null,
				"resource_limits":{}
			}
		},
	  "disk_mb": 512,
	  "memory_mb": 1024,
	  "cpu_weight": 42,
		"privileged": true,
	  "ports": [
	    5678
	  ],
	  "routes": [
	    "route-1",
	    "route-2"
	  ],
	  "log_guid": "log-guid",
	  "log_source": "the cloud",
	  "security_group_rules": [
		  {
				"protocol": "tcp",
				"destination": "0.0.0.0/0",
				"port_range": {
					"start": 1,
					"end": 1024
				}
			},
		  {
				"protocol": "udp",
				"destination": "8.8.0.0/16",
				"port_range": {
					"start": 53,
					"end": 53
				}
			}
		]
	}`

	BeforeEach(func() {
		lrp = DesiredLRP{
			Domain:      "some-domain",
			ProcessGuid: "some-guid",

			Instances:  1,
			Stack:      "some-stack",
			RootFSPath: "docker:///docker.com/docker",
			MemoryMB:   1024,
			DiskMB:     512,
			CPUWeight:  42,
			Privileged: true,
			Routes:     []string{"route-1", "route-2"},
			Annotation: "some-annotation",
			Ports: []uint32{
				5678,
			},
			LogGuid:   "log-guid",
			LogSource: "the cloud",
			EnvironmentVariables: []EnvironmentVariable{
				{
					Name:  "ENV_VAR_NAME",
					Value: "some environment variable value",
				},
			},
			Setup: &DownloadAction{
				From: "http://example.com",
				To:   "/tmp/internet",
			},
			Action: &RunAction{
				Path: "ls",
			},
			Monitor: &RunAction{
				Path: "reboot",
			},
			SecurityGroupRules: []SecurityGroupRule{
				{
					Protocol:    "tcp",
					Destination: "0.0.0.0/0",
					PortRange: &PortRange{
						Start: 1,
						End:   1024,
					},
				},
				{
					Protocol:    "udp",
					Destination: "8.8.0.0/16",
					PortRange: &PortRange{
						Start: 53,
						End:   53,
					},
				},
			},
		}
	})

	Describe("To JSON", func() {
		It("should JSONify", func() {
			json, err := ToJSON(&lrp)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(string(json)).Should(MatchJSON(lrpPayload))
		})
	})

	Describe("ApplyUpdate", func() {
		It("updates instances", func() {
			instances := 100
			update := DesiredLRPUpdate{Instances: &instances}

			expectedLRP := lrp
			expectedLRP.Instances = instances

			updatedLRP := lrp.ApplyUpdate(update)
			Ω(updatedLRP).Should(Equal(expectedLRP))
		})

		It("allows empty routes to be set", func() {
			update := DesiredLRPUpdate{
				Routes: []string{},
			}

			expectedLRP := lrp
			expectedLRP.Routes = []string{}

			updatedLRP := lrp.ApplyUpdate(update)
			Ω(updatedLRP).Should(Equal(expectedLRP))
		})

		It("allows annotation to be set", func() {
			annotation := "new-annotation"
			update := DesiredLRPUpdate{
				Annotation: &annotation,
			}

			expectedLRP := lrp
			expectedLRP.Annotation = annotation

			updatedLRP := lrp.ApplyUpdate(update)
			Ω(updatedLRP).Should(Equal(expectedLRP))
		})

		It("allows empty annotation to be set", func() {
			emptyAnnotation := ""
			update := DesiredLRPUpdate{
				Annotation: &emptyAnnotation,
			}

			expectedLRP := lrp
			expectedLRP.Annotation = emptyAnnotation

			updatedLRP := lrp.ApplyUpdate(update)
			Ω(updatedLRP).Should(Equal(expectedLRP))
		})

		It("updates routes", func() {
			update := DesiredLRPUpdate{
				Routes: []string{"new-route-1", "new-route-2"},
			}

			expectedLRP := lrp
			expectedLRP.Routes = []string{"new-route-1", "new-route-2"}

			updatedLRP := lrp.ApplyUpdate(update)
			Ω(updatedLRP).Should(Equal(expectedLRP))
		})
	})

	Describe("Validate", func() {
		var assertDesiredLRPValidationFailsWithMessage = func(lrp DesiredLRP, substring string) {
			validationErr := lrp.Validate()
			Ω(validationErr).Should(HaveOccurred())
			Ω(validationErr.Error()).Should(ContainSubstring(substring))
		}

		Context("process_guid only contains `A-Z`, `a-z`, `0-9`, `-`, and `_`", func() {
			validGuids := []string{"a", "A", "0", "-", "_", "-aaaa", "_-aaa", "09a87aaa-_aASKDn"}
			for _, validGuid := range validGuids {
				func(validGuid string) {
					It(fmt.Sprintf("'%s' is a valid process_guid", validGuid), func() {
						lrp.ProcessGuid = validGuid
						err := lrp.Validate()
						Ω(err).ShouldNot(HaveOccurred())
					})
				}(validGuid)
			}

			invalidGuids := []string{"", "bang!", "!!!", "\\slash", "star*", "params()", "invalid/key", "with.dots"}
			for _, invalidGuid := range invalidGuids {
				func(invalidGuid string) {
					It(fmt.Sprintf("'%s' is an invalid process_guid", invalidGuid), func() {
						lrp.ProcessGuid = invalidGuid
						assertDesiredLRPValidationFailsWithMessage(lrp, "process_guid")
					})
				}(invalidGuid)
			}
		})

		It("requires a positive nonzero number of instances", func() {
			lrp.Instances = -1
			assertDesiredLRPValidationFailsWithMessage(lrp, "instances")

			lrp.Instances = 0
			validationErr := lrp.Validate()
			Ω(validationErr).ShouldNot(HaveOccurred())

			lrp.Instances = 1
			validationErr = lrp.Validate()
			Ω(validationErr).ShouldNot(HaveOccurred())
		})

		It("requires a domain", func() {
			lrp.Domain = ""
			assertDesiredLRPValidationFailsWithMessage(lrp, "domain")
		})

		It("requires a stack", func() {
			lrp.Stack = ""
			assertDesiredLRPValidationFailsWithMessage(lrp, "stack")
		})

		It("requires an action", func() {
			lrp.Action = nil
			assertDesiredLRPValidationFailsWithMessage(lrp, "action")
		})

		It("requires a valid action", func() {
			lrp.Action = &UploadAction{
				From: "web_location",
			}
			assertDesiredLRPValidationFailsWithMessage(lrp, "to")
		})

		It("requires a valid setup action if specified", func() {
			lrp.Setup = &UploadAction{
				From: "web_location",
			}
			assertDesiredLRPValidationFailsWithMessage(lrp, "to")
		})

		It("requires a valid monitor action if specified", func() {
			lrp.Monitor = &UploadAction{
				From: "web_location",
			}
			assertDesiredLRPValidationFailsWithMessage(lrp, "to")
		})

		It("requires a valid CPU weight", func() {
			lrp.CPUWeight = 101
			assertDesiredLRPValidationFailsWithMessage(lrp, "cpu_weight")
		})

		Context("when security group is present", func() {
			It("must be valid", func() {
				lrp.SecurityGroupRules = []SecurityGroupRule{{
					Protocol: "foo",
				}}
				assertDesiredLRPValidationFailsWithMessage(lrp, "security_group_rules")
			})
		})

		Context("when security group is not present", func() {
			It("does not error", func() {
				lrp.SecurityGroupRules = []SecurityGroupRule{}

				validationErr := lrp.Validate()
				Ω(validationErr).ShouldNot(HaveOccurred())
			})
		})
	})

	Describe("ValidateModifications", func() {
		var newLrp DesiredLRP

		BeforeEach(func() {
			newLrp = lrp
		})

		It("does allow the instances to change", func() {
			newLrp.Instances = 5000
			err := lrp.ValidateModifications(newLrp)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("does allow the routes to change", func() {
			newLrp.Routes = []string{"my-new-route-1", "my-new-route-2"}
			err := lrp.ValidateModifications(newLrp)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("does allow the annotation to change", func() {
			newLrp.Annotation = "new-annotation"
			err := lrp.ValidateModifications(newLrp)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("does not allow the process-guid to change", func() {
			newLrp.ProcessGuid = "new-process-guid"
			err := lrp.ValidateModifications(newLrp)
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(ContainSubstring("process_guid"))
		})

		It("does not allow the domain to change", func() {
			newLrp.Domain = "new-domain"
			err := lrp.ValidateModifications(newLrp)
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(ContainSubstring("domain"))
		})

		It("does not allow the rootfs to change", func() {
			newLrp.RootFSPath = "new-rootfs"
			err := lrp.ValidateModifications(newLrp)
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(ContainSubstring("root_fs"))
		})

		It("does not allow the stack to change", func() {
			newLrp.Stack = "new-stack"
			err := lrp.ValidateModifications(newLrp)
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(ContainSubstring("stack"))
		})

		It("does not allow the env vars to change", func() {
			newLrp.EnvironmentVariables = []EnvironmentVariable{
				{
					Name:  "NEW_ENV_VAR_NAME",
					Value: "new environment variable value",
				},
			}
			err := lrp.ValidateModifications(newLrp)
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(ContainSubstring("env"))
		})

		It("does not allow the actions to change", func() {
			newLrp.Action = &UploadAction{
				To:   "new-destination",
				From: "new-source",
			}

			err := lrp.ValidateModifications(newLrp)
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(ContainSubstring("action"))
		})

		It("does not allow the disk size to change", func() {
			newLrp.DiskMB = 128
			err := lrp.ValidateModifications(newLrp)
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(ContainSubstring("disk_mb"))
		})

		It("does not allow the memory size to change", func() {
			newLrp.MemoryMB = 64
			err := lrp.ValidateModifications(newLrp)
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(ContainSubstring("memory_mb"))
		})

		It("does not allow the cpu weight to change", func() {
			newLrp.CPUWeight = 4
			err := lrp.ValidateModifications(newLrp)
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(ContainSubstring("cpu_weight"))
		})

		It("does not allow the ports to change", func() {
			newLrp.Ports = []uint32{6789}
			err := lrp.ValidateModifications(newLrp)
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(ContainSubstring("ports"))
		})

		It("does not allow the log guid to change", func() {
			newLrp.LogGuid = "new-guid"
			err := lrp.ValidateModifications(newLrp)
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(ContainSubstring("log"))
		})

		It("does not allow the log source to change", func() {
			newLrp.LogSource = "new-source"
			err := lrp.ValidateModifications(newLrp)
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(ContainSubstring("log"))
		})
	})

	Describe("Unmarshal", func() {
		It("returns a LRP with correct fields", func() {
			decodedLRP := DesiredLRP{}
			err := FromJSON([]byte(lrpPayload), &decodedLRP)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(decodedLRP).Should(Equal(lrp))
		})

		Context("with an invalid payload", func() {
			It("returns the error", func() {
				decodedLRP := DesiredLRP{}
				err := FromJSON([]byte("aliens lol"), &decodedLRP)
				Ω(err).Should(HaveOccurred())

				Ω(decodedLRP).Should(BeZero())
			})
		})

		Context("with a missing action", func() {
			It("returns the error", func() {
				decodedLRP := DesiredLRP{}
				err := FromJSON([]byte(`{
				"domain": "some-domain",
				"process_guid": "process_guid",
				"stack": "some-stack"
			}`,
				), &decodedLRP)
				Ω(err).Should(HaveOccurred())
			})
		})

		Context("with invalid actions", func() {
			var expectedLRP DesiredLRP
			var payload string

			BeforeEach(func() {
				expectedLRP = DesiredLRP{}
			})

			Context("with null actions", func() {
				BeforeEach(func() {
					payload = `{
					"setup": null,
					"action": null,
					"monitor": null
				}`
				})

				It("unmarshals", func() {
					var actualLRP DesiredLRP
					err := json.Unmarshal([]byte(payload), &actualLRP)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(actualLRP).Should(Equal(expectedLRP))
				})
			})

			Context("with missing action", func() {
				BeforeEach(func() {
					payload = `{}`
				})

				It("unmarshals", func() {
					var actualLRP DesiredLRP
					err := json.Unmarshal([]byte(payload), &actualLRP)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(actualLRP).Should(Equal(expectedLRP))
				})
			})
		})

		for field, payload := range map[string]string{
			"process_guid": `{
				"domain": "some-domain",
				"stack": "some-stack",
				"action":
					{"download":{"from":"http://example.com","to":"/tmp/internet","cache_key":""}}
			}`,
			"stack": `{
				"domain": "some-domain",
				"process_guid": "process_guid",
				"action":
					{"download":{"from":"http://example.com","to":"/tmp/internet","cache_key":""}}
			}`,
			"domain": `{
				"stack": "some-stack",
				"process_guid": "process_guid",
				"action":
					{"download":{"from":"http://example.com","to":"/tmp/internet","cache_key":""}}
			}`,
			"annotation": `{
				"stack": "some-stack",
				"domain": "some-domain",
				"process_guid": "process_guid",
				"instances": 1,
				"action": {
					"download":{"from":"http://example.com","to":"/tmp/internet","cache_key":""}
				},
				"annotation":"` + strings.Repeat("a", 10*1024+1) + `"
			}`,
		} {
			missingField := field
			jsonBytes := payload

			Context("when the json is missing a "+missingField, func() {
				It("returns an error indicating so", func() {
					decodedLRP := &DesiredLRP{}

					err := FromJSON([]byte(jsonBytes), decodedLRP)
					Ω(err).Should(HaveOccurred())
					Ω(err.Error()).Should(ContainSubstring(missingField))
				})
			})
		}
	})
})
