package shared

import (
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry/storeadapter"
)

func RetryIndefinitelyOnStoreTimeout(callback func() error) error {
	for {
		err := callback()

		if err == storeadapter.ErrorTimeout {
			time.Sleep(time.Second)
			continue
		}

		return convertStoreError(err)
	}
}

func convertStoreError(originalErr error) error {
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
