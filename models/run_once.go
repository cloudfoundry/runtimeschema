package models

import (
	"encoding/json"
)

type RunOnce struct {
	Guid string `json:"guid"`

	ExecutorID string `json:"executor_id"`

	ContainerHandle string `json:"container_handle"`

	Failed        bool   `json:"failed"`
	FailureReason string `json:"failure_reason"`
}

func NewRunOnceFromJSON(payload []byte) (RunOnce, error) {
	var runOnce RunOnce

	err := json.Unmarshal(payload, &runOnce)
	if err != nil {
		return RunOnce{}, err
	}

	return runOnce, nil
}

func (self RunOnce) ToJSON() []byte {
	bytes, err := json.Marshal(self)
	if err != nil {
		panic(err)
	}

	return bytes
}
