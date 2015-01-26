package lrp_bbs_test

import (
	"fmt"
	"time"

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
	tests := []crashTest{}

	for _, s := range models.ActualLRPStates {
		var state = s
		name := fmt.Sprintf("when the lrp is %s and has crashes and Since is older than 5 minutes", state)

		lrpGenerator := func() models.ActualLRP {
			lrp := lrpForState(state, OverTime)
			lrp.CrashCount = 4
			return lrp
		}

		if state == models.ActualLRPStateRunning {
			tests = append(tests, crashTest{
				Name:   name,
				LRP:    lrpGenerator,
				Result: itUnclaimsTheLRP(),
			})
		} else {
			tests = append(tests, crashTest{
				Name:   name,
				LRP:    lrpGenerator,
				Result: itCrashesTheLRP(),
			})
		}
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
	State       models.ActualLRPState
	CrashCount  int
	Auction     bool
	ReturnedErr error
}

func itUnclaimsTheLRP() crashTestResult {
	return crashTestResult{
		CrashCount: 1,
		State:      models.ActualLRPStateUnclaimed,
		Auction:    true,
	}
}

func itCrashesTheLRP() crashTestResult {
	return crashTestResult{
		CrashCount: 5,
		State:      models.ActualLRPStateCrashed,
		Auction:    false,
	}
}

func (t crashTest) Test() {
	Context(t.Name, func() {
		var crashErr error
		var actualLRPKey models.ActualLRPKey
		var containerKey models.ActualLRPContainerKey
		var auctioneerPresence models.AuctioneerPresence

		BeforeEach(func() {
			actualLRP := t.LRP()
			actualLRPKey = actualLRP.ActualLRPKey
			containerKey = actualLRP.ActualLRPContainerKey

			auctioneerPresence = models.NewAuctioneerPresence("the-auctioneer-id", "the-address")

			desiredLRP := models.DesiredLRP{
				ProcessGuid: actualLRPKey.ProcessGuid,
				Domain:      actualLRPKey.Domain,
				Instances:   actualLRPKey.Index + 1,
				Stack:       "foo",
				Action:      &models.RunAction{Path: "true"},
			}

			registerAuctioneer(auctioneerPresence)
			createRawDesiredLRP(desiredLRP)
			createRawActualLRP(actualLRP)
		})

		JustBeforeEach(func() {
			clock.Increment(600)
			crashErr = bbs.CrashActualLRP(actualLRPKey, containerKey, logger)
		})

		if t.Result.ReturnedErr == nil {
			It("does not return an error", func() {
				Ω(crashErr).ShouldNot(HaveOccurred())
			})
		} else {
			It(fmt.Sprintf("returned error should be '%s'", t.Result.ReturnedErr.Error()), func() {
				Ω(crashErr).Should(Equal(t.Result.ReturnedErr))
			})
		}

		It(fmt.Sprintf("increments the crash count to %d", t.Result.CrashCount), func() {
			actualLRP := getActualLRP(actualLRPKey)
			Ω(actualLRP.CrashCount).Should(Equal(t.Result.CrashCount))
			Ω(actualLRP.Since).Should(Equal(clock.Now().UnixNano()))
		})

		It(fmt.Sprintf("CAS to %s", t.Result.State), func() {
			actualLRP := getActualLRP(actualLRPKey)
			Ω(actualLRP.State).Should(Equal(t.Result.State))
		})

		if t.Result.Auction {
			It("starts an auction", func() {
				Ω(fakeAuctioneerClient.RequestLRPAuctionsCallCount()).Should(Equal(1))

				requestAddress, requestedAuctions := fakeAuctioneerClient.RequestLRPAuctionsArgsForCall(0)
				Ω(requestAddress).Should(Equal(auctioneerPresence.AuctioneerAddress))
				Ω(requestedAuctions).Should(HaveLen(1))

				desiredLRP, err := bbs.DesiredLRPByProcessGuid(actualLRPKey.ProcessGuid)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(requestedAuctions[0].DesiredLRP).Should(Equal(desiredLRP))
				Ω(requestedAuctions[0].Indices).Should(ConsistOf(uint(actualLRPKey.Index)))
			})
		} else {
			It("does not start an auction", func() {
				Ω(fakeAuctioneerClient.RequestLRPAuctionsCallCount()).Should(Equal(0))
			})
		}
	})
}

func lrpForState(state models.ActualLRPState, timeInState time.Duration) models.ActualLRP {
	var actualLRPKey = models.NewActualLRPKey("some-process-guid", 1, "tests")
	var containerKey = models.NewActualLRPContainerKey("some-instance-guid", "some-cell")

	lrp := models.ActualLRP{
		ActualLRPKey: actualLRPKey,
		State:        state,
		Since:        clock.Now().Add(-timeInState).UnixNano(),
	}

	switch state {
	case models.ActualLRPStateUnclaimed, models.ActualLRPStateCrashed:
	case models.ActualLRPStateClaimed:
		lrp.ActualLRPContainerKey = containerKey
	case models.ActualLRPStateRunning:
		lrp.ActualLRPContainerKey = containerKey
		lrp.ActualLRPNetInfo = models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{{ContainerPort: 1234, HostPort: 5678}})
	}

	return lrp
}
