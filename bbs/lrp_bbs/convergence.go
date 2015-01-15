package lrp_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/prune"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

func (bbs *LRPBBS) GatherAndPruneForLRPConvergence(logger lager.Logger) (
	map[string]models.DesiredLRP,
	map[string]models.ActualLRPsByIndex,
	map[string]struct{},
	error,
) {

	actuals, err := bbs.gatherAndPruneActualLRPs(logger)
	if err != nil {
		return nil, nil, nil, err
	}

	domains, err := bbs.domains(logger)
	if err != nil {
		return nil, nil, nil, err
	}

	desireds, err := bbs.gatherAndPruneDesiredLRPs(logger, domains)
	if err != nil {
		return nil, nil, nil, err
	}

	return desireds, actuals, domains, nil
}

func (bbs *LRPBBS) domains(logger lager.Logger) (map[string]struct{}, error) {
	domains := map[string]struct{}{}

	domainRoot, err := bbs.store.ListRecursively(shared.DomainSchemaRoot)
	if err != nil && err != storeadapter.ErrorKeyNotFound {
		logger.Error("failed-to-fetch-domains", err)
		return nil, err
	}

	for _, node := range domainRoot.ChildNodes {
		domains[string(node.Value)] = struct{}{}
	}

	return domains, nil
}

func (bbs *LRPBBS) gatherAndPruneActualLRPs(logger lager.Logger) (map[string]models.ActualLRPsByIndex, error) {
	cellRoot, err := bbs.store.ListRecursively(shared.CellSchemaRoot)
	if err == storeadapter.ErrorKeyNotFound {
		cellRoot = storeadapter.StoreNode{}
	} else if err != nil {
		logger.Error("failed-to-get-cells", err)
		return nil, err
	}

	pruner := newActualPruner(logger, cellRoot)
	err = prune.Prune(bbs.store, shared.ActualLRPSchemaRoot, pruner.gatherAndPrune)
	if err != nil {
		logger.Error("failed-to-prune-actual-lrps", err)
		return nil, err
	}

	return pruner.ActualsByProcessGuid, nil
}

type actualPruner struct {
	logger lager.Logger
	bbs    *LRPBBS

	cellRoot             storeadapter.StoreNode
	ActualsByProcessGuid map[string]models.ActualLRPsByIndex
}

func newActualPruner(logger lager.Logger, cellRoot storeadapter.StoreNode) *actualPruner {
	return &actualPruner{
		logger:   logger,
		cellRoot: cellRoot,

		ActualsByProcessGuid: map[string]models.ActualLRPsByIndex{},
	}
}

func (p *actualPruner) gatherAndPrune(node storeadapter.StoreNode) bool {
	var actual models.ActualLRP
	err := models.FromJSON(node.Value, &actual)
	if err != nil {
		return false
	}

	switch actual.State {
	case models.ActualLRPStateUnclaimed:
	case models.ActualLRPStateCrashed:
	default:
		if _, ok := p.cellRoot.Lookup(actual.CellID); !ok {
			p.logger.Info("detected-actual-with-missing-cell", lager.Data{
				"actual":  actual,
				"cell-id": actual.CellID,
			})
			return false
		}
	}

	actuals, found := p.ActualsByProcessGuid[actual.ProcessGuid]
	if !found {
		actuals = models.ActualLRPsByIndex{}
		p.ActualsByProcessGuid[actual.ProcessGuid] = actuals
	}

	actuals[actual.Index] = actual

	return true
}

func (bbs *LRPBBS) gatherAndPruneDesiredLRPs(logger lager.Logger, domains map[string]struct{}) (map[string]models.DesiredLRP, error) {
	pruner := newDesiredPruner(logger, domains)
	err := prune.Prune(bbs.store, shared.DesiredLRPSchemaRoot, pruner.gatherAndPrune)
	if err != nil {
		logger.Error("failed-to-prune-desired-lrps", err)
		return nil, err
	}

	return pruner.DesiredLRPsByProcessGuid, nil
}

type desiredPruner struct {
	logger  lager.Logger
	domains map[string]struct{}

	DesiredLRPsByProcessGuid map[string]models.DesiredLRP
}

func newDesiredPruner(logger lager.Logger, domains map[string]struct{}) *desiredPruner {
	return &desiredPruner{
		logger:  logger,
		domains: domains,

		DesiredLRPsByProcessGuid: map[string]models.DesiredLRP{},
	}
}

func (p *desiredPruner) gatherAndPrune(node storeadapter.StoreNode) bool {
	var desiredLRP models.DesiredLRP

	err := models.FromJSON(node.Value, &desiredLRP)
	if err != nil {
		p.logger.Info("pruning-invalid-desired-lrp-json", lager.Data{
			"error":   err.Error(),
			"payload": node.Value,
		})
		return false
	}

	p.DesiredLRPsByProcessGuid[desiredLRP.ProcessGuid] = desiredLRP

	return true
}
