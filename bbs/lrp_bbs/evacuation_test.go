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
	Describe("EvacuateClaimedActualLRP (tabular)", func() {
		tests := []testable{
			evacuationTest{
				Name:        "when the instance is UNCLAIMED",
				InstanceLRP: lrp(models.ActualLRPStateUnclaimed, nullContainerKey, nullNetInfo),
				Result:      anUnchangedUnclaimedInstance(),
			},
			evacuationTest{
				Name:        "when the instance is CLAIMED locally",
				InstanceLRP: lrp(models.ActualLRPStateClaimed, localContainerKey, nullNetInfo),
				Result:      anUpdatedUnclaimedInstance(),
			},
			evacuationTest{
				Name:        "when the instance is CLAIMED elsewhere",
				InstanceLRP: lrp(models.ActualLRPStateClaimed, otherContainerKey, nullNetInfo),
				Result:      anUnchangedInstance(models.ActualLRPStateClaimed, otherContainerKey, nullNetInfo, bbserrors.ErrActualLRPCannotBeUnclaimed),
			},
			evacuationTest{
				Name:        "when the instance is RUNNING locally",
				InstanceLRP: lrp(models.ActualLRPStateRunning, localContainerKey, localNetInfo),
				Result:      anUpdatedUnclaimedInstance(),
			},
			evacuationTest{
				Name:        "when the instance is RUNNING elsewhere",
				InstanceLRP: lrp(models.ActualLRPStateRunning, otherContainerKey, otherNetInfo),
				Result:      anUnchangedInstance(models.ActualLRPStateRunning, otherContainerKey, otherNetInfo, bbserrors.ErrActualLRPCannotBeUnclaimed),
			},
			evacuationTest{
				Name:        "when the instance is CRASHED",
				InstanceLRP: lrp(models.ActualLRPStateCrashed, nullContainerKey, nullNetInfo),
				Result:      anUnchangedInstance(models.ActualLRPStateCrashed, nullContainerKey, nullNetInfo, bbserrors.ErrActualLRPCannotBeUnclaimed),
			},
		}

		for _, test := range tests {
			test.Test()
		}
	})
})

const (
	initialTimestamp  = 1138
	timeIncrement     = 2279
	finalTimestamp    = initialTimestamp + timeIncrement
	processGuid       = "process-guid"
	localInstanceGuid = "local-instance-guid"
	otherInstanceGuid = "other-instance-guid"
	localCellID       = "local-cell-id"
	otherCellID       = "other-cell-id"
	localAddress      = "local-address"
	otherAddress      = "other-address"
)

var (
	desiredLRP = models.DesiredLRP{
		ProcessGuid: processGuid,
		Domain:      "domain",
		Instances:   1,
		Stack:       "some-stack",
		Action:      &models.RunAction{Path: "/bin/true"},
	}

	index             = 0
	lrpKey            = models.NewActualLRPKey(desiredLRP.ProcessGuid, index, desiredLRP.Domain)
	localContainerKey = models.NewActualLRPContainerKey(localInstanceGuid, localCellID)
	otherContainerKey = models.NewActualLRPContainerKey(otherInstanceGuid, otherCellID)
	nullContainerKey  = models.ActualLRPContainerKey{}
	localPorts        = []models.PortMapping{{ContainerPort: 2349, HostPort: 9872}}
	localNetInfo      = models.NewActualLRPNetInfo(localAddress, localPorts)
	otherPorts        = []models.PortMapping{{ContainerPort: 2345, HostPort: 9876}}
	otherNetInfo      = models.NewActualLRPNetInfo(otherAddress, otherPorts)
	nullNetInfo       = models.ActualLRPNetInfo{}
)

type testable interface {
	Test()
}

type evacuationTest struct {
	Name        string
	InstanceLRP lrpSetupFunc
	Result      testResult
}

type lrpStatus struct {
	State models.ActualLRPState
	models.ActualLRPNetInfo
	models.ActualLRPContainerKey
	ShouldUpdate bool
}

type testResult struct {
	Instance         *lrpStatus
	AuctionRequested bool
	ReturnedError    error
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
		})

		JustBeforeEach(func() {
			clock.Increment(timeIncrement)
			evacuateErr = bbs.EvacuateClaimedActualLRP(logger, lrpKey, localContainerKey)
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
				_, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
				Ω(err).Should(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		} else {
			if t.Result.Instance.ShouldUpdate {
				It("updates the Since", func() {
					lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrpInBBS.Since).Should(Equal(clock.Now().UnixNano()))
				})
			} else {
				It("does not update the Since", func() {
					lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrpInBBS.Since).Should(Equal(initialTimestamp))
				})
			}

			It("has the expected state", func() {
				lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(lrpInBBS.State).Should(Equal(t.Result.Instance.State))
			})

			It("has the expected container key", func() {
				lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(lrpInBBS.ActualLRPContainerKey).Should(Equal(t.Result.Instance.ActualLRPContainerKey))
			})

			It("has the expected net info", func() {
				lrpInBBS, err := bbs.ActualLRPByProcessGuidAndIndex(processGuid, index)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(lrpInBBS.ActualLRPNetInfo).Should(Equal(t.Result.Instance.ActualLRPNetInfo))
			})
		}
	})
}

func anUnchangedUnclaimedInstance() testResult {
	return anUnchangedInstance(
		models.ActualLRPStateUnclaimed,
		models.ActualLRPContainerKey{},
		models.ActualLRPNetInfo{},
		nil,
	)
}

func anUnchangedInstance(state models.ActualLRPState, containerKey models.ActualLRPContainerKey, netInfo models.ActualLRPNetInfo, err error) testResult {
	return testResult{
		Instance: &lrpStatus{
			State: state,
			ActualLRPContainerKey: containerKey,
			ActualLRPNetInfo:      netInfo,
			ShouldUpdate:          false,
		},
		AuctionRequested: false,
		ReturnedError:    err,
	}
}

func anUpdatedUnclaimedInstance() testResult {
	return testResult{
		Instance: &lrpStatus{
			State:                 models.ActualLRPStateUnclaimed,
			ActualLRPNetInfo:      models.ActualLRPNetInfo{},
			ActualLRPContainerKey: models.ActualLRPContainerKey{},
			ShouldUpdate:          true,
		},
		AuctionRequested: true,
		ReturnedError:    nil,
	}
}
