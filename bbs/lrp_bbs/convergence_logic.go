package lrp_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"
)

// a given actual LRP should only occur in one of the following states;
// precedence is implemented below

type ConvergenceChanges struct {
	ActualLRPsForExtraIndices      []models.ActualLRP
	ActualLRPKeysForMissingIndices []models.ActualLRPKey
	ActualLRPsWithMissingCells     []models.ActualLRP
	RestartableCrashedActualLRPs   []models.ActualLRP
	StaleUnclaimedActualLRPs       []models.ActualLRP
}

func CalculateConvergence(
	logger lager.Logger,
	clock clock.Clock,
	restartCalculator models.RestartCalculator,
	input *ConvergenceInput,
) *ConvergenceChanges {
	sess := logger.Session("calculate-convergence")

	sess.Info("start")
	defer sess.Info("done")

	changes := &ConvergenceChanges{}

	now := clock.Now()

	for processGuid, _ := range input.AllProcessGuids {
		pLog := sess.WithData(lager.Data{
			"process-guid": processGuid,
		})

		desired, hasDesired := input.DesiredLRPs[processGuid]

		actualsByIndex := input.ActualLRPs[processGuid]

		if hasDesired {

			for i := 0; i < desired.Instances; i++ {
				if _, hasIndex := actualsByIndex[i]; !hasIndex {
					pLog.Info("missing", lager.Data{"index": i})
					changes.ActualLRPKeysForMissingIndices = append(
						changes.ActualLRPKeysForMissingIndices,
						models.NewActualLRPKey(desired.ProcessGuid, i, desired.Domain),
					)
				}
			}

			for i, actual := range actualsByIndex {
				if actual.Index >= desired.Instances && input.Domains.Contains(desired.Domain) {
					pLog.Info("extra", lager.Data{"index": i})
					changes.ActualLRPsForExtraIndices = append(changes.ActualLRPsForExtraIndices, actual)
					continue
				}

				if actual.ShouldRestartCrash(now, restartCalculator) {
					pLog.Info("restart-crash", lager.Data{"index": i})
					changes.RestartableCrashedActualLRPs = append(changes.RestartableCrashedActualLRPs, actual)
					continue
				}

				if actual.CellIsMissing(input.Cells) {
					pLog.Info("missing-cell", lager.Data{"index": i})
					changes.ActualLRPsWithMissingCells = append(changes.ActualLRPsWithMissingCells, actual)
					continue
				}

				if actual.ShouldStartUnclaimed(now) {
					pLog.Info("stale-unclaimed", lager.Data{"index": i})
					changes.StaleUnclaimedActualLRPs = append(changes.StaleUnclaimedActualLRPs, actual)
					continue
				}
			}
		} else {
			for i, actual := range actualsByIndex {
				if !input.Domains.Contains(actual.Domain) {
					pLog.Info("skipping-unfresh-domain")
					continue
				}

				pLog.Info("no-longer-desired", lager.Data{"index": i})
				changes.ActualLRPsForExtraIndices = append(changes.ActualLRPsForExtraIndices, actual)
			}
		}
	}

	return changes
}
