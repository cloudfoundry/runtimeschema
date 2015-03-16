package cc_messages_test

import (
	"encoding/json"

	"github.com/cloudfoundry-incubator/runtime-schema/cc_messages"
	"github.com/cloudfoundry-incubator/runtime-schema/diego_errors"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("StagingMessages", func() {
	Describe("StagingRequestFromCC", func() {
		ccJSON := `{
           "app_id" : "fake-app_id",
           "task_id" : "fake-task_id",
           "memory_mb" : 1024,
           "disk_mb" : 10000,
           "file_descriptors" : 3,
           "environment" : [{"name": "FOO", "value":"BAR"}],
           "stack" : "fake-stack",
           "timeout" : 900,
           "lifecycle": "buildpack",
           "lifecycle_data": {"foo": "bar"}
        }`

		It("should be mapped to the CC's staging request JSON", func() {
			var stagingRequest cc_messages.StagingRequestFromCC
			err := json.Unmarshal([]byte(ccJSON), &stagingRequest)
			Ω(err).ShouldNot(HaveOccurred())

			lifecycle_data := json.RawMessage([]byte(`{"foo": "bar"}`))
			Ω(stagingRequest).Should(Equal(cc_messages.StagingRequestFromCC{
				AppId:           "fake-app_id",
				TaskId:          "fake-task_id",
				MemoryMB:        1024,
				DiskMB:          10000,
				FileDescriptors: 3,
				Environment: cc_messages.Environment{
					{Name: "FOO", Value: "BAR"},
				},
				Stack:         "fake-stack",
				Timeout:       900,
				Lifecycle:     "buildpack",
				LifecycleData: &lifecycle_data,
			}))
		})
	})

	Describe("BuildpackLifecycleData", func() {
		lifecycleDataJSON := `{
				"app_bits_download_uri" : "http://fake-download_uri",
				"build_artifacts_cache_download_uri" : "http://a-nice-place-to-get-valuable-artifacts.com",
				"build_artifacts_cache_upload_uri" : "http://a-nice-place-to-upload-valuable-artifacts.com",
				"buildpacks" : [{"name":"fake-buildpack-name", "key":"fake-buildpack-key" ,"url":"fake-buildpack-url", "skip_detect":true}],
				"droplet_upload_uri" : "http://droplet-upload-uri"
			}`

		It("unmarshals correctly", func() {
			var lifecycleData cc_messages.BuildpackStagingData
			err := json.Unmarshal([]byte(lifecycleDataJSON), &lifecycleData)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(lifecycleData).Should(Equal(cc_messages.BuildpackStagingData{
				AppBitsDownloadUri:             "http://fake-download_uri",
				BuildArtifactsCacheDownloadUri: "http://a-nice-place-to-get-valuable-artifacts.com",
				BuildArtifactsCacheUploadUri:   "http://a-nice-place-to-upload-valuable-artifacts.com",
				Buildpacks: []cc_messages.Buildpack{
					{
						Name:       "fake-buildpack-name",
						Key:        "fake-buildpack-key",
						Url:        "fake-buildpack-url",
						SkipDetect: true,
					},
				},
				DropletUploadUri: "http://droplet-upload-uri",
			}))
		})
	})

	Describe("DockerStagingData", func() {
		lifecycleDataJSON := `{
      "docker_image" : "docker:///diego/image"
    }`

		It("should be mapped to the CC's staging request JSON", func() {
			var stagingData cc_messages.DockerStagingData
			err := json.Unmarshal([]byte(lifecycleDataJSON), &stagingData)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(stagingData).Should(Equal(cc_messages.DockerStagingData{
				DockerImageUrl: "docker:///diego/image",
			}))
		})
	})

	Describe("Environment", func() {
		It("translates into a []model.Environment", func() {
			env := cc_messages.Environment{
				{Name: "FOO", Value: "BAR"},
			}
			bbsEnv := env.BBSEnvironment()
			Ω(bbsEnv).Should(Equal([]models.EnvironmentVariable{{Name: "FOO", Value: "BAR"}}))
		})
	})

	Describe("Buildpack", func() {
		Context("when skipping the detect phase is not specified", func() {
			ccJSONFragment := `{
       "name": "ocaml-buildpack",
       "key": "ocaml-buildpack-guid",
       "url": "http://ocaml.org/buildpack.zip"
      }`

			It("extracts the name, key, and url values", func() {
				var buildpack cc_messages.Buildpack

				err := json.Unmarshal([]byte(ccJSONFragment), &buildpack)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(buildpack).To(Equal(cc_messages.Buildpack{
					Name: "ocaml-buildpack",
					Key:  "ocaml-buildpack-guid",
					Url:  "http://ocaml.org/buildpack.zip",
				}))
			})
		})

		Context("when skipping the detect phase is specified", func() {
			ccJSONFragment := `{
        "name": "ocaml-buildpack",
        "key": "ocaml-buildpack-guid",
        "url": "http://ocaml.org/buildpack.zip",
        "skip_detect": true
      }`

			It("extracts the name, key, url, and skip_detect values", func() {
				var buildpack cc_messages.Buildpack

				err := json.Unmarshal([]byte(ccJSONFragment), &buildpack)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(buildpack).To(Equal(cc_messages.Buildpack{
					Name:       "ocaml-buildpack",
					Key:        "ocaml-buildpack-guid",
					Url:        "http://ocaml.org/buildpack.zip",
					SkipDetect: true,
				}))
			})
		})
	})

	Describe("StagingResponseForCC", func() {
		var stagingResponseForCC cc_messages.StagingResponseForCC

		BeforeEach(func() {
			stagingResponseForCC = cc_messages.StagingResponseForCC{
				AppId:                "the-app-id",
				TaskId:               "the-task-id",
				ExecutionMetadata:    "the-execution-metadata",
				DetectedStartCommand: map[string]string{"web": "the-detected-start-command"},
			}
		})

		Context("without lifecycle data", func() {
			It("generates valid json without the lifecycle data", func() {
				Ω(json.Marshal(stagingResponseForCC)).Should(MatchJSON(`{
					"app_id": "the-app-id",
					"execution_metadata": "the-execution-metadata",
					"detected_start_command":{"web":"the-detected-start-command"},
					"task_id": "the-task-id"
				}`))
			})
		})

		Context("with lifecycle data", func() {
			BeforeEach(func() {
				lifecycleData := json.RawMessage(`{"foo": "bar"}`)
				stagingResponseForCC = cc_messages.StagingResponseForCC{
					AppId:                "the-app-id",
					TaskId:               "the-task-id",
					ExecutionMetadata:    "the-execution-metadata",
					DetectedStartCommand: map[string]string{"web": "the-detected-start-command"},
					LifecycleData:        &lifecycleData,
				}
			})

			It("generates valid json with lifecycle data", func() {
				Ω(json.Marshal(stagingResponseForCC)).Should(MatchJSON(`{
					"app_id": "the-app-id",
					"task_id": "the-task-id",
					"execution_metadata": "the-execution-metadata",
					"detected_start_command":{"web":"the-detected-start-command"},
					"lifecycle_data": {"foo": "bar"}
				}`))
			})
		})

		Context("without an error", func() {
			It("generates valid JSON", func() {
				Ω(json.Marshal(stagingResponseForCC)).Should(MatchJSON(`{
					"app_id": "the-app-id",
					"task_id": "the-task-id",
					"execution_metadata": "the-execution-metadata",
					"detected_start_command":{"web":"the-detected-start-command"}
				}`))
			})
		})

		Context("with an error", func() {
			It("generates valid JSON with the error", func() {
				stagingResponseForCC.Error = &cc_messages.StagingError{
					Id:      "StagingError",
					Message: "FAIL, missing camels!",
				}
				Ω(json.Marshal(stagingResponseForCC)).Should(MatchJSON(`{
					"error": { "id": "StagingError", "message": "FAIL, missing camels!" },

					"app_id": "the-app-id",
					"execution_metadata": "the-execution-metadata",
					"detected_start_command":{"web":"the-detected-start-command"},
					"task_id": "the-task-id"
				}`))
			})
		})
	})

	Describe("BuildpackStagingResponse", func() {
		var buildpackStagingResponse cc_messages.BuildpackStagingResponse

		BeforeEach(func() {
			buildpackStagingResponse = cc_messages.BuildpackStagingResponse{
				BuildpackKey:      "buildpack-key",
				DetectedBuildpack: "detected-buildpack",
			}
		})

		It("marshals correctly", func() {
			Ω(json.Marshal(buildpackStagingResponse)).Should(MatchJSON(`{
				"buildpack_key": "buildpack-key",
				"detected_buildpack": "detected-buildpack"
			}`))
		})

		It("marshals correctly", func() {
			response := cc_messages.BuildpackStagingResponse{}
			err := json.Unmarshal([]byte(`{
				"buildpack_key": "buildpack-key",
				"detected_buildpack": "detected-buildpack"
			}`), &response)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(response).Should(Equal(buildpackStagingResponse))
		})
	})

	Describe("StopStagingRequestForCC", func() {
		var stopStagingRequestFromCC cc_messages.StopStagingRequestFromCC

		BeforeEach(func() {
			stopStagingRequestFromCC = cc_messages.StopStagingRequestFromCC{
				AppId:  "the-app-id",
				TaskId: "the-task-id",
			}
		})

		It("unmarshals JSON", func() {
			stopStagingRequest := cc_messages.StopStagingRequestFromCC{}

			err := json.Unmarshal([]byte(`{
					"app_id": "the-app-id",
					"task_id": "the-task-id"
				}`), &stopStagingRequest)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(stopStagingRequest).Should(Equal(stopStagingRequestFromCC))
		})
	})

	Describe("SanitizeErrorMessage", func() {
		Context("when the message is InsufficientResources", func() {
			It("returns a InsufficientResources", func() {
				stagingErr := cc_messages.SanitizeErrorMessage(diego_errors.INSUFFICIENT_RESOURCES_MESSAGE)
				Ω(stagingErr.Id).Should(Equal(cc_messages.INSUFFICIENT_RESOURCES))
				Ω(stagingErr.Message).Should(Equal(diego_errors.INSUFFICIENT_RESOURCES_MESSAGE))
			})
		})

		Context("when the message is NoCompatibleCell", func() {
			It("returns a NoCompatibleCell", func() {
				stagingErr := cc_messages.SanitizeErrorMessage(diego_errors.CELL_MISMATCH_MESSAGE)
				Ω(stagingErr.Id).Should(Equal(cc_messages.NO_COMPATIBLE_CELL))
				Ω(stagingErr.Message).Should(Equal(diego_errors.CELL_MISMATCH_MESSAGE))
			})
		})

		Context("any other message", func() {
			It("returns a StagingError", func() {
				stagingErr := cc_messages.SanitizeErrorMessage("some-error")
				Ω(stagingErr.Id).Should(Equal(cc_messages.STAGING_ERROR))
				Ω(stagingErr.Message).Should(Equal("staging failed"))
			})
		})
	})
})
