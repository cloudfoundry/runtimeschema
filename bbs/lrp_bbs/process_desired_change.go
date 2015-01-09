package lrp_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/metric"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/pivotal-golang/lager"
)

var (
	lrpStartInstanceCounter = metric.Counter("LRPInstanceStartRequests")
	lrpStopInstanceCounter  = metric.Counter("LRPInstanceStopRequests")
)

type reconcileInfo struct {
	desiredLRP models.DesiredLRP
	actualLRPs models.ActualLRPsByIndex
	Result
}

func (bbs *LRPBBS) processDesiredCreateOrUpdate(desiredLRP models.DesiredLRP, logger lager.Logger) {
	changeLogger := logger.Session("process-desired-lrp-create-or-update", lager.Data{"desired-lrp": desiredLRP})

	actuals, err := bbs.ActualLRPsByProcessGuid(desiredLRP.ProcessGuid)
	if err != nil {
		changeLogger.Error("fetch-actuals-failed", err, lager.Data{"desired-lrp": desiredLRP})
		return
	}

	bbs.reconcile([]reconcileInfo{{desiredLRP, actuals, Reconcile(desiredLRP.Instances, actuals)}}, logger)
}

func (bbs *LRPBBS) processDesiredDelete(desiredLRP models.DesiredLRP, logger lager.Logger) {
	changeLogger := logger.Session("process-desired-lrp-delete", lager.Data{"desired-lrp": desiredLRP})

	actuals, err := bbs.ActualLRPsByProcessGuid(desiredLRP.ProcessGuid)
	if err != nil {
		changeLogger.Error("fetch-actuals-failed", err, lager.Data{"desired-lrp": desiredLRP})
		return
	}

	bbs.reconcile([]reconcileInfo{{desiredLRP, actuals, Reconcile(0, actuals)}}, logger)
}

func (bbs *LRPBBS) reconcile(infos []reconcileInfo, logger lager.Logger) {
	startAuctions := []models.LRPStartRequest{}
	lrpsToRetire := []models.ActualLRP{}

	for _, delta := range infos {
		if len(delta.IndicesToStart) > 0 {
			lrpStartInstanceCounter.Add(uint64(len(delta.IndicesToStart)))

			indices := make([]uint, 0, len(delta.IndicesToStart))

			for _, lrpIndex := range delta.IndicesToStart {
				err := bbs.createActualLRP(delta.desiredLRP, lrpIndex, logger)
				if err != nil {
					logger.Error("failed-to-create-actual-lrp", err, lager.Data{
						"process-guid": delta.desiredLRP.ProcessGuid,
						"index":        lrpIndex,
					})
					continue
				}

				logger.Info("request-start", lager.Data{
					"process-guid": delta.desiredLRP.ProcessGuid,
					"index":        lrpIndex,
				})

				indices = append(indices, uint(lrpIndex))
			}

			if len(indices) > 0 {
				startAuctions = append(startAuctions, models.LRPStartRequest{
					DesiredLRP: delta.desiredLRP,
					Indices:    indices,
				})
			}
		}

		for _, index := range delta.IndicesToStop {
			actualLRP := delta.actualLRPs[index]
			logger.Info("request-stop", lager.Data{
				"process-guid":  actualLRP.ProcessGuid,
				"instance-guid": actualLRP.InstanceGuid,
				"index":         index,
			})

			lrpsToRetire = append(lrpsToRetire, actualLRP)
		}
	}

	if len(startAuctions) > 0 {
		err := bbs.requestLRPAuctions(startAuctions)
		if err != nil {
			logger.Error("failed-to-request-start-auctions", err, lager.Data{"lrp-starts": startAuctions})
			// The creation succeeded, the start request error can be dropped
		}
	}

	if len(lrpsToRetire) > 0 {
		lrpStopInstanceCounter.Add(uint64(len(lrpsToRetire)))
		bbs.RetireActualLRPs(lrpsToRetire, logger)
	}
}
