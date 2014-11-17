package models_test

import (
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry-incubator/runtime-schema/models"
)

var _ = Describe("Actions", func() {
	itSerializesAndDeserializes := func(actionPayload string, action Action) {
		It("Action <-> JSON for "+string(action.ActionType()), func() {
			By("marshalling to JSON", func() {
				marshalledAction := action

				json, err := json.Marshal(&marshalledAction)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(json).Should(MatchJSON(actionPayload))
			})

			wrappedJSON := fmt.Sprintf(`{"%s":%s}`, action.ActionType(), actionPayload)
			By("wrapping", func() {
				marshalledAction := action

				json, err := MarshalAction(marshalledAction)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(json).Should(MatchJSON(wrappedJSON))
			})

			By("unwrapping", func() {
				var unmarshalledAction Action
				unmarshalledAction, err := UnmarshalAction([]byte(wrappedJSON))
				Ω(err).ShouldNot(HaveOccurred())
				Ω(unmarshalledAction).Should(Equal(action))
			})
		})
	}

	Describe("Download", func() {
		itSerializesAndDeserializes(
			`{
					"from": "web_location",
					"to": "local_location",
					"cache_key": "elephant"
			}`,
			&DownloadAction{
				From:     "web_location",
				To:       "local_location",
				CacheKey: "elephant",
			},
		)
	})

	Describe("Upload", func() {
		itSerializesAndDeserializes(
			`{
					"from": "local_location",
					"to": "web_location"
			}`,
			&UploadAction{
				From: "local_location",
				To:   "web_location",
			},
		)
	})

	Describe("Run", func() {
		itSerializesAndDeserializes(
			`{
					"path": "rm",
					"args": ["-rf", "/"],
					"env": [
						{"name":"FOO", "value":"1"},
						{"name":"BAR", "value":"2"}
					],
					"resource_limits":{},
					"privileged": true
			}`,
			&RunAction{
				Path: "rm",
				Args: []string{"-rf", "/"},
				Env: []EnvironmentVariable{
					{"FOO", "1"},
					{"BAR", "2"},
				},
				Privileged: true,
			},
		)
	})

	Describe("EmitProgressAction", func() {
		itSerializesAndDeserializes(
			`{
					"start_message": "reticulating splines",
					"success_message": "reticulated splines",
					"failure_message": "reticulation failed",
					"action": {
						"run": {
							"path": "echo",
							"args": null,
							"env": null,
							"resource_limits":{}
						}
					}
			}`,
			EmitProgressFor(
				&RunAction{
					Path: "echo",
				},
				"reticulating splines", "reticulated splines", "reticulation failed",
			),
		)
	})

	Describe("Timeout", func() {
		itSerializesAndDeserializes(
			`{
				"action": {
					"run": {
						"path": "echo",
						"args": null,
						"env": null,
						"resource_limits":{}
					}
				},
				"timeout": 10000000
			}`,
			Timeout(
				&RunAction{
					Path: "echo",
				},
				10*time.Millisecond,
			),
		)
	})

	Describe("Try", func() {
		itSerializesAndDeserializes(
			`{
					"action": {
						"run": {
							"path": "echo",
							"args": null,
							"env": null,
							"resource_limits":{}
						}
					}
			}`,
			Try(&RunAction{Path: "echo"}),
		)
	})

	Describe("Parallel", func() {
		itSerializesAndDeserializes(
			`{
					"actions": [
						{
							"download": {
								"cache_key": "elephant",
								"to": "local_location",
								"from": "web_location"
							}
						},
						{
							"run": {
								"resource_limits": {},
								"env": null,
								"path": "echo",
								"args": null
							}
						}
					]
			}`,
			Parallel(
				&DownloadAction{
					From:     "web_location",
					To:       "local_location",
					CacheKey: "elephant",
				},
				&RunAction{Path: "echo"},
			),
		)
	})

	Describe("Serial", func() {
		itSerializesAndDeserializes(
			`{
					"actions": [
						{
							"download": {
								"cache_key": "elephant",
								"to": "local_location",
								"from": "web_location"
							}
						},
						{
							"run": {
								"resource_limits": {},
								"env": null,
								"path": "echo",
								"args": null
							}
						}
					]
			}`,
			Serial(
				&DownloadAction{
					From:     "web_location",
					To:       "local_location",
					CacheKey: "elephant",
				},
				&RunAction{Path: "echo"},
			),
		)
	})
})
