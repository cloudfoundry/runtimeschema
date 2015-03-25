package services_bbs

import (
	"math/rand"
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/heartbeater"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/tedsuo/ifrit"
)

func (bbs *ServicesBBS) NewReceptorHeartbeat(receptorPresence models.ReceptorPresence, ttl, retryInterval time.Duration) ifrit.Runner {
	payload, err := models.ToJSON(receptorPresence)
	if err != nil {
		panic(err)
	}

	return heartbeater.New(bbs.consul, shared.ReceptorSchemaPath(receptorPresence.ReceptorID), payload, ttl, bbs.clock, retryInterval, bbs.logger)
}

func (bbs *ServicesBBS) Receptor() (models.ReceptorPresence, error) {
	receptorPresence := models.ReceptorPresence{}

	receptors, err := bbs.consul.ListPairsExtending(shared.ReceptorSchemaRoot)
	if err != nil {
		return receptorPresence, shared.ConvertConsulError(err)
	}

	if len(receptors) == 0 {
		return receptorPresence, bbserrors.ErrServiceUnavailable
	}

	randomIndex := rand.Intn(len(receptors))
	for _, value := range receptors {
		if randomIndex == 0 {
			err = models.FromJSON(value, &receptorPresence)
			if err != nil {
				return receptorPresence, err
			}

			return receptorPresence, nil
		}

		randomIndex--
	}

	panic("should not reach")
}
