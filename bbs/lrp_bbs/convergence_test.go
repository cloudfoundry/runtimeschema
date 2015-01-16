package lrp_bbs_test

import (
	"fmt"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/lrp_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type gatherTestData struct {
	actualsToKeep   models.ActualLRPsByProcessGuidAndIndex
	actualsToPrune  models.ActualLRPsByProcessGuidAndIndex
	desiredsToKeep  models.DesiredLRPsByProcessGuid
	desiredsToPrune models.DesiredLRPsByProcessGuid
	domains         models.DomainSet
	cells           models.CellSet
}

func newGatherTestData() *gatherTestData {
	return &gatherTestData{
		actualsToKeep:   models.ActualLRPsByProcessGuidAndIndex{},
		actualsToPrune:  models.ActualLRPsByProcessGuidAndIndex{},
		desiredsToKeep:  models.DesiredLRPsByProcessGuid{},
		desiredsToPrune: models.DesiredLRPsByProcessGuid{},
		domains:         models.DomainSet{},
		cells:           models.CellSet{},
	}
}

var _ = FDescribe("Convergence", func() {

	Describe("Gathering", func() {
		var gatherTest *gatherTestData

		BeforeEach(func() {
			cellID := "some-cell-id"
			missingCellID := "missing-cell-id"
			domain := "test-domain"
			processGuid := "process-guid"

			gatherTest = newGatherTestData()

			// DesiredLRPs
			// keep valid desiredLRP
			gatherTest.desiredsToKeep.Add(newDesiredLRP(processGuid, domain, 4))
			// prune invalid desiredLRP
			gatherTest.desiredsToPrune.Add(models.DesiredLRP{ProcessGuid: "invalid-desired-1"})
			gatherTest.desiredsToPrune.Add(models.DesiredLRP{ProcessGuid: "invalid-desired-2"})

			// ActualLRPs
			// keep valid unclaimed for valid desiredlrp
			gatherTest.actualsToKeep.Add(newUnclaimedActualLRP(processGuid, domain, 0))
			// keep valid claimed on present cell for valid desiredlrp
			gatherTest.actualsToKeep.Add(newClaimedActualLRP(processGuid, cellID, domain, 1))
			// keep valid running on present cell for valid desiredlrp
			gatherTest.actualsToKeep.Add(newRunningActualLRP(processGuid, cellID, domain, 2))
			// prune valid claimed on missing cell for valid desiredlrp
			gatherTest.actualsToPrune.Add(newClaimedActualLRP(processGuid, missingCellID, domain, 4))
			// prune valid running on missing cell for valid desiredlrp
			gatherTest.actualsToPrune.Add(newRunningActualLRP(processGuid, missingCellID, domain, 5))
			// keep valid crashed for valid desiredlrp
			gatherTest.actualsToKeep.Add(newCrashedActualLRP(processGuid, domain, 6))

			// Domains
			gatherTest.domains.Add(domain)

			// Cells
			gatherTest.cells.Add(newCellPresence(cellID))

			createGatherTestData(gatherTest)
		})

		var input *lrp_bbs.ConvergenceInput
		var gatherError error

		JustBeforeEach(func() {
			input, gatherError = bbs.GatherAndPruneLRPConvergenceInput(logger)
		})

		It("gets all valid desired LRPs", func() {
			Ω(input.DesiredLRPs).Should(HaveLen(len(gatherTest.desiredsToKeep)))

			gatherTest.desiredsToKeep.Each(func(expected models.DesiredLRP) {
				desired, ok := input.DesiredLRPs[expected.ProcessGuid]
				Ω(ok).Should(BeTrue(), fmt.Sprintf("expected desiredLRP for process '%s' to be present", expected.ProcessGuid))
				Ω(desired).Should(Equal(expected))
			})
		})

		It("prunes the correct desired LRPs", func() {
			gatherTest.desiredsToPrune.Each(func(expected models.DesiredLRP) {
				_, err := bbs.DesiredLRPByProcessGuid(expected.ProcessGuid)
				Ω(err).Should(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		})

		It("fetches the correct actualLRPs", func() {
			Ω(input.ActualLRPs).Should(HaveLen(len(gatherTest.actualsToKeep)))

			gatherTest.actualsToKeep.Each(func(expected models.ActualLRP) {
				actualIndex, ok := input.ActualLRPs[expected.ProcessGuid]
				Ω(ok).Should(BeTrue(), fmt.Sprintf("expected actualIndex for process '%s' to be present", expected.ProcessGuid))
				actual, ok := actualIndex[expected.Index]
				Ω(ok).Should(BeTrue(), fmt.Sprintf("expected actual for process '%s' and index %d to be present", expected.ProcessGuid, expected.Index))
				Ω(actual).Should(Equal(actual))
			})
		})

		It("prunes the correct actualLRPs", func() {
			gatherTest.actualsToPrune.Each(func(expected models.ActualLRP) {
				_, err := bbs.ActualLRPByProcessGuidAndIndex(expected.ProcessGuid, expected.Index)
				Ω(err).Should(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		})

		It("gets all the domains", func() {
			Ω(input.Domains).Should(HaveLen(len(gatherTest.domains)))
			gatherTest.domains.Each(func(domain string) {
				Ω(input.Domains).Should(HaveKey(domain))
			})
		})
	})
})

func createGatherTestData(test *gatherTestData) {
	test.desiredsToKeep.Each(func(desired models.DesiredLRP) {
		createRawDesiredLRP(desired)
	})

	test.desiredsToPrune.Each(func(desired models.DesiredLRP) {
		createMalformedDesiredLRP(desired.ProcessGuid)
	})

	test.actualsToKeep.Each(func(actual models.ActualLRP) {
		createRawActualLRP(actual)
	})

	test.actualsToPrune.Each(func(actual models.ActualLRP) {
		createRawActualLRP(actual)
	})

	test.domains.Each(func(domain string) {
		createRawDomain(domain)
	})

	test.cells.Each(func(cell models.CellPresence) {
		registerCell(cell)
	})
}
func createMalformedDesiredLRP(guid string) {
	createMalformedValueForKey(shared.DesiredLRPSchemaPath(models.DesiredLRP{ProcessGuid: guid}))
}

func createMalformedActualLRP(guid string, index int) {
	createMalformedValueForKey(shared.ActualLRPSchemaPath(guid, index))
}

func createMalformedValueForKey(key string) {
	err := shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return etcdClient.Create(storeadapter.StoreNode{
			Key:   key,
			Value: []byte("ßßßßßß"),
		})
	})

	Ω(err).ShouldNot(HaveOccurred())
}

func newCellPresence(cellID string) models.CellPresence {
	return models.NewCellPresence(cellID, "some-stack", "1.2.3.4", "az-1")
}

func newDesiredLRP(guid, domain string, instances int) models.DesiredLRP {
	return models.DesiredLRP{
		Domain:      domain,
		ProcessGuid: guid,
		Instances:   instances,
		Stack:       "some-stack",
		MemoryMB:    1024,
		DiskMB:      512,
		CPUWeight:   42,
		Action:      &models.RunAction{Path: "ls"},
	}
}

func newUnclaimedActualLRP(processGuid, domain string, index int) models.ActualLRP {
	return models.ActualLRP{
		ActualLRPKey: models.NewActualLRPKey(processGuid, index, domain),
		State:        models.ActualLRPStateUnclaimed,
		Since:        1138,
	}
}

func newClaimedActualLRP(processGuid, cellID, domain string, index int) models.ActualLRP {
	return models.ActualLRP{
		ActualLRPKey:          models.NewActualLRPKey(processGuid, index, domain),
		ActualLRPContainerKey: models.NewActualLRPContainerKey("instance-guid", cellID),
		State: models.ActualLRPStateClaimed,
		Since: 1138,
	}
}

func newRunningActualLRP(processGuid, cellID, domain string, index int) models.ActualLRP {
	return models.ActualLRP{
		ActualLRPKey:          models.NewActualLRPKey(processGuid, index, domain),
		ActualLRPContainerKey: models.NewActualLRPContainerKey("instance-guid", cellID),
		ActualLRPNetInfo:      models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{}),
		State:                 models.ActualLRPStateRunning,
		Since:                 1138,
	}
}

func newCrashedActualLRP(processGuid, domain string, index int) models.ActualLRP {
	return models.ActualLRP{
		ActualLRPKey: models.NewActualLRPKey(processGuid, index, domain),
		State:        models.ActualLRPStateCrashed,
		Since:        1138,
	}
}
