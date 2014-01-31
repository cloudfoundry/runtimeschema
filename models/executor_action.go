package models

type ExecutorAction struct {
	Name string    `json:"name"`
	Args Arguments `json:"args"`
}

type Arguments map[string]string
