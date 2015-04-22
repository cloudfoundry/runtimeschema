package cc_messages

import (
	"encoding/json"

	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

type HealthCheckType string

const UnspecifiedHealthCheckType HealthCheckType = "" // backwards-compatibility
const PortHealthCheckType HealthCheckType = "port"
const NoneHealthCheckType HealthCheckType = "none"

type DesireAppRequestFromCC struct {
	ProcessGuid                 string                     `json:"process_guid"`
	DropletUri                  string                     `json:"droplet_uri"`
	DockerImageUrl              string                     `json:"docker_image"`
	Stack                       string                     `json:"stack"`
	StartCommand                string                     `json:"start_command"`
	ExecutionMetadata           string                     `json:"execution_metadata"`
	Environment                 Environment                `json:"environment"`
	MemoryMB                    int                        `json:"memory_mb"`
	DiskMB                      int                        `json:"disk_mb"`
	FileDescriptors             uint64                     `json:"file_descriptors"`
	NumInstances                int                        `json:"num_instances"`
	Routes                      []string                   `json:"routes"`
	AllowSSH                    bool                       `json:"allow_ssh"`
	LogGuid                     string                     `json:"log_guid"`
	HealthCheckType             HealthCheckType            `json:"health_check_type"`
	HealthCheckTimeoutInSeconds uint                       `json:"health_check_timeout_in_seconds"`
	EgressRules                 []models.SecurityGroupRule `json:"egress_rules,omitempty"`
	ETag                        string                     `json:"etag"`
}

type CCDesiredStateServerResponse struct {
	Apps        []DesireAppRequestFromCC `json:"apps"`
	CCBulkToken *json.RawMessage         `json:"token"`
}

type CCDesiredAppFingerprint struct {
	ProcessGuid string `json:"process_guid"`
	ETag        string `json:"etag"`
}

type CCDesiredStateFingerprintResponse struct {
	Fingerprints []CCDesiredAppFingerprint `json:"fingerprints"`
	CCBulkToken  *json.RawMessage          `json:"token"`
}

type CCBulkToken struct {
	Id int `json:"id"`
}
