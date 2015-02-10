package lrp_bbs_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Actual LRP Getters", func() {
	const (
		cellID          = "cell-id"
		noExpirationTTL = 0

		baseProcessGuid  = "base-process-guid"
		baseDomain       = "base-domain"
		baseInstanceGuid = "base-instance-guid"

		baseIndex       = 1
		otherIndex      = 2
		yetAnotherIndex = 3

		evacuatingInstanceGuid = "evacuating-instance-guid"

		otherCellID            = "other-cell-id"
		otherCellIDProcessGuid = "other-cell-id-process-guid"

		otherDomainProcessGuid = "other-domain-process-guid"
		otherDomain            = "other-domain"

		otherProcessGuid = "other-process-guid"
	)

	var (
		baseLRP             models.ActualLRP
		otherIndexLRP       models.ActualLRP
		yetAnotherIndexLRP  models.ActualLRP
		evacuatingLRP       models.ActualLRP
		otherCellIDLRP      models.ActualLRP
		otherDomainLRP      models.ActualLRP
		otherProcessGuidLRP models.ActualLRP

		baseLRPKey          models.ActualLRPKey
		baseLRPContainerKey models.ActualLRPContainerKey

		netInfo models.ActualLRPNetInfo

		actualLRPs      []models.ActualLRP
		actualLRPGroups []models.ActualLRPGroup
	)

	BeforeEach(func() {
		baseLRPKey = models.NewActualLRPKey(baseProcessGuid, baseIndex, baseDomain)
		baseLRPContainerKey = models.NewActualLRPContainerKey(baseInstanceGuid, cellID)
		netInfo = models.NewActualLRPNetInfo("127.0.0.1", []models.PortMapping{{8080, 80}})

		baseLRP = models.ActualLRP{
			ActualLRPKey:          baseLRPKey,
			ActualLRPContainerKey: baseLRPContainerKey,
			ActualLRPNetInfo:      netInfo,
			State:                 models.ActualLRPStateRunning,
			Since:                 clock.Now().UnixNano(),
		}
		evacuatingLRP = models.ActualLRP{
			ActualLRPKey:          baseLRPKey,
			ActualLRPContainerKey: models.NewActualLRPContainerKey(evacuatingInstanceGuid, cellID),
			ActualLRPNetInfo:      netInfo,
			State:                 models.ActualLRPStateRunning,
			Since:                 clock.Now().UnixNano() - 1000,
		}

		otherIndexLRP = models.ActualLRP{
			ActualLRPKey:          models.NewActualLRPKey(baseProcessGuid, otherIndex, baseDomain),
			ActualLRPContainerKey: baseLRPContainerKey,
			State: models.ActualLRPStateClaimed,
			Since: clock.Now().UnixNano(),
		}

		yetAnotherIndexLRP = models.ActualLRP{
			ActualLRPKey:          models.NewActualLRPKey(baseProcessGuid, yetAnotherIndex, baseDomain),
			ActualLRPContainerKey: baseLRPContainerKey,
			ActualLRPNetInfo:      netInfo,
			State:                 models.ActualLRPStateRunning,
			Since:                 clock.Now().UnixNano(),
		}

		otherCellIDLRP = models.ActualLRP{
			ActualLRPKey:          models.NewActualLRPKey(otherCellIDProcessGuid, baseIndex, baseDomain),
			ActualLRPContainerKey: models.NewActualLRPContainerKey(baseInstanceGuid, otherCellID),
			ActualLRPNetInfo:      netInfo,
			State:                 models.ActualLRPStateRunning,
			Since:                 clock.Now().UnixNano(),
		}

		otherDomainLRP = models.ActualLRP{
			ActualLRPKey:          models.NewActualLRPKey(otherDomainProcessGuid, baseIndex, otherDomain),
			ActualLRPContainerKey: baseLRPContainerKey,
			ActualLRPNetInfo:      netInfo,
			State:                 models.ActualLRPStateRunning,
			Since:                 clock.Now().UnixNano(),
		}

		otherProcessGuidLRP = models.ActualLRP{
			ActualLRPKey:          models.NewActualLRPKey(otherProcessGuid, baseIndex, baseDomain),
			ActualLRPContainerKey: baseLRPContainerKey,
			State: models.ActualLRPStateUnclaimed,
			Since: clock.Now().UnixNano(),
		}
	})

	Describe("ActualLRPs", func() {
		JustBeforeEach(func() {
			var err error
			actualLRPs, err = bbs.ActualLRPs()
			Ω(err).ShouldNot(HaveOccurred())
		})

		Context("when there are both /instance and /evacuating LRPs", func() {
			BeforeEach(func() {
				createRawActualLRP(baseLRP)
				createRawEvacuatingActualLRP(evacuatingLRP, noExpirationTTL)
				createRawActualLRP(otherDomainLRP)
				createRawEvacuatingActualLRP(otherIndexLRP, noExpirationTTL)
			})

			It("returns all the /instance LRPs and no /evacuating LRPs", func() {
				Ω(actualLRPs).Should(ConsistOf(baseLRP, otherDomainLRP))
			})
		})

		Context("when there are no LRPs", func() {
			BeforeEach(func() {
				// leave some intermediate directories in the store
				createRawActualLRP(baseLRP)
				err := bbs.RemoveActualLRP(logger, baseLRPKey, baseLRPContainerKey)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("returns an empty list", func() {
				Ω(actualLRPs).ShouldNot(BeNil())
				Ω(actualLRPs).Should(BeEmpty())
			})
		})
	})

	Describe("ActualLRPGroups", func() {
		JustBeforeEach(func() {
			var err error
			actualLRPGroups, err = bbs.ActualLRPGroups()
			Ω(err).ShouldNot(HaveOccurred())
		})

		Context("when there are both /instance and /evacuating LRPs", func() {
			BeforeEach(func() {
				createRawActualLRP(baseLRP)
				createRawEvacuatingActualLRP(evacuatingLRP, noExpirationTTL)
				createRawActualLRP(otherDomainLRP)
				createRawEvacuatingActualLRP(otherIndexLRP, noExpirationTTL)
			})

			It("returns all the /instance LRPs and /evacuating LRPs in groups", func() {
				Ω(actualLRPGroups).Should(ConsistOf(
					models.ActualLRPGroup{Instance: &baseLRP, Evacuating: &evacuatingLRP},
					models.ActualLRPGroup{Instance: &otherDomainLRP, Evacuating: nil},
					models.ActualLRPGroup{Instance: nil, Evacuating: &otherIndexLRP},
				))
			})
		})

		Context("when there are no LRPs", func() {
			BeforeEach(func() {
				// leave some intermediate directories in the store
				createRawActualLRP(baseLRP)
				err := bbs.RemoveActualLRP(logger, baseLRPKey, baseLRPContainerKey)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("returns an empty list", func() {
				Ω(actualLRPGroups).ShouldNot(BeNil())
				Ω(actualLRPGroups).Should(BeEmpty())
			})
		})
	})

	Describe("ActualLRPsByProcessGuid", func() {
		var actualLRPsByIndex models.ActualLRPsByIndex

		JustBeforeEach(func() {
			var err error
			actualLRPsByIndex, err = bbs.ActualLRPsByProcessGuid(baseProcessGuid)
			Ω(err).ShouldNot(HaveOccurred())
		})

		Context("when there are both /instance and /evacuating LRPs", func() {
			BeforeEach(func() {
				createRawActualLRP(baseLRP)
				createRawActualLRP(otherIndexLRP)
				createRawEvacuatingActualLRP(yetAnotherIndexLRP, noExpirationTTL)
				createRawActualLRP(otherProcessGuidLRP)
			})

			It("returns only the /instance LRPs for the requested process guid", func() {
				Ω(actualLRPsByIndex).Should(Equal(models.ActualLRPsByIndex{
					baseIndex:  baseLRP,
					otherIndex: otherIndexLRP,
				}))
			})
		})

		Context("when there are no LRPs", func() {
			BeforeEach(func() {
				// leave some intermediate directories in the store
				createRawActualLRP(baseLRP)
				err := bbs.RemoveActualLRP(logger, baseLRPKey, baseLRPContainerKey)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("returns an empty map", func() {
				Ω(actualLRPsByIndex).ShouldNot(BeNil())
				Ω(actualLRPsByIndex).Should(BeEmpty())
			})
		})
	})

	Describe("ActualLRPGroupsByProcessGuid", func() {
		var actualLRPGroupsByIndex models.ActualLRPGroupsByIndex

		JustBeforeEach(func() {
			var err error
			actualLRPGroupsByIndex, err = bbs.ActualLRPGroupsByProcessGuid(baseProcessGuid)
			Ω(err).ShouldNot(HaveOccurred())
		})

		Context("when there are both /instance and /evacuating LRPs", func() {
			BeforeEach(func() {
				createRawActualLRP(baseLRP)
				createRawActualLRP(otherIndexLRP)
				createRawActualLRP(yetAnotherIndexLRP)
				createRawEvacuatingActualLRP(yetAnotherIndexLRP, noExpirationTTL)
				createRawActualLRP(otherProcessGuidLRP)
			})

			It("returns all the /instance LRPs and /evacuating LRPs in groups", func() {
				Ω(actualLRPGroupsByIndex).Should(Equal(models.ActualLRPGroupsByIndex{
					baseIndex:       {Instance: &baseLRP, Evacuating: nil},
					otherIndex:      {Instance: &otherIndexLRP, Evacuating: nil},
					yetAnotherIndex: {Instance: &yetAnotherIndexLRP, Evacuating: &yetAnotherIndexLRP},
				}))
			})
		})

		Context("when there are no LRPs", func() {
			BeforeEach(func() {
				// leave some intermediate directories in the store
				createRawActualLRP(baseLRP)
				err := bbs.RemoveActualLRP(logger, baseLRPKey, baseLRPContainerKey)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("returns an empty map", func() {
				Ω(actualLRPGroupsByIndex).ShouldNot(BeNil())
				Ω(actualLRPGroupsByIndex).Should(BeEmpty())
			})
		})
	})

	Describe("ActualLRPsByCellID", func() {
		JustBeforeEach(func() {
			var err error
			actualLRPs, err = bbs.ActualLRPsByCellID(cellID)
			Ω(err).ShouldNot(HaveOccurred())
		})

		Context("when there are /instance and /evacuating LRPs", func() {
			BeforeEach(func() {
				createRawActualLRP(baseLRP)
				createRawActualLRP(otherIndexLRP)
				createRawActualLRP(otherDomainLRP)
				createRawEvacuatingActualLRP(yetAnotherIndexLRP, noExpirationTTL)
				createRawActualLRP(otherCellIDLRP)
			})

			It("returns the /instance actual lrps belonging to the requested cell id", func() {
				Ω(actualLRPs).Should(ConsistOf(baseLRP, otherIndexLRP, otherDomainLRP))
				Ω(actualLRPs).ShouldNot(ContainElement(yetAnotherIndexLRP))
				Ω(actualLRPs).ShouldNot(ContainElement(otherCellIDLRP))
			})
		})

		Context("when there are no LRPs", func() {
			BeforeEach(func() {
				// leave some intermediate directories in the store
				createRawActualLRP(baseLRP)
				err := bbs.RemoveActualLRP(logger, baseLRPKey, baseLRPContainerKey)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("returns an empty list", func() {
				Ω(actualLRPs).ShouldNot(BeNil())
				Ω(actualLRPs).Should(BeEmpty())
			})
		})
	})

	Describe("ActualLRPGroupsByDomain", func() {
		JustBeforeEach(func() {
			var err error
			actualLRPGroups, err = bbs.ActualLRPGroupsByDomain(baseDomain)
			Ω(err).ShouldNot(HaveOccurred())
		})

		Context("when there are both /instance and /evacuating LRPs in the domain", func() {
			BeforeEach(func() {
				createRawActualLRP(baseLRP)
				createRawEvacuatingActualLRP(evacuatingLRP, noExpirationTTL)
				createRawActualLRP(yetAnotherIndexLRP)
				createRawActualLRP(otherDomainLRP)
				createRawEvacuatingActualLRP(otherIndexLRP, noExpirationTTL)
			})

			It("should fetch all the instance and evacuating LRPs for the specified domain", func() {
				Ω(actualLRPGroups).Should(ConsistOf(
					models.ActualLRPGroup{Instance: &baseLRP, Evacuating: &evacuatingLRP},
					models.ActualLRPGroup{Instance: &yetAnotherIndexLRP, Evacuating: nil},
					models.ActualLRPGroup{Instance: nil, Evacuating: &otherIndexLRP},
				))
			})
		})

		Context("when there are no actual LRPs in the requested domain", func() {
			BeforeEach(func() {
				createRawActualLRP(baseLRP)
				err := bbs.RemoveActualLRP(logger, baseLRPKey, baseLRPContainerKey)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("returns an empty list", func() {
				Ω(actualLRPGroups).ShouldNot(BeNil())
				Ω(actualLRPGroups).Should(HaveLen(0))
			})
		})
	})

	Describe("ActualLRPByProcessGuidAndIndex", func() {
		var (
			returnedLRP models.ActualLRP
			returnedErr error
		)

		JustBeforeEach(func() {
			returnedLRP, returnedErr = bbs.ActualLRPByProcessGuidAndIndex(baseProcessGuid, baseIndex)
		})

		Context("when there is an /instance entry", func() {
			BeforeEach(func() {
				createRawActualLRP(baseLRP)
			})

			It("returns the /instance entry", func() {
				Ω(returnedErr).ShouldNot(HaveOccurred())
				Ω(returnedLRP).Should(Equal(baseLRP))
			})

			Context("when there is also an /evacuating entry", func() {
				BeforeEach(func() {
					createRawEvacuatingActualLRP(evacuatingLRP, noExpirationTTL)
				})

				It("returns the /instance entry", func() {
					Ω(returnedErr).ShouldNot(HaveOccurred())
					Ω(returnedLRP).Should(Equal(baseLRP))
				})
			})
		})

		Context("when there is only an /evacuating entry", func() {
			BeforeEach(func() {
				createRawEvacuatingActualLRP(evacuatingLRP, noExpirationTTL)
			})

			It("returns an ErrStoreResourceNotFound", func() {
				Ω(returnedErr).Should(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		})

		Context("when there are no entries", func() {
			It("returns an ErrStoreResourceNotFound", func() {
				Ω(returnedErr).Should(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		})
	})

	Describe("ActualLRPGroupByProcessGuidAndIndex", func() {
		var (
			returnedLRPGroup models.ActualLRPGroup
			returnedErr      error
		)

		JustBeforeEach(func() {
			returnedLRPGroup, returnedErr = bbs.ActualLRPGroupByProcessGuidAndIndex(baseProcessGuid, baseIndex)
		})

		Context("when there is an /instance entry", func() {
			BeforeEach(func() {
				createRawActualLRP(baseLRP)
			})

			It("returns the /instance entry", func() {
				Ω(returnedErr).ShouldNot(HaveOccurred())
				Ω(returnedLRPGroup).Should(Equal(models.ActualLRPGroup{
					Instance:   &baseLRP,
					Evacuating: nil,
				}))
			})

			Context("when there is also an /evacuating entry", func() {
				BeforeEach(func() {
					createRawEvacuatingActualLRP(evacuatingLRP, noExpirationTTL)
				})

				It("returns both the /instance LRPs and /evacuating LRPs the group", func() {
					Ω(returnedErr).ShouldNot(HaveOccurred())
					Ω(returnedLRPGroup).Should(Equal(models.ActualLRPGroup{
						Instance:   &baseLRP,
						Evacuating: &evacuatingLRP,
					}))
				})
			})
		})

		Context("when there is only an /evacuating entry", func() {
			BeforeEach(func() {
				createRawEvacuatingActualLRP(evacuatingLRP, noExpirationTTL)
			})

			It("returns an ErrStoreResourceNotFound", func() {
				Ω(returnedErr).ShouldNot(HaveOccurred())
				Ω(returnedLRPGroup).Should(Equal(models.ActualLRPGroup{
					Instance:   nil,
					Evacuating: &evacuatingLRP,
				}))
			})
		})

		Context("when there are no entries", func() {
			It("returns an ErrStoreResourceNotFound", func() {
				Ω(returnedErr).Should(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		})
	})

	Describe("EvacuatingActualLRPByProcessGuidAndIndex", func() {
		var (
			returnedLRP models.ActualLRP
			returnedErr error
		)

		JustBeforeEach(func() {
			returnedLRP, returnedErr = bbs.EvacuatingActualLRPByProcessGuidAndIndex(baseProcessGuid, baseIndex)
		})

		Context("when there is both an /instance and an /evacuating entry", func() {
			BeforeEach(func() {
				createRawActualLRP(baseLRP)
				createRawEvacuatingActualLRP(evacuatingLRP, noExpirationTTL)
			})

			It("returns the /evacuating entry", func() {
				Ω(returnedErr).ShouldNot(HaveOccurred())
				Ω(returnedLRP).Should(Equal(evacuatingLRP))
			})
		})

		Context("when there is only an /instance entry", func() {
			BeforeEach(func() {
				createRawActualLRP(baseLRP)
			})

			It("returns ErrStoreResourceNotFound", func() {
				Ω(returnedErr).Should(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		})

		Context("when there is only an /evacuating entry", func() {
			BeforeEach(func() {
				createRawEvacuatingActualLRP(evacuatingLRP, noExpirationTTL)
			})

			It("returns the /evacuating entry", func() {
				Ω(returnedErr).ShouldNot(HaveOccurred())
				Ω(returnedLRP).Should(Equal(evacuatingLRP))
			})
		})

		Context("when there are no entries", func() {
			It("returns ErrStoreResourceNotFound", func() {
				Ω(returnedErr).Should(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		})
	})

	Describe("EvacuatingActualLRPsByCellID", func() {
		JustBeforeEach(func() {
			var err error
			actualLRPs, err = bbs.EvacuatingActualLRPsByCellID(cellID)
			Ω(err).ShouldNot(HaveOccurred())
		})

		Context("when there are both instance and evacuating LRPs on the requested cell", func() {
			BeforeEach(func() {
				createRawEvacuatingActualLRP(baseLRP, noExpirationTTL)
				createRawEvacuatingActualLRP(yetAnotherIndexLRP, noExpirationTTL)
				createRawActualLRP(otherIndexLRP)
				createRawEvacuatingActualLRP(otherCellIDLRP, noExpirationTTL)
			})

			It("returns only the evacuating LRPs", func() {
				Ω(actualLRPs).Should(HaveLen(2))
				Ω(actualLRPs).Should(ConsistOf(baseLRP, yetAnotherIndexLRP))
			})
		})

		Context("when there are no LRPs", func() {
			BeforeEach(func() {
				// leave some intermediate directories in the store
				createRawEvacuatingActualLRP(baseLRP, noExpirationTTL)
				err := bbs.RemoveEvacuatingActualLRP(logger, baseLRPKey, baseLRPContainerKey)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("returns an empty list", func() {
				Ω(actualLRPs).ShouldNot(BeNil())
				Ω(actualLRPs).Should(BeEmpty())
			})
		})
	})
})
