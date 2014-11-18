package models

import (
	"encoding/json"
	"errors"
	"time"
)

var ErrInvalidActionType = errors.New("invalid action type")

type DownloadAction struct {
	From     string `json:"from"`
	To       string `json:"to"`
	CacheKey string `json:"cache_key"`

	LogSource string `json:"log_source,omitempty"`
}

type UploadAction struct {
	To   string `json:"to"`
	From string `json:"from"`

	LogSource string `json:"log_source,omitempty"`
}

type RunAction struct {
	Path           string                `json:"path"`
	Args           []string              `json:"args"`
	Env            []EnvironmentVariable `json:"env"`
	ResourceLimits ResourceLimits        `json:"resource_limits"`
	Privileged     bool                  `json:"privileged,omitempty"`

	LogSource string `json:"log_source,omitempty"`
}

type EnvironmentVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type ResourceLimits struct {
	Nofile *uint64 `json:"nofile,omitempty"`
}

type TimeoutAction struct {
	Action  ExecutorAction `json:"action"`
	Timeout time.Duration  `json:"timeout"`

	LogSource string `json:"log_source,omitempty"`
}

type TryAction struct {
	Action ExecutorAction `json:"action"`

	LogSource string `json:"log_source,omitempty"`
}

type ParallelAction struct {
	Actions []ExecutorAction `json:"actions"`

	LogSource string `json:"log_source,omitempty"`
}

type SerialAction struct {
	Actions []ExecutorAction `json:"actions"`

	LogSource string `json:"log_source,omitempty"`
}

type EmitProgressAction struct {
	Action         ExecutorAction `json:"action"`
	StartMessage   string         `json:"start_message"`
	SuccessMessage string         `json:"success_message"`
	FailureMessage string         `json:"failure_message"`

	LogSource string `json:"log_source,omitempty"`
}

func EmitProgressFor(action ExecutorAction, startMessage string, successMessage string, failureMessage string) ExecutorAction {
	return ExecutorAction{
		EmitProgressAction{
			Action:         action,
			StartMessage:   startMessage,
			SuccessMessage: successMessage,
			FailureMessage: failureMessage,
		},
	}
}

func Timeout(action ExecutorAction, timeout time.Duration) ExecutorAction {
	return ExecutorAction{
		TimeoutAction{
			Action:  action,
			Timeout: timeout,
		},
	}
}

func Try(action ExecutorAction) ExecutorAction {
	return ExecutorAction{
		TryAction{
			Action: action,
		},
	}
}

func Parallel(actions ...ExecutorAction) ExecutorAction {
	return ExecutorAction{
		ParallelAction{
			Actions: actions,
		},
	}
}

func Serial(actions ...ExecutorAction) ExecutorAction {
	return ExecutorAction{
		SerialAction{
			Actions: actions,
		},
	}
}

type executorActionEnvelope struct {
	Name          string           `json:"action"`
	ActionPayload *json.RawMessage `json:"args"`
}

type ExecutorAction struct {
	Action interface{} `json:"-"`
}

func (a ExecutorAction) Validate() error {
	switch a.Action.(type) {
	case DownloadAction, RunAction, UploadAction, TimeoutAction, EmitProgressAction, TryAction, ParallelAction, SerialAction:
		return nil
	default:
		return ErrInvalidActionType
	}
}

func (a ExecutorAction) MarshalJSON() ([]byte, error) {
	var envelope executorActionEnvelope

	payload, err := json.Marshal(a.Action)

	if err != nil {
		return nil, err
	}

	switch a.Action.(type) {
	case DownloadAction:
		envelope.Name = "download"
	case RunAction:
		envelope.Name = "run"
	case UploadAction:
		envelope.Name = "upload"
	case TimeoutAction:
		envelope.Name = "timeout"
	case EmitProgressAction:
		envelope.Name = "emit_progress"
	case TryAction:
		envelope.Name = "try"
	case ParallelAction:
		envelope.Name = "parallel"
	case SerialAction:
		envelope.Name = "serial"
	}

	envelope.ActionPayload = (*json.RawMessage)(&payload)

	return json.Marshal(envelope)
}

func (a *ExecutorAction) UnmarshalJSON(bytes []byte) error {
	var envelope executorActionEnvelope

	err := json.Unmarshal(bytes, &envelope)
	if err != nil {
		return err
	}

	switch envelope.Name {
	case "download":
		action := DownloadAction{}
		err = json.Unmarshal(*envelope.ActionPayload, &action)
		a.Action = action
	case "run":
		action := RunAction{}
		err = json.Unmarshal(*envelope.ActionPayload, &action)
		a.Action = action
	case "upload":
		action := UploadAction{}
		err = json.Unmarshal(*envelope.ActionPayload, &action)
		a.Action = action
	case "emit_progress":
		action := EmitProgressAction{}
		err = json.Unmarshal(*envelope.ActionPayload, &action)
		a.Action = action
	case "timeout":
		action := TimeoutAction{}
		err = json.Unmarshal(*envelope.ActionPayload, &action)
		a.Action = action
	case "try":
		action := TryAction{}
		err = json.Unmarshal(*envelope.ActionPayload, &action)
		a.Action = action
	case "parallel":
		action := ParallelAction{}
		err = json.Unmarshal(*envelope.ActionPayload, &action)
		a.Action = action
	case "serial":
		action := SerialAction{}
		err = json.Unmarshal(*envelope.ActionPayload, &action)
		a.Action = action
	}

	return err
}
