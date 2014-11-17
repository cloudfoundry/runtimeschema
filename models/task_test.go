package models_test

import (
	"encoding/json"
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
				err := json.Unmarshal([]byte("aliens lol"), decodedTask)
				Ω(err).Should(HaveOccurred())
			})
		})

		Context("with a missing action", func() {
			It("returns the error", func() {
				taskPayload = `{
					"domain": "some-domain", "task_guid": "process-guid",
					"stack": "some-stack"}`
				decodedTask := &Task{}

				err := FromJSON([]byte(taskPayload), decodedTask)
				Ω(err).Should(HaveOccurred())
			})
		})

		for field, payload := range map[string]string{
			"task_guid": `{"domain": "some-domain", "stack": "some-stack", "action": {"run": {"path": "date"}}}`,
			"stack":     `{"domain": "some-domain", "task_guid": "process-guid", "action": {"run": {"path": "date"}}}`,
			"domain":    `{"stack": "some-stack", "task_guid": "process-guid", "action": {"run": {"path": "date"}}}`,
			"annotation": `{"domain": "some-domain", "stack": "some-stack", "task_guid": "process-guid", "instances": 1, "action": {"run": {"path": "date"}},
										"annotation":"` + strings.Repeat("a", 10*1024+1) + `"}`,
		} {
			missingField := field
			jsonPayload := payload

			Context("when the json is missing a "+missingField, func() {
				It("returns an error indicating so", func() {
					decodedTask := &Task{}
					err := FromJSON([]byte(jsonPayload), decodedTask)
					Ω(err).Should(HaveOccurred())
					Ω(err.Error()).Should(ContainSubstring(missingField))
				})
			})
		}

		Context("when the task GUID is present but invalid", func() {
			payload := `{"domain": "some-domain", "task_guid": "invalid/guid", "stack": "some-stack", "action": {"run": {"path": "date"}}}`

			It("returns an error indicating so", func() {
				decodedTask := &Task{}
				err := FromJSON([]byte(payload), decodedTask)
				Ω(err).Should(HaveOccurred())
				Ω(err.Error()).Should(ContainSubstring("task_guid"))
			})
		})

		Context("with an invalid CPU weight", func() {
			payload := `{"domain": "some-domain", "task_guid": "guid", "cpu_weight": 101, "stack": "some-stack", "action": {"run": {"path": "date"}}}`

			It("returns an error", func() {
				decodedTask := &Task{}
				err := FromJSON([]byte(payload), decodedTask)
				Ω(err).Should(HaveOccurred())
				Ω(err.Error()).Should(ContainSubstring("cpu_weight"))
			})
		})
	})
})
