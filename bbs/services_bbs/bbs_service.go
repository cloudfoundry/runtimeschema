package services_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

func (bbs *ServicesBBS) MasterURL() (string, error) {
	value, err := bbs.consul.GetAcquiredValue(shared.LockSchemaPath("bbs_lock"))
	if err != nil {
		return "", bbserrors.ErrServiceUnavailable
	}

	bbsPresence := models.BBSPresence{}
	err = models.FromJSON(value, &bbsPresence)
	if err != nil {
		return "", err
	}

	return bbsPresence.URL, nil
}
