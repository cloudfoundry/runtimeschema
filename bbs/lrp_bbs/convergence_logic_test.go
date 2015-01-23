package lrp_bbs_test

import (
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/lrp_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/timeprovider/faketimeprovider"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager/lagertest"
)

const staleUnclaimedDuration = 30 * time.Second

var _ = Describe("CalculateConvergence", func() {
	const domainA = "domain-a"
	const domainB = "domain-b"

	var cellA = models.CellPresence{
		CellID:     "cell-a",
		RepAddress: "some-rep-address",
		Zone:       "some-zone",
		Stack:      "some-stack",
	}

	var cellB = models.CellPresence{
		CellID:     "cell-b",
		RepAddress: "some-rep-address",
		Zone:       "some-zone",
		Stack:      "some-stack",
	}

	var lrpA = models.DesiredLRP{
		ProcessGuid: "process-guid-a",
		Instances:   2,
		Domain:      domainA,
	}

	var lrpB = models.DesiredLRP{
		ProcessGuid: "process-guid-b",
		Instances:   2,
		Domain:      domainB,
	}

	var (
		logger           *lagertest.TestLogger
		fakeTimeProvider *faketimeprovider.FakeTimeProvider
		input            *lrp_bbs.ConvergenceInput

		changes *lrp_bbs.ConvergenceChanges
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		fakeTimeProvider = faketimeprovider.New(time.Unix(0, 1138))
		input = nil
	})

	JustBeforeEach(func() {
		changes = lrp_bbs.CalculateConvergence(logger, fakeTimeProvider, models.NewDefaultRestartCalculator(), input)
	})

	Context("actual LRPs with missing cells", func() {
		BeforeEach(func() {
			input = lrp_bbs.NewConvergenceInput(
				desiredLRPs(lrpA),
				actualLRPs(
					newRunningActualLRP(lrpA, cellA.CellID, 0),
					newRunningActualLRP(lrpA, cellA.CellID, 1),
				),
				domainSet(domainA),
				cellSet(),
			)
		})

		It("reports them", func() {
			output := &lrp_bbs.ConvergenceChanges{
				ActualLRPsWithMissingCells: []models.ActualLRP{
					newRunningActualLRP(lrpA, cellA.CellID, 0),
					newRunningActualLRP(lrpA, cellA.CellID, 1),
				},
			}

			changesEqual(changes, output)
		})
	})

	Context("actual lrp keys for missing desired indices", func() {
		BeforeEach(func() {
			input = lrp_bbs.NewConvergenceInput(
				desiredLRPs(lrpA),
				actualLRPs(),
				domainSet(domainA),
				cellSet(cellA),
			)
		})

		It("reports them", func() {
			output := &lrp_bbs.ConvergenceChanges{
				ActualLRPKeysForMissingIndices: []models.ActualLRPKey{
					actualLRPKey(lrpA, 0),
					actualLRPKey(lrpA, 1),
				},
			}

			changesEqual(changes, output)
		})
	})

	Context("actualLRPs existing for indices we don't desire", func() {
		BeforeEach(func() {
			input = lrp_bbs.NewConvergenceInput(
				desiredLRPs(lrpA),
				actualLRPs(
					newRunningActualLRP(lrpA, cellA.CellID, 0),
					newRunningActualLRP(lrpA, cellA.CellID, 1),
					newRunningActualLRP(lrpA, cellA.CellID, 2),
				),
				domainSet(domainA),
				cellSet(cellA),
			)
		})

		It("reports them", func() {
			output := &lrp_bbs.ConvergenceChanges{
				ActualLRPsForExtraIndices: []models.ActualLRP{
					newRunningActualLRP(lrpA, cellA.CellID, 2),
				},
			}

			changesEqual(changes, output)
		})
	})

	Context("crashed actual LRPS ready to be restarted", func() {
		BeforeEach(func() {
			input = lrp_bbs.NewConvergenceInput(
				desiredLRPs(lrpA),
				actualLRPs(
					newStartableCrashedActualLRP(lrpA, 0),
					newUnstartableCrashedActualLRP(lrpA, 1),
				),
				domainSet(domainA),
				cellSet(cellA),
			)
		})

		It("reports them", func() {
			output := &lrp_bbs.ConvergenceChanges{
				RestartableCrashedActualLRPs: []models.ActualLRP{
					newStartableCrashedActualLRP(lrpA, 0),
				},
			}

			changesEqual(changes, output)
		})
	})

	Context("stale unclaimed actual LRPs", func() {
		BeforeEach(func() {
			input = lrp_bbs.NewConvergenceInput(
				desiredLRPs(lrpA),
				actualLRPs(
					newRunningActualLRP(lrpA, cellA.CellID, 0),
					newStaleUnclaimedActualLRP(lrpA, 1),
				),
				domainSet(domainA),
				cellSet(cellA),
			)
		})

		It("reports them", func() {
			output := &lrp_bbs.ConvergenceChanges{
				StaleUnclaimedActualLRPs: []models.ActualLRP{
					newStaleUnclaimedActualLRP(lrpA, 1),
				},
			}

			changesEqual(changes, output)
		})
	})

	Context("an unfresh domain", func() {
		BeforeEach(func() {
			input = lrp_bbs.NewConvergenceInput(
				desiredLRPs(lrpA, lrpB),
				actualLRPs(newRunningActualLRP(lrpA, cellA.CellID, 7)),
				domainSet(domainB),
				cellSet(cellA, cellB),
			)
		})

		It("performs all checks except stopping extra indices", func() {
			output := &lrp_bbs.ConvergenceChanges{
				ActualLRPKeysForMissingIndices: []models.ActualLRPKey{
					actualLRPKey(lrpA, 0),
					actualLRPKey(lrpA, 1),
					actualLRPKey(lrpB, 0),
					actualLRPKey(lrpB, 1),
				},
			}

			changesEqual(changes, output)
		})
	})

	Context("stable state", func() {
		BeforeEach(func() {
			input = lrp_bbs.NewConvergenceInput(
				desiredLRPs(lrpA),
				actualLRPs(
					newStableRunningActualLRP(lrpA, cellA.CellID, 0),
					newStableRunningActualLRP(lrpA, cellA.CellID, 1),
				),
				domainSet(domainA),
				cellSet(cellA),
			)
		})

		It("reports nothing", func() {
			changesEqual(changes, &lrp_bbs.ConvergenceChanges{})
		})
	})
})

func changesEqual(actual *lrp_bbs.ConvergenceChanges, expected *lrp_bbs.ConvergenceChanges) {
	Ω(actual.ActualLRPsWithMissingCells).Should(ConsistOf(expected.ActualLRPsWithMissingCells))
	Ω(actual.ActualLRPsForExtraIndices).Should(ConsistOf(expected.ActualLRPsForExtraIndices))
	Ω(actual.ActualLRPKeysForMissingIndices).Should(ConsistOf(expected.ActualLRPKeysForMissingIndices))
	Ω(actual.RestartableCrashedActualLRPs).Should(ConsistOf(expected.RestartableCrashedActualLRPs))
	Ω(actual.StaleUnclaimedActualLRPs).Should(ConsistOf(expected.StaleUnclaimedActualLRPs))
}

func domainSet(domains ...string) models.DomainSet {
	set := models.DomainSet{}

	for _, domain := range domains {
		set[domain] = struct{}{}
	}

	return set
}

func cellSet(cells ...models.CellPresence) models.CellSet {
	set := models.CellSet{}

	for _, cell := range cells {
		set.Add(cell)
	}

	return set
}

func desiredLRPs(lrps ...models.DesiredLRP) models.DesiredLRPsByProcessGuid {
	set := models.DesiredLRPsByProcessGuid{}

	for _, lrp := range lrps {
		set[lrp.ProcessGuid] = lrp
	}

	return set
}

func actualLRPs(lrps ...models.ActualLRP) models.ActualLRPsByProcessGuidAndIndex {
	set := models.ActualLRPsByProcessGuidAndIndex{}

	for _, lrp := range lrps {
		byIndex, found := set[lrp.ProcessGuid]
		if !found {
			byIndex = models.ActualLRPsByIndex{}
			set[lrp.ProcessGuid] = byIndex
		}

		byIndex[lrp.Index] = lrp
	}

	return set
}

func actualLRPKey(lrp models.DesiredLRP, index int) models.ActualLRPKey {
	return models.NewActualLRPKey(lrp.ProcessGuid, index, lrp.Domain)
}

func crashedActualReadyForRestart(lrp models.DesiredLRP, index int) models.ActualLRP {
	return models.ActualLRP{
		ActualLRPKey: actualLRPKey(lrp, index),
		CrashCount:   1,
		State:        models.ActualLRPStateCrashed,
		Since:        1138,
	}
}

func crashedActualNeverRestart(lrp models.DesiredLRP, index int) models.ActualLRP {
	return models.ActualLRP{
		ActualLRPKey: actualLRPKey(lrp, index),
		CrashCount:   201,
		State:        models.ActualLRPStateCrashed,
		Since:        1138,
	}
}

func newNotStaleUnclaimedActualLRP(lrp models.DesiredLRP, index int) models.ActualLRP {
	return models.ActualLRP{
		ActualLRPKey: actualLRPKey(lrp, index),
		State:        models.ActualLRPStateUnclaimed,
		Since:        1138,
	}
}

func newStaleUnclaimedActualLRP(lrp models.DesiredLRP, index int) models.ActualLRP {
	return models.ActualLRP{
		ActualLRPKey: actualLRPKey(lrp, index),
		State:        models.ActualLRPStateUnclaimed,
		Since:        1138 - (staleUnclaimedDuration + time.Second).Nanoseconds(),
	}
}

func newStableRunningActualLRP(lrp models.DesiredLRP, cellID string, index int) models.ActualLRP {
	return models.ActualLRP{
		ActualLRPKey:          actualLRPKey(lrp, index),
		ActualLRPContainerKey: models.NewActualLRPContainerKey("instance-guid", cellID),
		ActualLRPNetInfo:      models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{}),
		State:                 models.ActualLRPStateRunning,
		Since:                 1138 - (30 * time.Minute).Nanoseconds(),
	}
}
