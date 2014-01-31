package models

type ExecutorAction struct {
	Name string    `json:"name"`
	Args Arguments `json:"args"`
}

type Arguments map[string]string

func NewCopyAction(from string, to string) ExecutorAction {
	return ExecutorAction{
		Name: "copy",
		Args: Arguments{
			"from": from,
			"to":   to,
		},
	}
}
