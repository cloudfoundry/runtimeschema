package lrp_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/metric"
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

	convergenceInput, err := bbs.GatherAndPruneLRPConvergenceInput(logger)
	if err != nil {
		logger.Error("failed-to-gather-convergence-input", err)
		return
	}

	changes := CalculateConvergence(logger, bbs.clock, bbs.restartCalculator, convergenceInput)

	bbs.ResolveConvergence(logger, convergenceInput.DesiredLRPs, changes)
}
