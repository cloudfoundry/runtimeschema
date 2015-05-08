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

type testDataForConvergenceGatherer struct {
	actualsToKeep      models.ActualLRPsByProcessGuidAndIndex
	actualsToPrune     models.ActualLRPsByProcessGuidAndIndex
	evacuationsToKeep  models.ActualLRPsByProcessGuidAndIndex
	evacuationsToPrune models.ActualLRPsByProcessGuidAndIndex
	desiredsToKeep     models.DesiredLRPsByProcessGuid
	desiredsToPrune    models.DesiredLRPsByProcessGuid
	domains            models.DomainSet
	cells              models.CellSet
}

func newTestDataForConvergenceGatherer() *testDataForConvergenceGatherer {
	return &testDataForConvergenceGatherer{
		actualsToKeep:      models.ActualLRPsByProcessGuidAndIndex{},
		actualsToPrune:     models.ActualLRPsByProcessGuidAndIndex{},
		evacuationsToKeep:  models.ActualLRPsByProcessGuidAndIndex{},
		evacuationsToPrune: models.ActualLRPsByProcessGuidAndIndex{},
		desiredsToKeep:     models.DesiredLRPsByProcessGuid{},
		desiredsToPrune:    models.DesiredLRPsByProcessGuid{},
		domains:            models.DomainSet{},
		cells:              models.CellSet{},
	}
}

var _ = Describe("Convergence", func() {
	Describe("Gathering", func() {
		var testData *testDataForConvergenceGatherer
		cellID := "some-cell-id"
		domain := "test-domain"

		BeforeEach(func() {
			testData = newTestDataForConvergenceGatherer()

			// create some valid DesiredLRPs
			for i := 0; i < 200; i++ {
				testData.desiredsToKeep.Add(newDesiredLRP(fmt.Sprintf("valid-desired-%d", i), "domain", 1))
			}

			// create some invalid DesiredLRPs that will have valid corresponding ActualLRPs
			for i := 0; i < 20; i++ {
				testData.desiredsToPrune.Add(models.DesiredLRP{ProcessGuid: fmt.Sprintf("invalid-desired-with-valid-actual-%d", i)})
			}

			// create some invalid DesiredLRPs that will NOT have valid corresponding ActualLRPs
			for i := 0; i < 20; i++ {
				testData.desiredsToPrune.Add(models.DesiredLRP{ProcessGuid: fmt.Sprintf("invalid-desired-without-valid-actual-%d", i)})
			}

			// create some ActualLRPs, corresponding to valid DesiredLRPs
			// with all indices having bad data
			for i := 0; i < 20; i++ {
				invalidActual1 := newActualLRP(
					newDesiredLRP(fmt.Sprintf("valid-desired-%d", i), "domain", 1),
					9001,
				)
				invalidActual1.Since = 0
				testData.actualsToPrune.Add(invalidActual1)

				invalidActual2 := newActualLRP(
					newDesiredLRP(fmt.Sprintf("valid-desired-%d", i), "domain", 1),
					1337,
				)
				invalidActual2.Since = 0
				testData.actualsToPrune.Add(invalidActual2)
			}

			// create some ActualLRPs, corresponding to valid DesiredLRPs
			// with some indices with bad data, and some with good data
			for i := 20; i < 40; i++ {
				validActual := newActualLRP(
					newDesiredLRP(fmt.Sprintf("valid-desired-%d", i), "domain", 1),
					9001,
				)
				testData.actualsToKeep.Add(validActual)

				invalidActual := newActualLRP(
					newDesiredLRP(fmt.Sprintf("valid-desired-%d", i), "domain", 1),
					1337,
				)
				invalidActual.Since = 0
				testData.actualsToPrune.Add(invalidActual)
			}

			// create some ActualLRPs, corresponding to valid DesiredLRPs
			// with all indices having good data
			for i := 40; i < 60; i++ {
				validActual1 := newActualLRP(
					newDesiredLRP(fmt.Sprintf("valid-desired-%d", i), "domain", 1),
					9001,
				)
				testData.actualsToKeep.Add(validActual1)

				validActual2 := newActualLRP(
					newDesiredLRP(fmt.Sprintf("valid-desired-%d", i), "domain", 1),
					1337,
				)
				testData.actualsToKeep.Add(validActual2)
			}

			// create some ActualLRPs, corresponding to valid DesiredLRPs
			// with one index having good Evacuating data but bad Instance data,
			// another index having the opposite
			for i := 60; i < 80; i++ {
				validEvacuating := newActualLRP(
					newDesiredLRP(fmt.Sprintf("valid-desired-%d", i), "domain", 1),
					9001,
				)
				testData.evacuationsToKeep.Add(validEvacuating)

				invalidActual := newActualLRP(
					newDesiredLRP(fmt.Sprintf("valid-desired-%d", i), "domain", 1),
					9001,
				)
				invalidActual.Since = 0
				testData.actualsToPrune.Add(invalidActual)

				invalidEvacuating := newActualLRP(
					newDesiredLRP(fmt.Sprintf("valid-desired-%d", i), "domain", 1),
					1337,
				)
				invalidEvacuating.Since = 0
				testData.evacuationsToPrune.Add(invalidEvacuating)

				validActual := newActualLRP(
					newDesiredLRP(fmt.Sprintf("valid-desired-%d", i), "domain", 1),
					1337,
				)
				testData.actualsToKeep.Add(validActual)
			}

			// create similar ActualLRPs corresponding to invalid DesiredLRPs
			for i := 0; i < 5; i++ {
				invalidActual1 := newActualLRP(
					newDesiredLRP(fmt.Sprintf("invalid-desired-without-valid-actual-%d", i), "domain", 1),
					9001,
				)
				invalidActual1.Since = 0
				testData.actualsToPrune.Add(invalidActual1)

				invalidActual2 := newActualLRP(
					newDesiredLRP(fmt.Sprintf("invalid-desired-without-valid-actual-%d", i), "domain", 1),
					1337,
				)
				invalidActual2.Since = 0
				testData.actualsToPrune.Add(invalidActual2)
			}
			for i := 0; i < 5; i++ {
				validActual := newActualLRP(
					newDesiredLRP(fmt.Sprintf("invalid-desired-with-valid-actual-%d", i), "domain", 1),
					9001,
				)
				testData.actualsToKeep.Add(validActual)

				invalidActual := newActualLRP(
					newDesiredLRP(fmt.Sprintf("invalid-desired-with-valid-actual-%d", i), "domain", 1),
					1337,
				)
				invalidActual.Since = 0
				testData.actualsToPrune.Add(invalidActual)
			}
			for i := 5; i < 10; i++ {
				validActual1 := newActualLRP(
					newDesiredLRP(fmt.Sprintf("invalid-desired-with-valid-actual-%d", i), "domain", 1),
					9001,
				)
				testData.actualsToKeep.Add(validActual1)

				validActual2 := newActualLRP(
					newDesiredLRP(fmt.Sprintf("invalid-desired-with-valid-actual-%d", i), "domain", 1),
					1337,
				)
				testData.actualsToKeep.Add(validActual2)
			}
			for i := 10; i < 15; i++ {
				validEvacuating := newActualLRP(
					newDesiredLRP(fmt.Sprintf("invalid-desired-with-valid-actual-%d", i), "domain", 1),
					9001,
				)
				testData.evacuationsToKeep.Add(validEvacuating)

				invalidActual := newActualLRP(
					newDesiredLRP(fmt.Sprintf("invalid-desired-with-valid-actual-%d", i), "domain", 1),
					9001,
				)
				invalidActual.Since = 0
				testData.actualsToPrune.Add(invalidActual)

				invalidEvacuating := newActualLRP(
					newDesiredLRP(fmt.Sprintf("invalid-desired-with-valid-actual-%d", i), "domain", 1),
					1337,
				)
				invalidEvacuating.Since = 0
				testData.evacuationsToPrune.Add(invalidEvacuating)

				validActual := newActualLRP(
					newDesiredLRP(fmt.Sprintf("invalid-desired-with-valid-actual-%d", i), "domain", 1),
					1337,
				)
				testData.actualsToKeep.Add(validActual)
			}

			// create similar ActualLRPs corresponding to unknown DesiredLRPs
			for i := 0; i < 5; i++ {
				invalidActual1 := newActualLRP(
					newDesiredLRP(fmt.Sprintf("unknown-desired-without-valid-actual-%d", i), "domain", 1),
					9001,
				)
				invalidActual1.Since = 0
				testData.actualsToPrune.Add(invalidActual1)

				invalidActual2 := newActualLRP(
					newDesiredLRP(fmt.Sprintf("unknown-desired-without-valid-actual-%d", i), "domain", 1),
					1337,
				)
				invalidActual2.Since = 0
				testData.actualsToPrune.Add(invalidActual2)
			}
			for i := 0; i < 5; i++ {
				validActual := newActualLRP(
					newDesiredLRP(fmt.Sprintf("unknown-desired-with-valid-actual-%d", i), "domain", 1),
					9001,
				)
				testData.actualsToKeep.Add(validActual)

				invalidActual := newActualLRP(
					newDesiredLRP(fmt.Sprintf("unknown-desired-with-valid-actual-%d", i), "domain", 1),
					1337,
				)
				invalidActual.Since = 0
				testData.actualsToPrune.Add(invalidActual)
			}
			for i := 5; i < 10; i++ {
				validActual1 := newActualLRP(
					newDesiredLRP(fmt.Sprintf("unknown-desired-with-valid-actual-%d", i), "domain", 1),
					9001,
				)
				testData.actualsToKeep.Add(validActual1)

				validActual2 := newActualLRP(
					newDesiredLRP(fmt.Sprintf("unknown-desired-with-valid-actual-%d", i), "domain", 1),
					1337,
				)
				testData.actualsToKeep.Add(validActual2)
			}
			for i := 10; i < 15; i++ {
				validEvacuating := newActualLRP(
					newDesiredLRP(fmt.Sprintf("unknown-desired-with-valid-actual-%d", i), "domain", 1),
					9001,
				)
				testData.evacuationsToKeep.Add(validEvacuating)

				invalidActual := newActualLRP(
					newDesiredLRP(fmt.Sprintf("unknown-desired-with-valid-actual-%d", i), "domain", 1),
					9001,
				)
				invalidActual.Since = 0
				testData.actualsToPrune.Add(invalidActual)

				invalidEvacuating := newActualLRP(
					newDesiredLRP(fmt.Sprintf("unknown-desired-with-valid-actual-%d", i), "domain", 1),
					1337,
				)
				invalidEvacuating.Since = 0
				testData.evacuationsToPrune.Add(invalidEvacuating)

				validActual := newActualLRP(
					newDesiredLRP(fmt.Sprintf("unknown-desired-with-valid-actual-%d", i), "domain", 1),
					1337,
				)
				testData.actualsToKeep.Add(validActual)
			}

			// Domains
			testData.domains.Add(domain)

			// Cells
			testData.cells.Add(newCellPresence(cellID))
			testData.cells.Add(newCellPresence("other-cell"))

			saveTestData(testData)
		})

		var input *lrp_bbs.ConvergenceInput
		var gatherError error

		JustBeforeEach(func() {
			input, gatherError = bbs.GatherAndPruneLRPConvergenceInput(logger, servicesBBS.NewCellsLoader())
		})

		It("gets all processGuids in the system", func() {
			expectedGuids := map[string]struct{}{}

			// all valid DesiredLRPs' ProcessGuids should be present
			for i := 0; i < 200; i++ {
				expectedGuids[fmt.Sprintf("valid-desired-%d", i)] = struct{}{}
			}

			// invalid DesiredLRPs' ProcessGuids should be present if there
			// are corresponding valid ActualLRPs
			for i := 0; i < 15; i++ {
				expectedGuids[fmt.Sprintf("invalid-desired-with-valid-actual-%d", i)] = struct{}{}
			}

			// non-existent DesiredLRP ProcessGuids should be present if there
			// are corresponding valid ActualLRPs
			for i := 0; i < 15; i++ {
				expectedGuids[fmt.Sprintf("unknown-desired-with-valid-actual-%d", i)] = struct{}{}
			}

			Expect(input.AllProcessGuids).To(Equal(expectedGuids))
		})

		It("fetches the correct desired LRPs", func() {
			Expect(input.DesiredLRPs).To(HaveLen(len(testData.desiredsToKeep)))

			testData.desiredsToKeep.Each(func(expected models.DesiredLRP) {
				desired, ok := input.DesiredLRPs[expected.ProcessGuid]
				Expect(ok).To(BeTrue(), fmt.Sprintf("expected desiredLRP for process '%s' to be present", expected.ProcessGuid))
				Expect(desired).To(Equal(expected))
			})
		})

		It("prunes the correct desired LRPs", func() {
			testData.desiredsToPrune.Each(func(expected models.DesiredLRP) {
				_, err := bbs.DesiredLRPByProcessGuid(expected.ProcessGuid)
				Expect(err).To(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		})

		It("fetches the correct actualLRPs", func() {
			Expect(input.ActualLRPs).To(HaveLen(len(testData.actualsToKeep)))

			testData.actualsToKeep.Each(func(expected models.ActualLRP) {
				actualIndex, ok := input.ActualLRPs[expected.ProcessGuid]
				Expect(ok).To(BeTrue(), fmt.Sprintf("expected actualIndex for process '%s' to be present", expected.ProcessGuid))
				actual, ok := actualIndex[expected.Index]
				Expect(ok).To(BeTrue(), fmt.Sprintf("expected actual for process '%s' and index %d to be present", expected.ProcessGuid, expected.Index))
				Expect(actual).To(Equal(expected))
			})
		})

		It("prunes the correct actualLRPs", func() {
			for i := 0; i < 20; i++ {
				groups, err := bbs.ActualLRPGroupsByProcessGuid(fmt.Sprintf("valid-desired-%d", i))
				Expect(err).NotTo(HaveOccurred())
				Expect(groups).To(BeEmpty())
			}

			for i := 20; i < 40; i++ {
				groups, err := bbs.ActualLRPGroupsByProcessGuid(fmt.Sprintf("valid-desired-%d", i))
				Expect(err).NotTo(HaveOccurred())
				Expect(groups).To(HaveLen(1))
				Expect(groups).To(HaveKey(9001))
			}

			for i := 40; i < 60; i++ {
				groups, err := bbs.ActualLRPGroupsByProcessGuid(fmt.Sprintf("valid-desired-%d", i))
				Expect(err).NotTo(HaveOccurred())
				Expect(groups).To(HaveLen(2))
				Expect(groups).To(HaveKey(9001))
				Expect(groups).To(HaveKey(1337))
			}

			for i := 60; i < 80; i++ {
				group1, err := bbs.ActualLRPGroupByProcessGuidAndIndex(fmt.Sprintf("valid-desired-%d", i), 9001)
				Expect(err).NotTo(HaveOccurred())
				Expect(group1.Instance).To(BeNil())
				Expect(group1.Evacuating).NotTo(BeNil())

				group2, err := bbs.ActualLRPGroupByProcessGuidAndIndex(fmt.Sprintf("valid-desired-%d", i), 1337)
				Expect(err).NotTo(HaveOccurred())
				Expect(group2.Instance).NotTo(BeNil())
				Expect(group2.Evacuating).To(BeNil())
			}

			for i := 0; i < 5; i++ {
				groups, err := bbs.ActualLRPGroupsByProcessGuid(fmt.Sprintf("invalid-desired-without-valid-actual-%d", i))
				Expect(err).NotTo(HaveOccurred())
				Expect(groups).To(BeEmpty())
			}

			for i := 0; i < 5; i++ {
				groups, err := bbs.ActualLRPGroupsByProcessGuid(fmt.Sprintf("invalid-desired-with-valid-actual-%d", i))
				Expect(err).NotTo(HaveOccurred())
				Expect(groups).To(HaveLen(1))
				Expect(groups).To(HaveKey(9001))
			}

			for i := 5; i < 10; i++ {
				groups, err := bbs.ActualLRPGroupsByProcessGuid(fmt.Sprintf("invalid-desired-with-valid-actual-%d", i))
				Expect(err).NotTo(HaveOccurred())
				Expect(groups).To(HaveLen(2))
				Expect(groups).To(HaveKey(9001))
				Expect(groups).To(HaveKey(1337))
			}

			for i := 10; i < 15; i++ {
				group1, err := bbs.ActualLRPGroupByProcessGuidAndIndex(fmt.Sprintf("invalid-desired-with-valid-actual-%d", i), 9001)
				Expect(err).NotTo(HaveOccurred())
				Expect(group1.Instance).To(BeNil())
				Expect(group1.Evacuating).NotTo(BeNil())

				group2, err := bbs.ActualLRPGroupByProcessGuidAndIndex(fmt.Sprintf("invalid-desired-with-valid-actual-%d", i), 1337)
				Expect(err).NotTo(HaveOccurred())
				Expect(group2.Instance).NotTo(BeNil())
				Expect(group2.Evacuating).To(BeNil())
			}

			for i := 0; i < 5; i++ {
				groups, err := bbs.ActualLRPGroupsByProcessGuid(fmt.Sprintf("unknown-desired-without-valid-actual-%d", i))
				Expect(err).NotTo(HaveOccurred())
				Expect(groups).To(BeEmpty())
			}

			for i := 0; i < 5; i++ {
				groups, err := bbs.ActualLRPGroupsByProcessGuid(fmt.Sprintf("unknown-desired-with-valid-actual-%d", i))
				Expect(err).NotTo(HaveOccurred())
				Expect(groups).To(HaveLen(1))
				Expect(groups).To(HaveKey(9001))
			}

			for i := 5; i < 10; i++ {
				groups, err := bbs.ActualLRPGroupsByProcessGuid(fmt.Sprintf("unknown-desired-with-valid-actual-%d", i))
				Expect(err).NotTo(HaveOccurred())
				Expect(groups).To(HaveLen(2))
				Expect(groups).To(HaveKey(9001))
				Expect(groups).To(HaveKey(1337))
			}

			for i := 10; i < 15; i++ {
				group1, err := bbs.ActualLRPGroupByProcessGuidAndIndex(fmt.Sprintf("unknown-desired-with-valid-actual-%d", i), 9001)
				Expect(err).NotTo(HaveOccurred())
				Expect(group1.Instance).To(BeNil())
				Expect(group1.Evacuating).NotTo(BeNil())

				group2, err := bbs.ActualLRPGroupByProcessGuidAndIndex(fmt.Sprintf("unknown-desired-with-valid-actual-%d", i), 1337)
				Expect(err).NotTo(HaveOccurred())
				Expect(group2.Instance).NotTo(BeNil())
				Expect(group2.Evacuating).To(BeNil())
			}
		})

		It("gets all the domains", func() {
			Expect(input.Domains).To(HaveLen(len(testData.domains)))
			testData.domains.Each(func(domain string) {
				Expect(input.Domains).To(HaveKey(domain))
			})
		})

		It("gets all the cells", func() {
			Expect(input.Cells).To(HaveLen(len(testData.cells)))
			testData.cells.Each(func(cell models.CellPresence) {
				Expect(input.Cells).To(ContainElement(cell))
			})
		})
	})
})

func saveTestData(testData *testDataForConvergenceGatherer) {
	testData.desiredsToKeep.Each(func(desired models.DesiredLRP) {
		setRawDesiredLRP(desired)
	})

	testData.desiredsToPrune.Each(func(desired models.DesiredLRP) {
		createMalformedDesiredLRP(desired.ProcessGuid)
	})

	testData.actualsToKeep.Each(func(actual models.ActualLRP) {
		setRawActualLRP(actual)
	})

	testData.actualsToPrune.Each(func(actual models.ActualLRP) {
		setRawActualLRP(actual)
	})

	testData.evacuationsToKeep.Each(func(actual models.ActualLRP) {
		setRawEvacuatingActualLRP(actual, 100)
	})

	testData.evacuationsToPrune.Each(func(actual models.ActualLRP) {
		setRawEvacuatingActualLRP(actual, 100)
	})

	testData.domains.Each(func(domain string) {
		createRawDomain(domain)
	})

	testData.cells.Each(func(cell models.CellPresence) {
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
	err := etcdClient.Create(storeadapter.StoreNode{
		Key:   key,
		Value: []byte("ßßßßßß"),
	})

	Expect(err).NotTo(HaveOccurred())
}

func newCellPresence(cellID string) models.CellPresence {
	return models.NewCellPresence(cellID, "1.2.3.4", "az-1", models.CellCapacity{128, 1024, 3})
}

func newDesiredLRP(guid, domain string, instances int) models.DesiredLRP {
	return models.DesiredLRP{
		Domain:      domain,
		ProcessGuid: guid,
		Instances:   instances,
		RootFS:      "some:rootfs",
		MemoryMB:    1024,
		DiskMB:      512,
		CPUWeight:   42,
		Action:      &models.RunAction{Path: "ls"},
	}
}

func newActualLRP(d models.DesiredLRP, index int) models.ActualLRP {
	return models.ActualLRP{
		ActualLRPKey: models.NewActualLRPKey(d.ProcessGuid, index, d.Domain),
		State:        models.ActualLRPStateUnclaimed,
		Since:        1138,
	}
}

func newUnclaimedActualLRP(d models.DesiredLRP, index int) models.ActualLRP {
	return models.ActualLRP{
		ActualLRPKey: models.NewActualLRPKey(d.ProcessGuid, index, d.Domain),
		State:        models.ActualLRPStateUnclaimed,
		Since:        1138,
	}
}

func newClaimedActualLRP(d models.DesiredLRP, cellID string, index int) models.ActualLRP {
	return models.ActualLRP{
		ActualLRPKey:         models.NewActualLRPKey(d.ProcessGuid, index, d.Domain),
		ActualLRPInstanceKey: models.NewActualLRPInstanceKey("instance-guid", cellID),
		State:                models.ActualLRPStateClaimed,
		Since:                1138,
	}
}

func newRunningActualLRP(d models.DesiredLRP, cellID string, index int) models.ActualLRP {
	return models.ActualLRP{
		ActualLRPKey:         models.NewActualLRPKey(d.ProcessGuid, index, d.Domain),
		ActualLRPInstanceKey: models.NewActualLRPInstanceKey("instance-guid", cellID),
		ActualLRPNetInfo:     models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{}),
		State:                models.ActualLRPStateRunning,
		Since:                1138,
	}
}

func newStartableCrashedActualLRP(d models.DesiredLRP, index int) models.ActualLRP {
	return models.ActualLRP{
		ActualLRPKey: models.NewActualLRPKey(d.ProcessGuid, index, d.Domain),
		CrashCount:   1,
		State:        models.ActualLRPStateCrashed,
		Since:        1138,
	}
}

func newUnstartableCrashedActualLRP(d models.DesiredLRP, index int) models.ActualLRP {
	return models.ActualLRP{
		ActualLRPKey: models.NewActualLRPKey(d.ProcessGuid, index, d.Domain),
		CrashCount:   201,
		State:        models.ActualLRPStateCrashed,
		Since:        1138,
	}
}
