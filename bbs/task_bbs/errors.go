package task_bbs

import "errors"

var (
	ErrTaskNotFound      = errors.New("task not found")
	ErrTaskNotResolvable = errors.New("cannot start resolving task in non-completed state")
)

type UnexpectedTaskStateError string

func (e UnexpectedTaskStateError) Error() string {
	return string(e)
}
