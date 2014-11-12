package models

import (
	"encoding/json"
	"errors"
)

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
	ExecutorID   string `json:"executor_id"`
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
	executorID string,
	domain string,
	index int,
) (ActualLRP, error) {
	if processGuid == "" {
		return ActualLRP{}, errors.New("Cannot construct Actual LRP with empty process guid")
	}

	if instanceGuid == "" {
		return ActualLRP{}, errors.New("Cannot construct Actual LRP with empty instance guid")
	}

	if executorID == "" {
		return ActualLRP{}, errors.New("Cannot construct Actual LRP with empty executor ID")
	}

	if domain == "" {
		return ActualLRP{}, errors.New("Cannot construct Actual LRP with empty domain")
	}

	return ActualLRP{
		ProcessGuid:  processGuid,
		InstanceGuid: instanceGuid,
		ExecutorID:   executorID,
		Domain:       domain,

		Index: index,
	}, nil
}

func NewActualLRPFromJSON(payload []byte) (ActualLRP, error) {
	var actualLRP ActualLRP

	err := json.Unmarshal(payload, &actualLRP)
	if err != nil {
		return ActualLRP{}, err
	}

	if actualLRP.ProcessGuid == "" {
		return ActualLRP{}, ErrInvalidJSONMessage{"process_guid"}
	}

	if actualLRP.InstanceGuid == "" {
		return ActualLRP{}, ErrInvalidJSONMessage{"instance_guid"}
	}

	if actualLRP.ExecutorID == "" {
		return ActualLRP{}, ErrInvalidJSONMessage{"executor_id"}
	}

	if actualLRP.Domain == "" {
		return ActualLRP{}, ErrInvalidJSONMessage{"domain"}
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
