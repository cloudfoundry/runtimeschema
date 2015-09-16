package cc_messages

import (
	"encoding/json"

	"github.com/cloudfoundry-incubator/bbs/models"
)

type HealthCheckType string

const UnspecifiedHealthCheckType HealthCheckType = "" // backwards-compatibility
const PortHealthCheckType HealthCheckType = "port"
const NoneHealthCheckType HealthCheckType = "none"

const CC_HTTP_ROUTES = "http_routes"

type DesireAppRequestFromCC struct {
	ProcessGuid                 string                        `json:"process_guid"`
	DropletUri                  string                        `json:"droplet_uri"`
	DockerImageUrl              string                        `json:"docker_image"`
	DockerLoginServer           string                        `json:"docker_login_server,omitempty"`
	DockerUser                  string                        `json:"docker_user,omitempty"`
	DockerPassword              string                        `json:"docker_password,omitempty"`
	DockerEmail                 string                        `json:"docker_email,omitempty"`
	Stack                       string                        `json:"stack"`
	StartCommand                string                        `json:"start_command"`
	ExecutionMetadata           string                        `json:"execution_metadata"`
	Environment                 []*models.EnvironmentVariable `json:"environment"`
	MemoryMB                    int                           `json:"memory_mb"`
	DiskMB                      int                           `json:"disk_mb"`
	FileDescriptors             uint64                        `json:"file_descriptors"`
	NumInstances                int                           `json:"num_instances"`
	RoutingInfo                 CCRouteInfo                   `json:"routing_info"`
	AllowSSH                    bool                          `json:"allow_ssh"`
	LogGuid                     string                        `json:"log_guid"`
	HealthCheckType             HealthCheckType               `json:"health_check_type"`
	HealthCheckTimeoutInSeconds uint                          `json:"health_check_timeout_in_seconds"`
	EgressRules                 []*models.SecurityGroupRule   `json:"egress_rules,omitempty"`
	ETag                        string                        `json:"etag"`
}

type CCRouteInfo map[string]*json.RawMessage

type CCHTTPRoutes []CCHTTPRoute

func (r CCHTTPRoutes) CCRouteInfo() (CCRouteInfo, error) {
	routesJson, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	routesPayload := json.RawMessage(routesJson)
	routingInfo := make(map[string]*json.RawMessage)
	routingInfo[CC_HTTP_ROUTES] = &routesPayload
	return routingInfo, nil
}

type CCHTTPRoute struct {
	Hostname        string `json: "hostname"`
	RouteServiceUrl string `json: "route_service_url,omitempty"`
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
