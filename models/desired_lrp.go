package models

import (
	"encoding/json"
	"reflect"
	"regexp"
)

type DesiredLRP struct {
	ProcessGuid          string                `json:"process_guid"`
	Domain               string                `json:"domain"`
	RootFSPath           string                `json:"root_fs"`
	Instances            int                   `json:"instances"`
	Stack                string                `json:"stack"`
	EnvironmentVariables []EnvironmentVariable `json:"env,omitempty"`
	Actions              []ExecutorAction      `json:"actions"`
	DiskMB               int                   `json:"disk_mb"`
	MemoryMB             int                   `json:"memory_mb"`
	CPUWeight            uint                  `json:"cpu_weight"`
	Ports                []PortMapping         `json:"ports"`
	Routes               []string              `json:"routes"`
	Log                  LogConfig             `json:"log"`
	Annotation           string                `json:"annotation,omitempty"`
}

type DesiredLRPChange struct {
	Before *DesiredLRP
	After  *DesiredLRP
}

var processGuidPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func (desired DesiredLRP) Validate() error {
	if desired.Domain == "" {
		return ErrInvalidJSONMessage{"domain"}
	}

	if !processGuidPattern.MatchString(desired.ProcessGuid) {
		return ErrInvalidJSONMessage{"process_guid"}
	}

	if desired.Stack == "" {
		return ErrInvalidJSONMessage{"stack"}
	}

	if len(desired.Actions) == 0 {
		return ErrInvalidJSONMessage{"actions"}
	}

	if desired.Instances < 1 {
		return ErrInvalidJSONMessage{"instances"}
	}

	if desired.CPUWeight > 100 {
		return ErrInvalidJSONMessage{"cpu_weight"}
	}

	return nil
}

func (desired DesiredLRP) ValidateModifications(updatedModel DesiredLRP) error {
	if desired.ProcessGuid != updatedModel.ProcessGuid {
		return ErrInvalidModification{"process_guid"}
	}

	if desired.Domain != updatedModel.Domain {
		return ErrInvalidModification{"domain"}
	}

	if desired.RootFSPath != updatedModel.RootFSPath {
		return ErrInvalidModification{"root_fs"}
	}

	if desired.Stack != updatedModel.Stack {
		return ErrInvalidModification{"stack"}
	}

	if !reflect.DeepEqual(desired.EnvironmentVariables, updatedModel.EnvironmentVariables) {
		return ErrInvalidModification{"env"}
	}

	if !reflect.DeepEqual(desired.Actions, updatedModel.Actions) {
		return ErrInvalidModification{"actions"}
	}

	if desired.DiskMB != updatedModel.DiskMB {
		return ErrInvalidModification{"disk_mb"}
	}

	if desired.MemoryMB != updatedModel.MemoryMB {
		return ErrInvalidModification{"memory_mb"}
	}

	if desired.CPUWeight != updatedModel.CPUWeight {
		return ErrInvalidModification{"cpu_weight"}
	}

	if !reflect.DeepEqual(desired.Ports, updatedModel.Ports) {
		return ErrInvalidModification{"ports"}
	}

	if !reflect.DeepEqual(desired.Log, updatedModel.Log) {
		return ErrInvalidModification{"log"}
	}

	return nil
}

func NewDesiredLRPFromJSON(payload []byte) (DesiredLRP, error) {
	var lrp DesiredLRP

	err := json.Unmarshal(payload, &lrp)
	if err != nil {
		return DesiredLRP{}, err
	}

	err = lrp.Validate()
	if err != nil {
		return DesiredLRP{}, err
	}

	return lrp, nil
}

func (desired DesiredLRP) ToJSON() []byte {
	bytes, err := json.Marshal(desired)
	if err != nil {
		panic(err)
	}

	return bytes
}
