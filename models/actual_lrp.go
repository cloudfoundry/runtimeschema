package models

import "encoding/json"

type ActualLRPState int

const (
	ActualLRPStateInvalid ActualLRPState = iota
	ActualLRPStateStarting
	ActualLRPStateRunning
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
) (ActualLRP, error) {

	lrp := ActualLRP{
		ProcessGuid:  processGuid,
		InstanceGuid: instanceGuid,
		CellID:       cellID,
		Domain:       domain,

		Index: index,
	}

	err := lrp.Validate()
	if err != nil {
		return ActualLRP{}, err
	}

	return lrp, nil
}

func NewActualLRPFromJSON(payload []byte) (ActualLRP, error) {
	var actualLRP ActualLRP

	err := json.Unmarshal(payload, &actualLRP)
	if err != nil {
		return ActualLRP{}, err
	}

	err = actualLRP.Validate()
	if err != nil {
		return ActualLRP{}, err
	}

	return actualLRP, nil
}

func (actual ActualLRP) ToJSON() []byte {
	bytes, err := json.Marshal(actual)
	if err != nil {
		panic(err)
	}

	return bytes
}

func (actual ActualLRP) Validate() error {
	var validationError ValidationError

	if actual.ProcessGuid == "" {
		validationError = append(validationError, ErrInvalidJSONMessage{"process_guid"})
	}

	if actual.InstanceGuid == "" {
		validationError = append(validationError, ErrInvalidJSONMessage{"instance_guid"})
	}

	if actual.CellID == "" {
		validationError = append(validationError, ErrInvalidJSONMessage{"cell_id"})
	}

	if actual.Domain == "" {
		validationError = append(validationError, ErrInvalidJSONMessage{"domain"})
	}

	if len(validationError) > 0 {
		return validationError
	}

	return nil
}
