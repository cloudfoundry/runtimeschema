package lrp_bbs_test

import (
	"fmt"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/dropsonde/metric_sender/fake"
	"github.com/cloudfoundry/dropsonde/metrics"

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

	validDesiredGuidsWithSomeValidActuals     []string
	validDesiredGuidsWithNoActuals            []string
	validDesiredGuidsWithOnlyInvalidActuals   []string
	invalidDesiredGuidsWithSomeValidActuals   []string
	invalidDesiredGuidsWithNoActuals          []string
	invalidDesiredGuidsWithOnlyInvalidActuals []string
	unknownDesiredGuidsWithSomeValidActuals   []string
	unknownDesiredGuidsWithNoActuals          []string
	unknownDesiredGuidsWithOnlyInvalidActuals []string
}

const cellID = "some-cell-id"
const domain = "test-domain"

// ActualLRPs with indices that don't make sense for their corresponding
// DesiredLRPs are *not* pruned at this phase
const randomIndex1 = 9001
const randomIndex2 = 1337

var _ = Describe("Convergence", func() {
	var metricSender *fake.FakeMetricSender
	var testData *testDataForConvergenceGatherer

	Describe("Gathering Behaviour", func() {
		BeforeEach(func() {
			metricSender = fake.NewFakeMetricSender()
			metrics.Initialize(metricSender)
			testData = createTestData(3, 1, 1, 3, 1, 1, 3, 1, 1)
		})

		It("provides all processGuids in the system", func() {
			input, gatherError := lrpBBS.GatherAndPruneLRPConvergenceInput(logger, servicesBBS.NewCellsLoader())
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
			input, gatherError := lrpBBS.GatherAndPruneLRPConvergenceInput(logger, servicesBBS.NewCellsLoader())
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
			_, gatherError := lrpBBS.GatherAndPruneLRPConvergenceInput(logger, servicesBBS.NewCellsLoader())
			Expect(gatherError).NotTo(HaveOccurred())

			for _, desiredGuid := range testData.validDesiredGuidsWithSomeValidActuals {
				_, err := lrpBBS.DesiredLRPByProcessGuid(desiredGuid)
				Expect(err).NotTo(HaveOccurred())
			}
			for _, desiredGuid := range testData.validDesiredGuidsWithNoActuals {
				_, err := lrpBBS.DesiredLRPByProcessGuid(desiredGuid)
				Expect(err).NotTo(HaveOccurred())
			}
			for _, desiredGuid := range testData.validDesiredGuidsWithOnlyInvalidActuals {
				_, err := lrpBBS.DesiredLRPByProcessGuid(desiredGuid)
				Expect(err).NotTo(HaveOccurred())
			}

			for _, desiredGuid := range testData.invalidDesiredGuidsWithSomeValidActuals {
				_, err := lrpBBS.DesiredLRPByProcessGuid(desiredGuid)
				Expect(err).To(Equal(bbserrors.ErrStoreResourceNotFound))
			}
			for _, desiredGuid := range testData.invalidDesiredGuidsWithNoActuals {
				_, err := lrpBBS.DesiredLRPByProcessGuid(desiredGuid)
				Expect(err).To(Equal(bbserrors.ErrStoreResourceNotFound))
			}
			for _, desiredGuid := range testData.invalidDesiredGuidsWithOnlyInvalidActuals {
				_, err := lrpBBS.DesiredLRPByProcessGuid(desiredGuid)
				Expect(err).To(Equal(bbserrors.ErrStoreResourceNotFound))
			}
		})

		It("emits a metric for the number of pruned DesiredLRPs", func() {
			_, gatherError := lrpBBS.GatherAndPruneLRPConvergenceInput(logger, servicesBBS.NewCellsLoader())
			Expect(gatherError).NotTo(HaveOccurred())

			expectedMetric := len(testData.invalidDesiredGuidsWithSomeValidActuals) +
				len(testData.invalidDesiredGuidsWithNoActuals) +
				len(testData.invalidDesiredGuidsWithOnlyInvalidActuals) +
				len(testData.unknownDesiredGuidsWithSomeValidActuals) +
				len(testData.unknownDesiredGuidsWithNoActuals) +
				len(testData.unknownDesiredGuidsWithOnlyInvalidActuals)
			Expect(metricSender.GetCounter("ConvergenceLRPPreProcessingDesiredLRPsDeleted")).To(BeNumerically("==", expectedMetric))
		})

		It("emits a metric for the number of pruned ActualLRPs", func() {
			_, gatherError := lrpBBS.GatherAndPruneLRPConvergenceInput(logger, servicesBBS.NewCellsLoader())
			Expect(gatherError).NotTo(HaveOccurred())

			expectedMetric := len(testData.instanceKeysToPrune) +
				len(testData.evacuatingKeysToPrune)
			Expect(metricSender.GetCounter("ConvergenceLRPPreProcessingActualLRPsDeleted")).To(BeNumerically("==", expectedMetric))
		})

		It("provides the correct actualLRPs", func() {
			input, gatherError := lrpBBS.GatherAndPruneLRPConvergenceInput(logger, servicesBBS.NewCellsLoader())
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
			_, gatherError := lrpBBS.GatherAndPruneLRPConvergenceInput(logger, servicesBBS.NewCellsLoader())
			Expect(gatherError).NotTo(HaveOccurred())

			for _, guid := range testData.validDesiredGuidsWithOnlyInvalidActuals {
				groups, err := lrpBBS.ActualLRPGroupsByProcessGuid(guid)
				Expect(err).NotTo(HaveOccurred())
				Expect(groups).To(BeEmpty())
			}

			for i, guid := range testData.validDesiredGuidsWithSomeValidActuals {
				switch i % 3 {
				case 0:
					groups, err := lrpBBS.ActualLRPGroupsByProcessGuid(guid)
					Expect(err).NotTo(HaveOccurred())
					Expect(groups).To(HaveLen(2))
					Expect(groups).To(HaveKey(randomIndex1))
					Expect(groups).To(HaveKey(randomIndex2))
				case 1:
					groups, err := lrpBBS.ActualLRPGroupsByProcessGuid(guid)
					Expect(err).NotTo(HaveOccurred())
					Expect(groups).To(HaveLen(1))
					Expect(groups).To(HaveKey(randomIndex1))
				case 2:
					group1, err := lrpBBS.ActualLRPGroupByProcessGuidAndIndex(guid, randomIndex1)
					Expect(err).NotTo(HaveOccurred())
					Expect(group1.Instance).To(BeNil())
					Expect(group1.Evacuating).NotTo(BeNil())

					group2, err := lrpBBS.ActualLRPGroupByProcessGuidAndIndex(guid, randomIndex2)
					Expect(err).NotTo(HaveOccurred())
					Expect(group2.Instance).NotTo(BeNil())
					Expect(group2.Evacuating).To(BeNil())
				}
			}

			for _, guid := range testData.invalidDesiredGuidsWithOnlyInvalidActuals {
				groups, err := lrpBBS.ActualLRPGroupsByProcessGuid(guid)
				Expect(err).NotTo(HaveOccurred())
				Expect(groups).To(BeEmpty())
			}

			for i, guid := range testData.invalidDesiredGuidsWithSomeValidActuals {
				switch i % 3 {
				case 0:
					groups, err := lrpBBS.ActualLRPGroupsByProcessGuid(guid)
					Expect(err).NotTo(HaveOccurred())
					Expect(groups).To(HaveLen(2))
					Expect(groups).To(HaveKey(randomIndex1))
					Expect(groups).To(HaveKey(randomIndex2))
				case 1:
					groups, err := lrpBBS.ActualLRPGroupsByProcessGuid(guid)
					Expect(err).NotTo(HaveOccurred())
					Expect(groups).To(HaveLen(1))
					Expect(groups).To(HaveKey(randomIndex1))
				case 2:
					group1, err := lrpBBS.ActualLRPGroupByProcessGuidAndIndex(guid, randomIndex1)
					Expect(err).NotTo(HaveOccurred())
					Expect(group1.Instance).To(BeNil())
					Expect(group1.Evacuating).NotTo(BeNil())

					group2, err := lrpBBS.ActualLRPGroupByProcessGuidAndIndex(guid, randomIndex2)
					Expect(err).NotTo(HaveOccurred())
					Expect(group2.Instance).NotTo(BeNil())
					Expect(group2.Evacuating).To(BeNil())
				}
			}

			for _, guid := range testData.unknownDesiredGuidsWithOnlyInvalidActuals {
				groups, err := lrpBBS.ActualLRPGroupsByProcessGuid(guid)
				Expect(err).NotTo(HaveOccurred())
				Expect(groups).To(BeEmpty())
			}

			for i, guid := range testData.unknownDesiredGuidsWithSomeValidActuals {
				switch i % 3 {
				case 0:
					groups, err := lrpBBS.ActualLRPGroupsByProcessGuid(guid)
					Expect(err).NotTo(HaveOccurred())
					Expect(groups).To(HaveLen(2))
					Expect(groups).To(HaveKey(randomIndex1))
					Expect(groups).To(HaveKey(randomIndex2))
				case 1:
					groups, err := lrpBBS.ActualLRPGroupsByProcessGuid(guid)
					Expect(err).NotTo(HaveOccurred())
					Expect(groups).To(HaveLen(1))
					Expect(groups).To(HaveKey(randomIndex1))
				case 2:
					group1, err := lrpBBS.ActualLRPGroupByProcessGuidAndIndex(guid, randomIndex1)
					Expect(err).NotTo(HaveOccurred())
					Expect(group1.Instance).To(BeNil())
					Expect(group1.Evacuating).NotTo(BeNil())

					group2, err := lrpBBS.ActualLRPGroupByProcessGuidAndIndex(guid, randomIndex2)
					Expect(err).NotTo(HaveOccurred())
					Expect(group2.Instance).NotTo(BeNil())
					Expect(group2.Evacuating).To(BeNil())
				}
			}
		})

		It("gets all the domains", func() {
			input, gatherError := lrpBBS.GatherAndPruneLRPConvergenceInput(logger, servicesBBS.NewCellsLoader())
			Expect(gatherError).NotTo(HaveOccurred())

			Expect(input.Domains).To(HaveLen(len(testData.domains)))
			testData.domains.Each(func(domain string) {
				Expect(input.Domains).To(HaveKey(domain))
			})
		})

		It("gets all the cells", func() {
			input, gatherError := lrpBBS.GatherAndPruneLRPConvergenceInput(logger, servicesBBS.NewCellsLoader())
			Expect(gatherError).NotTo(HaveOccurred())

			Expect(input.Cells).To(HaveLen(len(testData.cells)))
			testData.cells.Each(func(cell models.CellPresence) {
				Expect(input.Cells).To(ContainElement(cell))
			})
		})

		Context("when root nodes are missing", func() {
			BeforeEach(func() {
				etcdRunner.Reset()
				consulRunner.Reset()
			})

			It("does not error", func() {
				_, gatherError := lrpBBS.GatherAndPruneLRPConvergenceInput(logger, servicesBBS.NewCellsLoader())
				Expect(gatherError).NotTo(HaveOccurred())
			})

			It("returns empty convergence input", func() {
				input, _ := lrpBBS.GatherAndPruneLRPConvergenceInput(logger, servicesBBS.NewCellsLoader())

				Expect(input.AllProcessGuids).To(BeEmpty())
				Expect(input.DesiredLRPs).To(BeEmpty())
				Expect(input.ActualLRPs).To(BeEmpty())
				Expect(input.Domains).To(BeEmpty())
				Expect(input.Cells).To(BeEmpty())
			})
		})
	})
})

func createTestData(
	numValidDesiredGuidsWithSomeValidActuals,
	numValidDesiredGuidsWithNoActuals,
	numValidDesiredGuidsWithOnlyInvalidActuals,
	numInvalidDesiredGuidsWithSomeValidActuals,
	numInvalidDesiredGuidsWithNoActuals,
	numInvalidDesiredGuidsWithOnlyInvalidActuals,
	numUnknownDesiredGuidsWithSomeValidActuals,
	numUnknownDesiredGuidsWithNoActuals,
	numUnknownDesiredGuidsWithOnlyInvalidActuals int,
) *testDataForConvergenceGatherer {
	testData := &testDataForConvergenceGatherer{
		instanceKeysToKeep:    map[processGuidAndIndex]struct{}{},
		instanceKeysToPrune:   map[processGuidAndIndex]struct{}{},
		evacuatingKeysToKeep:  map[processGuidAndIndex]struct{}{},
		evacuatingKeysToPrune: map[processGuidAndIndex]struct{}{},
		domains:               models.DomainSet{},
		cells:                 models.CellSet{},

		validDesiredGuidsWithSomeValidActuals:     []string{},
		validDesiredGuidsWithNoActuals:            []string{},
		validDesiredGuidsWithOnlyInvalidActuals:   []string{},
		invalidDesiredGuidsWithSomeValidActuals:   []string{},
		invalidDesiredGuidsWithNoActuals:          []string{},
		invalidDesiredGuidsWithOnlyInvalidActuals: []string{},
		unknownDesiredGuidsWithSomeValidActuals:   []string{},
		unknownDesiredGuidsWithNoActuals:          []string{},
		unknownDesiredGuidsWithOnlyInvalidActuals: []string{},
	}

	for i := 0; i < numValidDesiredGuidsWithSomeValidActuals; i++ {
		guid := fmt.Sprintf("valid-desired-with-some-valid-actuals-%d", i)
		testData.validDesiredGuidsWithSomeValidActuals = append(
			testData.validDesiredGuidsWithSomeValidActuals,
			guid,
		)

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

	for i := 0; i < numValidDesiredGuidsWithNoActuals; i++ {
		guid := fmt.Sprintf("valid-desired-with-no-actuals-%d", i)
		testData.validDesiredGuidsWithNoActuals = append(
			testData.validDesiredGuidsWithNoActuals,
			guid,
		)
	}

	for i := 0; i < numValidDesiredGuidsWithOnlyInvalidActuals; i++ {
		guid := fmt.Sprintf("valid-desired-with-only-invalid-actuals-%d", i)
		testData.validDesiredGuidsWithOnlyInvalidActuals = append(
			testData.validDesiredGuidsWithOnlyInvalidActuals,
			guid,
		)

		testData.instanceKeysToPrune[processGuidAndIndex{guid, randomIndex1}] = struct{}{}
		testData.instanceKeysToPrune[processGuidAndIndex{guid, randomIndex2}] = struct{}{}
	}

	for i := 0; i < numInvalidDesiredGuidsWithSomeValidActuals; i++ {
		guid := fmt.Sprintf("invalid-desired-with-some-valid-actuals-%d", i)
		testData.invalidDesiredGuidsWithSomeValidActuals = append(
			testData.invalidDesiredGuidsWithSomeValidActuals,
			guid,
		)

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

	for i := 0; i < numInvalidDesiredGuidsWithNoActuals; i++ {
		guid := fmt.Sprintf("invalid-desired-with-no-actuals-%d", i)
		testData.invalidDesiredGuidsWithNoActuals = append(
			testData.invalidDesiredGuidsWithNoActuals,
			guid,
		)
	}

	for i := 0; i < numInvalidDesiredGuidsWithOnlyInvalidActuals; i++ {
		guid := fmt.Sprintf("invalid-desired-with-only-invalid-actuals-%d", i)
		testData.invalidDesiredGuidsWithOnlyInvalidActuals = append(
			testData.invalidDesiredGuidsWithOnlyInvalidActuals,
			guid,
		)

		testData.instanceKeysToPrune[processGuidAndIndex{guid, randomIndex1}] = struct{}{}
		testData.instanceKeysToPrune[processGuidAndIndex{guid, randomIndex2}] = struct{}{}
	}

	for i := 0; i < numUnknownDesiredGuidsWithSomeValidActuals; i++ {
		guid := fmt.Sprintf("unknown-desired-with-some-valid-actuals-%d", i)
		testData.unknownDesiredGuidsWithSomeValidActuals = append(
			testData.unknownDesiredGuidsWithSomeValidActuals,
			guid,
		)

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

	for i := 0; i < numUnknownDesiredGuidsWithNoActuals; i++ {
		guid := fmt.Sprintf("unknown-desired-with-no-actuals-%d", i)
		testData.unknownDesiredGuidsWithNoActuals = append(
			testData.unknownDesiredGuidsWithNoActuals,
			guid,
		)
	}

	for i := 0; i < numUnknownDesiredGuidsWithOnlyInvalidActuals; i++ {
		guid := fmt.Sprintf("unknown-desired-with-only-invalid-actuals-%d", i)
		testData.unknownDesiredGuidsWithOnlyInvalidActuals = append(
			testData.unknownDesiredGuidsWithOnlyInvalidActuals,
			guid,
		)

		testData.instanceKeysToPrune[processGuidAndIndex{guid, randomIndex1}] = struct{}{}
		testData.instanceKeysToPrune[processGuidAndIndex{guid, randomIndex2}] = struct{}{}
	}

	testData.domains = models.DomainSet{domain: struct{}{}}

	testData.cells = models.CellSet{
		cellID:       newCellPresence(cellID),
		"other-cell": newCellPresence("other-cell"),
	}

	for actualData := range testData.instanceKeysToKeep {
		testHelper.CreateValidActualLRP(actualData.processGuid, actualData.index)
	}
	for actualData := range testData.instanceKeysToPrune {
		testHelper.CreateMalformedActualLRP(actualData.processGuid, actualData.index)
	}
	for actualData := range testData.evacuatingKeysToKeep {
		testHelper.CreateValidEvacuatingLRP(actualData.processGuid, actualData.index)
	}
	for actualData := range testData.evacuatingKeysToPrune {
		testHelper.CreateMalformedEvacuatingLRP(actualData.processGuid, actualData.index)
	}

	for _, guid := range testData.validDesiredGuidsWithSomeValidActuals {
		testHelper.CreateValidDesiredLRP(guid)
	}
	for _, guid := range testData.validDesiredGuidsWithNoActuals {
		testHelper.CreateValidDesiredLRP(guid)
	}
	for _, guid := range testData.validDesiredGuidsWithOnlyInvalidActuals {
		testHelper.CreateValidDesiredLRP(guid)
	}

	for _, guid := range testData.invalidDesiredGuidsWithSomeValidActuals {
		testHelper.CreateMalformedDesiredLRP(guid)
	}
	for _, guid := range testData.invalidDesiredGuidsWithNoActuals {
		testHelper.CreateMalformedDesiredLRP(guid)
	}
	for _, guid := range testData.invalidDesiredGuidsWithOnlyInvalidActuals {
		testHelper.CreateMalformedDesiredLRP(guid)
	}

	for _, guid := range testData.unknownDesiredGuidsWithSomeValidActuals {
		testHelper.CreateMalformedDesiredLRP(guid)
	}
	for _, guid := range testData.unknownDesiredGuidsWithNoActuals {
		testHelper.CreateMalformedDesiredLRP(guid)
	}
	for _, guid := range testData.unknownDesiredGuidsWithOnlyInvalidActuals {
		testHelper.CreateMalformedDesiredLRP(guid)
	}

	testData.domains.Each(func(domain string) {
		err := domainBBS.UpsertDomain(domain, 0)
		Expect(err).NotTo(HaveOccurred())
	})

	testData.cells.Each(func(cell models.CellPresence) {
		testHelper.RegisterCell(cell)
	})

	return testData
}

func newCellPresence(cellID string) models.CellPresence {
	return models.NewCellPresence(cellID, "1.2.3.4", "az-1", models.CellCapacity{128, 1024, 3})
}
