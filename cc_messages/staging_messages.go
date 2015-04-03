package cc_messages

import (
	"encoding/json"

	"github.com/cloudfoundry-incubator/runtime-schema/diego_errors"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

type StagingErrorID string

var (
	STAGING_ERROR          StagingErrorID = "StagingError"
	INSUFFICIENT_RESOURCES StagingErrorID = "InsufficientResources"
	NO_COMPATIBLE_CELL     StagingErrorID = "NoCompatibleCell"
)

type StagingError struct {
	Id      StagingErrorID `json:"id"`
	Message string         `json:"message"`
}

type StagingRequestFromCC struct {
	AppId           string                     `json:"app_id"`
	FileDescriptors int                        `json:"file_descriptors"`
	MemoryMB        int                        `json:"memory_mb"`
	DiskMB          int                        `json:"disk_mb"`
	Environment     Environment                `json:"environment"`
	EgressRules     []models.SecurityGroupRule `json:"egress_rules,omitempty"`
	Timeout         int                        `json:"timeout"`
	LogGuid         string                     `json:"log_guid"`
	Lifecycle       string                     `json:"lifecycle"`
	LifecycleData   *json.RawMessage           `json:"lifecycle_data,omitempty"`
}

type BuildpackStagingData struct {
	AppBitsDownloadUri             string      `json:"app_bits_download_uri"`
	BuildArtifactsCacheDownloadUri string      `json:"build_artifacts_cache_download_uri,omitempty"`
	BuildArtifactsCacheUploadUri   string      `json:"build_artifacts_cache_upload_uri"`
	Buildpacks                     []Buildpack `json:"buildpacks"`
	DropletUploadUri               string      `json:"droplet_upload_uri"`
	Stack                          string      `json:"stack"`
}

type DockerStagingData struct {
	DockerImageUrl string `json:"docker_image"`
}

const CUSTOM_BUILDPACK = "custom"

type Buildpack struct {
	Name       string `json:"name"`
	Key        string `json:"key"`
	Url        string `json:"url"`
	SkipDetect bool   `json:"skip_detect"`
}

type StagingResponseForCC struct {
	ExecutionMetadata    string            `json:"execution_metadata"`
	DetectedStartCommand map[string]string `json:"detected_start_command"`
	Error                *StagingError     `json:"error,omitempty"`
	LifecycleData        *json.RawMessage  `json:"lifecycle_data,omitempty"`
}

type BuildpackStagingResponse struct {
	BuildpackKey      string `json:"buildpack_key"`
	DetectedBuildpack string `json:"detected_buildpack"`
}

type StagingTaskAnnotation struct {
	Lifecycle string `json:"lifecycle"`
}

func SanitizeErrorMessage(message string) *StagingError {
	id := STAGING_ERROR
	switch message {
	case diego_errors.INSUFFICIENT_RESOURCES_MESSAGE:
		id = INSUFFICIENT_RESOURCES
	case diego_errors.CELL_MISMATCH_MESSAGE:
		id = NO_COMPATIBLE_CELL
	default:
		message = "staging failed"
	}

	return &StagingError{
		Id:      id,
		Message: message,
	}
}
