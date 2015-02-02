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
		evacuateClaimedActualLRP := func() error {
			return bbs.EvacuateClaimedActualLRP(logger, lrpKey, localContainerKey)
		}
		tests := []testable{
			evacuationTest{
				Name:    "when there is no instance or evacuating LRP",
				Subject: evacuateClaimedActualLRP,
				Result:  noInstanceOrEvacuatingLRP(),
			},
			evacuationTest{
				Name:        "when the instance is UNCLAIMED",
				Subject:     evacuateClaimedActualLRP,
				InstanceLRP: lrp(models.ActualLRPStateUnclaimed, nullContainerKey, nullNetInfo),
				Result:      anUnchangedUnclaimedInstanceLRP(),
			},
			evacuationTest{
				Name:        "when the instance is CLAIMED locally",
				Subject:     evacuateClaimedActualLRP,
				InstanceLRP: lrp(models.ActualLRPStateClaimed, localContainerKey, nullNetInfo),
				Result:      anUpdatedUnclaimedInstanceLRP(),
			},
			evacuationTest{
				Name:        "when the instance is CLAIMED elsewhere",
				Subject:     evacuateClaimedActualLRP,
				InstanceLRP: lrp(models.ActualLRPStateClaimed, otherContainerKey, nullNetInfo),
				Result:      anUnchangedInstanceLRP(models.ActualLRPStateClaimed, otherContainerKey, nullNetInfo, bbserrors.ErrActualLRPCannotBeUnclaimed),
			},
			evacuationTest{
				Name:        "when the instance is RUNNING locally",
				Subject:     evacuateClaimedActualLRP,
				InstanceLRP: lrp(models.ActualLRPStateRunning, localContainerKey, localNetInfo),
				Result:      anUpdatedUnclaimedInstanceLRP(),
			},
			evacuationTest{
				Name:        "when the instance is RUNNING elsewhere",
				Subject:     evacuateClaimedActualLRP,
				InstanceLRP: lrp(models.ActualLRPStateRunning, otherContainerKey, otherNetInfo),
				Result:      anUnchangedInstanceLRP(models.ActualLRPStateRunning, otherContainerKey, otherNetInfo, bbserrors.ErrActualLRPCannotBeUnclaimed),
			},
			evacuationTest{
				Name:        "when the instance is CRASHED",
				Subject:     evacuateClaimedActualLRP,
				InstanceLRP: lrp(models.ActualLRPStateCrashed, nullContainerKey, nullNetInfo),
				Result:      anUnchangedInstanceLRP(models.ActualLRPStateCrashed, nullContainerKey, nullNetInfo, bbserrors.ErrActualLRPCannotBeUnclaimed),
			},
			evacuationTest{
				Name:          "when the evacuating LRP is RUNNING locally",
				Subject:       evacuateClaimedActualLRP,
				EvacuatingLRP: lrp(models.ActualLRPStateRunning, localContainerKey, localNetInfo),
				Result:        noInstanceOrEvacuatingLRP(),
			},
			evacuationTest{
				Name:          "when the evacuating LRP is RUNNING elsewhere",
				Subject:       evacuateClaimedActualLRP,
				EvacuatingLRP: lrp(models.ActualLRPStateRunning, otherContainerKey, otherNetInfo),
				Result:        anUnchangedEvacuatingLRP(models.ActualLRPStateRunning, otherContainerKey, otherNetInfo),
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
	Name          string
	Subject       func() error
	InstanceLRP   lrpSetupFunc
	EvacuatingLRP lrpSetupFunc
	Result        testResult
}

type lrpStatus struct {
	State models.ActualLRPState
	models.ActualLRPNetInfo
	models.ActualLRPContainerKey
	ShouldUpdate bool
}

type testResult struct {
	Instance         *lrpStatus
	Evacuating       *lrpStatus
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

func anUnchangedUnclaimedInstanceLRP() testResult {
	return anUnchangedInstanceLRP(
		models.ActualLRPStateUnclaimed,
		models.ActualLRPContainerKey{},
		models.ActualLRPNetInfo{},
		nil,
	)
}

func anUnchangedInstanceLRP(state models.ActualLRPState, containerKey models.ActualLRPContainerKey, netInfo models.ActualLRPNetInfo, err error) testResult {
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

func anUnchangedEvacuatingLRP(state models.ActualLRPState, containerKey models.ActualLRPContainerKey, netInfo models.ActualLRPNetInfo) testResult {
	return testResult{
		Evacuating: &lrpStatus{
			State: state,
			ActualLRPContainerKey: containerKey,
			ActualLRPNetInfo:      netInfo,
			ShouldUpdate:          false,
		},
		AuctionRequested: false,
		ReturnedError:    nil,
	}
}

func anUpdatedUnclaimedInstanceLRP() testResult {
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

func noInstanceOrEvacuatingLRP() testResult {
	return testResult{
		AuctionRequested: false,
		ReturnedError:    nil,
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
			if t.EvacuatingLRP != nil {
				createRawEvacuatingActualLRP(t.EvacuatingLRP())
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
				_, err := getEvacuatingActualLRP(lrpKey)
				Ω(err).Should(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		} else {
			if t.Result.Evacuating.ShouldUpdate {
				It("updates the /evacuating Since", func() {
					lrpInBBS, err := getEvacuatingActualLRP(lrpKey)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrpInBBS.Since).Should(Equal(clock.Now().UnixNano()))
				})
			} else {
				It("does not update the /evacuating Since", func() {
					lrpInBBS, err := getEvacuatingActualLRP(lrpKey)
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrpInBBS.Since).Should(Equal(initialTimestamp))
				})
			}

			It("has the expected /evacuating state", func() {
				lrpInBBS, err := getEvacuatingActualLRP(lrpKey)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(lrpInBBS.State).Should(Equal(t.Result.Evacuating.State))
			})

			It("has the expected /evacuating container key", func() {
				lrpInBBS, err := getEvacuatingActualLRP(lrpKey)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(lrpInBBS.ActualLRPContainerKey).Should(Equal(t.Result.Evacuating.ActualLRPContainerKey))
			})

			It("has the expected /evacuating net info", func() {
				lrpInBBS, err := getEvacuatingActualLRP(lrpKey)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(lrpInBBS.ActualLRPNetInfo).Should(Equal(t.Result.Evacuating.ActualLRPNetInfo))
			})
		}
	})
}
