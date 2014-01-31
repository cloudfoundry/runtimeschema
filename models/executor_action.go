package models

type ExecutorAction struct {
	Name string    `json:"name"`
	Args Arguments `json:"args"`
}

type Arguments map[string]interface{}

func NewCopyAction(from string, to string, extract bool, compress bool) ExecutorAction {
	return ExecutorAction{
		Name: "copy",
		Args: Arguments{
			"from":     from,
			"to":       to,
			"extract":  extract,
			"compress": compress,
		},
	}
}
