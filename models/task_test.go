package models_test

import (
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry-incubator/runtime-schema/models"
)

var _ = Describe("Task", func() {
	var taskPayload string
	var task Task

	BeforeEach(func() {
		taskPayload = `{
		"task_guid":"some-guid",
		"domain":"some-domain",
		"root_fs": "docker:///docker.com/docker",
		"stack":"some-stack",
		"env":[
			{
				"name":"ENV_VAR_NAME",
				"value":"an environmment value"
			}
		],
		"cell_id":"cell",
		"action": {
			"download":{
				"from":"old_location",
				"to":"new_location",
				"cache_key":"the-cache-key"
			}
		},
		"result_file":"some-file.txt",
		"result": "turboencabulated",
		"failed":true,
		"failure_reason":"because i said so",
		"memory_mb":256,
		"disk_mb":1024,
		"cpu_weight": 42,
		"log_guid": "123",
		"log_source": "APP",
		"created_at": 1393371971000000000,
		"updated_at": 1393371971000000010,
		"first_completed_at": 1393371971000000030,
		"state": 1,
		"annotation": "[{\"anything\": \"you want!\"}]... dude"
	}`

		task = Task{
			TaskGuid:   "some-guid",
			Domain:     "some-domain",
			RootFSPath: "docker:///docker.com/docker",
			Stack:      "some-stack",
			EnvironmentVariables: []EnvironmentVariable{
				{
					Name:  "ENV_VAR_NAME",
					Value: "an environmment value",
				},
			},
			Action: &DownloadAction{
				From:     "old_location",
				To:       "new_location",
				CacheKey: "the-cache-key",
			},
			MemoryMB:         256,
			DiskMB:           1024,
			CPUWeight:        42,
			LogSource:        "APP",
			LogGuid:          "123",
			CreatedAt:        time.Date(2014, time.February, 25, 23, 46, 11, 00, time.UTC).UnixNano(),
			UpdatedAt:        time.Date(2014, time.February, 25, 23, 46, 11, 10, time.UTC).UnixNano(),
			FirstCompletedAt: time.Date(2014, time.February, 25, 23, 46, 11, 30, time.UTC).UnixNano(),
			ResultFile:       "some-file.txt",
			State:            TaskStatePending,
			CellID:           "cell",

			Result:        "turboencabulated",
			Failed:        true,
			FailureReason: "because i said so",

			Annotation: `[{"anything": "you want!"}]... dude`,
		}
	})

	Describe("Validate", func() {
		Context("when the task has a domain, valid guid, stack, and valid action", func() {
			It("is valid", func() {
				task = Task{
					Domain:   "some-domain",
					TaskGuid: "some-task-guid",
					Stack:    "some-stack",
					Action: &RunAction{
						Path: "ls",
					},
				}

				err := task.Validate()
				Ω(err).ShouldNot(HaveOccurred())
			})
		})

		Context("when the task GUID is present but invalid", func() {
			It("returns an error indicating so", func() {
				task = Task{
					Domain:   "some-domain",
					TaskGuid: "invalid/guid",
					Stack:    "some-stack",
					Action: &RunAction{
						Path: "ls",
					},
				}

				err := task.Validate()
				Ω(err).Should(HaveOccurred())
				Ω(err.Error()).Should(ContainSubstring("task_guid"))
			})
		})

		for field, task := range map[string]Task{
			"task_guid": Task{
				Domain: "some-domain",
				Stack:  "some-stack",
				Action: &RunAction{
					Path: "ls",
				},
			},
			"stack": Task{
				Domain:   "some-domain",
				TaskGuid: "some-stack",
				Action: &RunAction{
					Path: "ls",
				},
			},
			"domain": Task{
				TaskGuid: "some-stack",
				Stack:    "some-stack",
				Action: &RunAction{
					Path: "ls",
				},
			},
			"action": Task{
				Domain:   "some-domain",
				TaskGuid: "some-stack",
				Stack:    "some-stack",
			},
			"path": Task{
				Domain:   "some-domain",
				TaskGuid: "some-stack",
				Stack:    "some-stack",
				Action:   &RunAction{},
			},
			"annotation": Task{
				Domain:   "some-domain",
				TaskGuid: "some-stack",
				Stack:    "some-stack",
				Action: &RunAction{
					Path: "ls",
				},
				Annotation: strings.Repeat("a", 10*1024+1),
			},
			"cpu_weight": Task{
				Domain:   "some-domain",
				TaskGuid: "some-stack",
				Stack:    "some-stack",
				Action: &RunAction{
					Path: "ls",
				},
				CPUWeight: 101,
			},
		} {
			missingField := field
			invalidTask := task

			Context("when the field "+missingField+" is invalid", func() {
				It("returns an error indicating so", func() {
					err := invalidTask.Validate()
					Ω(err).Should(HaveOccurred())
					Ω(err.Error()).Should(ContainSubstring(missingField))
				})
			})
		}
	})

	Describe("Marshal", func() {
		It("should JSONify", func() {
			json, err := ToJSON(&task)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(string(json)).Should(MatchJSON(taskPayload))
		})
	})

	Describe("Unmarshal", func() {
		It("returns a Task with correct fields", func() {
			decodedTask := &Task{}
			err := FromJSON([]byte(taskPayload), decodedTask)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(decodedTask).Should(Equal(&task))
		})

		Context("with an invalid payload", func() {
			It("returns the error", func() {
				decodedTask := &Task{}
				err := FromJSON([]byte("aliens lol"), decodedTask)
				Ω(err).Should(HaveOccurred())
			})
		})
	})
})
