package models_test

import (
	"fmt"

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
	  "env":[
	    {
	      "name": "ENV_VAR_NAME",
	      "value": "some environment variable value"
	    }
	  ],
	  "actions": [
	    {
	      "action": "download",
	      "args": {
	        "from": "http://example.com",
	        "to": "/tmp/internet",
	        "cache_key": ""
	      }
	    }
	  ],
	  "disk_mb": 512,
	  "memory_mb": 1024,
	  "cpu_weight": 42,
	  "ports": [
	    {
	      "container_port": 5678,
	      "host_port": 1234
	    }
	  ],
	  "routes": [
	    "route-1",
	    "route-2"
	  ],
	  "log_guid": "log-guid",
	  "log_source": "the cloud"
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
			Routes:     []string{"route-1", "route-2"},
			Annotation: "some-annotation",
			Ports: []PortMapping{
				{HostPort: 1234, ContainerPort: 5678},
			},
			LogGuid:   "log-guid",
			LogSource: "the cloud",
			EnvironmentVariables: []EnvironmentVariable{
				{
					Name:  "ENV_VAR_NAME",
					Value: "some environment variable value",
				},
			},
			Actions: []ExecutorAction{
				{
					Action: DownloadAction{
						From: "http://example.com",
						To:   "/tmp/internet",
					},
				},
			},
		}
	})

	Describe("ToJSON", func() {
		It("should JSONify", func() {
			json := lrp.ToJSON()
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

		It("requires a positive number of instances", func() {
			lrp.Instances = 0
			assertDesiredLRPValidationFailsWithMessage(lrp, "instances")
		})

		It("requires a domain", func() {
			lrp.Domain = ""
			assertDesiredLRPValidationFailsWithMessage(lrp, "domain")
		})

		It("requires a stack", func() {
			lrp.Stack = ""
			assertDesiredLRPValidationFailsWithMessage(lrp, "stack")
		})

		It("requires actions", func() {
			lrp.Actions = nil
			assertDesiredLRPValidationFailsWithMessage(lrp, "actions")
		})

		It("requires a valid CPU weight", func() {
			lrp.CPUWeight = 101
			assertDesiredLRPValidationFailsWithMessage(lrp, "cpu_weight")
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
			newLrp.Actions = []ExecutorAction{
				{UploadAction{
					To:   "new-destination",
					From: "new-source",
				}},
			}
			err := lrp.ValidateModifications(newLrp)
			Ω(err).Should(HaveOccurred())
			Ω(err.Error()).Should(ContainSubstring("actions"))
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
			newLrp.Ports = []PortMapping{
				{HostPort: 2345, ContainerPort: 6789},
			}
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

	Describe("NewDesiredLRPFromJSON", func() {
		It("returns a LRP with correct fields", func() {
			decodedStartAuction, err := NewDesiredLRPFromJSON([]byte(lrpPayload))
			Ω(err).ShouldNot(HaveOccurred())

			Ω(decodedStartAuction).Should(Equal(lrp))
		})

		Context("with an invalid payload", func() {
			It("returns the error", func() {
				decodedStartAuction, err := NewDesiredLRPFromJSON([]byte("aliens lol"))
				Ω(err).Should(HaveOccurred())

				Ω(decodedStartAuction).Should(BeZero())
			})
		})

		for field, payload := range map[string]string{
			"process_guid": `{
				"domain": "some-domain",
				"actions": [
					{"action":"download","args":{"from":"http://example.com","to":"/tmp/internet","cache_key":""}}
				],
				"stack": "some-stack"
			}`,
			"actions": `{
				"domain": "some-domain",
				"process_guid": "process_guid",
				"stack": "some-stack"
			}`,
			"stack": `{
				"domain": "some-domain",
				"process_guid": "process_guid",
				"actions": [
					{"action":"download","args":{"from":"http://example.com","to":"/tmp/internet","cache_key":""}}
				]
			}`,
			"domain": `{
				"stack": "some-stack",
				"process_guid": "process_guid",
				"actions": [
					{"action":"download","args":{"from":"http://example.com","to":"/tmp/internet","cache_key":""}}
				]
			}`,
		} {
			json := payload
			missingField := field

			Context("when the json is missing a "+missingField, func() {
				It("returns an error indicating so", func() {
					decodedStartAuction, err := NewDesiredLRPFromJSON([]byte(json))
					Ω(err).Should(HaveOccurred())
					Ω(err.Error()).Should(Equal("JSON has missing/invalid field: " + missingField))

					Ω(decodedStartAuction).Should(BeZero())
				})
			})
		}
	})
})
