package task_bbs

import "errors"

var (
	ErrTaskNotFound      = errors.New("cannot complete non-existing task")
	ErrTaskNotResolvable = errors.New("cannot start resolving task in non-completed state")
)
