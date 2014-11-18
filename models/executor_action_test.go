package models_test

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry-incubator/runtime-schema/models"
)

var _ = Describe("ExecutorAction", func() {
	Describe("Validate", func() {
		Context("With an invalid action", func() {
			It("should fail to marshal", func() {
				invalidAction := []string{"aliens", "from", "mars"}
				executorAction := ExecutorAction{Action: invalidAction}
				err := executorAction.Validate()
				Ω(err).Should(Equal(ErrInvalidActionType))
			})
		})
	})

	itSerializesAndDeserializes := func(actionPayload string, action interface{}) {
		Describe("Converting to JSON", func() {
			It("creates a json representation of the object", func() {
				marshalledAction := action

				json, err := json.Marshal(&marshalledAction)
				Ω(err).Should(BeNil())
				Ω(json).Should(MatchJSON(actionPayload))
			})
		})

		Describe("Converting from JSON", func() {
			It("constructs an object from the json string", func() {
				var unmarshalledAction *ExecutorAction
				err := json.Unmarshal([]byte(actionPayload), &unmarshalledAction)
				Ω(err).Should(BeNil())
				Ω(*unmarshalledAction).Should(Equal(action))
			})
		})
	}

	Describe("Download", func() {
		itSerializesAndDeserializes(
			`{
				"action": "download",
				"args": {
					"from": "web_location",
					"to": "local_location",
					"cache_key": "elephant"
				}
			}`,
			ExecutorAction{
				Action: DownloadAction{
					From:     "web_location",
					To:       "local_location",
					CacheKey: "elephant",
				},
			},
		)
	})

	Describe("Upload", func() {
		itSerializesAndDeserializes(
			`{
				"action": "upload",
				"args": {
					"from": "local_location",
					"to": "web_location"
				}
			}`,
			ExecutorAction{
				Action: UploadAction{
					From: "local_location",
					To:   "web_location",
				},
			},
		)
	})

	Describe("Run", func() {
		itSerializesAndDeserializes(
			`{
				"action": "run",
				"args": {
					"path": "rm",
					"args": ["-rf", "/"],
					"env": [
						{"name":"FOO", "value":"1"},
						{"name":"BAR", "value":"2"}
					],
					"resource_limits":{},
					"privileged": true
				}
			}`,
			ExecutorAction{
				Action: RunAction{
					Path: "rm",
					Args: []string{"-rf", "/"},
					Env: []EnvironmentVariable{
						{"FOO", "1"},
						{"BAR", "2"},
					},
					Privileged: true,
				},
			},
		)
	})

	Describe("EmitProgressAction", func() {
		itSerializesAndDeserializes(
			`{
				"action": "emit_progress",
				"args": {
					"start_message": "reticulating splines",
					"success_message": "reticulated splines",
					"failure_message": "reticulation failed",
					"action": {
						"action": "run",
						"args": {
							"path": "echo",
							"args": null,
							"env": null,
							"resource_limits":{}
						}
					}
				}
			}`,
			EmitProgressFor(
				ExecutorAction{
					RunAction{
						Path: "echo",
					},
				}, "reticulating splines", "reticulated splines", "reticulation failed"),
		)
	})

	Describe("Timeout", func() {
		itSerializesAndDeserializes(
			`{
				"action": "timeout",
				"args": {
					"action": {
						"action": "run",
						"args": {
							"path": "echo",
							"args": null,
							"env": null,
							"resource_limits":{}
						}
					},
					"timeout": 10000000
				}
			}`,
			Timeout(
				ExecutorAction{
					RunAction{Path: "echo"},
				},
				10*time.Millisecond,
			),
		)
	})

	Describe("Try", func() {
		itSerializesAndDeserializes(
			`{
				"action": "try",
				"args": {
					"action": {
						"action": "run",
						"args": {
							"path": "echo",
							"args": null,
							"env": null,
							"resource_limits":{}
						}
					}
				}
			}`,
			Try(ExecutorAction{
				RunAction{Path: "echo"},
			}),
		)
	})

	Describe("Parallel", func() {
		itSerializesAndDeserializes(
			`{
				"action": "parallel",
				"args": {
					"actions": [
						{
							"action": "download",
							"args": {
								"cache_key": "elephant",
								"to": "local_location",
								"from": "web_location"
							}
						},
						{
							"action": "run",
							"args": {
								"resource_limits": {},
								"env": null,
								"path": "echo",
								"args": null
							}
						}
					]
				}
			}`,
			Parallel(
				ExecutorAction{
					DownloadAction{
						From:     "web_location",
						To:       "local_location",
						CacheKey: "elephant",
					},
				},
				ExecutorAction{
					RunAction{Path: "echo"},
				},
			),
		)
	})

	Describe("Serial", func() {
		itSerializesAndDeserializes(
			`{
				"action": "serial",
				"args": {
					"actions": [
						{
							"action": "download",
							"args": {
								"cache_key": "elephant",
								"to": "local_location",
								"from": "web_location"
							}
						},
						{
							"action": "run",
							"args": {
								"resource_limits": {},
								"env": null,
								"path": "echo",
								"args": null
							}
						}
					]
				}
			}`,
			Serial(
				ExecutorAction{
					DownloadAction{
						From:     "web_location",
						To:       "local_location",
						CacheKey: "elephant",
					},
				},
				ExecutorAction{
					RunAction{Path: "echo"},
				},
			),
		)
	})
})
