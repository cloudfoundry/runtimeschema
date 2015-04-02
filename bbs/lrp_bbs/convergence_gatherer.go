package lrp_bbs

import (
	"path"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/prune"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/services_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

type ConvergenceInput struct {
	AllProcessGuids map[string]struct{}
	DesiredLRPs     models.DesiredLRPsByProcessGuid
	ActualLRPs      models.ActualLRPsByProcessGuidAndIndex
	Domains         models.DomainSet
	Cells           models.CellSet
}

func NewConvergenceInput(
	desireds models.DesiredLRPsByProcessGuid,
	actuals models.ActualLRPsByProcessGuidAndIndex,
	domains models.DomainSet,
	cells models.CellSet,
) *ConvergenceInput {

	guids := map[string]struct{}{}
	token := struct{}{}
	for guid, _ := range desireds {
		guids[guid] = token
	}
	for guid, _ := range actuals {
		guids[guid] = token
	}

	return &ConvergenceInput{
		AllProcessGuids: guids,
		DesiredLRPs:     desireds,
		ActualLRPs:      actuals,
		Domains:         domains,
		Cells:           cells,
	}
}

func (bbs *LRPBBS) GatherAndPruneLRPConvergenceInput(logger lager.Logger, cellsLoader *services_bbs.CellsLoader) (*ConvergenceInput, error) {
	// always fetch actualLRPs before desiredLRPs to ensure correctness
	actuals, err := bbs.gatherAndPruneActualLRPs(logger)
	if err != nil {
		return &ConvergenceInput{}, err
	}

	domains, err := bbs.domains(logger)
	if err != nil {
		return &ConvergenceInput{}, err
	}

	// always fetch desiredLRPs after actualLRPs to ensure correctness
	desireds, err := bbs.gatherAndPruneDesiredLRPs(logger, domains)
	if err != nil {
		return &ConvergenceInput{}, err
	}

	cellSet, err := cellsLoader.Cells()
	if err != nil {
		return &ConvergenceInput{}, err
	}

	return NewConvergenceInput(desireds, actuals, domains, cellSet), nil
}

func (bbs *LRPBBS) domains(logger lager.Logger) (map[string]struct{}, error) {
	domains := map[string]struct{}{}

	domainRoot, err := bbs.store.ListRecursively(shared.DomainSchemaRoot)
	if err != nil && err != storeadapter.ErrorKeyNotFound {
		logger.Error("failed-to-fetch-domains", err)
		return nil, err
	}

	for _, node := range domainRoot.ChildNodes {
		domains[path.Base(node.Key)] = struct{}{}
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
	err = prune.Prune(logger, bbs.store, shared.ActualLRPSchemaRoot, pruner.gatherAndPrune)
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
		p.logger.Error("pruning-invalid-actual-lrp", err, lager.Data{
			"payload": node.Value,
		})

		return false
	}

	p.Actuals.Add(actual)

	return true
}

func (bbs *LRPBBS) gatherAndPruneDesiredLRPs(logger lager.Logger, domains map[string]struct{}) (map[string]models.DesiredLRP, error) {
	pruner := newDesiredPruner(logger, domains)

	err := prune.Prune(logger, bbs.store, shared.DesiredLRPSchemaRoot, pruner.gatherAndPrune)
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
		p.logger.Error("pruning-invalid-desired-lrp-json", err, lager.Data{
			"payload": node.Value,
		})
		return false
	}

	p.DesiredLRPs.Add(desiredLRP)

	return true
}
