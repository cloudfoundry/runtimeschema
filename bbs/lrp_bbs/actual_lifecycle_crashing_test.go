package lrp_bbs_test

import (
	"fmt"
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/lrp_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const OverTime = lrp_bbs.CrashResetTimeout + time.Minute

var _ = Describe("CrashActualLRP", func() {
	var crashTests = []crashTest{
		{
			Name: "when the lrp is RUNNING and the crash count is greater than 3",
			LRP: func() models.ActualLRP {
				lrp := lrpForState(models.ActualLRPStateRunning, time.Minute)
				lrp.CrashCount = 4
				return lrp
			},
			Result: itCrashesTheLRP(),
		},
		{
			Name: "when the lrp is RUNNING and the crash count is less than 3",
			LRP: func() models.ActualLRP {
				return lrpForState(models.ActualLRPStateRunning, time.Minute)
			},
			Result: itUnclaimsTheLRP(),
		},
		{
			Name: "when the lrp is RUNNING and has crashes and Since is older than 5 minutes",
			LRP: func() models.ActualLRP {
				lrp := lrpForState(models.ActualLRPStateRunning, OverTime)
				lrp.CrashCount = 4
				return lrp
			},
			Result: itUnclaimsTheLRP(),
		},
	}

	crashTests = append(crashTests, resetOnlyRunningLRPsThatHaveNotCrashedRecently()...)

	for _, t := range crashTests {
		var crashTest = t
		crashTest.Test()
	}
})

func resetOnlyRunningLRPsThatHaveNotCrashedRecently() []crashTest {
	lrpGenerator := func(state models.ActualLRPState) lrpSetupFunc {
		return func() models.ActualLRP {
			lrp := lrpForState(state, OverTime)
			lrp.CrashCount = 4
			return lrp
		}
	}

	nameGenerator := func(state models.ActualLRPState) string {
		return fmt.Sprintf("when the lrp is %s and has crashes and Since is older than 5 minutes", state)
	}

	tests := []crashTest{
		crashTest{
			Name:   nameGenerator(models.ActualLRPStateUnclaimed),
			LRP:    lrpGenerator(models.ActualLRPStateUnclaimed),
			Result: itDoesNotChangeTheUnclaimedLRP(),
		},
		crashTest{
			Name:   nameGenerator(models.ActualLRPStateClaimed),
			LRP:    lrpGenerator(models.ActualLRPStateClaimed),
			Result: itCrashesTheLRP(),
		},
		crashTest{
			Name:   nameGenerator(models.ActualLRPStateRunning),
			LRP:    lrpGenerator(models.ActualLRPStateRunning),
			Result: itUnclaimsTheLRP(),
		},
		crashTest{
			Name:   nameGenerator(models.ActualLRPStateCrashed),
			LRP:    lrpGenerator(models.ActualLRPStateCrashed),
			Result: itDoesNotChangeTheCrashedLRP(),
		},
	}

	return tests
}

type lrpSetupFunc func() models.ActualLRP

type crashTest struct {
	Name   string
	LRP    lrpSetupFunc
	Result crashTestResult
}

type crashTestResult struct {
	State        models.ActualLRPState
	CrashCount   int
	CrashReason  string
	ShouldUpdate bool
	Auction      bool
	ReturnedErr  error
}

func itUnclaimsTheLRP() crashTestResult {
	return crashTestResult{
		CrashCount:   1,
		CrashReason:  "crashed",
		State:        models.ActualLRPStateUnclaimed,
		ShouldUpdate: true,
		Auction:      true,
		ReturnedErr:  nil,
	}
}

func itCrashesTheLRP() crashTestResult {
	return crashTestResult{
		CrashCount:   5,
		CrashReason:  "crashed",
		State:        models.ActualLRPStateCrashed,
		ShouldUpdate: true,
		Auction:      false,
		ReturnedErr:  nil,
	}
}

func itDoesNotChangeTheUnclaimedLRP() crashTestResult {
	return crashTestResult{
		CrashCount:   4,
		State:        models.ActualLRPStateUnclaimed,
		ShouldUpdate: false,
		Auction:      false,
		ReturnedErr:  bbserrors.ErrActualLRPCannotBeCrashed,
	}
}

func itDoesNotChangeTheCrashedLRP() crashTestResult {
	return crashTestResult{
		CrashCount:   4,
		CrashReason:  "crashed",
		State:        models.ActualLRPStateCrashed,
		ShouldUpdate: false,
		Auction:      false,
		ReturnedErr:  bbserrors.ErrActualLRPCannotBeCrashed,
	}
}

func (t crashTest) Test() {
	Context(t.Name, func() {
		var crashErr error
		var actualLRPKey models.ActualLRPKey
		var instanceKey models.ActualLRPInstanceKey
		var auctioneerPresence models.AuctioneerPresence
		var initialTimestamp int64
		var initialModificationIndex uint

		BeforeEach(func() {
			actualLRP := t.LRP()
			actualLRPKey = actualLRP.ActualLRPKey
			instanceKey = actualLRP.ActualLRPInstanceKey

			auctioneerPresence = models.NewAuctioneerPresence("the-auctioneer-id", "the-address")
			initialTimestamp = actualLRP.Since
			initialModificationIndex = actualLRP.ModificationTag.Index

			desiredLRP := models.DesiredLRP{
				ProcessGuid: actualLRPKey.ProcessGuid,
				Domain:      actualLRPKey.Domain,
				Instances:   actualLRPKey.Index + 1,
				RootFS:      "foo:bar",
				Action:      &models.RunAction{Path: "true"},
			}

			testHelper.RegisterAuctioneer(auctioneerPresence)
			testHelper.SetRawDesiredLRP(desiredLRP)
			testHelper.SetRawActualLRP(actualLRP)
		})

		JustBeforeEach(func() {
			clock.Increment(600)
			crashErr = lrpBBS.CrashActualLRP(logger, actualLRPKey, instanceKey, "crashed")
		})

		if t.Result.ReturnedErr == nil {
			It("does not return an error", func() {
				Expect(crashErr).NotTo(HaveOccurred())
			})
		} else {
			It(fmt.Sprintf("returned error should be '%s'", t.Result.ReturnedErr.Error()), func() {
				Expect(crashErr).To(Equal(t.Result.ReturnedErr))
			})
		}

		It(fmt.Sprintf("has crash count %d", t.Result.CrashCount), func() {
			actualLRP, err := testHelper.GetInstanceActualLRP(actualLRPKey)
			Expect(err).NotTo(HaveOccurred())
			Expect(actualLRP.CrashCount).To(Equal(t.Result.CrashCount))
		})

		It(fmt.Sprintf("has crash reason %s", t.Result.CrashReason), func() {
			actualLRP, err := testHelper.GetInstanceActualLRP(actualLRPKey)
			Expect(err).NotTo(HaveOccurred())
			Expect(actualLRP.CrashReason).To(Equal(t.Result.CrashReason))
		})

		if t.Result.ShouldUpdate {
			It("updates the Since", func() {
				actualLRP, err := testHelper.GetInstanceActualLRP(actualLRPKey)
				Expect(err).NotTo(HaveOccurred())
				Expect(actualLRP.Since).To(Equal(clock.Now().UnixNano()))
			})

			It("updates the ModificationIndex", func() {
				actualLRP, err := testHelper.GetInstanceActualLRP(actualLRPKey)
				Expect(err).NotTo(HaveOccurred())
				Expect(actualLRP.ModificationTag.Index).To(Equal(initialModificationIndex + 1))
			})
		} else {
			It("does not update the Since", func() {
				actualLRP, err := testHelper.GetInstanceActualLRP(actualLRPKey)
				Expect(err).NotTo(HaveOccurred())
				Expect(actualLRP.Since).To(Equal(initialTimestamp))
			})

			It("does not update the ModificationIndex", func() {
				actualLRP, err := testHelper.GetInstanceActualLRP(actualLRPKey)
				Expect(err).NotTo(HaveOccurred())
				Expect(actualLRP.ModificationTag.Index).To(Equal(initialModificationIndex))
			})
		}

		It(fmt.Sprintf("CAS to %s", t.Result.State), func() {
			actualLRP, err := testHelper.GetInstanceActualLRP(actualLRPKey)
			Expect(err).NotTo(HaveOccurred())
			Expect(actualLRP.State).To(Equal(t.Result.State))
		})

		if t.Result.Auction {
			It("starts an auction", func() {
				Expect(fakeAuctioneerClient.RequestLRPAuctionsCallCount()).To(Equal(1))

				requestAddress, requestedAuctions := fakeAuctioneerClient.RequestLRPAuctionsArgsForCall(0)
				Expect(requestAddress).To(Equal(auctioneerPresence.AuctioneerAddress))
				Expect(requestedAuctions).To(HaveLen(1))

				desiredLRP, err := lrpBBS.DesiredLRPByProcessGuid(logger, actualLRPKey.ProcessGuid)
				Expect(err).NotTo(HaveOccurred())
				Expect(requestedAuctions[0].DesiredLRP).To(Equal(desiredLRP))
				Expect(requestedAuctions[0].Indices).To(ConsistOf(uint(actualLRPKey.Index)))
			})

			Context("when the desired LRP no longer exists", func() {
				BeforeEach(func() {
					err := lrpBBS.RemoveDesiredLRPByProcessGuid(logger, actualLRPKey.ProcessGuid)
					Expect(err).NotTo(HaveOccurred())
				})

				It("the actual LRP is also deleted", func() {
					Expect(crashErr).NotTo(HaveOccurred())

					_, err := lrpBBS.ActualLRPGroupByProcessGuidAndIndex(logger, actualLRPKey.ProcessGuid, actualLRPKey.Index)
					Expect(err).To(Equal(bbserrors.ErrStoreResourceNotFound))
				})
			})
		} else {
			It("does not start an auction", func() {
				Expect(fakeAuctioneerClient.RequestLRPAuctionsCallCount()).To(Equal(0))
			})
		}

		Context("when crashing a different instance key", func() {
			var beforeActualGroup models.ActualLRPGroup

			BeforeEach(func() {
				var err error
				beforeActualGroup, err = lrpBBS.ActualLRPGroupByProcessGuidAndIndex(logger, actualLRPKey.ProcessGuid, actualLRPKey.Index)
				Expect(err).NotTo(HaveOccurred())
				instanceKey.InstanceGuid = "another-guid"
			})

			It("does not crash", func() {
				Expect(crashErr).To(Equal(bbserrors.ErrActualLRPCannotBeCrashed))

				afterActualGroup, err := lrpBBS.ActualLRPGroupByProcessGuidAndIndex(logger, actualLRPKey.ProcessGuid, actualLRPKey.Index)
				Expect(err).NotTo(HaveOccurred())
				Expect(afterActualGroup).To(Equal(beforeActualGroup))
			})
		})
	})
}

func lrpForState(state models.ActualLRPState, timeInState time.Duration) models.ActualLRP {
	var actualLRPKey = models.NewActualLRPKey("some-process-guid", 1, "tests")
	var instanceKey = models.NewActualLRPInstanceKey("some-instance-guid", "some-cell")

	lrp := models.ActualLRP{
		ActualLRPKey: actualLRPKey,
		State:        state,
		Since:        clock.Now().Add(-timeInState).UnixNano(),
	}

	switch state {
	case models.ActualLRPStateUnclaimed:
	case models.ActualLRPStateCrashed:
		lrp.CrashReason = "crashed"
	case models.ActualLRPStateClaimed:
		lrp.ActualLRPInstanceKey = instanceKey
	case models.ActualLRPStateRunning:
		lrp.ActualLRPInstanceKey = instanceKey
		lrp.ActualLRPNetInfo = models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{{ContainerPort: 1234, HostPort: 5678}})
	}

	return lrp
}
