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

func (bbs *ServicesBBS) NewReceptorHeartbeat(receptorPresence models.ReceptorPresence, interval time.Duration) ifrit.Runner {
	payload, err := models.ToJSON(receptorPresence)
	if err != nil {
		panic(err)
	}

	return heartbeater.New(bbs.store, bbs.timeProvider, shared.ReceptorSchemaPath(receptorPresence.ReceptorID), string(payload), interval, bbs.logger)
}

func (bbs *ServicesBBS) Receptor() (models.ReceptorPresence, error) {
	receptorPresence := models.ReceptorPresence{}

	node, err := bbs.store.ListRecursively(shared.ReceptorSchemaRoot)
	if err != nil {
		return receptorPresence, shared.ConvertStoreError(err)
	}

	receptors := node.ChildNodes

	if len(receptors) == 0 {
		return receptorPresence, bbserrors.ErrServiceUnavailable
	}

	receptorNode := receptors[rand.Intn(len(receptors))]

	err = models.FromJSON(receptorNode.Value, &receptorPresence)
	if err != nil {
		return receptorPresence, err
	}

	return receptorPresence, nil
}
