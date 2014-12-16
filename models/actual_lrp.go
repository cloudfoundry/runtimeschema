package models

import "errors"

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

func (key ActualLRPKey) Validate() error {
	var validationError ValidationError

	if key.ProcessGuid == "" {
		validationError = validationError.Append(ErrInvalidField{"process_guid"})
	}

	if key.Domain == "" {
		validationError = validationError.Append(ErrInvalidField{"domain"})
	}

	if len(validationError) > 0 {
		return validationError
	}

	return nil
}

type ActualLRPContainerKey struct {
	InstanceGuid string `json:"instance_guid"`
	CellID       string `json:"cell_id"`
}

var emptyActualLRPContainerKey = ActualLRPContainerKey{}

func (key *ActualLRPContainerKey) Empty() bool {
	return *key == emptyActualLRPContainerKey
}

func NewActualLRPContainerKey(instanceGuid string, cellID string) ActualLRPContainerKey {
	return ActualLRPContainerKey{
		InstanceGuid: instanceGuid,
		CellID:       cellID,
	}
}

func (key ActualLRPContainerKey) Validate() error {
	var validationError ValidationError

	if key.CellID == "" {
		validationError = validationError.Append(ErrInvalidField{"cell_id"})
	}

	if key.InstanceGuid == "" {
		validationError = validationError.Append(ErrInvalidField{"instance_guid"})
	}

	if len(validationError) > 0 {
		return validationError
	}
	return nil
}

type ActualLRPNetInfo struct {
	Host  string        `json:"host"`
	Ports []PortMapping `json:"ports"`
}

func (info *ActualLRPNetInfo) Empty() bool {
	return info.Host == "" && len(info.Ports) == 0
}

func NewActualLRPNetInfo(host string, ports []PortMapping) ActualLRPNetInfo {
	return ActualLRPNetInfo{
		Host:  host,
		Ports: ports,
	}
}

func (key ActualLRPNetInfo) Validate() error {
	var validationError ValidationError

	if key.Host == "" {
		return append(validationError, ErrInvalidField{"host"})
	}

	return nil
}

type ActualLRP struct {
	ActualLRPKey
	ActualLRPContainerKey
	ActualLRPNetInfo
	State ActualLRPState `json:"state"`
	Since int64          `json:"since"`
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

	err := actual.ActualLRPKey.Validate()
	if err != nil {
		validationError = validationError.Append(err)
	}

	if actual.Since == 0 {
		validationError = validationError.Append(ErrInvalidField{"since"})
	}

	switch actual.State {
	case ActualLRPStateUnclaimed:
		if !actual.ActualLRPContainerKey.Empty() {
			validationError = validationError.Append(errors.New("container key cannot be set when state is unclaimed"))
		}
		if !actual.ActualLRPNetInfo.Empty() {
			validationError = validationError.Append(errors.New("net info cannot be set when state is unclaimed"))
		}

	case ActualLRPStateClaimed:
		if err := actual.ActualLRPContainerKey.Validate(); err != nil {
			validationError = validationError.Append(err)
		}
		if !actual.ActualLRPNetInfo.Empty() {
			validationError = validationError.Append(errors.New("net info cannot be set when state is claimed"))
		}

	case ActualLRPStateRunning:
		if err := actual.ActualLRPContainerKey.Validate(); err != nil {
			validationError = validationError.Append(err)
		}
		if err := actual.ActualLRPNetInfo.Validate(); err != nil {
			validationError = validationError.Append(err)
		}

	default:
		validationError = validationError.Append(ErrInvalidField{"state"})
	}

	if len(validationError) > 0 {
		return validationError
	}

	return nil
}
