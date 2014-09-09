package cc_messages

type DockerStagingRequestFromCC struct {
	AppId           string      `json:"app_id"`
	TaskId          string      `json:"task_id"`
	Stack           string      `json:"stack"`
	DockerImageUrl  string      `json:"docker_image"`
	FileDescriptors int         `json:"file_descriptors"`
	MemoryMB        int         `json:"memory_mb"`
	DiskMB          int         `json:"disk_mb"`
	Environment     Environment `json:"environment"`
}

type DockerStagingResponseForCC struct {
	AppId  string `json:"app_id,omitempty"`
	TaskId string `json:"task_id,omitempty"`
	Error  string `json:"error,omitempty"`
}

type StagingRequestFromCC struct {
	AppId                          string      `json:"app_id"`
	TaskId                         string      `json:"task_id"`
	Stack                          string      `json:"stack"`
	AppBitsDownloadUri             string      `json:"app_bits_download_uri"`
	BuildArtifactsCacheDownloadUri string      `json:"build_artifacts_cache_download_uri,omitempty"`
	FileDescriptors                int         `json:"file_descriptors"`
	MemoryMB                       int         `json:"memory_mb"`
	DiskMB                         int         `json:"disk_mb"`
	Buildpacks                     []Buildpack `json:"buildpacks"`
	Environment                    Environment `json:"environment"`
}

type Buildpack struct {
	Name string `json:"name"`
	Key  string `json:"key"`
	Url  string `json:"url"`
}

type StagingResponseForCC struct {
	AppId             string `json:"app_id,omitempty"`
	TaskId            string `json:"task_id,omitempty"`
	BuildpackKey      string `json:"buildpack_key,omitempty"`
	DetectedBuildpack string `json:"detected_buildpack,omitempty"`
	ExecutionMetadata string `json:"execution_metadata,omitempty"`
	Error             string `json:"error,omitempty"`
}
