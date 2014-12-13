package models

type ActualLRPState string

const (
	ActualLRPStateUnclaimed ActualLRPState = "UNCLAIMED"
	ActualLRPStateClaimed   ActualLRPState = "CLAIMED"
	ActualLRPStateRunning   ActualLRPState = "RUNNING"
)

type ActualLRPChange struct {
	Before *ActualLRP
	After  *ActualLRP
}

type ActualLRPKey struct {
	ProcessGuid string `json:"process_guid"`
	Index       int    `json:"index"`
	Domain      string `json:"domain"`
}

func NewActualLRPKey(processGuid string, index int, domain string) ActualLRPKey {
	return ActualLRPKey{
		ProcessGuid: processGuid,
		Index:       index,
		Domain:      domain,
	}
}

type ActualLRPContainerKey struct {
	InstanceGuid string `json:"instance_guid"`
	CellID       string `json:"cell_id"`
}

func NewActualLRPContainerKey(instanceGuid string, cellID string) ActualLRPContainerKey {
	return ActualLRPContainerKey{
		InstanceGuid: instanceGuid,
		CellID:       cellID,
	}
}

type ActualLRPNetInfo struct {
	Host  string        `json:"host"`
	Ports []PortMapping `json:"ports"`
}

func NewActualLRPNetInfo(host string, ports []PortMapping) ActualLRPNetInfo {
	return ActualLRPNetInfo{
		Host:  host,
		Ports: ports,
	}
}

type ActualLRP struct {
	ActualLRPKey
	ActualLRPContainerKey
	ActualLRPNetInfo
	State ActualLRPState `json:"state"`
	Since int64          `json:"since"`
}

func (actual ActualLRP) IsEquivalentTo(other ActualLRP) bool {
	return actual.CellID == other.CellID &&
		actual.Domain == other.Domain &&
		actual.Index == other.Index &&
		actual.InstanceGuid == other.InstanceGuid &&
		actual.ProcessGuid == other.ProcessGuid &&
		actual.State == other.State
}

func (before ActualLRP) AllowsTransitionTo(lrpKey ActualLRPKey, containerKey ActualLRPContainerKey, newState ActualLRPState) bool {
	if before.ActualLRPKey != lrpKey {
		return false
	}

	if before.State == ActualLRPStateClaimed && newState == ActualLRPStateRunning {
		return true
	}

	if (before.State == ActualLRPStateClaimed || before.State == ActualLRPStateRunning) &&
		(newState == ActualLRPStateClaimed || newState == ActualLRPStateRunning) &&
		(before.ActualLRPContainerKey != containerKey) {
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

	if actual.State == ActualLRPStateUnclaimed {
		if actual.InstanceGuid != "" {
			validationError = append(validationError, ErrInvalidField{"instance_guid"})
		}

		if actual.CellID != "" {
			validationError = append(validationError, ErrInvalidField{"cell_id"})
		}
	} else {
		if actual.InstanceGuid == "" {
			validationError = append(validationError, ErrInvalidField{"instance_guid"})
		}

		if actual.CellID == "" {
			validationError = append(validationError, ErrInvalidField{"cell_id"})
		}
	}

	if len(validationError) > 0 {
		return validationError
	}

	return nil
}
