package cc_messages

import "github.com/cloudfoundry-incubator/runtime-schema/models"
import "github.com/cloudfoundry-incubator/runtime-schema/diego_errors"

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

type DockerStagingRequestFromCC struct {
	AppId           string                     `json:"app_id"`
	TaskId          string                     `json:"task_id"`
	Stack           string                     `json:"stack"`
	DockerImageUrl  string                     `json:"docker_image"`
	FileDescriptors int                        `json:"file_descriptors"`
	MemoryMB        int                        `json:"memory_mb"`
	DiskMB          int                        `json:"disk_mb"`
	Environment     Environment                `json:"environment"`
	EgressRules     []models.SecurityGroupRule `json:"egress_rules,omitempty"`
	Timeout         int                        `json:"timeout"`
}

type DockerStagingResponseForCC struct {
	AppId                string            `json:"app_id"`
	TaskId               string            `json:"task_id"`
	ExecutionMetadata    string            `json:"execution_metadata"`
	DetectedStartCommand map[string]string `json:"detected_start_command"`
	Error                *StagingError     `json:"error,omitempty"`
}

type StagingRequestFromCC struct {
	AppId                          string                     `json:"app_id"`
	TaskId                         string                     `json:"task_id"`
	Stack                          string                     `json:"stack"`
	AppBitsDownloadUri             string                     `json:"app_bits_download_uri"`
	BuildArtifactsCacheDownloadUri string                     `json:"build_artifacts_cache_download_uri,omitempty"`
	BuildArtifactsCacheUploadUri   string                     `json:"build_artifacts_cache_upload_uri"`
	FileDescriptors                int                        `json:"file_descriptors"`
	MemoryMB                       int                        `json:"memory_mb"`
	DiskMB                         int                        `json:"disk_mb"`
	Buildpacks                     []Buildpack                `json:"buildpacks"`
	Environment                    Environment                `json:"environment"`
	DropletUploadUri               string                     `json:"droplet_upload_uri"`
	EgressRules                    []models.SecurityGroupRule `json:"egress_rules,omitempty"`
	Timeout                        int                        `json:"timeout"`
}

const CUSTOM_BUILDPACK = "custom"

type Buildpack struct {
	Name string `json:"name"`
	Key  string `json:"key"`
	Url  string `json:"url"`
}

type StagingResponseForCC struct {
	AppId                string            `json:"app_id"`
	TaskId               string            `json:"task_id"`
	BuildpackKey         string            `json:"buildpack_key"`
	DetectedBuildpack    string            `json:"detected_buildpack"`
	ExecutionMetadata    string            `json:"execution_metadata"`
	DetectedStartCommand map[string]string `json:"detected_start_command"`
	Error                *StagingError     `json:"error,omitempty"`
}

type StopStagingRequestFromCC struct {
	AppId  string `json:"app_id"`
	TaskId string `json:"task_id"`
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
