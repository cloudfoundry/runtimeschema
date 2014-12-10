package bbserrors

import (
	"errors"
	"fmt"

	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

var (
	ErrActualLRPCannotBeClaimed = errors.New("cannot claim actual LRP")
	ErrActualLRPCannotBeStarted = errors.New("cannot start actual LRP")

	ErrStoreResourceNotFound             = errors.New("the requested resource could not be found in the store")
	ErrStoreExpectedNonCollectionRequest = errors.New("unable to access single (non-collection) store resource, request body refers to a collection")
	ErrStoreExpectedCollectionRequest    = errors.New("unable to access collection of store resources, request body refers to a single object")
	ErrStoreTimeout                      = errors.New("store request timed out")
	ErrStoreInvalidFormat                = errors.New("store request has invalid format")
	ErrStoreInvalidTTL                   = errors.New("store request has invalid TTL")
	ErrStoreResourceExists               = errors.New("the requested store resource already exists")
	ErrStoreComparisonFailed             = errors.New("store resource comparison failed")

	ErrServiceUnavailable = errors.New("service unavailable")
)

type TaskNotFoundError struct{}

func (e TaskNotFoundError) Error() string {
	return "task not found"
}

func NewTaskStateTransitionError(from, to models.TaskState) TaskStateTransitionError {
	return TaskStateTransitionError{from, to}
}

type TaskStateTransitionError struct {
	from models.TaskState
	to   models.TaskState
}

func (e TaskStateTransitionError) Error() string {
	return "Cannot transition from " + stateString(e.from) + " to " + stateString(e.to)
}

func NewTaskCannotBeResolvedError(from models.TaskState) taskCannotBeResolvedError {
	return taskCannotBeResolvedError{from}
}

type taskCannotBeResolvedError struct {
	from models.TaskState
}

func (e taskCannotBeResolvedError) Error() string {
	return "Cannot resolve task from " + stateString(e.from) + " state"
}

func stateString(state models.TaskState) string {
	switch state {
	case models.TaskStateCompleted:
		return "COMPLETED"
	case models.TaskStateInvalid:
		return "INVALID"
	case models.TaskStatePending:
		return "PENDING"
	case models.TaskStateRunning:
		return "RUNNNING"
	case models.TaskStateResolving:
		return "RESOLVING"
	default:
		panic(fmt.Sprintf("Unknown task state: %v", state))
	}
}
