package lrp_bbs_test

import (
	"errors"
	"fmt"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/diego_errors"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Evacuation", func() {
	Describe("Tabular tests", func() {
		claimedTest := func(base evacuationTest) evacuationTest {
			return evacuationTest{
				Name: base.Name,
				Subject: func() (shared.ContainerRetainment, error) {
					return bbs.EvacuateClaimedActualLRP(logger, lrpKey, alphaInstanceKey)
				},
				InstanceLRP:   base.InstanceLRP,
				EvacuatingLRP: base.EvacuatingLRP,
				Result:        base.Result,
			}
		}

		runningTest := func(base evacuationTest) evacuationTest {
			return evacuationTest{
				Name: base.Name,
				Subject: func() (shared.ContainerRetainment, error) {
					return bbs.EvacuateRunningActualLRP(logger, lrpKey, alphaInstanceKey, alphaNetInfo, alphaEvacuationTTL)
				},
				InstanceLRP:   base.InstanceLRP,
				EvacuatingLRP: base.EvacuatingLRP,
				Result:        base.Result,
			}
		}

		stoppedTest := func(base evacuationTest) evacuationTest {
			return evacuationTest{
				Name: base.Name,
				Subject: func() (shared.ContainerRetainment, error) {
					return bbs.EvacuateStoppedActualLRP(logger, lrpKey, alphaInstanceKey)
				},
				InstanceLRP:   base.InstanceLRP,
				EvacuatingLRP: base.EvacuatingLRP,
				Result:        base.Result,
			}
		}

		crashedTest := func(base evacuationTest) evacuationTest {
			return evacuationTest{
				Name: base.Name,
				Subject: func() (shared.ContainerRetainment, error) {
					return bbs.EvacuateCrashedActualLRP(logger, lrpKey, alphaInstanceKey, "crashed")
				},
				InstanceLRP:   base.InstanceLRP,
				EvacuatingLRP: base.EvacuatingLRP,
				Result:        base.Result,
			}
		}

		removalTest := func(base evacuationTest) evacuationTest {
			return evacuationTest{
				Name: base.Name,
				Subject: func() (shared.ContainerRetainment, error) {
					err := bbs.RemoveEvacuatingActualLRP(logger, lrpKey, alphaInstanceKey)
					return shared.DeleteContainer, err
				},
				InstanceLRP:   base.InstanceLRP,
				EvacuatingLRP: base.EvacuatingLRP,
				Result:        base.Result,
			}
		}

		claimedTests := []testable{
			claimedTest(evacuationTest{
				Name:   "when there is no instance or evacuating LRP",
				Result: noInstanceNoEvacuating(shared.DeleteContainer, nil),
			}),
			claimedTest(evacuationTest{
				Name:        "when the instance is UNCLAIMED",
				InstanceLRP: unclaimedLRP(),
				Result:      instanceNoEvacuating(anUnchangedUnclaimedInstanceLRP(), shared.DeleteContainer, nil),
			}),
			claimedTest(evacuationTest{
				Name:        "when the instance is CLAIMED on alpha",
				InstanceLRP: claimedLRP(alphaInstanceKey),
				Result:      instanceNoEvacuating(anUpdatedUnclaimedInstanceLRP(), shared.DeleteContainer, nil),
			}),
			claimedTest(evacuationTest{
				Name:        "when the instance is CLAIMED on omega",
				InstanceLRP: claimedLRP(omegaInstanceKey),
				Result: instanceNoEvacuating(
					anUnchangedClaimedInstanceLRP(omegaInstanceKey),
					shared.DeleteContainer,
					bbserrors.ErrActualLRPCannotBeUnclaimed,
				),
			}),
			claimedTest(evacuationTest{
				Name:        "when the instance is RUNNING on alpha",
				InstanceLRP: runningLRP(alphaInstanceKey, alphaNetInfo),
				Result:      instanceNoEvacuating(anUpdatedUnclaimedInstanceLRP(), shared.DeleteContainer, nil),
			}),
			claimedTest(evacuationTest{
				Name:        "when the instance is RUNNING on omega",
				InstanceLRP: runningLRP(omegaInstanceKey, omegaNetInfo),
				Result: instanceNoEvacuating(
					anUnchangedRunningInstanceLRP(omegaInstanceKey, omegaNetInfo),
					shared.DeleteContainer,
					bbserrors.ErrActualLRPCannotBeUnclaimed,
				),
			}),
			claimedTest(evacuationTest{
				Name:        "when the instance is CRASHED",
				InstanceLRP: crashedLRP(),
				Result: instanceNoEvacuating(
					anUnchangedCrashedInstanceLRP(),
					shared.DeleteContainer,
					bbserrors.ErrActualLRPCannotBeUnclaimed,
				),
			}),
			claimedTest(evacuationTest{
				Name:          "when the evacuating LRP is RUNNING on alpha",
				EvacuatingLRP: runningLRP(alphaInstanceKey, alphaNetInfo),
				Result:        noInstanceNoEvacuating(shared.DeleteContainer, nil),
			}),
			claimedTest(evacuationTest{
				Name:          "when the evacuating LRP is RUNNING on beta",
				EvacuatingLRP: runningLRP(betaInstanceKey, betaNetInfo),
				Result:        evacuatingNoInstance(anUnchangedBetaEvacuatingLRP(), shared.DeleteContainer, nil),
			}),
		}

		runningTests := []testable{
			runningTest(evacuationTest{
				Name:        "when the instance is UNCLAIMED and there is no evacuating LRP",
				InstanceLRP: unclaimedLRP(),
				Result:      newTestResult(anUnchangedUnclaimedInstanceLRP(), anUpdatedAlphaEvacuatingLRP(), shared.KeepContainer, nil),
			}),
			runningTest(evacuationTest{
				Name:        "when the instance is UNCLAIMED with a placement error and there is no evacuating LRP",
				InstanceLRP: unclaimedLRPWithPlacementError(),
				Result: instanceNoEvacuating(
					anUnchangedUnclaimedInstanceLRP(),
					shared.DeleteContainer,
					nil,
				),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is UNCLAIMED and an evacuating LRP is RUNNING on alpha",
				InstanceLRP:   unclaimedLRP(),
				EvacuatingLRP: runningLRP(alphaInstanceKey, alphaNetInfo),
				Result:        newTestResult(anUnchangedUnclaimedInstanceLRP(), anUnchangedAlphaEvacuatingLRP(), shared.KeepContainer, nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is UNCLAIMED with a placement error and an evacuating LRP is RUNNING on alpha",
				InstanceLRP:   unclaimedLRPWithPlacementError(),
				EvacuatingLRP: runningLRP(alphaInstanceKey, alphaNetInfo),
				Result: instanceNoEvacuating(
					anUnchangedUnclaimedInstanceLRP(),
					shared.DeleteContainer,
					nil,
				),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is UNCLAIMED and an evacuating LRP is RUNNING on alpha with out-of-date net info",
				InstanceLRP:   unclaimedLRP(),
				EvacuatingLRP: runningLRP(alphaInstanceKey, betaNetInfo),
				Result:        newTestResult(anUnchangedUnclaimedInstanceLRP(), anUpdatedAlphaEvacuatingLRP(), shared.KeepContainer, nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is UNCLAIMED and an evacuating LRP is RUNNING on beta",
				InstanceLRP:   unclaimedLRP(),
				EvacuatingLRP: runningLRP(betaInstanceKey, betaNetInfo),
				Result: newTestResult(
					anUnchangedUnclaimedInstanceLRP(),
					anUnchangedBetaEvacuatingLRP(),
					shared.DeleteContainer,
					nil,
				),
			}),
			runningTest(evacuationTest{
				Name:        "when the instance is CLAIMED on alpha and there is no evacuating LRP",
				InstanceLRP: claimedLRP(alphaInstanceKey),
				Result:      newTestResult(anUpdatedUnclaimedInstanceLRP(), anUpdatedAlphaEvacuatingLRP(), shared.KeepContainer, nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is CLAIMED on alpha and an evacuating LRP is RUNNING on alpha",
				InstanceLRP:   claimedLRP(alphaInstanceKey),
				EvacuatingLRP: runningLRP(alphaInstanceKey, alphaNetInfo),
				Result:        newTestResult(anUpdatedUnclaimedInstanceLRP(), anUnchangedAlphaEvacuatingLRP(), shared.KeepContainer, nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is CLAIMED on alpha and an evacuating LRP is RUNNING on alpha with out-of-date net info",
				InstanceLRP:   claimedLRP(alphaInstanceKey),
				EvacuatingLRP: runningLRP(alphaInstanceKey, betaNetInfo),
				Result:        newTestResult(anUpdatedUnclaimedInstanceLRP(), anUpdatedAlphaEvacuatingLRP(), shared.KeepContainer, nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is CLAIMED on alpha and an evacuating LRP is RUNNING on beta",
				InstanceLRP:   claimedLRP(alphaInstanceKey),
				EvacuatingLRP: runningLRP(betaInstanceKey, betaNetInfo),
				Result:        newTestResult(anUpdatedUnclaimedInstanceLRP(), anUpdatedAlphaEvacuatingLRP(), shared.KeepContainer, nil),
			}),
			runningTest(evacuationTest{
				Name:        "when the instance is CLAIMED remotely and there is no evacuating LRP",
				InstanceLRP: claimedLRP(omegaInstanceKey),
				Result:      newTestResult(anUnchangedClaimedInstanceLRP(omegaInstanceKey), anUpdatedAlphaEvacuatingLRP(), shared.KeepContainer, nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is CLAIMED remotely and an evacuating LRP is RUNNING on alpha",
				InstanceLRP:   claimedLRP(omegaInstanceKey),
				EvacuatingLRP: runningLRP(alphaInstanceKey, alphaNetInfo),
				Result:        newTestResult(anUnchangedClaimedInstanceLRP(omegaInstanceKey), anUnchangedAlphaEvacuatingLRP(), shared.KeepContainer, nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is CLAIMED remotely and an evacuating LRP is RUNNING on alpha with out-of-date net info",
				InstanceLRP:   claimedLRP(omegaInstanceKey),
				EvacuatingLRP: runningLRP(alphaInstanceKey, betaNetInfo),
				Result:        newTestResult(anUnchangedClaimedInstanceLRP(omegaInstanceKey), anUpdatedAlphaEvacuatingLRP(), shared.KeepContainer, nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is CLAIMED remotely and an evacuating LRP is RUNNING on beta",
				InstanceLRP:   claimedLRP(omegaInstanceKey),
				EvacuatingLRP: runningLRP(betaInstanceKey, betaNetInfo),
				Result: newTestResult(
					anUnchangedClaimedInstanceLRP(omegaInstanceKey),
					anUnchangedBetaEvacuatingLRP(),
					shared.DeleteContainer,
					nil,
				),
			}),
			runningTest(evacuationTest{
				Name:        "when the instance is RUNNING on alpha and there is no evacuating LRP",
				InstanceLRP: runningLRP(alphaInstanceKey, alphaNetInfo),
				Result:      newTestResult(anUpdatedUnclaimedInstanceLRP(), anUpdatedAlphaEvacuatingLRP(), shared.KeepContainer, nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is RUNNING on alpha and an evacuating LRP is RUNNING on alpha",
				InstanceLRP:   runningLRP(alphaInstanceKey, alphaNetInfo),
				EvacuatingLRP: runningLRP(alphaInstanceKey, alphaNetInfo),
				Result:        newTestResult(anUpdatedUnclaimedInstanceLRP(), anUnchangedAlphaEvacuatingLRP(), shared.KeepContainer, nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is RUNNING on alpha and an evacuating LRP is RUNNING on alpha with out-of-date net info",
				InstanceLRP:   runningLRP(alphaInstanceKey, alphaNetInfo),
				EvacuatingLRP: runningLRP(alphaInstanceKey, betaNetInfo),
				Result:        newTestResult(anUpdatedUnclaimedInstanceLRP(), anUpdatedAlphaEvacuatingLRP(), shared.KeepContainer, nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is RUNNING on alpha and an evacuating LRP is RUNNING on beta",
				InstanceLRP:   runningLRP(alphaInstanceKey, alphaNetInfo),
				EvacuatingLRP: runningLRP(betaInstanceKey, betaNetInfo),
				Result:        newTestResult(anUpdatedUnclaimedInstanceLRP(), anUpdatedAlphaEvacuatingLRP(), shared.KeepContainer, nil),
			}),
			runningTest(evacuationTest{
				Name:        "when the instance is RUNNING on omega and there is no evacuating LRP",
				InstanceLRP: runningLRP(omegaInstanceKey, omegaNetInfo),
				Result: instanceNoEvacuating(
					anUnchangedRunningInstanceLRP(omegaInstanceKey, omegaNetInfo),
					shared.DeleteContainer,
					nil,
				),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is RUNNING on omega and an evacuating LRP is RUNNING on alpha",
				InstanceLRP:   runningLRP(omegaInstanceKey, omegaNetInfo),
				EvacuatingLRP: runningLRP(alphaInstanceKey, alphaNetInfo),
				Result: instanceNoEvacuating(
					anUnchangedRunningInstanceLRP(omegaInstanceKey, omegaNetInfo),
					shared.DeleteContainer,
					nil,
				),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is RUNNING on omega and an evacuating LRP is RUNNING on beta",
				InstanceLRP:   runningLRP(omegaInstanceKey, omegaNetInfo),
				EvacuatingLRP: runningLRP(betaInstanceKey, betaNetInfo),
				Result: newTestResult(
					anUnchangedRunningInstanceLRP(omegaInstanceKey, omegaNetInfo),
					anUnchangedBetaEvacuatingLRP(),
					shared.DeleteContainer,
					nil,
				),
			}),
			runningTest(evacuationTest{
				Name:        "when the instance is CRASHED and there is no evacuating LRP",
				InstanceLRP: crashedLRP(),
				Result: instanceNoEvacuating(
					anUnchangedCrashedInstanceLRP(),
					shared.DeleteContainer,
					nil,
				),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is CRASHED and an evacuating LRP is RUNNING on alpha",
				InstanceLRP:   crashedLRP(),
				EvacuatingLRP: runningLRP(alphaInstanceKey, alphaNetInfo),
				Result: instanceNoEvacuating(
					anUnchangedCrashedInstanceLRP(),
					shared.DeleteContainer,
					nil,
				),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is CRASHED and an evacuating LRP is RUNNING on beta",
				InstanceLRP:   crashedLRP(),
				EvacuatingLRP: runningLRP(betaInstanceKey, betaNetInfo),
				Result: newTestResult(
					anUnchangedCrashedInstanceLRP(),
					anUnchangedBetaEvacuatingLRP(),
					shared.DeleteContainer,
					nil,
				),
			}),
			runningTest(evacuationTest{
				Name: "when the instance is MISSING and there is no evacuating LRP",
				Result: noInstanceNoEvacuating(
					shared.DeleteContainer,
					nil,
				),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is MISSING and an evacuating LRP is RUNNING on alpha",
				EvacuatingLRP: runningLRP(alphaInstanceKey, alphaNetInfo),
				Result: noInstanceNoEvacuating(
					shared.DeleteContainer,
					nil,
				),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is MISSING and an evacuating LRP is RUNNING on beta",
				EvacuatingLRP: runningLRP(betaInstanceKey, betaNetInfo),
				Result: evacuatingNoInstance(
					anUnchangedBetaEvacuatingLRP(),
					shared.DeleteContainer,
					nil,
				),
			}),
		}

		stoppedTests := []testable{
			stoppedTest(evacuationTest{
				Name:   "when there is no instance or evacuating LRP",
				Result: noInstanceNoEvacuating(shared.DeleteContainer, nil),
			}),
			stoppedTest(evacuationTest{
				Name:        "when the instance is UNCLAIMED",
				InstanceLRP: unclaimedLRP(),
				Result: instanceNoEvacuating(
					anUnchangedUnclaimedInstanceLRP(),
					shared.DeleteContainer,
					bbserrors.ErrActualLRPCannotBeRemoved,
				),
			}),
			stoppedTest(evacuationTest{
				Name:        "when the instance is CLAIMED on alpha",
				InstanceLRP: claimedLRP(alphaInstanceKey),
				Result:      noInstanceNoEvacuating(shared.DeleteContainer, nil),
			}),
			stoppedTest(evacuationTest{
				Name:        "when the instance is CLAIMED on omega",
				InstanceLRP: claimedLRP(omegaInstanceKey),
				Result: instanceNoEvacuating(
					anUnchangedClaimedInstanceLRP(omegaInstanceKey),
					shared.DeleteContainer,
					bbserrors.ErrActualLRPCannotBeRemoved,
				),
			}),
			stoppedTest(evacuationTest{
				Name:        "when the instance is RUNNING on alpha",
				InstanceLRP: runningLRP(alphaInstanceKey, alphaNetInfo),
				Result:      noInstanceNoEvacuating(shared.DeleteContainer, nil),
			}),
			stoppedTest(evacuationTest{
				Name:        "when the instance is RUNNING on omega",
				InstanceLRP: runningLRP(omegaInstanceKey, omegaNetInfo),
				Result: instanceNoEvacuating(
					anUnchangedRunningInstanceLRP(omegaInstanceKey, omegaNetInfo),
					shared.DeleteContainer,
					bbserrors.ErrActualLRPCannotBeRemoved,
				),
			}),
			stoppedTest(evacuationTest{
				Name:        "when the instance is CRASHED",
				InstanceLRP: crashedLRP(),
				Result: instanceNoEvacuating(
					anUnchangedCrashedInstanceLRP(),
					shared.DeleteContainer,
					bbserrors.ErrActualLRPCannotBeRemoved,
				),
			}),
			stoppedTest(evacuationTest{
				Name:          "when the evacuating LRP is RUNNING on alpha",
				EvacuatingLRP: runningLRP(alphaInstanceKey, alphaNetInfo),
				Result:        noInstanceNoEvacuating(shared.DeleteContainer, nil),
			}),
			stoppedTest(evacuationTest{
				Name:          "when the evacuating LRP is RUNNING on beta",
				EvacuatingLRP: runningLRP(betaInstanceKey, betaNetInfo),
				Result:        evacuatingNoInstance(anUnchangedBetaEvacuatingLRP(), shared.DeleteContainer, nil),
			}),
		}

		crashedTests := []testable{
			crashedTest(evacuationTest{
				Name:   "when there is no instance or evacuating LRP",
				Result: noInstanceNoEvacuating(shared.DeleteContainer, nil),
			}),
			crashedTest(evacuationTest{
				Name:        "when the instance is UNCLAIMED",
				InstanceLRP: unclaimedLRP(),
				Result: instanceNoEvacuating(
					anUnchangedUnclaimedInstanceLRP(),
					shared.DeleteContainer,
					bbserrors.ErrActualLRPCannotBeCrashed,
				),
			}),
			crashedTest(evacuationTest{
				Name:        "when the instance is CLAIMED on alpha",
				InstanceLRP: claimedLRP(alphaInstanceKey),
				Result:      instanceNoEvacuating(anUpdatedUnclaimedInstanceLRPWithCrashCount(1), shared.DeleteContainer, nil),
			}),
			crashedTest(evacuationTest{
				Name:        "when the instance is CLAIMED on omega",
				InstanceLRP: claimedLRP(omegaInstanceKey),
				Result: instanceNoEvacuating(
					anUnchangedClaimedInstanceLRP(omegaInstanceKey),
					shared.DeleteContainer,
					bbserrors.ErrActualLRPCannotBeCrashed,
				),
			}),
			crashedTest(evacuationTest{
				Name:        "when the instance is RUNNING on alpha",
				InstanceLRP: runningLRP(alphaInstanceKey, alphaNetInfo),
				Result:      instanceNoEvacuating(anUpdatedUnclaimedInstanceLRPWithCrashCount(1), shared.DeleteContainer, nil),
			}),
			crashedTest(evacuationTest{
				Name:        "when the instance is RUNNING on omega",
				InstanceLRP: runningLRP(omegaInstanceKey, omegaNetInfo),
				Result: instanceNoEvacuating(
					anUnchangedRunningInstanceLRP(omegaInstanceKey, omegaNetInfo),
					shared.DeleteContainer,
					bbserrors.ErrActualLRPCannotBeCrashed,
				),
			}),
			crashedTest(evacuationTest{
				Name:        "when the instance is CRASHED",
				InstanceLRP: crashedLRP(),
				Result: instanceNoEvacuating(
					anUnchangedCrashedInstanceLRP(),
					shared.DeleteContainer,
					bbserrors.ErrActualLRPCannotBeCrashed,
				),
			}),
			crashedTest(evacuationTest{
				Name:          "when the evacuating LRP is RUNNING on alpha",
				EvacuatingLRP: runningLRP(alphaInstanceKey, alphaNetInfo),
				Result:        noInstanceNoEvacuating(shared.DeleteContainer, nil),
			}),
			crashedTest(evacuationTest{
				Name:          "when the evacuating LRP is RUNNING on beta",
				EvacuatingLRP: runningLRP(betaInstanceKey, betaNetInfo),
				Result:        evacuatingNoInstance(anUnchangedBetaEvacuatingLRP(), shared.DeleteContainer, nil),
			}),
		}

		removalTests := []testable{
			removalTest(evacuationTest{
				Name:   "when there is no instance or evacuating LRP",
				Result: noInstanceNoEvacuating(shared.DeleteContainer, nil),
			}),
			removalTest(evacuationTest{
				Name:        "when the instance is UNCLAIMED",
				InstanceLRP: unclaimedLRP(),
				Result:      instanceNoEvacuating(anUnchangedUnclaimedInstanceLRP(), shared.DeleteContainer, nil),
			}),
			removalTest(evacuationTest{
				Name:        "when the instance is CLAIMED on alpha",
				InstanceLRP: claimedLRP(alphaInstanceKey),
				Result:      instanceNoEvacuating(anUnchangedClaimedInstanceLRP(alphaInstanceKey), shared.DeleteContainer, nil),
			}),
			removalTest(evacuationTest{
				Name:        "when the instance is CLAIMED on omega",
				InstanceLRP: claimedLRP(omegaInstanceKey),
				Result:      instanceNoEvacuating(anUnchangedClaimedInstanceLRP(omegaInstanceKey), shared.DeleteContainer, nil),
			}),
			removalTest(evacuationTest{
				Name:        "when the instance is RUNNING on alpha",
				InstanceLRP: runningLRP(alphaInstanceKey, alphaNetInfo),
				Result:      instanceNoEvacuating(anUnchangedRunningInstanceLRP(alphaInstanceKey, alphaNetInfo), shared.DeleteContainer, nil),
			}),
			removalTest(evacuationTest{
				Name:        "when the instance is RUNNING on omega",
				InstanceLRP: runningLRP(omegaInstanceKey, omegaNetInfo),
				Result:      instanceNoEvacuating(anUnchangedRunningInstanceLRP(omegaInstanceKey, omegaNetInfo), shared.DeleteContainer, nil),
			}),
			removalTest(evacuationTest{
				Name:        "when the instance is CRASHED",
				InstanceLRP: crashedLRP(),
				Result:      instanceNoEvacuating(anUnchangedCrashedInstanceLRP(), shared.DeleteContainer, nil),
			}),
			removalTest(evacuationTest{
				Name:          "when the evacuating LRP is RUNNING on alpha",
				EvacuatingLRP: runningLRP(alphaInstanceKey, alphaNetInfo),
				Result:        noInstanceNoEvacuating(shared.DeleteContainer, nil),
			}),
			removalTest(evacuationTest{
				Name:          "when the evacuating LRP is RUNNING on beta",
				EvacuatingLRP: runningLRP(betaInstanceKey, betaNetInfo),
				Result: evacuatingNoInstance(
					anUnchangedBetaEvacuatingLRP(),
					shared.DeleteContainer,
					bbserrors.ErrActualLRPCannotBeRemoved,
				),
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

		Context("when the LRP is to be CRASHED", func() {
			for _, test := range crashedTests {
				test.Test()
			}
		})

		Context("when the evacuating LRP is to be removed", func() {
			for _, test := range removalTests {
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

	alphaInstanceKey = models.NewActualLRPInstanceKey(alphaInstanceGuid, alphaCellID)
	betaInstanceKey  = models.NewActualLRPInstanceKey(betaInstanceGuid, betaCellID)
	omegaInstanceKey = models.NewActualLRPInstanceKey(omegaInstanceGuid, omegaCellID)
	emptyInstanceKey = models.ActualLRPInstanceKey{}

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
	Subject       func() (shared.ContainerRetainment, error)
	InstanceLRP   lrpSetupFunc
	EvacuatingLRP lrpSetupFunc
	Result        testResult
}

func lrp(state models.ActualLRPState, instanceKey models.ActualLRPInstanceKey, netInfo models.ActualLRPNetInfo, placementError string) lrpSetupFunc {
	return func() models.ActualLRP {
		return models.ActualLRP{
			ActualLRPKey:         lrpKey,
			ActualLRPInstanceKey: instanceKey,
			ActualLRPNetInfo:     netInfo,
			State:                state,
			Since:                clock.Now().UnixNano(),
			PlacementError:       placementError,
		}
	}
}

func unclaimedLRP() lrpSetupFunc {
	return lrp(models.ActualLRPStateUnclaimed, emptyInstanceKey, emptyNetInfo, "")
}

func unclaimedLRPWithPlacementError() lrpSetupFunc {
	return lrp(models.ActualLRPStateUnclaimed, emptyInstanceKey, emptyNetInfo, diego_errors.INSUFFICIENT_RESOURCES_MESSAGE)
}

func claimedLRP(instanceKey models.ActualLRPInstanceKey) lrpSetupFunc {
	return lrp(models.ActualLRPStateClaimed, instanceKey, emptyNetInfo, "")
}

func runningLRP(instanceKey models.ActualLRPInstanceKey, netInfo models.ActualLRPNetInfo) lrpSetupFunc {
	return lrp(models.ActualLRPStateRunning, instanceKey, netInfo, "")
}

func crashedLRP() lrpSetupFunc {
	actualFunc := lrp(models.ActualLRPStateCrashed, emptyInstanceKey, emptyNetInfo, "")
	return func() models.ActualLRP {
		actual := actualFunc()
		actual.CrashReason = "crashed"
		return actual
	}
}

type lrpStatus struct {
	State models.ActualLRPState
	models.ActualLRPInstanceKey
	models.ActualLRPNetInfo
	ShouldUpdate bool
}

type instanceLRPStatus struct {
	lrpStatus
	CrashCount  int
	CrashReason string
}

type evacuatingLRPStatus struct {
	lrpStatus
	TTL uint64
}

type testResult struct {
	Instance         *instanceLRPStatus
	Evacuating       *evacuatingLRPStatus
	AuctionRequested bool
	ReturnedError    error
	RetainContainer  shared.ContainerRetainment
}

func anUpdatedAlphaEvacuatingLRP() *evacuatingLRPStatus {
	return newEvacuatingLRPStatus(alphaInstanceKey, alphaNetInfo, true)
}

func anUnchangedAlphaEvacuatingLRP() *evacuatingLRPStatus {
	return newEvacuatingLRPStatus(alphaInstanceKey, alphaNetInfo, false)
}

func anUnchangedBetaEvacuatingLRP() *evacuatingLRPStatus {
	return newEvacuatingLRPStatus(betaInstanceKey, betaNetInfo, false)
}

func newEvacuatingLRPStatus(instanceKey models.ActualLRPInstanceKey, netInfo models.ActualLRPNetInfo, shouldUpdate bool) *evacuatingLRPStatus {
	status := &evacuatingLRPStatus{
		lrpStatus: lrpStatus{
			State:                models.ActualLRPStateRunning,
			ActualLRPInstanceKey: instanceKey,
			ActualLRPNetInfo:     netInfo,
			ShouldUpdate:         shouldUpdate,
		},
	}

	if shouldUpdate {
		status.TTL = alphaEvacuationTTL
	}

	return status
}

func anUpdatedUnclaimedInstanceLRP() *instanceLRPStatus {
	return anUpdatedUnclaimedInstanceLRPWithCrashCount(0)
}

func anUpdatedUnclaimedInstanceLRPWithCrashCount(crashCount int) *instanceLRPStatus {
	reason := ""
	if crashCount > 0 {
		reason = "crashed"
	}
	return &instanceLRPStatus{
		lrpStatus: lrpStatus{
			State:                models.ActualLRPStateUnclaimed,
			ActualLRPInstanceKey: emptyInstanceKey,
			ActualLRPNetInfo:     emptyNetInfo,
			ShouldUpdate:         true,
		},
		CrashCount:  crashCount,
		CrashReason: reason,
	}
}

func anUnchangedInstanceLRP(state models.ActualLRPState, instanceKey models.ActualLRPInstanceKey, netInfo models.ActualLRPNetInfo) *instanceLRPStatus {
	return &instanceLRPStatus{
		lrpStatus: lrpStatus{
			State:                state,
			ActualLRPInstanceKey: instanceKey,
			ActualLRPNetInfo:     netInfo,
			ShouldUpdate:         false,
		},
	}
}

func anUnchangedUnclaimedInstanceLRP() *instanceLRPStatus {
	return anUnchangedInstanceLRP(models.ActualLRPStateUnclaimed, emptyInstanceKey, emptyNetInfo)
}

func anUnchangedClaimedInstanceLRP(instanceKey models.ActualLRPInstanceKey) *instanceLRPStatus {
	return anUnchangedInstanceLRP(models.ActualLRPStateClaimed, instanceKey, emptyNetInfo)
}

func anUnchangedRunningInstanceLRP(instanceKey models.ActualLRPInstanceKey, netInfo models.ActualLRPNetInfo) *instanceLRPStatus {
	return anUnchangedInstanceLRP(models.ActualLRPStateRunning, instanceKey, netInfo)
}

func anUnchangedCrashedInstanceLRP() *instanceLRPStatus {
	instance := anUnchangedInstanceLRP(models.ActualLRPStateCrashed, emptyInstanceKey, emptyNetInfo)
	instance.CrashReason = "crashed"
	return instance
}

func newTestResult(instanceStatus *instanceLRPStatus, evacuatingStatus *evacuatingLRPStatus, retainContainer shared.ContainerRetainment, err error) testResult {
	result := testResult{
		Instance:        instanceStatus,
		Evacuating:      evacuatingStatus,
		ReturnedError:   err,
		RetainContainer: retainContainer,
	}

	if instanceStatus != nil && instanceStatus.ShouldUpdate {
		result.AuctionRequested = true
	}

	return result
}

func instanceNoEvacuating(instanceStatus *instanceLRPStatus, retainContainer shared.ContainerRetainment, err error) testResult {
	return newTestResult(instanceStatus, nil, retainContainer, err)
}

func evacuatingNoInstance(evacuatingStatus *evacuatingLRPStatus, retainContainer shared.ContainerRetainment, err error) testResult {
	return newTestResult(nil, evacuatingStatus, retainContainer, err)
}

func noInstanceNoEvacuating(retainContainer shared.ContainerRetainment, err error) testResult {
	return newTestResult(nil, nil, retainContainer, err)
}

func (t evacuationTest) Test() {
	Context(t.Name, func() {
		var evacuateErr error
		var auctioneerPresence models.AuctioneerPresence
		var initialTimestamp int64
		var initialInstanceModificationIndex uint
		var initialEvacuatingModificationIndex uint
		var retainContainer shared.ContainerRetainment

		BeforeEach(func() {
			auctioneerPresence = models.NewAuctioneerPresence("the-auctioneer-id", "the-address")
			initialTimestamp = clock.Now().UnixNano()

			registerAuctioneer(auctioneerPresence)
			setRawDesiredLRP(desiredLRP)
			if t.InstanceLRP != nil {
				actualLRP := t.InstanceLRP()
				initialInstanceModificationIndex = actualLRP.ModificationTag.Index
				setRawActualLRP(actualLRP)
			}
			if t.EvacuatingLRP != nil {
				evacuatingLRP := t.EvacuatingLRP()
				initialEvacuatingModificationIndex = evacuatingLRP.ModificationTag.Index
				setRawEvacuatingActualLRP(evacuatingLRP, omegaEvacuationTTL)
			}
		})

		JustBeforeEach(func() {
			clock.Increment(timeIncrement)
			retainContainer, evacuateErr = t.Subject()
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

		if t.Result.RetainContainer == shared.KeepContainer {
			It("returns KeepContainer", func() {
				Ω(retainContainer).Should(Equal(shared.KeepContainer))
			})

		} else {
			It("returns DeleteContainer", func() {
				Ω(retainContainer).Should(Equal(shared.DeleteContainer))
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

			Context("when the desired LRP no longer exists", func() {
				BeforeEach(func() {
					err := bbs.RemoveDesiredLRPByProcessGuid(logger, desiredLRP.ProcessGuid)
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("the actual LRP is also deleted", func() {
					Ω(evacuateErr).ShouldNot(HaveOccurred())

					lrpGroup, _ := bbs.ActualLRPGroupByProcessGuidAndIndex(t.InstanceLRP().ProcessGuid, t.InstanceLRP().Index)
					Ω(lrpGroup.Instance).Should(BeNil())
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

				It("updates the /instance ModificationTag", func() {
					lrpInBBS, err := getInstanceActualLRP(lrpKey)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrpInBBS.ModificationTag.Index).Should(Equal(initialInstanceModificationIndex + 1))
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

			It("has the expected /instance crash count", func() {
				lrpInBBS, err := getInstanceActualLRP(lrpKey)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(lrpInBBS.CrashCount).Should(Equal(t.Result.Instance.CrashCount))
			})

			It("has the expected /instance crash reason", func() {
				lrpInBBS, err := getInstanceActualLRP(lrpKey)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(lrpInBBS.CrashReason).Should(Equal(t.Result.Instance.CrashReason))
			})

			It("has the expected /instance instance key", func() {
				lrpInBBS, err := getInstanceActualLRP(lrpKey)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(lrpInBBS.ActualLRPInstanceKey).Should(Equal(t.Result.Instance.ActualLRPInstanceKey))
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

				It("updates the /evacuating ModificationTag", func() {
					lrpInBBS, _, err := getEvacuatingActualLRP(lrpKey)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrpInBBS.ModificationTag.Index).Should(Equal(initialEvacuatingModificationIndex + 1))
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

			It("has the expected /evacuating instance key", func() {
				lrpInBBS, _, err := getEvacuatingActualLRP(lrpKey)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(lrpInBBS.ActualLRPInstanceKey).Should(Equal(t.Result.Evacuating.ActualLRPInstanceKey))
			})

			It("has the expected /evacuating net info", func() {
				lrpInBBS, _, err := getEvacuatingActualLRP(lrpKey)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(lrpInBBS.ActualLRPNetInfo).Should(Equal(t.Result.Evacuating.ActualLRPNetInfo))
			})
		}
	})
}
