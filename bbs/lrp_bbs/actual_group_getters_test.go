package lrp_bbs_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
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

		baseLRPKey         models.ActualLRPKey
		baseLRPInstanceKey models.ActualLRPInstanceKey

		netInfo models.ActualLRPNetInfo

		actualLRPs      []models.ActualLRP
		actualLRPGroups []models.ActualLRPGroup
	)

	BeforeEach(func() {
		baseLRPKey = models.NewActualLRPKey(baseProcessGuid, baseIndex, baseDomain)
		baseLRPInstanceKey = models.NewActualLRPInstanceKey(baseInstanceGuid, cellID)
		netInfo = models.NewActualLRPNetInfo("127.0.0.1", []models.PortMapping{{8080, 80}})

		baseLRP = models.ActualLRP{
			ActualLRPKey:         baseLRPKey,
			ActualLRPInstanceKey: baseLRPInstanceKey,
			ActualLRPNetInfo:     netInfo,
			State:                models.ActualLRPStateRunning,
			Since:                clock.Now().UnixNano(),
		}
		evacuatingLRP = models.ActualLRP{
			ActualLRPKey:         baseLRPKey,
			ActualLRPInstanceKey: models.NewActualLRPInstanceKey(evacuatingInstanceGuid, cellID),
			ActualLRPNetInfo:     netInfo,
			State:                models.ActualLRPStateRunning,
			Since:                clock.Now().UnixNano() - 1000,
		}

		otherIndexLRP = models.ActualLRP{
			ActualLRPKey:         models.NewActualLRPKey(baseProcessGuid, otherIndex, baseDomain),
			ActualLRPInstanceKey: baseLRPInstanceKey,
			State:                models.ActualLRPStateClaimed,
			Since:                clock.Now().UnixNano(),
		}

		yetAnotherIndexLRP = models.ActualLRP{
			ActualLRPKey:         models.NewActualLRPKey(baseProcessGuid, yetAnotherIndex, baseDomain),
			ActualLRPInstanceKey: baseLRPInstanceKey,
			ActualLRPNetInfo:     netInfo,
			State:                models.ActualLRPStateRunning,
			Since:                clock.Now().UnixNano(),
		}

		otherCellIDLRP = models.ActualLRP{
			ActualLRPKey:         models.NewActualLRPKey(otherCellIDProcessGuid, baseIndex, baseDomain),
			ActualLRPInstanceKey: models.NewActualLRPInstanceKey(baseInstanceGuid, otherCellID),
			ActualLRPNetInfo:     netInfo,
			State:                models.ActualLRPStateRunning,
			Since:                clock.Now().UnixNano(),
		}

		otherDomainLRP = models.ActualLRP{
			ActualLRPKey:         models.NewActualLRPKey(otherDomainProcessGuid, baseIndex, otherDomain),
			ActualLRPInstanceKey: baseLRPInstanceKey,
			ActualLRPNetInfo:     netInfo,
			State:                models.ActualLRPStateRunning,
			Since:                clock.Now().UnixNano(),
		}

		otherProcessGuidLRP = models.ActualLRP{
			ActualLRPKey:         models.NewActualLRPKey(otherProcessGuid, baseIndex, baseDomain),
			ActualLRPInstanceKey: baseLRPInstanceKey,
			State:                models.ActualLRPStateUnclaimed,
			Since:                clock.Now().UnixNano(),
		}
	})

	Describe("ActualLRPs", func() {
		JustBeforeEach(func() {
			var err error
			actualLRPs, err = bbs.ActualLRPs()
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when there are both /instance and /evacuating LRPs", func() {
			BeforeEach(func() {
				setRawActualLRP(baseLRP)
				setRawEvacuatingActualLRP(evacuatingLRP, noExpirationTTL)
				setRawActualLRP(otherDomainLRP)
				setRawEvacuatingActualLRP(otherIndexLRP, noExpirationTTL)
			})

			It("returns all the /instance LRPs and no /evacuating LRPs", func() {
				Expect(actualLRPs).To(ConsistOf(baseLRP, otherDomainLRP))
			})
		})

		Context("when there are no LRPs", func() {
			BeforeEach(func() {
				// leave some intermediate directories in the store
				setRawActualLRP(baseLRP)
				err := bbs.RemoveActualLRP(logger, baseLRPKey, baseLRPInstanceKey)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns an empty list", func() {
				Expect(actualLRPs).NotTo(BeNil())
				Expect(actualLRPs).To(BeEmpty())
			})
		})
	})

	Describe("ActualLRPGroups", func() {
		JustBeforeEach(func() {
			var err error
			actualLRPGroups, err = bbs.ActualLRPGroups()
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when there are both /instance and /evacuating LRPs", func() {
			BeforeEach(func() {
				setRawActualLRP(baseLRP)
				setRawEvacuatingActualLRP(evacuatingLRP, noExpirationTTL)
				setRawActualLRP(otherDomainLRP)
				setRawEvacuatingActualLRP(otherIndexLRP, noExpirationTTL)
			})

			It("returns all the /instance LRPs and /evacuating LRPs in groups", func() {
				Expect(actualLRPGroups).To(ConsistOf(
					models.ActualLRPGroup{Instance: &baseLRP, Evacuating: &evacuatingLRP},
					models.ActualLRPGroup{Instance: &otherDomainLRP, Evacuating: nil},
					models.ActualLRPGroup{Instance: nil, Evacuating: &otherIndexLRP},
				))

			})
		})

		Context("when there are no LRPs", func() {
			BeforeEach(func() {
				// leave some intermediate directories in the store
				setRawActualLRP(baseLRP)
				err := bbs.RemoveActualLRP(logger, baseLRPKey, baseLRPInstanceKey)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns an empty list", func() {
				Expect(actualLRPGroups).NotTo(BeNil())
				Expect(actualLRPGroups).To(BeEmpty())
			})
		})
	})

	Describe("ActualLRPGroupsByProcessGuid", func() {
		var (
			actualLRPGroupsByIndex models.ActualLRPGroupsByIndex
			err                    error
		)

		Context("when there are both /instance and /evacuating LRPs", func() {
			BeforeEach(func() {
				setRawActualLRP(baseLRP)
				setRawActualLRP(otherIndexLRP)
				setRawActualLRP(yetAnotherIndexLRP)
				setRawEvacuatingActualLRP(yetAnotherIndexLRP, noExpirationTTL)
				setRawActualLRP(otherProcessGuidLRP)
			})

			It("returns all the /instance LRPs and /evacuating LRPs in groups", func() {
				actualLRPGroupsByIndex, err = bbs.ActualLRPGroupsByProcessGuid(baseProcessGuid)
				Expect(err).NotTo(HaveOccurred())
				Expect(actualLRPGroupsByIndex).To(Equal(models.ActualLRPGroupsByIndex{
					baseIndex:       {Instance: &baseLRP, Evacuating: nil},
					otherIndex:      {Instance: &otherIndexLRP, Evacuating: nil},
					yetAnotherIndex: {Instance: &yetAnotherIndexLRP, Evacuating: &yetAnotherIndexLRP},
				}))

			})
		})

		Context("when there are no LRPs", func() {
			BeforeEach(func() {
				// leave some intermediate directories in the store
				setRawActualLRP(baseLRP)
				err := bbs.RemoveActualLRP(logger, baseLRPKey, baseLRPInstanceKey)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns an empty map", func() {
				actualLRPGroupsByIndex, err = bbs.ActualLRPGroupsByProcessGuid(baseProcessGuid)
				Expect(err).NotTo(HaveOccurred())
				Expect(actualLRPGroupsByIndex).NotTo(BeNil())
				Expect(actualLRPGroupsByIndex).To(BeEmpty())
			})
		})

		Context("when given an empty process guid", func() {
			It("returns an error", func() {
				_, err = bbs.ActualLRPGroupsByProcessGuid("")
				Expect(err).To(Equal(bbserrors.ErrNoProcessGuid))
			})
		})
	})

	Describe("ActualLRPGroupsByCellID", func() {
		var err error

		Context("when there are /instance and /evacuating LRPs", func() {
			BeforeEach(func() {
				setRawActualLRP(baseLRP)
				setRawActualLRP(otherIndexLRP)
				setRawActualLRP(otherDomainLRP)
				setRawEvacuatingActualLRP(otherDomainLRP, noExpirationTTL)
				setRawEvacuatingActualLRP(yetAnotherIndexLRP, noExpirationTTL)
				setRawActualLRP(otherCellIDLRP)
			})

			It("returns both /instance and /evacuting actual lrps for the requested cell id", func() {
				actualLRPGroups, err = bbs.ActualLRPGroupsByCellID(cellID)
				Expect(err).NotTo(HaveOccurred())
				Expect(actualLRPGroups).To(ConsistOf(
					models.ActualLRPGroup{Instance: &baseLRP, Evacuating: nil},
					models.ActualLRPGroup{Instance: &otherIndexLRP, Evacuating: nil},
					models.ActualLRPGroup{Instance: &otherDomainLRP, Evacuating: &otherDomainLRP},
					models.ActualLRPGroup{Instance: nil, Evacuating: &yetAnotherIndexLRP},
				))

			})
		})

		Context("when there are no LRPs", func() {
			BeforeEach(func() {
				// leave some intermediate directories in the store
				setRawActualLRP(baseLRP)
				err := bbs.RemoveActualLRP(logger, baseLRPKey, baseLRPInstanceKey)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns an empty list", func() {
				actualLRPGroups, err = bbs.ActualLRPGroupsByCellID(cellID)
				Expect(err).NotTo(HaveOccurred())
				Expect(actualLRPGroups).NotTo(BeNil())
				Expect(actualLRPGroups).To(BeEmpty())
			})
		})

		Context("when given an empty cell id", func() {
			It("returns an error", func() {
				_, err = bbs.ActualLRPGroupsByCellID("")
				Expect(err).To(Equal(bbserrors.ErrNoCellID))
			})
		})
	})

	Describe("ActualLRPGroupsByDomain", func() {
		var err error

		Context("when there are both /instance and /evacuating LRPs in the domain", func() {
			BeforeEach(func() {
				setRawActualLRP(baseLRP)
				setRawEvacuatingActualLRP(evacuatingLRP, noExpirationTTL)
				setRawActualLRP(yetAnotherIndexLRP)
				setRawActualLRP(otherDomainLRP)
				setRawEvacuatingActualLRP(otherIndexLRP, noExpirationTTL)
			})

			It("should fetch all the instance and evacuating LRPs for the specified domain", func() {
				actualLRPGroups, err = bbs.ActualLRPGroupsByDomain(baseDomain)
				Expect(err).NotTo(HaveOccurred())
				Expect(actualLRPGroups).To(ConsistOf(
					models.ActualLRPGroup{Instance: &baseLRP, Evacuating: &evacuatingLRP},
					models.ActualLRPGroup{Instance: &yetAnotherIndexLRP, Evacuating: nil},
					models.ActualLRPGroup{Instance: nil, Evacuating: &otherIndexLRP},
				))

			})
		})

		Context("when there are no actual LRPs in the requested domain", func() {
			BeforeEach(func() {
				setRawActualLRP(baseLRP)
				err := bbs.RemoveActualLRP(logger, baseLRPKey, baseLRPInstanceKey)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns an empty list", func() {
				actualLRPGroups, err = bbs.ActualLRPGroupsByDomain(baseDomain)
				Expect(err).NotTo(HaveOccurred())
				Expect(actualLRPGroups).NotTo(BeNil())
				Expect(actualLRPGroups).To(HaveLen(0))
			})
		})

		Context("when given an empty domain", func() {
			It("returns an error", func() {
				_, err = bbs.ActualLRPGroupsByDomain("")
				Expect(err).To(Equal(bbserrors.ErrNoDomain))
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
				setRawActualLRP(baseLRP)
			})

			It("returns the /instance entry", func() {
				Expect(returnedErr).NotTo(HaveOccurred())
				Expect(returnedLRPGroup).To(Equal(models.ActualLRPGroup{
					Instance:   &baseLRP,
					Evacuating: nil,
				}))

			})

			Context("when there is also an /evacuating entry", func() {
				BeforeEach(func() {
					setRawEvacuatingActualLRP(evacuatingLRP, noExpirationTTL)
				})

				It("returns both the /instance LRPs and /evacuating LRPs the group", func() {
					Expect(returnedErr).NotTo(HaveOccurred())
					Expect(returnedLRPGroup).To(Equal(models.ActualLRPGroup{
						Instance:   &baseLRP,
						Evacuating: &evacuatingLRP,
					}))

				})
			})
		})

		Context("when there is only an /evacuating entry", func() {
			BeforeEach(func() {
				setRawEvacuatingActualLRP(evacuatingLRP, noExpirationTTL)
			})

			It("returns an ErrStoreResourceNotFound", func() {
				Expect(returnedErr).NotTo(HaveOccurred())
				Expect(returnedLRPGroup).To(Equal(models.ActualLRPGroup{
					Instance:   nil,
					Evacuating: &evacuatingLRP,
				}))

			})
		})

		Context("when there are no entries", func() {
			It("returns an ErrStoreResourceNotFound", func() {
				_, returnedErr = bbs.ActualLRPGroupByProcessGuidAndIndex(baseProcessGuid, baseIndex)
				Expect(returnedErr).To(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		})

		Context("when given an empty process guid", func() {
			It("returns an error", func() {
				_, returnedErr = bbs.ActualLRPGroupByProcessGuidAndIndex("", baseIndex)
				Expect(returnedErr).To(Equal(bbserrors.ErrNoProcessGuid))
			})
		})

		Context("when there is an index entry without /instance or /evacuating", func() {
			BeforeEach(func() {
				setRawActualLRP(baseLRP)
				err := etcdClient.Delete(shared.ActualLRPSchemaPath(baseLRP.ProcessGuid, baseLRP.Index))
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns an ErrStoreResourceNotFound", func() {
				Expect(returnedErr).To(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		})
	})

	Describe("EvacuatingActualLRPByProcessGuidAndIndex", func() {
		var (
			returnedLRP models.ActualLRP
			returnedErr error
		)

		Context("when there is both an /instance and an /evacuating entry", func() {
			BeforeEach(func() {
				setRawActualLRP(baseLRP)
				setRawEvacuatingActualLRP(evacuatingLRP, noExpirationTTL)
			})

			It("returns the /evacuating entry", func() {
				returnedLRP, returnedErr = bbs.EvacuatingActualLRPByProcessGuidAndIndex(baseProcessGuid, baseIndex)
				Expect(returnedErr).NotTo(HaveOccurred())
				Expect(returnedLRP).To(Equal(evacuatingLRP))
			})
		})

		Context("when there is only an /instance entry", func() {
			BeforeEach(func() {
				setRawActualLRP(baseLRP)
			})

			It("returns ErrStoreResourceNotFound", func() {
				returnedLRP, returnedErr = bbs.EvacuatingActualLRPByProcessGuidAndIndex(baseProcessGuid, baseIndex)
				Expect(returnedErr).To(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		})

		Context("when there is only an /evacuating entry", func() {
			BeforeEach(func() {
				setRawEvacuatingActualLRP(evacuatingLRP, noExpirationTTL)
			})

			It("returns the /evacuating entry", func() {
				returnedLRP, returnedErr = bbs.EvacuatingActualLRPByProcessGuidAndIndex(baseProcessGuid, baseIndex)
				Expect(returnedErr).NotTo(HaveOccurred())
				Expect(returnedLRP).To(Equal(evacuatingLRP))
			})
		})

		Context("when there are no entries", func() {
			It("returns ErrStoreResourceNotFound", func() {
				_, returnedErr = bbs.EvacuatingActualLRPByProcessGuidAndIndex(baseProcessGuid, baseIndex)
				Expect(returnedErr).To(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		})

		Context("when given an empty process guid", func() {
			It("returns an error", func() {
				_, returnedErr = bbs.EvacuatingActualLRPByProcessGuidAndIndex("", baseIndex)
				Expect(returnedErr).To(Equal(bbserrors.ErrNoProcessGuid))
			})
		})
	})
})
