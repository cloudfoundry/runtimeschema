package models

import (
	"encoding/json"
	"net/url"
	"regexp"
)

type TaskState int

const (
	TaskStateInvalid TaskState = iota
	TaskStatePending
	TaskStateClaimed
	TaskStateRunning
	TaskStateCompleted
	TaskStateResolving
)

var taskGuidPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

type Task struct {
	TaskGuid             string                `json:"task_guid"`
	Domain               string                `json:"domain"`
	RootFSPath           string                `json:"root_fs"`
	Stack                string                `json:"stack"`
	EnvironmentVariables []EnvironmentVariable `json:"env,omitempty"`
	Action               ExecutorAction        `json:"action"`
	MemoryMB             int                   `json:"memory_mb"`
	DiskMB               int                   `json:"disk_mb"`
	CPUWeight            uint                  `json:"cpu_weight"`
	LogSource            string                `json:"log_source"`
	LogGuid              string                `json:"log_guid"`
	CreatedAt            int64                 `json:"created_at"` //  the number of nanoseconds elapsed since January 1, 1970 UTC
	UpdatedAt            int64                 `json:"updated_at"`
	FirstCompletedAt     int64                 `json:"first_completed_at"`
	ResultFile           string                `json:"result_file"`

	State TaskState `json:"state"`

	CellID string `json:"cell_id"`

	Result        string `json:"result"`
	Failed        bool   `json:"failed"`
	FailureReason string `json:"failure_reason"`

	CompletionCallbackURL *url.URL `json:"completion_callback_url,omitempty"`
	Annotation            string   `json:"annotation,omitempty"`
}

type StagingResult struct {
	BuildpackKey         string            `json:"buildpack_key,omitempty"`
	DetectedBuildpack    string            `json:"detected_buildpack"`
	ExecutionMetadata    string            `json:"execution_metadata"`
	DetectedStartCommand map[string]string `json:"detected_start_command"`
}

type StagingDockerResult struct {
	ExecutionMetadata    string            `json:"execution_metadata"`
	DetectedStartCommand map[string]string `json:"detected_start_command"`
}

type StagingTaskAnnotation struct {
	AppId  string `json:"app_id"`
	TaskId string `json:"task_id"`
}

func NewTaskFromJSON(payload []byte) (Task, error) {
	var task Task

	err := json.Unmarshal(payload, &task)
	if err != nil {
		return Task{}, err
	}

	err = task.Validate()
	if err != nil {
		return Task{}, err
	}

	return task, nil
}

func (task Task) Validate() error {
	if task.Domain == "" {
		return ErrInvalidJSONMessage{"domain"}
	}

	if !taskGuidPattern.MatchString(task.TaskGuid) {
		return ErrInvalidJSONMessage{"task_guid"}
	}

	if task.Stack == "" {
		return ErrInvalidJSONMessage{"stack"}
	}

	if err := task.Action.Validate(); err != nil {
		return err
	}

	if task.CPUWeight > 100 {
		return ErrInvalidJSONMessage{"cpu_weight"}
	}

	return nil
}

func (task Task) ToJSON() []byte {
	bytes, err := json.Marshal(task)
	if err != nil {
		panic(err)
	}

	return bytes
}
