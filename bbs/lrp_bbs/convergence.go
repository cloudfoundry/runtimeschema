package lrp_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/prune"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

type ConvergenceInput struct {
	DesiredLRPs models.DesiredLRPsByProcessGuid
	ActualLRPs  models.ActualLRPsByProcessGuidAndIndex
	Domains     models.DomainSet
}

func NewConvergenceInput(
	desireds models.DesiredLRPsByProcessGuid,
	actuals models.ActualLRPsByProcessGuidAndIndex,
	domains models.DomainSet,
) *ConvergenceInput {
	return &ConvergenceInput{
		DesiredLRPs: desireds,
		ActualLRPs:  actuals,
		Domains:     domains,
	}
}

func (bbs *LRPBBS) GatherAndPruneLRPConvergenceInput(logger lager.Logger) (*ConvergenceInput, error) {
	actuals, err := bbs.gatherAndPruneActualLRPs(logger)
	if err != nil {
		return &ConvergenceInput{}, err
	}

	domains, err := bbs.domains(logger)
	if err != nil {
		return &ConvergenceInput{}, err
	}

	desireds, err := bbs.gatherAndPruneDesiredLRPs(logger, domains)
	if err != nil {
		return &ConvergenceInput{}, err
	}

	return NewConvergenceInput(desireds, actuals, domains), nil
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

	return pruner.Actuals, nil
}

type actualPruner struct {
	logger lager.Logger
	bbs    *LRPBBS

	cellRoot storeadapter.StoreNode
	Actuals  models.ActualLRPsByProcessGuidAndIndex
}

func newActualPruner(logger lager.Logger, cellRoot storeadapter.StoreNode) *actualPruner {
	return &actualPruner{
		logger:   logger,
		cellRoot: cellRoot,

		Actuals: models.ActualLRPsByProcessGuidAndIndex{},
	}
}

func (p *actualPruner) gatherAndPrune(node storeadapter.StoreNode) bool {
	var actual models.ActualLRP
	err := models.FromJSON(node.Value, &actual)
	if err != nil {
		return false
	}

	if actual.CellID != "" {
		if _, ok := p.cellRoot.Lookup(actual.CellID); !ok {
			p.logger.Info("detected-actual-with-missing-cell", lager.Data{
				"actual":  actual,
				"cell-id": actual.CellID,
			})
			return false
		}
	}

	p.Actuals.Add(actual)
	return true
}

func (bbs *LRPBBS) gatherAndPruneDesiredLRPs(logger lager.Logger, domains map[string]struct{}) (map[string]models.DesiredLRP, error) {
	pruner := newDesiredPruner(logger, domains)
	err := prune.Prune(bbs.store, shared.DesiredLRPSchemaRoot, pruner.gatherAndPrune)
	if err != nil {
		logger.Error("failed-to-prune-desired-lrps", err)
		return nil, err
	}

	return pruner.DesiredLRPs, nil
}

type desiredPruner struct {
	logger  lager.Logger
	domains map[string]struct{}

	DesiredLRPs models.DesiredLRPsByProcessGuid
}

func newDesiredPruner(logger lager.Logger, domains map[string]struct{}) *desiredPruner {
	return &desiredPruner{
		logger:  logger,
		domains: domains,

		DesiredLRPs: models.DesiredLRPsByProcessGuid{},
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

	p.DesiredLRPs.Add(desiredLRP)

	return true
}
