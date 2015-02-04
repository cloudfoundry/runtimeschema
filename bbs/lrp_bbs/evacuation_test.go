package lrp_bbs_test

import (
	"errors"
	"fmt"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Evacuation", func() {
	Describe("Tabular tests", func() {
		claimedTest := func(base evacuationTest) evacuationTest {
			return evacuationTest{
				Name: base.Name,
				Subject: func() error {
					return bbs.EvacuateClaimedActualLRP(logger, lrpKey, alphaContainerKey)
				},
				InstanceLRP:   base.InstanceLRP,
				EvacuatingLRP: base.EvacuatingLRP,
				Result:        base.Result,
			}
		}

		runningTest := func(base evacuationTest) evacuationTest {
			return evacuationTest{
				Name: base.Name,
				Subject: func() error {
					return bbs.EvacuateRunningActualLRP(logger, lrpKey, alphaContainerKey, alphaNetInfo, alphaEvacuationTTL)
				},
				InstanceLRP:   base.InstanceLRP,
				EvacuatingLRP: base.EvacuatingLRP,
				Result:        base.Result,
			}
		}

		stoppedTest := func(base evacuationTest) evacuationTest {
			return evacuationTest{
				Name: base.Name,
				Subject: func() error {
					return bbs.EvacuateStoppedActualLRP(logger, lrpKey, alphaContainerKey)
				},
				InstanceLRP:   base.InstanceLRP,
				EvacuatingLRP: base.EvacuatingLRP,
				Result:        base.Result,
			}
		}

		claimedTests := []testable{
			claimedTest(evacuationTest{
				Name:   "when there is no instance or evacuating LRP",
				Result: noInstanceNoEvacuating(nil),
			}),
			claimedTest(evacuationTest{
				Name:        "when the instance is UNCLAIMED",
				InstanceLRP: unclaimedLRP(),
				Result:      instanceNoEvacuating(anUnchangedUnclaimedInstanceLRP(), nil),
			}),
			claimedTest(evacuationTest{
				Name:        "when the instance is CLAIMED on alpha",
				InstanceLRP: claimedLRP(alphaContainerKey),
				Result:      instanceNoEvacuating(anUpdatedUnclaimedInstanceLRP(), nil),
			}),
			claimedTest(evacuationTest{
				Name:        "when the instance is CLAIMED on omega",
				InstanceLRP: claimedLRP(omegaContainerKey),
				Result: instanceNoEvacuating(
					anUnchangedClaimedInstanceLRP(omegaContainerKey),
					bbserrors.ErrActualLRPCannotBeUnclaimed,
				),
			}),
			claimedTest(evacuationTest{
				Name:        "when the instance is RUNNING on alpha",
				InstanceLRP: runningLRP(alphaContainerKey, alphaNetInfo),
				Result:      instanceNoEvacuating(anUpdatedUnclaimedInstanceLRP(), nil),
			}),
			claimedTest(evacuationTest{
				Name:        "when the instance is RUNNING on omega",
				InstanceLRP: runningLRP(omegaContainerKey, omegaNetInfo),
				Result: instanceNoEvacuating(
					anUnchangedRunningInstanceLRP(omegaContainerKey, omegaNetInfo),
					bbserrors.ErrActualLRPCannotBeUnclaimed,
				),
			}),
			claimedTest(evacuationTest{
				Name:        "when the instance is CRASHED",
				InstanceLRP: crashedLRP(),
				Result: instanceNoEvacuating(
					anUnchangedCrashedInstanceLRP(),
					bbserrors.ErrActualLRPCannotBeUnclaimed,
				),
			}),
			claimedTest(evacuationTest{
				Name:          "when the evacuating LRP is RUNNING on alpha",
				EvacuatingLRP: runningLRP(alphaContainerKey, alphaNetInfo),
				Result:        noInstanceNoEvacuating(nil),
			}),
			claimedTest(evacuationTest{
				Name:          "when the evacuating LRP is RUNNING on beta",
				EvacuatingLRP: runningLRP(betaContainerKey, betaNetInfo),
				Result:        evacuatingNoInstance(anUnchangedBetaEvacuatingLRP(), nil),
			}),
		}

		runningTests := []testable{
			runningTest(evacuationTest{
				Name:        "when the instance is UNCLAIMED and there is no evacuating LRP",
				InstanceLRP: unclaimedLRP(),
				Result:      newTestResult(anUnchangedUnclaimedInstanceLRP(), anUpdatedAlphaEvacuatingLRP(), nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is UNCLAIMED and an evacuating LRP is RUNNING on alpha",
				InstanceLRP:   unclaimedLRP(),
				EvacuatingLRP: runningLRP(alphaContainerKey, alphaNetInfo),
				Result:        newTestResult(anUnchangedUnclaimedInstanceLRP(), anUnchangedAlphaEvacuatingLRP(), nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is UNCLAIMED and an evacuating LRP is RUNNING on alpha with out-of-date net info",
				InstanceLRP:   unclaimedLRP(),
				EvacuatingLRP: runningLRP(alphaContainerKey, betaNetInfo),
				Result:        newTestResult(anUnchangedUnclaimedInstanceLRP(), anUpdatedAlphaEvacuatingLRP(), nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is UNCLAIMED and an evacuating LRP is RUNNING on beta",
				InstanceLRP:   unclaimedLRP(),
				EvacuatingLRP: runningLRP(betaContainerKey, betaNetInfo),
				Result: newTestResult(
					anUnchangedUnclaimedInstanceLRP(),
					anUnchangedBetaEvacuatingLRP(),
					bbserrors.ErrActualLRPCannotBeEvacuated,
				),
			}),
			runningTest(evacuationTest{
				Name:        "when the instance is CLAIMED on alpha and there is no evacuating LRP",
				InstanceLRP: claimedLRP(alphaContainerKey),
				Result:      newTestResult(anUpdatedUnclaimedInstanceLRP(), anUpdatedAlphaEvacuatingLRP(), nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is CLAIMED on alpha and an evacuating LRP is RUNNING on alpha",
				InstanceLRP:   claimedLRP(alphaContainerKey),
				EvacuatingLRP: runningLRP(alphaContainerKey, alphaNetInfo),
				Result:        newTestResult(anUpdatedUnclaimedInstanceLRP(), anUnchangedAlphaEvacuatingLRP(), nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is CLAIMED on alpha and an evacuating LRP is RUNNING on alpha with out-of-date net info",
				InstanceLRP:   claimedLRP(alphaContainerKey),
				EvacuatingLRP: runningLRP(alphaContainerKey, betaNetInfo),
				Result:        newTestResult(anUpdatedUnclaimedInstanceLRP(), anUpdatedAlphaEvacuatingLRP(), nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is CLAIMED on alpha and an evacuating LRP is RUNNING on beta",
				InstanceLRP:   claimedLRP(alphaContainerKey),
				EvacuatingLRP: runningLRP(betaContainerKey, betaNetInfo),
				Result:        newTestResult(anUpdatedUnclaimedInstanceLRP(), anUpdatedAlphaEvacuatingLRP(), nil),
			}),
			runningTest(evacuationTest{
				Name:        "when the instance is CLAIMED remotely and there is no evacuating LRP",
				InstanceLRP: claimedLRP(omegaContainerKey),
				Result:      newTestResult(anUnchangedClaimedInstanceLRP(omegaContainerKey), anUpdatedAlphaEvacuatingLRP(), nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is CLAIMED remotely and an evacuating LRP is RUNNING on alpha",
				InstanceLRP:   claimedLRP(omegaContainerKey),
				EvacuatingLRP: runningLRP(alphaContainerKey, alphaNetInfo),
				Result:        newTestResult(anUnchangedClaimedInstanceLRP(omegaContainerKey), anUnchangedAlphaEvacuatingLRP(), nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is CLAIMED remotely and an evacuating LRP is RUNNING on alpha with out-of-date net info",
				InstanceLRP:   claimedLRP(omegaContainerKey),
				EvacuatingLRP: runningLRP(alphaContainerKey, betaNetInfo),
				Result:        newTestResult(anUnchangedClaimedInstanceLRP(omegaContainerKey), anUpdatedAlphaEvacuatingLRP(), nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is CLAIMED remotely and an evacuating LRP is RUNNING on beta",
				InstanceLRP:   claimedLRP(omegaContainerKey),
				EvacuatingLRP: runningLRP(betaContainerKey, betaNetInfo),
				Result: newTestResult(
					anUnchangedClaimedInstanceLRP(omegaContainerKey),
					anUnchangedBetaEvacuatingLRP(),
					bbserrors.ErrActualLRPCannotBeEvacuated,
				),
			}),
			runningTest(evacuationTest{
				Name:        "when the instance is RUNNING on alpha and there is no evacuating LRP",
				InstanceLRP: runningLRP(alphaContainerKey, alphaNetInfo),
				Result:      newTestResult(anUpdatedUnclaimedInstanceLRP(), anUpdatedAlphaEvacuatingLRP(), nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is RUNNING on alpha and an evacuating LRP is RUNNING on alpha",
				InstanceLRP:   runningLRP(alphaContainerKey, alphaNetInfo),
				EvacuatingLRP: runningLRP(alphaContainerKey, alphaNetInfo),
				Result:        newTestResult(anUpdatedUnclaimedInstanceLRP(), anUnchangedAlphaEvacuatingLRP(), nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is RUNNING on alpha and an evacuating LRP is RUNNING on alpha with out-of-date net info",
				InstanceLRP:   runningLRP(alphaContainerKey, alphaNetInfo),
				EvacuatingLRP: runningLRP(alphaContainerKey, betaNetInfo),
				Result:        newTestResult(anUpdatedUnclaimedInstanceLRP(), anUpdatedAlphaEvacuatingLRP(), nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is RUNNING on alpha and an evacuating LRP is RUNNING on beta",
				InstanceLRP:   runningLRP(alphaContainerKey, alphaNetInfo),
				EvacuatingLRP: runningLRP(betaContainerKey, betaNetInfo),
				Result:        newTestResult(anUpdatedUnclaimedInstanceLRP(), anUpdatedAlphaEvacuatingLRP(), nil),
			}),
			runningTest(evacuationTest{
				Name:        "when the instance is RUNNING on omega and there is no evacuating LRP",
				InstanceLRP: runningLRP(omegaContainerKey, omegaNetInfo),
				Result: instanceNoEvacuating(
					anUnchangedRunningInstanceLRP(omegaContainerKey, omegaNetInfo),
					bbserrors.ErrActualLRPCannotBeEvacuated,
				),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is RUNNING on omega and an evacuating LRP is RUNNING on alpha",
				InstanceLRP:   runningLRP(omegaContainerKey, omegaNetInfo),
				EvacuatingLRP: runningLRP(alphaContainerKey, alphaNetInfo),
				Result: instanceNoEvacuating(
					anUnchangedRunningInstanceLRP(omegaContainerKey, omegaNetInfo),
					bbserrors.ErrActualLRPCannotBeEvacuated,
				),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is RUNNING on omega and an evacuating LRP is RUNNING on beta",
				InstanceLRP:   runningLRP(omegaContainerKey, omegaNetInfo),
				EvacuatingLRP: runningLRP(betaContainerKey, betaNetInfo),
				Result: newTestResult(
					anUnchangedRunningInstanceLRP(omegaContainerKey, omegaNetInfo),
					anUnchangedBetaEvacuatingLRP(),
					bbserrors.ErrActualLRPCannotBeEvacuated,
				),
			}),
			runningTest(evacuationTest{
				Name:        "when the instance is CRASHED and there is no evacuating LRP",
				InstanceLRP: crashedLRP(),
				Result: instanceNoEvacuating(
					anUnchangedCrashedInstanceLRP(),
					bbserrors.ErrActualLRPCannotBeEvacuated,
				),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is CRASHED and an evacuating LRP is RUNNING on alpha",
				InstanceLRP:   crashedLRP(),
				EvacuatingLRP: runningLRP(alphaContainerKey, alphaNetInfo),
				Result: instanceNoEvacuating(
					anUnchangedCrashedInstanceLRP(),
					bbserrors.ErrActualLRPCannotBeEvacuated,
				),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is CRASHED and an evacuating LRP is RUNNING on beta",
				InstanceLRP:   crashedLRP(),
				EvacuatingLRP: runningLRP(betaContainerKey, betaNetInfo),
				Result: newTestResult(
					anUnchangedCrashedInstanceLRP(),
					anUnchangedBetaEvacuatingLRP(),
					bbserrors.ErrActualLRPCannotBeEvacuated,
				),
			}),
			runningTest(evacuationTest{
				Name: "when the instance is MISSING and there is no evacuating LRP",
				Result: noInstanceNoEvacuating(
					bbserrors.ErrActualLRPCannotBeEvacuated,
				),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is MISSING and an evacuating LRP is RUNNING on alpha",
				EvacuatingLRP: runningLRP(alphaContainerKey, alphaNetInfo),
				Result: noInstanceNoEvacuating(
					bbserrors.ErrActualLRPCannotBeEvacuated,
				),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is MISSING and an evacuating LRP is RUNNING on beta",
				EvacuatingLRP: runningLRP(betaContainerKey, betaNetInfo),
				Result: evacuatingNoInstance(
					anUnchangedBetaEvacuatingLRP(),
					bbserrors.ErrActualLRPCannotBeEvacuated,
				),
			}),
		}

		stoppedTests := []testable{
			stoppedTest(evacuationTest{
				Name:   "when there is no instance or evacuating LRP",
				Result: noInstanceNoEvacuating(nil),
			}),
			stoppedTest(evacuationTest{
				Name:        "when the instance is UNCLAIMED",
				InstanceLRP: unclaimedLRP(),
				Result: instanceNoEvacuating(
					anUnchangedUnclaimedInstanceLRP(),
					bbserrors.ErrActualLRPCannotBeRemoved,
				),
			}),
			stoppedTest(evacuationTest{
				Name:        "when the instance is CLAIMED on alpha",
				InstanceLRP: claimedLRP(alphaContainerKey),
				Result:      noInstanceNoEvacuating(nil),
			}),
			stoppedTest(evacuationTest{
				Name:        "when the instance is CLAIMED on omega",
				InstanceLRP: claimedLRP(omegaContainerKey),
				Result: instanceNoEvacuating(
					anUnchangedClaimedInstanceLRP(omegaContainerKey),
					bbserrors.ErrActualLRPCannotBeRemoved,
				),
			}),
			stoppedTest(evacuationTest{
				Name:        "when the instance is RUNNING on alpha",
				InstanceLRP: runningLRP(alphaContainerKey, alphaNetInfo),
				Result:      noInstanceNoEvacuating(nil),
			}),
			stoppedTest(evacuationTest{
				Name:        "when the instance is RUNNING on omega",
				InstanceLRP: runningLRP(omegaContainerKey, omegaNetInfo),
				Result: instanceNoEvacuating(
					anUnchangedRunningInstanceLRP(omegaContainerKey, omegaNetInfo),
					bbserrors.ErrActualLRPCannotBeRemoved,
				),
			}),
			stoppedTest(evacuationTest{
				Name:        "when the instance is CRASHED",
				InstanceLRP: crashedLRP(),
				Result: instanceNoEvacuating(
					anUnchangedCrashedInstanceLRP(),
					bbserrors.ErrActualLRPCannotBeRemoved,
				),
			}),
			stoppedTest(evacuationTest{
				Name:          "when the evacuating LRP is RUNNING on alpha",
				EvacuatingLRP: runningLRP(alphaContainerKey, alphaNetInfo),
				Result:        noInstanceNoEvacuating(nil),
			}),
			stoppedTest(evacuationTest{
				Name:          "when the evacuating LRP is RUNNING on beta",
				EvacuatingLRP: runningLRP(betaContainerKey, betaNetInfo),
				Result:        evacuatingNoInstance(anUnchangedBetaEvacuatingLRP(), nil),
			}),
		}

		Context("when the LRP is to be CLAIMED", func() {
			for _, test := range claimedTests {
				test.Test()
			}
		})

		Context("when the LRP is to be RUNNING", func() {
			for _, test := range runningTests {
				test.Test()
			}
		})

		Context("when the LRP is to be STOPPED", func() {
			for _, test := range stoppedTests {
				test.Test()
			}
		})
	})
})

const (
	initialTimestamp = 1138
	timeIncrement    = 2279
	finalTimestamp   = initialTimestamp + timeIncrement

	alphaEvacuationTTL = 30
	omegaEvacuationTTL = 1000
	allowedTTLDecay    = 2

	processGuid       = "process-guid"
	alphaInstanceGuid = "alpha-instance-guid"
	betaInstanceGuid  = "beta-instance-guid"
	omegaInstanceGuid = "omega-instance-guid"
	alphaCellID       = "alpha-cell-id"
	betaCellID        = "beta-cell-id"
	omegaCellID       = "omega-cell-id"
	alphaAddress      = "alpha-address"
	betaAddress       = "beta-address"
	omegaAddress      = "omega-address"
)

var (
	desiredLRP = models.DesiredLRP{
		ProcessGuid: processGuid,
		Domain:      "domain",
		Instances:   1,
		Stack:       "some-stack",
		Action:      &models.RunAction{Path: "/bin/true"},
	}

	index  = 0
	lrpKey = models.NewActualLRPKey(desiredLRP.ProcessGuid, index, desiredLRP.Domain)

	alphaContainerKey = models.NewActualLRPContainerKey(alphaInstanceGuid, alphaCellID)
	betaContainerKey  = models.NewActualLRPContainerKey(betaInstanceGuid, betaCellID)
	omegaContainerKey = models.NewActualLRPContainerKey(omegaInstanceGuid, omegaCellID)
	emptyContainerKey = models.ActualLRPContainerKey{}

	alphaPorts   = []models.PortMapping{{ContainerPort: 2349, HostPort: 9872}}
	alphaNetInfo = models.NewActualLRPNetInfo(alphaAddress, alphaPorts)
	betaPorts    = []models.PortMapping{{ContainerPort: 2353, HostPort: 9868}}
	betaNetInfo  = models.NewActualLRPNetInfo(betaAddress, betaPorts)
	omegaPorts   = []models.PortMapping{{ContainerPort: 2345, HostPort: 9876}}
	omegaNetInfo = models.NewActualLRPNetInfo(omegaAddress, omegaPorts)
	emptyNetInfo = models.EmptyActualLRPNetInfo()
)

type testable interface {
	Test()
}

type evacuationTest struct {
	Name          string
	Subject       func() error
	InstanceLRP   lrpSetupFunc
	EvacuatingLRP lrpSetupFunc
	Result        testResult
}

func lrp(state models.ActualLRPState, containerKey models.ActualLRPContainerKey, netInfo models.ActualLRPNetInfo) lrpSetupFunc {
	return func() models.ActualLRP {
		return models.ActualLRP{
			ActualLRPKey:          lrpKey,
			ActualLRPContainerKey: containerKey,
			ActualLRPNetInfo:      netInfo,
			State:                 state,
			Since:                 clock.Now().UnixNano(),
		}
	}
}

func unclaimedLRP() lrpSetupFunc {
	return lrp(models.ActualLRPStateUnclaimed, emptyContainerKey, emptyNetInfo)
}

func claimedLRP(containerKey models.ActualLRPContainerKey) lrpSetupFunc {
	return lrp(models.ActualLRPStateClaimed, containerKey, emptyNetInfo)
}

func runningLRP(containerKey models.ActualLRPContainerKey, netInfo models.ActualLRPNetInfo) lrpSetupFunc {
	return lrp(models.ActualLRPStateRunning, containerKey, netInfo)
}

func crashedLRP() lrpSetupFunc {
	return lrp(models.ActualLRPStateCrashed, emptyContainerKey, emptyNetInfo)
}

type lrpStatus struct {
	State models.ActualLRPState
	models.ActualLRPContainerKey
	models.ActualLRPNetInfo
	ShouldUpdate bool
}

type evacuatingLRPStatus struct {
	lrpStatus
	TTL uint64
}

type testResult struct {
	Instance         *lrpStatus
	Evacuating       *evacuatingLRPStatus
	AuctionRequested bool
	ReturnedError    error
}

func anUpdatedAlphaEvacuatingLRP() *evacuatingLRPStatus {
	return newEvacuatingLRPStatus(alphaContainerKey, alphaNetInfo, true)
}

func anUnchangedAlphaEvacuatingLRP() *evacuatingLRPStatus {
	return newEvacuatingLRPStatus(alphaContainerKey, alphaNetInfo, false)
}

func anUnchangedBetaEvacuatingLRP() *evacuatingLRPStatus {
	return newEvacuatingLRPStatus(betaContainerKey, betaNetInfo, false)
}

func newEvacuatingLRPStatus(containerKey models.ActualLRPContainerKey, netInfo models.ActualLRPNetInfo, shouldUpdate bool) *evacuatingLRPStatus {
	status := &evacuatingLRPStatus{
		lrpStatus: lrpStatus{
			State: models.ActualLRPStateRunning,
			ActualLRPContainerKey: containerKey,
			ActualLRPNetInfo:      netInfo,
			ShouldUpdate:          shouldUpdate,
		},
	}

	if shouldUpdate {
		status.TTL = alphaEvacuationTTL
	}

	return status
}

func anUpdatedUnclaimedInstanceLRP() *lrpStatus {
	return &lrpStatus{
		State: models.ActualLRPStateUnclaimed,
		ActualLRPContainerKey: emptyContainerKey,
		ActualLRPNetInfo:      emptyNetInfo,
		ShouldUpdate:          true,
	}
}

func anUnchangedInstanceLRP(state models.ActualLRPState, containerKey models.ActualLRPContainerKey, netInfo models.ActualLRPNetInfo) *lrpStatus {
	return &lrpStatus{
		State: state,
		ActualLRPContainerKey: containerKey,
		ActualLRPNetInfo:      netInfo,
		ShouldUpdate:          false,
	}
}

func anUnchangedUnclaimedInstanceLRP() *lrpStatus {
	return anUnchangedInstanceLRP(models.ActualLRPStateUnclaimed, emptyContainerKey, emptyNetInfo)
}

func anUnchangedClaimedInstanceLRP(containerKey models.ActualLRPContainerKey) *lrpStatus {
	return anUnchangedInstanceLRP(models.ActualLRPStateClaimed, containerKey, emptyNetInfo)
}

func anUnchangedRunningInstanceLRP(containerKey models.ActualLRPContainerKey, netInfo models.ActualLRPNetInfo) *lrpStatus {
	return anUnchangedInstanceLRP(models.ActualLRPStateRunning, containerKey, netInfo)
}

func anUnchangedCrashedInstanceLRP() *lrpStatus {
	return anUnchangedInstanceLRP(models.ActualLRPStateCrashed, emptyContainerKey, emptyNetInfo)
}

func newTestResult(instanceStatus *lrpStatus, evacuatingStatus *evacuatingLRPStatus, err error) testResult {
	result := testResult{
		Instance:      instanceStatus,
		Evacuating:    evacuatingStatus,
		ReturnedError: err,
	}

	if instanceStatus != nil && instanceStatus.ShouldUpdate {
		result.AuctionRequested = true
	}

	return result
}

func instanceNoEvacuating(instanceStatus *lrpStatus, err error) testResult {
	return newTestResult(instanceStatus, nil, err)
}

func evacuatingNoInstance(evacuatingStatus *evacuatingLRPStatus, err error) testResult {
	return newTestResult(nil, evacuatingStatus, err)
}

func noInstanceNoEvacuating(err error) testResult {
	return newTestResult(nil, nil, err)
}

func (t evacuationTest) Test() {
	Context(t.Name, func() {
		var evacuateErr error
		var auctioneerPresence models.AuctioneerPresence
		var initialTimestamp int64

		BeforeEach(func() {
			auctioneerPresence = models.NewAuctioneerPresence("the-auctioneer-id", "the-address")
			initialTimestamp = clock.Now().UnixNano()

			registerAuctioneer(auctioneerPresence)
			createRawDesiredLRP(desiredLRP)
			if t.InstanceLRP != nil {
				createRawActualLRP(t.InstanceLRP())
			}
			if t.EvacuatingLRP != nil {
				createRawEvacuatingActualLRP(t.EvacuatingLRP(), omegaEvacuationTTL)
			}
		})

		JustBeforeEach(func() {
			clock.Increment(timeIncrement)
			evacuateErr = t.Subject()
		})

		if t.Result.ReturnedError == nil {
			It("does not return an error", func() {
				Ω(evacuateErr).ShouldNot(HaveOccurred())
			})
		} else {
			It(fmt.Sprintf("returned error should be '%s'", t.Result.ReturnedError.Error()), func() {
				Ω(evacuateErr).Should(Equal(t.Result.ReturnedError))
			})
		}

		if t.Result.AuctionRequested {
			It("starts an auction", func() {
				Ω(fakeAuctioneerClient.RequestLRPAuctionsCallCount()).Should(Equal(1))

				requestAddress, requestedAuctions := fakeAuctioneerClient.RequestLRPAuctionsArgsForCall(0)
				Ω(requestAddress).Should(Equal(auctioneerPresence.AuctioneerAddress))
				Ω(requestedAuctions).Should(HaveLen(1))

				Ω(requestedAuctions[0].DesiredLRP).Should(Equal(desiredLRP))
				Ω(requestedAuctions[0].Indices).Should(ConsistOf(uint(index)))
			})

			Context("when starting the auction fails", func() {
				var err error
				BeforeEach(func() {
					err = errors.New("error")
					fakeAuctioneerClient.RequestLRPAuctionsReturns(err)
				})

				It("returns an error", func() {
					Ω(evacuateErr).Should(Equal(err))
				})
			})
		} else {
			It("does not start an auction", func() {
				Ω(fakeAuctioneerClient.RequestLRPAuctionsCallCount()).Should(Equal(0))
			})
		}

		if t.Result.Instance == nil {
			It("removes the /instance actualLRP", func() {
				_, err := getInstanceActualLRP(lrpKey)
				Ω(err).Should(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		} else {
			if t.Result.Instance.ShouldUpdate {
				It("updates the /instance Since", func() {
					lrpInBBS, err := getInstanceActualLRP(lrpKey)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrpInBBS.Since).Should(Equal(clock.Now().UnixNano()))
				})
			} else {
				It("does not update the /instance Since", func() {
					lrpInBBS, err := getInstanceActualLRP(lrpKey)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrpInBBS.Since).Should(Equal(initialTimestamp))
				})
			}

			It("has the expected /instance state", func() {
				lrpInBBS, err := getInstanceActualLRP(lrpKey)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(lrpInBBS.State).Should(Equal(t.Result.Instance.State))
			})

			It("has the expected /instance container key", func() {
				lrpInBBS, err := getInstanceActualLRP(lrpKey)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(lrpInBBS.ActualLRPContainerKey).Should(Equal(t.Result.Instance.ActualLRPContainerKey))
			})

			It("has the expected /instance net info", func() {
				lrpInBBS, err := getInstanceActualLRP(lrpKey)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(lrpInBBS.ActualLRPNetInfo).Should(Equal(t.Result.Instance.ActualLRPNetInfo))
			})
		}

		if t.Result.Evacuating == nil {
			It("removes the /evacuating actualLRP", func() {
				_, _, err := getEvacuatingActualLRP(lrpKey)
				Ω(err).Should(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		} else {
			if t.Result.Evacuating.ShouldUpdate {
				It("updates the /evacuating Since", func() {
					lrpInBBS, _, err := getEvacuatingActualLRP(lrpKey)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrpInBBS.Since).Should(Equal(clock.Now().UnixNano()))
				})

				It("updates the /evacuating TTL to the desired value", func() {
					_, ttl, err := getEvacuatingActualLRP(lrpKey)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(ttl).Should(BeNumerically("~", t.Result.Evacuating.TTL, allowedTTLDecay))
				})
			} else {
				It("does not update the /evacuating Since", func() {
					lrpInBBS, _, err := getEvacuatingActualLRP(lrpKey)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrpInBBS.Since).Should(Equal(initialTimestamp))
				})

				It("does not update the /evacuating TTL", func() {
					_, ttl, err := getEvacuatingActualLRP(lrpKey)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(ttl).Should(BeNumerically("~", omegaEvacuationTTL, allowedTTLDecay))
				})
			}

			It("has the expected /evacuating state", func() {
				lrpInBBS, _, err := getEvacuatingActualLRP(lrpKey)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(lrpInBBS.State).Should(Equal(t.Result.Evacuating.State))
			})

			It("has the expected /evacuating container key", func() {
				lrpInBBS, _, err := getEvacuatingActualLRP(lrpKey)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(lrpInBBS.ActualLRPContainerKey).Should(Equal(t.Result.Evacuating.ActualLRPContainerKey))
			})

			It("has the expected /evacuating net info", func() {
				lrpInBBS, _, err := getEvacuatingActualLRP(lrpKey)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(lrpInBBS.ActualLRPNetInfo).Should(Equal(t.Result.Evacuating.ActualLRPNetInfo))
			})
		}
	})
}
