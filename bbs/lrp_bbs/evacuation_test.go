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
				Subject: func() (shared.ContainerRetainment, error) {
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
				Subject: func() (shared.ContainerRetainment, error) {
					return bbs.EvacuateStoppedActualLRP(logger, lrpKey, alphaContainerKey)
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
					return bbs.EvacuateCrashedActualLRP(logger, lrpKey, alphaContainerKey)
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
					err := bbs.RemoveEvacuatingActualLRP(logger, lrpKey, alphaContainerKey)
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
				InstanceLRP: claimedLRP(alphaContainerKey),
				Result:      instanceNoEvacuating(anUpdatedUnclaimedInstanceLRP(), shared.DeleteContainer, nil),
			}),
			claimedTest(evacuationTest{
				Name:        "when the instance is CLAIMED on omega",
				InstanceLRP: claimedLRP(omegaContainerKey),
				Result: instanceNoEvacuating(
					anUnchangedClaimedInstanceLRP(omegaContainerKey),
					shared.DeleteContainer,
					bbserrors.ErrActualLRPCannotBeUnclaimed,
				),
			}),
			claimedTest(evacuationTest{
				Name:        "when the instance is RUNNING on alpha",
				InstanceLRP: runningLRP(alphaContainerKey, alphaNetInfo),
				Result:      instanceNoEvacuating(anUpdatedUnclaimedInstanceLRP(), shared.DeleteContainer, nil),
			}),
			claimedTest(evacuationTest{
				Name:        "when the instance is RUNNING on omega",
				InstanceLRP: runningLRP(omegaContainerKey, omegaNetInfo),
				Result: instanceNoEvacuating(
					anUnchangedRunningInstanceLRP(omegaContainerKey, omegaNetInfo),
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
				EvacuatingLRP: runningLRP(alphaContainerKey, alphaNetInfo),
				Result:        noInstanceNoEvacuating(shared.DeleteContainer, nil),
			}),
			claimedTest(evacuationTest{
				Name:          "when the evacuating LRP is RUNNING on beta",
				EvacuatingLRP: runningLRP(betaContainerKey, betaNetInfo),
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
				EvacuatingLRP: runningLRP(alphaContainerKey, alphaNetInfo),
				Result:        newTestResult(anUnchangedUnclaimedInstanceLRP(), anUnchangedAlphaEvacuatingLRP(), shared.KeepContainer, nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is UNCLAIMED with a placement error and an evacuating LRP is RUNNING on alpha",
				InstanceLRP:   unclaimedLRPWithPlacementError(),
				EvacuatingLRP: runningLRP(alphaContainerKey, alphaNetInfo),
				Result: instanceNoEvacuating(
					anUnchangedUnclaimedInstanceLRP(),
					shared.DeleteContainer,
					nil,
				),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is UNCLAIMED and an evacuating LRP is RUNNING on alpha with out-of-date net info",
				InstanceLRP:   unclaimedLRP(),
				EvacuatingLRP: runningLRP(alphaContainerKey, betaNetInfo),
				Result:        newTestResult(anUnchangedUnclaimedInstanceLRP(), anUpdatedAlphaEvacuatingLRP(), shared.KeepContainer, nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is UNCLAIMED and an evacuating LRP is RUNNING on beta",
				InstanceLRP:   unclaimedLRP(),
				EvacuatingLRP: runningLRP(betaContainerKey, betaNetInfo),
				Result: newTestResult(
					anUnchangedUnclaimedInstanceLRP(),
					anUnchangedBetaEvacuatingLRP(),
					shared.DeleteContainer,
					nil,
				),
			}),
			runningTest(evacuationTest{
				Name:        "when the instance is CLAIMED on alpha and there is no evacuating LRP",
				InstanceLRP: claimedLRP(alphaContainerKey),
				Result:      newTestResult(anUpdatedUnclaimedInstanceLRP(), anUpdatedAlphaEvacuatingLRP(), shared.KeepContainer, nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is CLAIMED on alpha and an evacuating LRP is RUNNING on alpha",
				InstanceLRP:   claimedLRP(alphaContainerKey),
				EvacuatingLRP: runningLRP(alphaContainerKey, alphaNetInfo),
				Result:        newTestResult(anUpdatedUnclaimedInstanceLRP(), anUnchangedAlphaEvacuatingLRP(), shared.KeepContainer, nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is CLAIMED on alpha and an evacuating LRP is RUNNING on alpha with out-of-date net info",
				InstanceLRP:   claimedLRP(alphaContainerKey),
				EvacuatingLRP: runningLRP(alphaContainerKey, betaNetInfo),
				Result:        newTestResult(anUpdatedUnclaimedInstanceLRP(), anUpdatedAlphaEvacuatingLRP(), shared.KeepContainer, nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is CLAIMED on alpha and an evacuating LRP is RUNNING on beta",
				InstanceLRP:   claimedLRP(alphaContainerKey),
				EvacuatingLRP: runningLRP(betaContainerKey, betaNetInfo),
				Result:        newTestResult(anUpdatedUnclaimedInstanceLRP(), anUpdatedAlphaEvacuatingLRP(), shared.KeepContainer, nil),
			}),
			runningTest(evacuationTest{
				Name:        "when the instance is CLAIMED remotely and there is no evacuating LRP",
				InstanceLRP: claimedLRP(omegaContainerKey),
				Result:      newTestResult(anUnchangedClaimedInstanceLRP(omegaContainerKey), anUpdatedAlphaEvacuatingLRP(), shared.KeepContainer, nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is CLAIMED remotely and an evacuating LRP is RUNNING on alpha",
				InstanceLRP:   claimedLRP(omegaContainerKey),
				EvacuatingLRP: runningLRP(alphaContainerKey, alphaNetInfo),
				Result:        newTestResult(anUnchangedClaimedInstanceLRP(omegaContainerKey), anUnchangedAlphaEvacuatingLRP(), shared.KeepContainer, nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is CLAIMED remotely and an evacuating LRP is RUNNING on alpha with out-of-date net info",
				InstanceLRP:   claimedLRP(omegaContainerKey),
				EvacuatingLRP: runningLRP(alphaContainerKey, betaNetInfo),
				Result:        newTestResult(anUnchangedClaimedInstanceLRP(omegaContainerKey), anUpdatedAlphaEvacuatingLRP(), shared.KeepContainer, nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is CLAIMED remotely and an evacuating LRP is RUNNING on beta",
				InstanceLRP:   claimedLRP(omegaContainerKey),
				EvacuatingLRP: runningLRP(betaContainerKey, betaNetInfo),
				Result: newTestResult(
					anUnchangedClaimedInstanceLRP(omegaContainerKey),
					anUnchangedBetaEvacuatingLRP(),
					shared.DeleteContainer,
					nil,
				),
			}),
			runningTest(evacuationTest{
				Name:        "when the instance is RUNNING on alpha and there is no evacuating LRP",
				InstanceLRP: runningLRP(alphaContainerKey, alphaNetInfo),
				Result:      newTestResult(anUpdatedUnclaimedInstanceLRP(), anUpdatedAlphaEvacuatingLRP(), shared.KeepContainer, nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is RUNNING on alpha and an evacuating LRP is RUNNING on alpha",
				InstanceLRP:   runningLRP(alphaContainerKey, alphaNetInfo),
				EvacuatingLRP: runningLRP(alphaContainerKey, alphaNetInfo),
				Result:        newTestResult(anUpdatedUnclaimedInstanceLRP(), anUnchangedAlphaEvacuatingLRP(), shared.KeepContainer, nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is RUNNING on alpha and an evacuating LRP is RUNNING on alpha with out-of-date net info",
				InstanceLRP:   runningLRP(alphaContainerKey, alphaNetInfo),
				EvacuatingLRP: runningLRP(alphaContainerKey, betaNetInfo),
				Result:        newTestResult(anUpdatedUnclaimedInstanceLRP(), anUpdatedAlphaEvacuatingLRP(), shared.KeepContainer, nil),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is RUNNING on alpha and an evacuating LRP is RUNNING on beta",
				InstanceLRP:   runningLRP(alphaContainerKey, alphaNetInfo),
				EvacuatingLRP: runningLRP(betaContainerKey, betaNetInfo),
				Result:        newTestResult(anUpdatedUnclaimedInstanceLRP(), anUpdatedAlphaEvacuatingLRP(), shared.KeepContainer, nil),
			}),
			runningTest(evacuationTest{
				Name:        "when the instance is RUNNING on omega and there is no evacuating LRP",
				InstanceLRP: runningLRP(omegaContainerKey, omegaNetInfo),
				Result: instanceNoEvacuating(
					anUnchangedRunningInstanceLRP(omegaContainerKey, omegaNetInfo),
					shared.DeleteContainer,
					nil,
				),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is RUNNING on omega and an evacuating LRP is RUNNING on alpha",
				InstanceLRP:   runningLRP(omegaContainerKey, omegaNetInfo),
				EvacuatingLRP: runningLRP(alphaContainerKey, alphaNetInfo),
				Result: instanceNoEvacuating(
					anUnchangedRunningInstanceLRP(omegaContainerKey, omegaNetInfo),
					shared.DeleteContainer,
					nil,
				),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is RUNNING on omega and an evacuating LRP is RUNNING on beta",
				InstanceLRP:   runningLRP(omegaContainerKey, omegaNetInfo),
				EvacuatingLRP: runningLRP(betaContainerKey, betaNetInfo),
				Result: newTestResult(
					anUnchangedRunningInstanceLRP(omegaContainerKey, omegaNetInfo),
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
				EvacuatingLRP: runningLRP(alphaContainerKey, alphaNetInfo),
				Result: instanceNoEvacuating(
					anUnchangedCrashedInstanceLRP(),
					shared.DeleteContainer,
					nil,
				),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is CRASHED and an evacuating LRP is RUNNING on beta",
				InstanceLRP:   crashedLRP(),
				EvacuatingLRP: runningLRP(betaContainerKey, betaNetInfo),
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
				EvacuatingLRP: runningLRP(alphaContainerKey, alphaNetInfo),
				Result: noInstanceNoEvacuating(
					shared.DeleteContainer,
					nil,
				),
			}),
			runningTest(evacuationTest{
				Name:          "when the instance is MISSING and an evacuating LRP is RUNNING on beta",
				EvacuatingLRP: runningLRP(betaContainerKey, betaNetInfo),
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
				InstanceLRP: claimedLRP(alphaContainerKey),
				Result:      noInstanceNoEvacuating(shared.DeleteContainer, nil),
			}),
			stoppedTest(evacuationTest{
				Name:        "when the instance is CLAIMED on omega",
				InstanceLRP: claimedLRP(omegaContainerKey),
				Result: instanceNoEvacuating(
					anUnchangedClaimedInstanceLRP(omegaContainerKey),
					shared.DeleteContainer,
					bbserrors.ErrActualLRPCannotBeRemoved,
				),
			}),
			stoppedTest(evacuationTest{
				Name:        "when the instance is RUNNING on alpha",
				InstanceLRP: runningLRP(alphaContainerKey, alphaNetInfo),
				Result:      noInstanceNoEvacuating(shared.DeleteContainer, nil),
			}),
			stoppedTest(evacuationTest{
				Name:        "when the instance is RUNNING on omega",
				InstanceLRP: runningLRP(omegaContainerKey, omegaNetInfo),
				Result: instanceNoEvacuating(
					anUnchangedRunningInstanceLRP(omegaContainerKey, omegaNetInfo),
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
				EvacuatingLRP: runningLRP(alphaContainerKey, alphaNetInfo),
				Result:        noInstanceNoEvacuating(shared.DeleteContainer, nil),
			}),
			stoppedTest(evacuationTest{
				Name:          "when the evacuating LRP is RUNNING on beta",
				EvacuatingLRP: runningLRP(betaContainerKey, betaNetInfo),
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
				InstanceLRP: claimedLRP(alphaContainerKey),
				Result:      instanceNoEvacuating(anUpdatedUnclaimedInstanceLRPWithCrashCount(1), shared.DeleteContainer, nil),
			}),
			crashedTest(evacuationTest{
				Name:        "when the instance is CLAIMED on omega",
				InstanceLRP: claimedLRP(omegaContainerKey),
				Result: instanceNoEvacuating(
					anUnchangedClaimedInstanceLRP(omegaContainerKey),
					shared.DeleteContainer,
					bbserrors.ErrActualLRPCannotBeCrashed,
				),
			}),
			crashedTest(evacuationTest{
				Name:        "when the instance is RUNNING on alpha",
				InstanceLRP: runningLRP(alphaContainerKey, alphaNetInfo),
				Result:      instanceNoEvacuating(anUpdatedUnclaimedInstanceLRPWithCrashCount(1), shared.DeleteContainer, nil),
			}),
			crashedTest(evacuationTest{
				Name:        "when the instance is RUNNING on omega",
				InstanceLRP: runningLRP(omegaContainerKey, omegaNetInfo),
				Result: instanceNoEvacuating(
					anUnchangedRunningInstanceLRP(omegaContainerKey, omegaNetInfo),
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
				EvacuatingLRP: runningLRP(alphaContainerKey, alphaNetInfo),
				Result:        noInstanceNoEvacuating(shared.DeleteContainer, nil),
			}),
			crashedTest(evacuationTest{
				Name:          "when the evacuating LRP is RUNNING on beta",
				EvacuatingLRP: runningLRP(betaContainerKey, betaNetInfo),
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
				InstanceLRP: claimedLRP(alphaContainerKey),
				Result:      instanceNoEvacuating(anUnchangedClaimedInstanceLRP(alphaContainerKey), shared.DeleteContainer, nil),
			}),
			removalTest(evacuationTest{
				Name:        "when the instance is CLAIMED on omega",
				InstanceLRP: claimedLRP(omegaContainerKey),
				Result:      instanceNoEvacuating(anUnchangedClaimedInstanceLRP(omegaContainerKey), shared.DeleteContainer, nil),
			}),
			removalTest(evacuationTest{
				Name:        "when the instance is RUNNING on alpha",
				InstanceLRP: runningLRP(alphaContainerKey, alphaNetInfo),
				Result:      instanceNoEvacuating(anUnchangedRunningInstanceLRP(alphaContainerKey, alphaNetInfo), shared.DeleteContainer, nil),
			}),
			removalTest(evacuationTest{
				Name:        "when the instance is RUNNING on omega",
				InstanceLRP: runningLRP(omegaContainerKey, omegaNetInfo),
				Result:      instanceNoEvacuating(anUnchangedRunningInstanceLRP(omegaContainerKey, omegaNetInfo), shared.DeleteContainer, nil),
			}),
			removalTest(evacuationTest{
				Name:        "when the instance is CRASHED",
				InstanceLRP: crashedLRP(),
				Result:      instanceNoEvacuating(anUnchangedCrashedInstanceLRP(), shared.DeleteContainer, nil),
			}),
			removalTest(evacuationTest{
				Name:          "when the evacuating LRP is RUNNING on alpha",
				EvacuatingLRP: runningLRP(alphaContainerKey, alphaNetInfo),
				Result:        noInstanceNoEvacuating(shared.DeleteContainer, nil),
			}),
			removalTest(evacuationTest{
				Name:          "when the evacuating LRP is RUNNING on beta",
				EvacuatingLRP: runningLRP(betaContainerKey, betaNetInfo),
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
	Subject       func() (shared.ContainerRetainment, error)
	InstanceLRP   lrpSetupFunc
	EvacuatingLRP lrpSetupFunc
	Result        testResult
}

func lrp(state models.ActualLRPState, containerKey models.ActualLRPContainerKey, netInfo models.ActualLRPNetInfo, placementError string) lrpSetupFunc {
	return func() models.ActualLRP {
		return models.ActualLRP{
			ActualLRPKey:          lrpKey,
			ActualLRPContainerKey: containerKey,
			ActualLRPNetInfo:      netInfo,
			State:                 state,
			Since:                 clock.Now().UnixNano(),
			PlacementError:        placementError,
		}
	}
}

func unclaimedLRP() lrpSetupFunc {
	return lrp(models.ActualLRPStateUnclaimed, emptyContainerKey, emptyNetInfo, "")
}

func unclaimedLRPWithPlacementError() lrpSetupFunc {
	return lrp(models.ActualLRPStateUnclaimed, emptyContainerKey, emptyNetInfo, diego_errors.INSUFFICIENT_RESOURCES_MESSAGE)
}

func claimedLRP(containerKey models.ActualLRPContainerKey) lrpSetupFunc {
	return lrp(models.ActualLRPStateClaimed, containerKey, emptyNetInfo, "")
}

func runningLRP(containerKey models.ActualLRPContainerKey, netInfo models.ActualLRPNetInfo) lrpSetupFunc {
	return lrp(models.ActualLRPStateRunning, containerKey, netInfo, "")
}

func crashedLRP() lrpSetupFunc {
	return lrp(models.ActualLRPStateCrashed, emptyContainerKey, emptyNetInfo, "")
}

type lrpStatus struct {
	State models.ActualLRPState
	models.ActualLRPContainerKey
	models.ActualLRPNetInfo
	ShouldUpdate bool
}

type instanceLRPStatus struct {
	lrpStatus
	CrashCount int
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

func anUpdatedUnclaimedInstanceLRP() *instanceLRPStatus {
	return anUpdatedUnclaimedInstanceLRPWithCrashCount(0)
}

func anUpdatedUnclaimedInstanceLRPWithCrashCount(crashCount int) *instanceLRPStatus {
	return &instanceLRPStatus{
		lrpStatus: lrpStatus{
			State: models.ActualLRPStateUnclaimed,
			ActualLRPContainerKey: emptyContainerKey,
			ActualLRPNetInfo:      emptyNetInfo,
			ShouldUpdate:          true,
		},
		CrashCount: crashCount,
	}
}

func anUnchangedInstanceLRP(state models.ActualLRPState, containerKey models.ActualLRPContainerKey, netInfo models.ActualLRPNetInfo) *instanceLRPStatus {
	return &instanceLRPStatus{
		lrpStatus: lrpStatus{
			State: state,
			ActualLRPContainerKey: containerKey,
			ActualLRPNetInfo:      netInfo,
			ShouldUpdate:          false,
		},
	}
}

func anUnchangedUnclaimedInstanceLRP() *instanceLRPStatus {
	return anUnchangedInstanceLRP(models.ActualLRPStateUnclaimed, emptyContainerKey, emptyNetInfo)
}

func anUnchangedClaimedInstanceLRP(containerKey models.ActualLRPContainerKey) *instanceLRPStatus {
	return anUnchangedInstanceLRP(models.ActualLRPStateClaimed, containerKey, emptyNetInfo)
}

func anUnchangedRunningInstanceLRP(containerKey models.ActualLRPContainerKey, netInfo models.ActualLRPNetInfo) *instanceLRPStatus {
	return anUnchangedInstanceLRP(models.ActualLRPStateRunning, containerKey, netInfo)
}

func anUnchangedCrashedInstanceLRP() *instanceLRPStatus {
	return anUnchangedInstanceLRP(models.ActualLRPStateCrashed, emptyContainerKey, emptyNetInfo)
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
