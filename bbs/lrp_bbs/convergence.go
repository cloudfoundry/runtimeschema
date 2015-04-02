package lrp_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/metric"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/pivotal-golang/lager"
)

const (
	convergeLRPRunsCounter = metric.Counter("ConvergenceLRPRuns")
	convergeLRPDuration    = metric.Duration("ConvergenceLRPDuration")

	lrpsDeletedCounter = metric.Counter("ConvergenceLRPsDeleted")
)

func (bbs *LRPBBS) ConvergeLRPs(logger lager.Logger) {
	logger = logger.Session("converge-lrps")
	logger.Info("starting-convergence")
	defer logger.Info("finished-convergence")

	convergeLRPRunsCounter.Increment()

	// make sure to get funcy here otherwise the time will be precomputed
	convergeStart := bbs.clock.Now()
	defer func() {
		convergeLRPDuration.Send(bbs.clock.Now().Sub(convergeStart))
	}()

	logger.Debug("gathering-convergence-input")
	convergenceInput, err := bbs.GatherAndPruneLRPConvergenceInput(logger)
	if err != nil {
		logger.Error("failed-gathering-convergence-input", err)
		return
	}
	logger.Debug("succeeded-gathering-convergence-input")

	changes := CalculateConvergence(logger, bbs.clock, models.NewDefaultRestartCalculator(), convergenceInput)

	bbs.ResolveConvergence(logger, convergenceInput.DesiredLRPs, changes)
}
