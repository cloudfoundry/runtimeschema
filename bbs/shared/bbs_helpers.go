package shared

import (
	"github.com/cloudfoundry-incubator/consuladapter"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry/storeadapter"
)

type ContainerRetainment bool

const (
	KeepContainer   ContainerRetainment = true
	DeleteContainer ContainerRetainment = false
)

func ConvertStoreError(originalErr error) error {
	switch originalErr {
	case storeadapter.ErrorKeyNotFound:
		return bbserrors.ErrStoreResourceNotFound
	case storeadapter.ErrorNodeIsDirectory:
		return bbserrors.ErrStoreExpectedNonCollectionRequest
	case storeadapter.ErrorNodeIsNotDirectory:
		return bbserrors.ErrStoreExpectedCollectionRequest
	case storeadapter.ErrorTimeout:
		return bbserrors.ErrStoreTimeout
	case storeadapter.ErrorInvalidFormat:
		return bbserrors.ErrStoreInvalidFormat
	case storeadapter.ErrorInvalidTTL:
		return bbserrors.ErrStoreInvalidTTL
	case storeadapter.ErrorKeyExists:
		return bbserrors.ErrStoreResourceExists
	case storeadapter.ErrorKeyComparisonFailed:
		return bbserrors.ErrStoreComparisonFailed
	default:
		return originalErr
	}
}

func ConvertConsulError(originalErr error) error {
	switch originalErr.(type) {
	case consuladapter.KeyNotFoundError:
		return bbserrors.ErrStoreResourceNotFound
	case consuladapter.PrefixNotFoundError:
		return bbserrors.ErrStoreResourceNotFound
	default:
		return originalErr
	}
}
