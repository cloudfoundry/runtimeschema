package domain_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

type DomainBBS struct {
	store  storeadapter.StoreAdapter
	logger lager.Logger
}

func New(
	store storeadapter.StoreAdapter,
	logger lager.Logger,
) *DomainBBS {
	return &DomainBBS{
		store:  store,
		logger: logger,
	}
}

func (bbs *DomainBBS) UpsertDomain(domain string, ttlInSeconds int) error {
	logger := bbs.logger.Session("upsert-domain", lager.Data{
		"domain": domain,
		"ttl":    ttlInSeconds,
	})
	defer logger.Info("finished")

	var validationError models.ValidationError

	if domain == "" {
		validationError = validationError.Append(models.ErrInvalidParameter{"domain"})
	}

	if ttlInSeconds < 0 {
		validationError = validationError.Append(models.ErrInvalidParameter{"ttlInSeconds"})
	}

	if !validationError.Empty() {
		return validationError
	}

	logger.Info("starting")
	return shared.ConvertStoreError(bbs.store.SetMulti([]storeadapter.StoreNode{
		{
			Key: shared.DomainSchemaPath(domain),
			TTL: uint64(ttlInSeconds),
		},
	}))
}
