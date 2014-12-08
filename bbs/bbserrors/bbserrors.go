package bbserrors

import "errors"

var (
	ErrTaskNotFound                  = errors.New("task not found")
	ErrTaskCannotBeResolved          = errors.New("cannot resolve task from non-resolving state")
	ErrTaskCannotBeMarkedAsResolving = errors.New("cannot mark task as resolving from non-completed state")
	ErrTaskCannotBeStarted           = errors.New("cannot start task from non-pending state")
	ErrTaskCannotBeCompleted         = errors.New("cannot complete task from non-running state")
	ErrTaskCannotBeCancelled         = errors.New("cannot cancel task from non-pending/non-running state")

	ErrActualLRPCannotBeClaimed = errors.New("cannot claim actual LRP")

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
