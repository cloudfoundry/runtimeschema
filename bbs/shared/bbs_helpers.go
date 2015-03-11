package shared

import (
	"database/sql"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry/storeadapter"
	"github.com/go-sql-driver/mysql"
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
	case sql.ErrNoRows:
		return bbserrors.ErrStoreResourceNotFound
	default:
		err, ok := originalErr.(*mysql.MySQLError)
		if ok {
			switch err.Number {
			case 1062:
				return bbserrors.ErrStoreResourceExists
			default:
				return originalErr
			}
		}

		return originalErr
	}
}
