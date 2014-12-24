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

func (bbs *LRPBBS) processDesiredChange(desiredChange models.DesiredLRPChange, logger lager.Logger) {
	var desiredLRP models.DesiredLRP

	changeLogger := logger.Session("desired-lrp-change", lager.Data{
		"desired-lrp": desiredLRP,
	})

	if desiredChange.After == nil {
		desiredLRP = *desiredChange.Before
		desiredLRP.Instances = 0
	} else {
		desiredLRP = *desiredChange.After
	}

	actuals, err := bbs.ActualLRPsByProcessGuid(desiredLRP.ProcessGuid)
	if err != nil {
		changeLogger.Error("fetch-actuals-failed", err, lager.Data{"desired-app-message": desiredLRP})
		return
	}

	bbs.reconcile(desiredLRP, actuals, logger)
}

func (bbs *LRPBBS) reconcile(desiredLRP models.DesiredLRP, actuals models.ActualLRPsByIndex, logger lager.Logger) {
	delta := Reconcile(desiredLRP.Instances, actuals)

	for _, lrpIndex := range delta.IndicesToStart {
		logger.Info("request-start", lager.Data{
			"index": lrpIndex,
		})

		lrpStartInstanceCounter.Increment()

		err := bbs.CreateActualLRP(desiredLRP, lrpIndex, logger)
		if err != nil {
			logger.Error("failed-to-create-actual-lrp", err, lager.Data{
				"index": lrpIndex,
			})
		}
	}

	lrpsToRetire := []models.ActualLRP{}
	for _, index := range delta.IndicesToStop {
		logger.Info("request-stop", lager.Data{
			"index": index,
		})

		lrpsToRetire = append(lrpsToRetire, actuals[index])
	}

	lrpStopInstanceCounter.Add(uint64(len(lrpsToRetire)))

	err := bbs.RetireActualLRPs(lrpsToRetire, logger)
	if err != nil {
		logger.Error("failed-to-retire-actual-lrps", err)
	}
}
