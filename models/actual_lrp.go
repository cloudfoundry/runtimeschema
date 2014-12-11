package models

type ActualLRPState string

const (
	ActualLRPStateInvalid   ActualLRPState = "INVALID"
	ActualLRPStateUnclaimed ActualLRPState = "UNCLAIMED"
	ActualLRPStateClaimed   ActualLRPState = "CLAIMED"
	ActualLRPStateRunning   ActualLRPState = "RUNNING"
)

type ActualLRPChange struct {
	Before *ActualLRP
	After  *ActualLRP
}

type ActualLRP struct {
	ProcessGuid  string `json:"process_guid"`
	InstanceGuid string `json:"instance_guid"`
	CellID       string `json:"cell_id"`
	Domain       string `json:"domain"`

	Index int `json:"index"`

	Host  string        `json:"host"`
	Ports []PortMapping `json:"ports"`

	State ActualLRPState `json:"state"`
	Since int64          `json:"since"`
}

func NewActualLRP(
	processGuid string,
	instanceGuid string,
	cellID string,
	domain string,
	index int,
	state ActualLRPState,
) ActualLRP {

	lrp := ActualLRP{
		ProcessGuid:  processGuid,
		InstanceGuid: instanceGuid,
		CellID:       cellID,
		Domain:       domain,

		Index: index,
		State: state,
	}

	return lrp
}

func (actual ActualLRP) IsEquivalentTo(other ActualLRP) bool {
	return actual.CellID == other.CellID &&
		actual.Domain == other.Domain &&
		actual.Index == other.Index &&
		actual.InstanceGuid == other.InstanceGuid &&
		actual.ProcessGuid == other.ProcessGuid &&
		actual.State == other.State
}

func (before ActualLRP) AllowsTransitionTo(after ActualLRP) bool {
	if before.ProcessGuid != after.ProcessGuid {
		return false
	}

	if before.InstanceGuid != after.InstanceGuid &&
		!(before.State != ActualLRPStateRunning && after.State == ActualLRPStateRunning) {
		return false
	}

	if before.Index != after.Index {
		return false
	}

	if before.Domain != after.Domain {
		return false
	}

	if before.State == ActualLRPStateClaimed && after.State == ActualLRPStateRunning {
		return true
	}

	if (before.State == ActualLRPStateClaimed || before.State == ActualLRPStateRunning) &&
		(after.State == ActualLRPStateClaimed || after.State == ActualLRPStateRunning) &&
		(before.CellID != after.CellID) {
		return false
	}

	return true
}

func (actual ActualLRP) Validate() error {
	var validationError ValidationError

	if actual.ProcessGuid == "" {
		validationError = append(validationError, ErrInvalidField{"process_guid"})
	}

	if actual.Domain == "" {
		validationError = append(validationError, ErrInvalidField{"domain"})
	}

	if actual.InstanceGuid == "" {
		validationError = append(validationError, ErrInvalidField{"instance_guid"})
	}

	if actual.State == ActualLRPStateUnclaimed {
		if actual.CellID != "" {
			validationError = append(validationError, ErrInvalidField{"cell_id"})
		}
	} else {
		if actual.CellID == "" {
			validationError = append(validationError, ErrInvalidField{"cell_id"})
		}
	}

	if len(validationError) > 0 {
		return validationError
	}

	return nil
}
