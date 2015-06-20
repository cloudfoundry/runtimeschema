package test_helpers

import (
	"encoding/json"
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/gomega"
)

func (t *TestHelper) NewValidDesiredLRP(guid string) models.DesiredLRP {
	myRouterJSON := json.RawMessage(`{"foo":"bar"}`)
	desiredLRP := models.DesiredLRP{
		ProcessGuid:          guid,
		Domain:               "some-domain",
		RootFS:               "some:rootfs",
		Instances:            1,
		EnvironmentVariables: []models.EnvironmentVariable{{Name: "FOO", Value: "bar"}},
		Setup:                &models.RunAction{Path: "ls", User: "name"},
		Action:               &models.RunAction{Path: "ls", User: "name"},
		StartTimeout:         15,
		Monitor: models.EmitProgressFor(
			models.Timeout(
				models.Try(models.Parallel(models.Serial(&models.RunAction{Path: "ls", User: "name"}))),
				10*time.Second,
			),
			"start-message",
			"success-message",
			"failure-message",
		),
		DiskMB:      512,
		MemoryMB:    1024,
		CPUWeight:   42,
		Routes:      map[string]*json.RawMessage{"my-router": &myRouterJSON},
		LogSource:   "some-log-source",
		LogGuid:     "some-log-guid",
		MetricsGuid: "some-metrics-guid",
		Annotation:  "some-annotation",
		EgressRules: []models.SecurityGroupRule{{
			Protocol:     models.TCPProtocol,
			Destinations: []string{"1.1.1.1/32", "2.2.2.2/32"},
			PortRange:    &models.PortRange{Start: 10, End: 16000},
		}},
	}
	err := desiredLRP.Validate()
	Expect(err).NotTo(HaveOccurred())

	return desiredLRP
}

func (t *TestHelper) NewValidActualLRP(guid string, index int) models.ActualLRP {
	actualLRP := models.ActualLRP{
		ActualLRPKey:         models.NewActualLRPKey(guid, index, "some-domain"),
		ActualLRPInstanceKey: models.NewActualLRPInstanceKey("some-guid", "some-cell"),
		ActualLRPNetInfo:     models.NewActualLRPNetInfo("some-address", []models.PortMapping{{HostPort: 2222, ContainerPort: 4444}}),
		CrashCount:           33,
		CrashReason:          "badness",
		State:                models.ActualLRPStateRunning,
		Since:                1138,
		ModificationTag:      models.ModificationTag{Epoch: "some-epoch", Index: 999},
	}
	err := actualLRP.Validate()
	Expect(err).NotTo(HaveOccurred())

	return actualLRP
}
