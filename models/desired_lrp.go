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
	Setup                *ExecutorAction       `json:"setup,omitempty"`
	Action               ExecutorAction        `json:"action"`
	Monitor              *ExecutorAction       `json:"monitor,omitempty"`
	DiskMB               int                   `json:"disk_mb"`
	MemoryMB             int                   `json:"memory_mb"`
	CPUWeight            uint                  `json:"cpu_weight"`
	Ports                []uint32              `json:"ports"`
	Routes               []string              `json:"routes"`
	LogSource            string                `json:"log_source"`
	LogGuid              string                `json:"log_guid"`
	Annotation           string                `json:"annotation,omitempty"`
}

type DesiredLRPChange struct {
	Before *DesiredLRP
	After  *DesiredLRP
}

type DesiredLRPUpdate struct {
	Instances  *int
	Routes     []string
	Annotation *string
}

func (desired DesiredLRP) ApplyUpdate(update DesiredLRPUpdate) DesiredLRP {
	if update.Instances != nil {
		desired.Instances = *update.Instances
	}
	if update.Routes != nil {
		desired.Routes = update.Routes
	}
	if update.Annotation != nil {
		desired.Annotation = *update.Annotation
	}
	return desired
}

var processGuidPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func (desired DesiredLRP) Validate() error {
	var validationError ValidationError

	if desired.Domain == "" {
		validationError = append(validationError, ErrInvalidJSONMessage{"domain"})
	}

	if !processGuidPattern.MatchString(desired.ProcessGuid) {
		validationError = append(validationError, ErrInvalidJSONMessage{"process_guid"})
	}

	if desired.Stack == "" {
		validationError = append(validationError, ErrInvalidJSONMessage{"stack"})
	}

	if err := desired.Action.Validate(); err != nil {
		validationError = append(validationError, err)
	}

	if desired.Instances < 1 {
		validationError = append(validationError, ErrInvalidJSONMessage{"instances"})
	}

	if desired.CPUWeight > 100 {
		validationError = append(validationError, ErrInvalidJSONMessage{"cpu_weight"})
	}

	if len(validationError) > 0 {
		return validationError
	}

	if len(desired.Annotation) > maximumAnnotationLength {
		return ErrInvalidJSONMessage{"annotation"}
	}

	return nil
}

func (desired DesiredLRP) ValidateModifications(updatedModel DesiredLRP) error {
	var validationError ValidationError

	if desired.ProcessGuid != updatedModel.ProcessGuid {
		validationError = append(validationError, ErrInvalidModification{"process_guid"})
	}

	if desired.Domain != updatedModel.Domain {
		validationError = append(validationError, ErrInvalidModification{"domain"})
	}

	if desired.RootFSPath != updatedModel.RootFSPath {
		validationError = append(validationError, ErrInvalidModification{"root_fs"})
	}

	if desired.Stack != updatedModel.Stack {
		validationError = append(validationError, ErrInvalidModification{"stack"})
	}

	if !reflect.DeepEqual(desired.EnvironmentVariables, updatedModel.EnvironmentVariables) {
		validationError = append(validationError, ErrInvalidModification{"env"})
	}

	if !reflect.DeepEqual(desired.Action, updatedModel.Action) {
		validationError = append(validationError, ErrInvalidModification{"action"})
	}

	if desired.DiskMB != updatedModel.DiskMB {
		validationError = append(validationError, ErrInvalidModification{"disk_mb"})
	}

	if desired.MemoryMB != updatedModel.MemoryMB {
		validationError = append(validationError, ErrInvalidModification{"memory_mb"})
	}

	if desired.CPUWeight != updatedModel.CPUWeight {
		validationError = append(validationError, ErrInvalidModification{"cpu_weight"})
	}

	if !reflect.DeepEqual(desired.Ports, updatedModel.Ports) {
		validationError = append(validationError, ErrInvalidModification{"ports"})
	}

	if desired.LogSource != updatedModel.LogSource {
		validationError = append(validationError, ErrInvalidModification{"log_source"})
	}

	if desired.LogGuid != updatedModel.LogGuid {
		validationError = append(validationError, ErrInvalidModification{"log_guid"})
	}

	if len(validationError) > 0 {
		return validationError
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
