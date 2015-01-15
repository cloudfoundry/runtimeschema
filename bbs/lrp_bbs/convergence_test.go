package lrp_bbs_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = FDescribe("Convergence", func() {
	var (
		desireds map[string]models.DesiredLRP
		actuals  map[string]models.ActualLRPsByIndex
		domains  map[string]struct{}

		gatherError error
	)

	BeforeEach(func() {
	})

	JustBeforeEach(func() {
		desireds, actuals, domains, gatherError = bbs.GatherAndPruneForLRPConvergence(logger)
	})

	Describe("Gathering", func() {
		var desiredLRP models.DesiredLRP
		var actualLRP models.ActualLRP
		var cell models.CellPresence

		BeforeEach(func() {
			cell = models.NewCellPresence("some-cell-id", "some-stack", "1.2.3.4", "az-1")

			desiredLRP = models.DesiredLRP{
				Domain:      "test-domain",
				ProcessGuid: "process-guid",
				Instances:   1,
				Stack:       "some-stack",
				MemoryMB:    1024,
				DiskMB:      512,
				CPUWeight:   42,
				Action:      &models.RunAction{Path: "ls"},
			}

			lrpKey := models.NewActualLRPKey("process-guid", 0, "test-domain")
			containerKey := models.NewActualLRPContainerKey("instance-guid", "some-cell-id")
			netInfo := models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{})
			actualLRP = models.ActualLRP{
				ActualLRPKey:          lrpKey,
				ActualLRPContainerKey: containerKey,
				ActualLRPNetInfo:      netInfo,
				State:                 models.ActualLRPStateRunning,
				Since:                 1138,
			}

			createRawDomain("test-domain")
			registerCell(cell)
			createRawDesiredLRP(desiredLRP)
			createRawActualLRP(actualLRP)
		})

		It("gets desired LRPs", func() {
			Ω(desireds).Should(HaveLen(1))
			Ω(desireds).Should(HaveKey("process-guid"))
			Ω(desireds["process-guid"]).Should(Equal(desiredLRP))
		})

		It("gets actual LRPs", func() {
			expectedActuals := models.ActualLRPsByIndex{}
			expectedActuals[0] = actualLRP

			Ω(actuals).Should(HaveLen(1))
			Ω(actuals).Should(HaveKey("process-guid"))
			Ω(actuals["process-guid"]).Should(Equal(expectedActuals))
		})

		It("gets the domains", func() {
			Ω(domains).Should(HaveLen(1))
			Ω(domains).Should(HaveKey("test-domain"))
		})
	})
})
