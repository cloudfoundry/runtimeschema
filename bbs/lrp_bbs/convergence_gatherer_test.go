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

type processGuidAndIndex struct {
	processGuid string
	index       int
}

type testDataForConvergenceGatherer struct {
	instanceKeysToKeep                     map[processGuidAndIndex]struct{}
	instanceKeysToPrune                    map[processGuidAndIndex]struct{}
	evacuatingKeysToKeep                   map[processGuidAndIndex]struct{}
	evacuatingKeysToPrune                  map[processGuidAndIndex]struct{}
	validDesiredGuids                      []string
	invalidDesiredGuidsWithValidActuals    []string
	invalidDesiredGuidsWithoutValidActuals []string
	unknownDesiredGuidsWithValidActuals    []string
	unknownDesiredGuidsWithoutValidActuals []string
	domains                                models.DomainSet
	cells                                  models.CellSet
}

func newTestDataForConvergenceGatherer() *testDataForConvergenceGatherer {
	return &testDataForConvergenceGatherer{
		instanceKeysToKeep:                     map[processGuidAndIndex]struct{}{},
		instanceKeysToPrune:                    map[processGuidAndIndex]struct{}{},
		evacuatingKeysToKeep:                   map[processGuidAndIndex]struct{}{},
		evacuatingKeysToPrune:                  map[processGuidAndIndex]struct{}{},
		validDesiredGuids:                      []string{},
		invalidDesiredGuidsWithValidActuals:    []string{},
		invalidDesiredGuidsWithoutValidActuals: []string{},
		unknownDesiredGuidsWithValidActuals:    []string{},
		unknownDesiredGuidsWithoutValidActuals: []string{},
		domains: models.DomainSet{},
		cells:   models.CellSet{},
	}
}

var _ = Describe("Convergence", func() {
	Describe("Gathering", func() {
		const cellID = "some-cell-id"
		const domain = "test-domain"
		const numValidDesiredGuids = 200
		const numInvalidDesiredGuidsWithValidActuals = 20
		const numInvalidDesiredGuidsWithoutValidActuals = 20
		const numUnknownDesiredGuidsWithValidActuals = 20
		const numUnknownDesiredGuidsWithoutValidActuals = 20

		// ActualLRPs with indices that don't make sense for their corresponding
		// DesiredLRPs are *not* pruned at this phase
		const randomIndex1 = 9001
		const randomIndex2 = 1337

		var testData *testDataForConvergenceGatherer

		BeforeEach(func() {
			testData = newTestDataForConvergenceGatherer()

			// create some valid DesiredLRP guids
			for i := 0; i < numValidDesiredGuids; i++ {
				testData.validDesiredGuids = append(
					testData.validDesiredGuids,
					fmt.Sprintf("valid-desired-%d", i),
				)
			}

			// create some invalid DesiredLRP guids that will have valid corresponding ActualLRPs
			for i := 0; i < numInvalidDesiredGuidsWithValidActuals; i++ {
				testData.invalidDesiredGuidsWithValidActuals = append(
					testData.invalidDesiredGuidsWithValidActuals,
					fmt.Sprintf("invalid-desired-with-valid-actual-%d", i),
				)
			}

			// create some invalid DesiredLRP guids that will NOT have valid corresponding ActualLRPs
			for i := 0; i < numInvalidDesiredGuidsWithoutValidActuals; i++ {
				testData.invalidDesiredGuidsWithoutValidActuals = append(
					testData.invalidDesiredGuidsWithoutValidActuals,
					fmt.Sprintf("invalid-desired-without-valid-actual-%d", i),
				)
			}

			// create some unknown DesiredLRP guids that will have valid corresponding ActualLRPs
			for i := 0; i < numUnknownDesiredGuidsWithValidActuals; i++ {
				testData.unknownDesiredGuidsWithValidActuals = append(
					testData.unknownDesiredGuidsWithValidActuals,
					fmt.Sprintf("unknown-desired-with-valid-actual-%d", i),
				)
			}

			// create some unknown DesiredLRP guids that will NOT have valid corresponding ActualLRPs
			for i := 0; i < numUnknownDesiredGuidsWithoutValidActuals; i++ {
				testData.unknownDesiredGuidsWithoutValidActuals = append(
					testData.unknownDesiredGuidsWithoutValidActuals,
					fmt.Sprintf("unknown-desired-without-valid-actual-%d", i),
				)
			}

			// create some ActualLRPs, corresponding to valid DesiredLRPs with all indices having
			// bad data
			for i := 0; i < numValidDesiredGuids/10; i++ {
				testData.instanceKeysToPrune[processGuidAndIndex{testData.validDesiredGuids[i], randomIndex1}] = struct{}{}

				testData.instanceKeysToPrune[processGuidAndIndex{testData.validDesiredGuids[i], randomIndex2}] = struct{}{}
			}

			// create some ActualLRPs, corresponding to valid DesiredLRPs
			// with some indices with bad data, and some with good data
			for i := numValidDesiredGuids / 10; i < 2*numValidDesiredGuids/10; i++ {
				testData.instanceKeysToKeep[processGuidAndIndex{testData.validDesiredGuids[i], randomIndex1}] = struct{}{}

				testData.instanceKeysToPrune[processGuidAndIndex{testData.validDesiredGuids[i], randomIndex2}] = struct{}{}
			}

			// create some ActualLRPs, corresponding to valid DesiredLRPs
			// with all indices having good data
			for i := 2 * numValidDesiredGuids / 10; i < 3*numValidDesiredGuids/10; i++ {
				testData.instanceKeysToKeep[processGuidAndIndex{testData.validDesiredGuids[i], randomIndex1}] = struct{}{}

				testData.instanceKeysToKeep[processGuidAndIndex{testData.validDesiredGuids[i], randomIndex2}] = struct{}{}
			}

			// create some ActualLRPs, corresponding to valid DesiredLRPs
			// with one index having good Evacuating data but bad Instance data,
			// another index having the opposite
			for i := 3 * numValidDesiredGuids / 10; i < 4*numValidDesiredGuids/10; i++ {
				testData.evacuatingKeysToKeep[processGuidAndIndex{testData.validDesiredGuids[i], randomIndex1}] = struct{}{}

				testData.instanceKeysToPrune[processGuidAndIndex{testData.validDesiredGuids[i], randomIndex1}] = struct{}{}

				testData.evacuatingKeysToPrune[processGuidAndIndex{testData.validDesiredGuids[i], randomIndex2}] = struct{}{}

				testData.instanceKeysToKeep[processGuidAndIndex{testData.validDesiredGuids[i], randomIndex2}] = struct{}{}
			}

			// create similar ActualLRPs corresponding to invalid DesiredLRPs
			for i := 0; i < numInvalidDesiredGuidsWithoutValidActuals/10; i++ {
				testData.instanceKeysToPrune[processGuidAndIndex{testData.invalidDesiredGuidsWithoutValidActuals[i], randomIndex1}] = struct{}{}

				testData.instanceKeysToPrune[processGuidAndIndex{testData.invalidDesiredGuidsWithoutValidActuals[i], randomIndex2}] = struct{}{}
			}
			for i := 0; i < numInvalidDesiredGuidsWithValidActuals/10; i++ {
				testData.instanceKeysToKeep[processGuidAndIndex{testData.invalidDesiredGuidsWithValidActuals[i], randomIndex1}] = struct{}{}

				testData.instanceKeysToPrune[processGuidAndIndex{testData.invalidDesiredGuidsWithValidActuals[i], randomIndex2}] = struct{}{}
			}
			for i := numInvalidDesiredGuidsWithValidActuals / 10; i < 2*numInvalidDesiredGuidsWithValidActuals/10; i++ {
				testData.instanceKeysToKeep[processGuidAndIndex{testData.invalidDesiredGuidsWithValidActuals[i], randomIndex1}] = struct{}{}

				testData.instanceKeysToKeep[processGuidAndIndex{testData.invalidDesiredGuidsWithValidActuals[i], randomIndex2}] = struct{}{}
			}
			for i := 2 * numInvalidDesiredGuidsWithValidActuals / 10; i < numInvalidDesiredGuidsWithValidActuals; i++ {
				testData.evacuatingKeysToKeep[processGuidAndIndex{testData.invalidDesiredGuidsWithValidActuals[i], randomIndex1}] = struct{}{}

				testData.instanceKeysToPrune[processGuidAndIndex{testData.invalidDesiredGuidsWithValidActuals[i], randomIndex1}] = struct{}{}

				testData.evacuatingKeysToPrune[processGuidAndIndex{testData.invalidDesiredGuidsWithValidActuals[i], randomIndex2}] = struct{}{}

				testData.instanceKeysToKeep[processGuidAndIndex{testData.invalidDesiredGuidsWithValidActuals[i], randomIndex2}] = struct{}{}
			}

			// create similar ActualLRPs corresponding to unknown DesiredLRPs
			for i := 0; i < numUnknownDesiredGuidsWithoutValidActuals/10; i++ {
				testData.instanceKeysToPrune[processGuidAndIndex{testData.unknownDesiredGuidsWithoutValidActuals[i], randomIndex1}] = struct{}{}

				testData.instanceKeysToPrune[processGuidAndIndex{testData.unknownDesiredGuidsWithoutValidActuals[i], randomIndex2}] = struct{}{}
			}
			for i := 0; i < numUnknownDesiredGuidsWithValidActuals/10; i++ {
				testData.instanceKeysToKeep[processGuidAndIndex{testData.unknownDesiredGuidsWithValidActuals[i], randomIndex1}] = struct{}{}

				testData.instanceKeysToPrune[processGuidAndIndex{testData.unknownDesiredGuidsWithValidActuals[i], randomIndex2}] = struct{}{}
			}
			for i := numUnknownDesiredGuidsWithValidActuals / 10; i < 2*numUnknownDesiredGuidsWithValidActuals/10; i++ {
				testData.instanceKeysToKeep[processGuidAndIndex{testData.unknownDesiredGuidsWithValidActuals[i], randomIndex1}] = struct{}{}

				testData.instanceKeysToKeep[processGuidAndIndex{testData.unknownDesiredGuidsWithValidActuals[i], randomIndex2}] = struct{}{}
			}
			for i := 2 * numUnknownDesiredGuidsWithValidActuals / 10; i < numUnknownDesiredGuidsWithValidActuals; i++ {
				testData.evacuatingKeysToKeep[processGuidAndIndex{testData.unknownDesiredGuidsWithValidActuals[i], randomIndex1}] = struct{}{}

				testData.instanceKeysToPrune[processGuidAndIndex{testData.unknownDesiredGuidsWithValidActuals[i], randomIndex1}] = struct{}{}

				testData.evacuatingKeysToPrune[processGuidAndIndex{testData.unknownDesiredGuidsWithValidActuals[i], randomIndex2}] = struct{}{}

				testData.instanceKeysToKeep[processGuidAndIndex{testData.unknownDesiredGuidsWithValidActuals[i], randomIndex2}] = struct{}{}
			}

			// Domains
			testData.domains = models.DomainSet{domain: struct{}{}}

			// Cells
			testData.cells = models.CellSet{
				cellID:       newCellPresence(cellID),
				"other-cell": newCellPresence("other-cell"),
			}

			createTestData(testData)
		})

		var input *lrp_bbs.ConvergenceInput
		var gatherError error

		JustBeforeEach(func() {
			input, gatherError = bbs.GatherAndPruneLRPConvergenceInput(logger, servicesBBS.NewCellsLoader())
		})

		It("provides all processGuids in the system", func() {
			expectedGuids := map[string]struct{}{}
			for _, desiredGuid := range testData.validDesiredGuids {
				expectedGuids[desiredGuid] = struct{}{}
			}
			for _, desiredGuid := range testData.invalidDesiredGuidsWithValidActuals {
				expectedGuids[desiredGuid] = struct{}{}
			}
			for _, desiredGuid := range testData.unknownDesiredGuidsWithValidActuals {
				expectedGuids[desiredGuid] = struct{}{}
			}

			Expect(input.AllProcessGuids).To(Equal(expectedGuids))
		})

		It("provides the correct desired LRPs", func() {
			Expect(input.DesiredLRPs).To(HaveLen(len(testData.validDesiredGuids)))

			for _, desiredGuid := range testData.validDesiredGuids {
				_, ok := input.DesiredLRPs[desiredGuid]
				Expect(ok).To(BeTrue(), fmt.Sprintf("expected desiredLRP for process '%s' to be present", desiredGuid))
			}
		})

		It("prunes only the invalid DesiredLRPs from the datastore", func() {
			for _, desiredGuid := range testData.validDesiredGuids {
				_, err := bbs.DesiredLRPByProcessGuid(desiredGuid)
				Expect(err).NotTo(HaveOccurred())
			}

			for _, desiredGuid := range testData.invalidDesiredGuidsWithValidActuals {
				_, err := bbs.DesiredLRPByProcessGuid(desiredGuid)
				Expect(err).To(Equal(bbserrors.ErrStoreResourceNotFound))
			}

			for _, desiredGuid := range testData.invalidDesiredGuidsWithoutValidActuals {
				_, err := bbs.DesiredLRPByProcessGuid(desiredGuid)
				Expect(err).To(Equal(bbserrors.ErrStoreResourceNotFound))
			}
		})

		It("provides the correct actualLRPs", func() {
			for actualData := range testData.instanceKeysToKeep {
				actualIndex, ok := input.ActualLRPs[actualData.processGuid]
				Expect(ok).To(BeTrue(), fmt.Sprintf("expected actualIndex for process '%s' to be present", actualData.processGuid))

				_, ok = actualIndex[actualData.index]
				Expect(ok).To(BeTrue(), fmt.Sprintf("expected actual for process '%s' and index %d to be present", actualData.processGuid, actualData.index))
			}

			for guid, actuals := range input.ActualLRPs {
				for index, _ := range actuals {
					_, ok := testData.instanceKeysToKeep[processGuidAndIndex{guid, index}]
					Expect(ok).To(BeTrue(), fmt.Sprintf("did not expect actual for process '%s' and index %d to be present", guid, index))
				}
			}
		})

		It("prunes only the invalid ActualLRPs from the datastore", func() {
			for i := 0; i < numValidDesiredGuids/10; i++ {
				groups, err := bbs.ActualLRPGroupsByProcessGuid(testData.validDesiredGuids[i])
				Expect(err).NotTo(HaveOccurred())
				Expect(groups).To(BeEmpty())
			}

			for i := numValidDesiredGuids / 10; i < 2*numValidDesiredGuids/10; i++ {
				groups, err := bbs.ActualLRPGroupsByProcessGuid(testData.validDesiredGuids[i])
				Expect(err).NotTo(HaveOccurred())
				Expect(groups).To(HaveLen(1))
				Expect(groups).To(HaveKey(randomIndex1))
			}

			for i := 2 * numValidDesiredGuids / 10; i < 3*numValidDesiredGuids/10; i++ {
				groups, err := bbs.ActualLRPGroupsByProcessGuid(testData.validDesiredGuids[i])
				Expect(err).NotTo(HaveOccurred())
				Expect(groups).To(HaveLen(2))
				Expect(groups).To(HaveKey(randomIndex1))
				Expect(groups).To(HaveKey(randomIndex2))
			}

			for i := 3 * numValidDesiredGuids / 10; i < 4*numValidDesiredGuids/10; i++ {
				group1, err := bbs.ActualLRPGroupByProcessGuidAndIndex(testData.validDesiredGuids[i], randomIndex1)
				Expect(err).NotTo(HaveOccurred())
				Expect(group1.Instance).To(BeNil())
				Expect(group1.Evacuating).NotTo(BeNil())

				group2, err := bbs.ActualLRPGroupByProcessGuidAndIndex(testData.validDesiredGuids[i], randomIndex2)
				Expect(err).NotTo(HaveOccurred())
				Expect(group2.Instance).NotTo(BeNil())
				Expect(group2.Evacuating).To(BeNil())
			}

			for i := 0; i < numInvalidDesiredGuidsWithoutValidActuals/10; i++ {
				groups, err := bbs.ActualLRPGroupsByProcessGuid(testData.invalidDesiredGuidsWithoutValidActuals[i])
				Expect(err).NotTo(HaveOccurred())
				Expect(groups).To(BeEmpty())
			}

			for i := 0; i < numInvalidDesiredGuidsWithValidActuals/10; i++ {
				groups, err := bbs.ActualLRPGroupsByProcessGuid(testData.invalidDesiredGuidsWithValidActuals[i])
				Expect(err).NotTo(HaveOccurred())
				Expect(groups).To(HaveLen(1))
				Expect(groups).To(HaveKey(randomIndex1))
			}

			for i := numInvalidDesiredGuidsWithValidActuals / 10; i < 2*numInvalidDesiredGuidsWithValidActuals/10; i++ {
				groups, err := bbs.ActualLRPGroupsByProcessGuid(testData.invalidDesiredGuidsWithValidActuals[i])
				Expect(err).NotTo(HaveOccurred())
				Expect(groups).To(HaveLen(2))
				Expect(groups).To(HaveKey(randomIndex1))
				Expect(groups).To(HaveKey(randomIndex2))
			}

			for i := 2 * numInvalidDesiredGuidsWithValidActuals / 10; i < 3*numInvalidDesiredGuidsWithValidActuals/10; i++ {
				group1, err := bbs.ActualLRPGroupByProcessGuidAndIndex(testData.invalidDesiredGuidsWithValidActuals[i], randomIndex1)
				Expect(err).NotTo(HaveOccurred())
				Expect(group1.Instance).To(BeNil())
				Expect(group1.Evacuating).NotTo(BeNil())

				group2, err := bbs.ActualLRPGroupByProcessGuidAndIndex(testData.invalidDesiredGuidsWithValidActuals[i], randomIndex2)
				Expect(err).NotTo(HaveOccurred())
				Expect(group2.Instance).NotTo(BeNil())
				Expect(group2.Evacuating).To(BeNil())
			}

			for i := 0; i < numUnknownDesiredGuidsWithoutValidActuals/10; i++ {
				groups, err := bbs.ActualLRPGroupsByProcessGuid(testData.unknownDesiredGuidsWithoutValidActuals[i])
				Expect(err).NotTo(HaveOccurred())
				Expect(groups).To(BeEmpty())
			}

			for i := 0; i < numUnknownDesiredGuidsWithValidActuals/10; i++ {
				groups, err := bbs.ActualLRPGroupsByProcessGuid(testData.unknownDesiredGuidsWithValidActuals[i])
				Expect(err).NotTo(HaveOccurred())
				Expect(groups).To(HaveLen(1))
				Expect(groups).To(HaveKey(randomIndex1))
			}

			for i := numUnknownDesiredGuidsWithValidActuals / 10; i < 2*numUnknownDesiredGuidsWithValidActuals/10; i++ {
				groups, err := bbs.ActualLRPGroupsByProcessGuid(testData.unknownDesiredGuidsWithValidActuals[i])
				Expect(err).NotTo(HaveOccurred())
				Expect(groups).To(HaveLen(2))
				Expect(groups).To(HaveKey(randomIndex1))
				Expect(groups).To(HaveKey(randomIndex2))
			}

			for i := 2 * numUnknownDesiredGuidsWithValidActuals / 10; i < 3*numUnknownDesiredGuidsWithValidActuals/10; i++ {
				group1, err := bbs.ActualLRPGroupByProcessGuidAndIndex(testData.unknownDesiredGuidsWithValidActuals[i], randomIndex1)
				Expect(err).NotTo(HaveOccurred())
				Expect(group1.Instance).To(BeNil())
				Expect(group1.Evacuating).NotTo(BeNil())

				group2, err := bbs.ActualLRPGroupByProcessGuidAndIndex(testData.unknownDesiredGuidsWithValidActuals[i], randomIndex2)
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

func createTestData(testData *testDataForConvergenceGatherer) {
	for actualData := range testData.instanceKeysToKeep {
		createValidActualLRP(actualData.processGuid, actualData.index)
	}
	for actualData := range testData.instanceKeysToPrune {
		createMalformedActualLRP(actualData.processGuid, actualData.index)
	}
	for actualData := range testData.evacuatingKeysToKeep {
		createValidEvacuatingLRP(actualData.processGuid, actualData.index)
	}
	for actualData := range testData.evacuatingKeysToPrune {
		createMalformedEvacuatingLRP(actualData.processGuid, actualData.index)
	}
	for _, guid := range testData.validDesiredGuids {
		createValidDesiredLRP(guid)
	}
	for _, guid := range testData.invalidDesiredGuidsWithValidActuals {
		createMalformedDesiredLRP(guid)
	}
	for _, guid := range testData.invalidDesiredGuidsWithoutValidActuals {
		createMalformedDesiredLRP(guid)
	}

	testData.domains.Each(func(domain string) {
		createRawDomain(domain)
	})

	testData.cells.Each(func(cell models.CellPresence) {
		registerCell(cell)
	})
}

func createValidDesiredLRP(guid string) {
	setRawDesiredLRP(models.DesiredLRP{
		ProcessGuid: guid,
		Domain:      "some-domain",
		Instances:   1,
		RootFS:      "some:rootfs",
		MemoryMB:    1024,
		DiskMB:      512,
		CPUWeight:   42,
		Action:      &models.RunAction{Path: "ls"},
	})
}

func createMalformedDesiredLRP(guid string) {
	createMalformedValueForKey(shared.DesiredLRPSchemaPath(models.DesiredLRP{ProcessGuid: guid}))
}

func createValidActualLRP(guid string, index int) {
	setRawActualLRP(models.ActualLRP{
		ActualLRPKey: models.NewActualLRPKey(guid, index, "some-domain"),
		State:        models.ActualLRPStateUnclaimed,
		Since:        1138,
	})
}

func createMalformedActualLRP(guid string, index int) {
	createMalformedValueForKey(shared.ActualLRPSchemaPath(guid, index))
}

func createValidEvacuatingLRP(guid string, index int) {
	setRawEvacuatingActualLRP(
		models.ActualLRP{
			ActualLRPKey: models.NewActualLRPKey(guid, index, "some-domain"),
			State:        models.ActualLRPStateUnclaimed,
			Since:        1138,
		},
		100,
	)
}

func createMalformedEvacuatingLRP(guid string, index int) {
	createMalformedValueForKey(shared.EvacuatingActualLRPSchemaPath(guid, index))
}

func createMalformedValueForKey(key string) {
	err := etcdClient.Create(storeadapter.StoreNode{
		Key:   key,
		Value: []byte("ßßßßßß"),
	})

	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("error occurred at key '%s'", key))
}

func newCellPresence(cellID string) models.CellPresence {
	return models.NewCellPresence(cellID, "1.2.3.4", "az-1", models.CellCapacity{128, 1024, 3})
}
