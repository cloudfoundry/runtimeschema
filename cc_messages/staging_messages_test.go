package cc_messages_test

import (
	"encoding/json"

	. "github.com/cloudfoundry-incubator/runtime-schema/cc_messages"
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
           "app_bits_download_uri" : "http://fake-download_uri",
           "build_artifacts_cache_download_uri" : "http://a-nice-place-to-get-valuable-artifacts.com",
           "build_artifacts_cache_upload_uri" : "http://a-nice-place-to-upload-valuable-artifacts.com",
           "buildpacks" : [{"name":"fake-buildpack-name", "key":"fake-buildpack-key" ,"url":"fake-buildpack-url"}],
           "droplet_upload_uri" : "http://droplet-upload-uri",
           "timeout" : 900
        }`

		It("should be mapped to the CC's staging request JSON", func() {
			var stagingRequest StagingRequestFromCC
			err := json.Unmarshal([]byte(ccJSON), &stagingRequest)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(stagingRequest).Should(Equal(StagingRequestFromCC{
				AppId:                          "fake-app_id",
				TaskId:                         "fake-task_id",
				Stack:                          "fake-stack",
				AppBitsDownloadUri:             "http://fake-download_uri",
				BuildArtifactsCacheDownloadUri: "http://a-nice-place-to-get-valuable-artifacts.com",
				BuildArtifactsCacheUploadUri:   "http://a-nice-place-to-upload-valuable-artifacts.com",
				MemoryMB:                       1024,
				FileDescriptors:                3,
				DiskMB:                         10000,
				Buildpacks: []Buildpack{
					{
						Name: "fake-buildpack-name",
						Key:  "fake-buildpack-key",
						Url:  "fake-buildpack-url",
					},
				},
				Environment: Environment{
					{Name: "FOO", Value: "BAR"},
				},
				DropletUploadUri: "http://droplet-upload-uri",
				Timeout:          900,
			}))
		})
	})

	Describe("DockerStagingRequestFromCC", func() {
		ccJSON := `{
           "app_id" : "fake-app_id",
           "task_id" : "fake-task_id",
           "docker_image" : "docker:///diego/image",
           "memory_mb" : 1024,
           "disk_mb" : 10000,
           "file_descriptors" : 3,
           "environment" : [{"name": "FOO", "value":"BAR"}],
           "stack" : "fake-stack",
           "timeout" : 900
        }`

		It("should be mapped to the CC's staging request JSON", func() {
			var stagingRequest DockerStagingRequestFromCC
			err := json.Unmarshal([]byte(ccJSON), &stagingRequest)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(stagingRequest).Should(Equal(DockerStagingRequestFromCC{
				AppId:           "fake-app_id",
				TaskId:          "fake-task_id",
				DockerImageUrl:  "docker:///diego/image",
				Stack:           "fake-stack",
				MemoryMB:        1024,
				FileDescriptors: 3,
				DiskMB:          10000,
				Environment: Environment{
					{Name: "FOO", Value: "BAR"},
				},
				Timeout: 900,
			}))
		})
	})

	Describe("Environment", func() {
		It("translates into a []model.Environment", func() {
			env := Environment{
				{Name: "FOO", Value: "BAR"},
			}
			bbsEnv := env.BBSEnvironment()
			Ω(bbsEnv).Should(Equal([]models.EnvironmentVariable{{Name: "FOO", Value: "BAR"}}))
		})
	})

	Describe("Buildpack", func() {
		ccJSONFragment := `{
						"name": "ocaml-buildpack",
            "key": "ocaml-buildpack-guid",
            "url": "http://ocaml.org/buildpack.zip"
          }`

		It("extracts key and url", func() {
			var buildpack Buildpack

			err := json.Unmarshal([]byte(ccJSONFragment), &buildpack)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(buildpack).To(Equal(Buildpack{
				Name: "ocaml-buildpack",
				Key:  "ocaml-buildpack-guid",
				Url:  "http://ocaml.org/buildpack.zip",
			}))
		})
	})

	Describe("StagingResponseForCC", func() {
		var stagingResponseForCC StagingResponseForCC

		BeforeEach(func() {
			stagingResponseForCC = StagingResponseForCC{
				AppId:                "the-app-id",
				TaskId:               "the-task-id",
				BuildpackKey:         "the-buildpack-key",
				DetectedBuildpack:    "the-detected-buildpack",
				ExecutionMetadata:    "the-execution-metadata",
				DetectedStartCommand: map[string]string{"web": "the-detected-start-command"},
			}
		})

		Context("without an error", func() {
			It("generates valid JSON", func() {
				Ω(json.Marshal(stagingResponseForCC)).Should(MatchJSON(`{
					"app_id": "the-app-id",
					"buildpack_key": "the-buildpack-key",
					"detected_buildpack": "the-detected-buildpack",
					"execution_metadata": "the-execution-metadata",
					"detected_start_command":{"web":"the-detected-start-command"},
					"task_id": "the-task-id"
				}`))
			})
		})

		Context("with an error", func() {
			It("generates valid JSON with the error", func() {
				stagingResponseForCC.Error = &StagingError{
					Id:      "StagingError",
					Message: "FAIL, missing camels!",
				}
				Ω(json.Marshal(stagingResponseForCC)).Should(MatchJSON(`{
					"error": { "id": "StagingError", "message": "FAIL, missing camels!" },

					"app_id": "the-app-id",
					"buildpack_key": "the-buildpack-key",
					"detected_buildpack": "the-detected-buildpack",
					"execution_metadata": "the-execution-metadata",
					"detected_start_command":{"web":"the-detected-start-command"},
					"task_id": "the-task-id"
				}`))
			})
		})
	})

	Describe("DockerStagingResponseForCC", func() {
		var stagingResponseForCC DockerStagingResponseForCC

		BeforeEach(func() {
			stagingResponseForCC = DockerStagingResponseForCC{
				AppId:                "the-app-id",
				TaskId:               "the-task-id",
				ExecutionMetadata:    "the-execution-metadata",
				DetectedStartCommand: map[string]string{"web": "the-detected-start-command"},
			}
		})

		Context("without an error", func() {
			It("generates valid JSON", func() {
				Ω(json.Marshal(stagingResponseForCC)).Should(MatchJSON(`{
					"app_id": "the-app-id",
					"execution_metadata": "the-execution-metadata",
					"detected_start_command":{"web":"the-detected-start-command"},
					"task_id": "the-task-id"
				}`))
			})
		})

		Context("with an error", func() {
			It("generates valid JSON with the error", func() {
				stagingResponseForCC.Error = &StagingError{
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

	Describe("StopStagingRequestForCC", func() {
		var stopStagingRequestFromCC StopStagingRequestFromCC

		BeforeEach(func() {
			stopStagingRequestFromCC = StopStagingRequestFromCC{
				AppId:  "the-app-id",
				TaskId: "the-task-id",
			}
		})

		It("unmarshals JSON", func() {
			stopStagingRequest := StopStagingRequestFromCC{}

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
				stagingErr := SanitizeErrorMessage(diego_errors.INSUFFICIENT_RESOURCES_MESSAGE)
				Ω(stagingErr.Id).Should(Equal("InsufficientResources"))
				Ω(stagingErr.Message).Should(Equal(diego_errors.INSUFFICIENT_RESOURCES_MESSAGE))
			})
		})
		Context("when the message is NoCompatibleCell", func() {
			It("returns a NoCompatibleCell", func() {
				stagingErr := SanitizeErrorMessage(diego_errors.CELL_MISMATCH_MESSAGE)
				Ω(stagingErr.Id).Should(Equal("NoCompatibleCell"))
				Ω(stagingErr.Message).Should(Equal(diego_errors.CELL_MISMATCH_MESSAGE))
			})
		})

		Context("any other message", func() {
			It("returns a StagingError", func() {
				stagingErr := SanitizeErrorMessage("some-error")
				Ω(stagingErr.Id).Should(Equal("StagingError"))
				Ω(stagingErr.Message).Should(Equal("some-error"))
			})
		})
	})
})
