package lrp_bbs_test

import (
	"fmt"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
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
	instanceKeysToKeep    map[processGuidAndIndex]struct{}
	instanceKeysToPrune   map[processGuidAndIndex]struct{}
	evacuatingKeysToKeep  map[processGuidAndIndex]struct{}
	evacuatingKeysToPrune map[processGuidAndIndex]struct{}
	domains               models.DomainSet
	cells                 models.CellSet

	validDesiredGuidsWithSomeValidActuals     [300]string
	validDesiredGuidsWithNoActuals            [300]string
	validDesiredGuidsWithOnlyInvalidActuals   [300]string
	invalidDesiredGuidsWithSomeValidActuals   [30]string
	invalidDesiredGuidsWithNoActuals          [30]string
	invalidDesiredGuidsWithOnlyInvalidActuals [30]string
	unknownDesiredGuidsWithSomeValidActuals   [30]string
	unknownDesiredGuidsWithNoActuals          [30]string
	unknownDesiredGuidsWithOnlyInvalidActuals [30]string
}

func newTestDataForConvergenceGatherer() *testDataForConvergenceGatherer {
	return &testDataForConvergenceGatherer{
		instanceKeysToKeep:    map[processGuidAndIndex]struct{}{},
		instanceKeysToPrune:   map[processGuidAndIndex]struct{}{},
		evacuatingKeysToKeep:  map[processGuidAndIndex]struct{}{},
		evacuatingKeysToPrune: map[processGuidAndIndex]struct{}{},
		domains:               models.DomainSet{},
		cells:                 models.CellSet{},
	}
}

var _ = Describe("Convergence", func() {
	Describe("Gathering", func() {
		const cellID = "some-cell-id"
		const domain = "test-domain"

		// ActualLRPs with indices that don't make sense for their corresponding
		// DesiredLRPs are *not* pruned at this phase
		const randomIndex1 = 9001
		const randomIndex2 = 1337

		var testData *testDataForConvergenceGatherer

		BeforeEach(func() {
			testData = newTestDataForConvergenceGatherer()

			for i := range testData.validDesiredGuidsWithSomeValidActuals {
				guid := fmt.Sprintf("valid-desired-with-some-valid-actuals-%d", i)
				testData.validDesiredGuidsWithSomeValidActuals[i] = guid

				switch i % 3 {
				case 0:
					testData.instanceKeysToKeep[processGuidAndIndex{guid, randomIndex1}] = struct{}{}
					testData.instanceKeysToKeep[processGuidAndIndex{guid, randomIndex2}] = struct{}{}
				case 1:
					testData.instanceKeysToKeep[processGuidAndIndex{guid, randomIndex1}] = struct{}{}
					testData.instanceKeysToPrune[processGuidAndIndex{guid, randomIndex2}] = struct{}{}
				case 2:
					testData.evacuatingKeysToKeep[processGuidAndIndex{guid, randomIndex1}] = struct{}{}
					testData.instanceKeysToPrune[processGuidAndIndex{guid, randomIndex1}] = struct{}{}
					testData.evacuatingKeysToPrune[processGuidAndIndex{guid, randomIndex2}] = struct{}{}
					testData.instanceKeysToKeep[processGuidAndIndex{guid, randomIndex2}] = struct{}{}
				}
			}

			for i := range testData.validDesiredGuidsWithNoActuals {
				guid := fmt.Sprintf("valid-desired-with-no-actuals-%d", i)
				testData.validDesiredGuidsWithNoActuals[i] = guid
			}

			for i := range testData.validDesiredGuidsWithOnlyInvalidActuals {
				guid := fmt.Sprintf("valid-desired-with-only-invalid-actuals-%d", i)
				testData.validDesiredGuidsWithOnlyInvalidActuals[i] = guid

				testData.instanceKeysToPrune[processGuidAndIndex{guid, randomIndex1}] = struct{}{}
				testData.instanceKeysToPrune[processGuidAndIndex{guid, randomIndex2}] = struct{}{}
			}

			for i := range testData.invalidDesiredGuidsWithSomeValidActuals {
				guid := fmt.Sprintf("invalid-desired-with-some-valid-actuals-%d", i)
				testData.invalidDesiredGuidsWithSomeValidActuals[i] = guid

				switch i % 3 {
				case 0:
					testData.instanceKeysToKeep[processGuidAndIndex{guid, randomIndex1}] = struct{}{}
					testData.instanceKeysToKeep[processGuidAndIndex{guid, randomIndex2}] = struct{}{}
				case 1:
					testData.instanceKeysToKeep[processGuidAndIndex{guid, randomIndex1}] = struct{}{}
					testData.instanceKeysToPrune[processGuidAndIndex{guid, randomIndex2}] = struct{}{}
				case 2:
					testData.evacuatingKeysToKeep[processGuidAndIndex{guid, randomIndex1}] = struct{}{}
					testData.instanceKeysToPrune[processGuidAndIndex{guid, randomIndex1}] = struct{}{}
					testData.evacuatingKeysToPrune[processGuidAndIndex{guid, randomIndex2}] = struct{}{}
					testData.instanceKeysToKeep[processGuidAndIndex{guid, randomIndex2}] = struct{}{}
				}
			}

			for i := range testData.invalidDesiredGuidsWithNoActuals {
				guid := fmt.Sprintf("invalid-desired-with-no-actuals-%d", i)
				testData.invalidDesiredGuidsWithNoActuals[i] = guid
			}

			for i := range testData.invalidDesiredGuidsWithOnlyInvalidActuals {
				guid := fmt.Sprintf("invalid-desired-with-only-invalid-actuals-%d", i)
				testData.invalidDesiredGuidsWithOnlyInvalidActuals[i] = guid

				testData.instanceKeysToPrune[processGuidAndIndex{guid, randomIndex1}] = struct{}{}
				testData.instanceKeysToPrune[processGuidAndIndex{guid, randomIndex2}] = struct{}{}
			}

			for i := range testData.unknownDesiredGuidsWithSomeValidActuals {
				guid := fmt.Sprintf("unknown-desired-with-some-valid-actuals-%d", i)
				testData.unknownDesiredGuidsWithSomeValidActuals[i] = guid

				switch i % 3 {
				case 0:
					testData.instanceKeysToKeep[processGuidAndIndex{guid, randomIndex1}] = struct{}{}
					testData.instanceKeysToKeep[processGuidAndIndex{guid, randomIndex2}] = struct{}{}
				case 1:
					testData.instanceKeysToKeep[processGuidAndIndex{guid, randomIndex1}] = struct{}{}
					testData.instanceKeysToPrune[processGuidAndIndex{guid, randomIndex2}] = struct{}{}
				case 2:
					testData.evacuatingKeysToKeep[processGuidAndIndex{guid, randomIndex1}] = struct{}{}
					testData.instanceKeysToPrune[processGuidAndIndex{guid, randomIndex1}] = struct{}{}
					testData.evacuatingKeysToPrune[processGuidAndIndex{guid, randomIndex2}] = struct{}{}
					testData.instanceKeysToKeep[processGuidAndIndex{guid, randomIndex2}] = struct{}{}
				}
			}

			for i := range testData.unknownDesiredGuidsWithNoActuals {
				guid := fmt.Sprintf("unknown-desired-with-no-actuals-%d", i)
				testData.unknownDesiredGuidsWithNoActuals[i] = guid
			}

			for i := range testData.unknownDesiredGuidsWithOnlyInvalidActuals {
				guid := fmt.Sprintf("unknown-desired-with-only-invalid-actuals-%d", i)
				testData.unknownDesiredGuidsWithOnlyInvalidActuals[i] = guid

				testData.instanceKeysToPrune[processGuidAndIndex{guid, randomIndex1}] = struct{}{}
				testData.instanceKeysToPrune[processGuidAndIndex{guid, randomIndex2}] = struct{}{}
			}

			testData.domains = models.DomainSet{domain: struct{}{}}

			testData.cells = models.CellSet{
				cellID:       newCellPresence(cellID),
				"other-cell": newCellPresence("other-cell"),
			}

			createTestData(testData)
		})

		It("provides all processGuids in the system", func() {
			input, gatherError := bbs.GatherAndPruneLRPConvergenceInput(logger, servicesBBS.NewCellsLoader())
			Expect(gatherError).NotTo(HaveOccurred())

			expectedGuids := map[string]struct{}{}
			for _, desiredGuid := range testData.validDesiredGuidsWithSomeValidActuals {
				expectedGuids[desiredGuid] = struct{}{}
			}
			for _, desiredGuid := range testData.validDesiredGuidsWithNoActuals {
				expectedGuids[desiredGuid] = struct{}{}
			}
			for _, desiredGuid := range testData.validDesiredGuidsWithOnlyInvalidActuals {
				expectedGuids[desiredGuid] = struct{}{}
			}
			for _, desiredGuid := range testData.invalidDesiredGuidsWithSomeValidActuals {
				expectedGuids[desiredGuid] = struct{}{}
			}
			for _, desiredGuid := range testData.unknownDesiredGuidsWithSomeValidActuals {
				expectedGuids[desiredGuid] = struct{}{}
			}

			Expect(input.AllProcessGuids).To(Equal(expectedGuids))
		})

		It("provides the correct desired LRPs", func() {
			input, gatherError := bbs.GatherAndPruneLRPConvergenceInput(logger, servicesBBS.NewCellsLoader())
			Expect(gatherError).NotTo(HaveOccurred())

			expectedLength := len(testData.validDesiredGuidsWithSomeValidActuals) +
				len(testData.validDesiredGuidsWithNoActuals) +
				len(testData.validDesiredGuidsWithOnlyInvalidActuals)
			Expect(input.DesiredLRPs).To(HaveLen(expectedLength))

			for _, desiredGuid := range testData.validDesiredGuidsWithSomeValidActuals {
				Expect(input.DesiredLRPs).To(HaveKey(desiredGuid))
			}
			for _, desiredGuid := range testData.validDesiredGuidsWithNoActuals {
				Expect(input.DesiredLRPs).To(HaveKey(desiredGuid))
			}
			for _, desiredGuid := range testData.validDesiredGuidsWithOnlyInvalidActuals {
				Expect(input.DesiredLRPs).To(HaveKey(desiredGuid))
			}
		})

		It("prunes only the invalid DesiredLRPs from the datastore", func() {
			_, gatherError := bbs.GatherAndPruneLRPConvergenceInput(logger, servicesBBS.NewCellsLoader())
			Expect(gatherError).NotTo(HaveOccurred())

			for _, desiredGuid := range testData.validDesiredGuidsWithSomeValidActuals {
				_, err := bbs.DesiredLRPByProcessGuid(desiredGuid)
				Expect(err).NotTo(HaveOccurred())
			}
			for _, desiredGuid := range testData.validDesiredGuidsWithNoActuals {
				_, err := bbs.DesiredLRPByProcessGuid(desiredGuid)
				Expect(err).NotTo(HaveOccurred())
			}
			for _, desiredGuid := range testData.validDesiredGuidsWithOnlyInvalidActuals {
				_, err := bbs.DesiredLRPByProcessGuid(desiredGuid)
				Expect(err).NotTo(HaveOccurred())
			}

			for _, desiredGuid := range testData.invalidDesiredGuidsWithSomeValidActuals {
				_, err := bbs.DesiredLRPByProcessGuid(desiredGuid)
				Expect(err).To(Equal(bbserrors.ErrStoreResourceNotFound))
			}
			for _, desiredGuid := range testData.invalidDesiredGuidsWithNoActuals {
				_, err := bbs.DesiredLRPByProcessGuid(desiredGuid)
				Expect(err).To(Equal(bbserrors.ErrStoreResourceNotFound))
			}
			for _, desiredGuid := range testData.invalidDesiredGuidsWithOnlyInvalidActuals {
				_, err := bbs.DesiredLRPByProcessGuid(desiredGuid)
				Expect(err).To(Equal(bbserrors.ErrStoreResourceNotFound))
			}
		})

		It("provides the correct actualLRPs", func() {
			input, gatherError := bbs.GatherAndPruneLRPConvergenceInput(logger, servicesBBS.NewCellsLoader())
			Expect(gatherError).NotTo(HaveOccurred())

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
			_, gatherError := bbs.GatherAndPruneLRPConvergenceInput(logger, servicesBBS.NewCellsLoader())
			Expect(gatherError).NotTo(HaveOccurred())

			for _, guid := range testData.validDesiredGuidsWithOnlyInvalidActuals {
				groups, err := bbs.ActualLRPGroupsByProcessGuid(guid)
				Expect(err).NotTo(HaveOccurred())
				Expect(groups).To(BeEmpty())
			}

			for i, guid := range testData.validDesiredGuidsWithSomeValidActuals {
				switch i % 3 {
				case 0:
					groups, err := bbs.ActualLRPGroupsByProcessGuid(guid)
					Expect(err).NotTo(HaveOccurred())
					Expect(groups).To(HaveLen(2))
					Expect(groups).To(HaveKey(randomIndex1))
					Expect(groups).To(HaveKey(randomIndex2))
				case 1:
					groups, err := bbs.ActualLRPGroupsByProcessGuid(guid)
					Expect(err).NotTo(HaveOccurred())
					Expect(groups).To(HaveLen(1))
					Expect(groups).To(HaveKey(randomIndex1))
				case 2:
					group1, err := bbs.ActualLRPGroupByProcessGuidAndIndex(guid, randomIndex1)
					Expect(err).NotTo(HaveOccurred())
					Expect(group1.Instance).To(BeNil())
					Expect(group1.Evacuating).NotTo(BeNil())

					group2, err := bbs.ActualLRPGroupByProcessGuidAndIndex(guid, randomIndex2)
					Expect(err).NotTo(HaveOccurred())
					Expect(group2.Instance).NotTo(BeNil())
					Expect(group2.Evacuating).To(BeNil())
				}
			}

			for _, guid := range testData.invalidDesiredGuidsWithOnlyInvalidActuals {
				groups, err := bbs.ActualLRPGroupsByProcessGuid(guid)
				Expect(err).NotTo(HaveOccurred())
				Expect(groups).To(BeEmpty())
			}

			for i, guid := range testData.invalidDesiredGuidsWithSomeValidActuals {
				switch i % 3 {
				case 0:
					groups, err := bbs.ActualLRPGroupsByProcessGuid(guid)
					Expect(err).NotTo(HaveOccurred())
					Expect(groups).To(HaveLen(2))
					Expect(groups).To(HaveKey(randomIndex1))
					Expect(groups).To(HaveKey(randomIndex2))
				case 1:
					groups, err := bbs.ActualLRPGroupsByProcessGuid(guid)
					Expect(err).NotTo(HaveOccurred())
					Expect(groups).To(HaveLen(1))
					Expect(groups).To(HaveKey(randomIndex1))
				case 2:
					group1, err := bbs.ActualLRPGroupByProcessGuidAndIndex(guid, randomIndex1)
					Expect(err).NotTo(HaveOccurred())
					Expect(group1.Instance).To(BeNil())
					Expect(group1.Evacuating).NotTo(BeNil())

					group2, err := bbs.ActualLRPGroupByProcessGuidAndIndex(guid, randomIndex2)
					Expect(err).NotTo(HaveOccurred())
					Expect(group2.Instance).NotTo(BeNil())
					Expect(group2.Evacuating).To(BeNil())
				}
			}

			for _, guid := range testData.unknownDesiredGuidsWithOnlyInvalidActuals {
				groups, err := bbs.ActualLRPGroupsByProcessGuid(guid)
				Expect(err).NotTo(HaveOccurred())
				Expect(groups).To(BeEmpty())
			}

			for i, guid := range testData.unknownDesiredGuidsWithSomeValidActuals {
				switch i % 3 {
				case 0:
					groups, err := bbs.ActualLRPGroupsByProcessGuid(guid)
					Expect(err).NotTo(HaveOccurred())
					Expect(groups).To(HaveLen(2))
					Expect(groups).To(HaveKey(randomIndex1))
					Expect(groups).To(HaveKey(randomIndex2))
				case 1:
					groups, err := bbs.ActualLRPGroupsByProcessGuid(guid)
					Expect(err).NotTo(HaveOccurred())
					Expect(groups).To(HaveLen(1))
					Expect(groups).To(HaveKey(randomIndex1))
				case 2:
					group1, err := bbs.ActualLRPGroupByProcessGuidAndIndex(guid, randomIndex1)
					Expect(err).NotTo(HaveOccurred())
					Expect(group1.Instance).To(BeNil())
					Expect(group1.Evacuating).NotTo(BeNil())

					group2, err := bbs.ActualLRPGroupByProcessGuidAndIndex(guid, randomIndex2)
					Expect(err).NotTo(HaveOccurred())
					Expect(group2.Instance).NotTo(BeNil())
					Expect(group2.Evacuating).To(BeNil())
				}
			}
		})

		It("gets all the domains", func() {
			input, gatherError := bbs.GatherAndPruneLRPConvergenceInput(logger, servicesBBS.NewCellsLoader())
			Expect(gatherError).NotTo(HaveOccurred())

			Expect(input.Domains).To(HaveLen(len(testData.domains)))
			testData.domains.Each(func(domain string) {
				Expect(input.Domains).To(HaveKey(domain))
			})
		})

		It("gets all the cells", func() {
			input, gatherError := bbs.GatherAndPruneLRPConvergenceInput(logger, servicesBBS.NewCellsLoader())
			Expect(gatherError).NotTo(HaveOccurred())

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

	for _, guid := range testData.validDesiredGuidsWithSomeValidActuals {
		createValidDesiredLRP(guid)
	}
	for _, guid := range testData.validDesiredGuidsWithNoActuals {
		createValidDesiredLRP(guid)
	}
	for _, guid := range testData.validDesiredGuidsWithOnlyInvalidActuals {
		createValidDesiredLRP(guid)
	}

	for _, guid := range testData.invalidDesiredGuidsWithSomeValidActuals {
		createMalformedDesiredLRP(guid)
	}
	for _, guid := range testData.invalidDesiredGuidsWithNoActuals {
		createMalformedDesiredLRP(guid)
	}
	for _, guid := range testData.invalidDesiredGuidsWithOnlyInvalidActuals {
		createMalformedDesiredLRP(guid)
	}

	for _, guid := range testData.unknownDesiredGuidsWithSomeValidActuals {
		createMalformedDesiredLRP(guid)
	}
	for _, guid := range testData.unknownDesiredGuidsWithNoActuals {
		createMalformedDesiredLRP(guid)
	}
	for _, guid := range testData.unknownDesiredGuidsWithOnlyInvalidActuals {
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
