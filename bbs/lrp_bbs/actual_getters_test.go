package lrp_bbs_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Actual LRP Getters", func() {
	const (
		cellID               = "cell-id"
		defaultEvacuationTTL = 0

		baseProcessGuid  = "base-process-guid"
		baseDomain       = "base-domain"
		baseInstanceGuid = "base-instance-guid"

		baseIndex       = 1
		otherIndex      = 2
		yetAnotherIndex = 3

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
		otherCellIDLRP      models.ActualLRP
		otherDomainLRP      models.ActualLRP
		otherProcessGuidLRP models.ActualLRP

		baseLRPKey          models.ActualLRPKey
		baseLRPContainerKey models.ActualLRPContainerKey

		netInfo models.ActualLRPNetInfo

		actualLRPs []models.ActualLRP
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
				createRawActualLRP(otherIndexLRP)
				createRawActualLRP(otherDomainLRP)
				createRawEvacuatingActualLRP(yetAnotherIndexLRP, defaultEvacuationTTL)
			})

			It("returns all the /instance LRPs and no /evacuating LRPs", func() {
				Ω(actualLRPs).Should(HaveLen(3))
				Ω(actualLRPs).Should(ContainElement(baseLRP))
				Ω(actualLRPs).Should(ContainElement(otherIndexLRP))
				Ω(actualLRPs).Should(ContainElement(otherDomainLRP))
				Ω(actualLRPs).ShouldNot(ContainElement(yetAnotherIndexLRP))
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

	Describe("ActualLRPsByProcessGuid", func() {
		var actualLRPsByIndex models.ActualLRPsByIndex

		JustBeforeEach(func() {
			var err error
			actualLRPsByIndex, err = bbs.ActualLRPsByProcessGuid(baseProcessGuid)
			Ω(err).ShouldNot(HaveOccurred())
		})

		Context("when there are only /instance LRPs", func() {
			BeforeEach(func() {
				createRawActualLRP(baseLRP)
				createRawActualLRP(otherIndexLRP)
				createRawEvacuatingActualLRP(yetAnotherIndexLRP, defaultEvacuationTTL)
				createRawActualLRP(otherProcessGuidLRP)
			})

			It("returns only the /instance LRPs for the requested process guid", func() {
				Ω(actualLRPsByIndex).Should(HaveLen(2))
				Ω(actualLRPsByIndex[baseIndex]).Should(Equal(baseLRP))
				Ω(actualLRPsByIndex[otherIndex]).Should(Equal(otherIndexLRP))

				actualLRPs = actualLRPsByIndex.Slice()
				Ω(actualLRPs).ShouldNot(ContainElement(yetAnotherIndexLRP))
				Ω(actualLRPs).ShouldNot(ContainElement(otherProcessGuidLRP))
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
				createRawEvacuatingActualLRP(yetAnotherIndexLRP, defaultEvacuationTTL)
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

	Describe("ActualLRPsByDomain", func() {
		JustBeforeEach(func() {
			var err error
			actualLRPs, err = bbs.ActualLRPsByDomain(baseDomain)
			Ω(err).ShouldNot(HaveOccurred())
		})

		Context("when there are both /instance and /evacuating LRPs in the domain", func() {
			BeforeEach(func() {
				createRawActualLRP(baseLRP)
				createRawActualLRP(yetAnotherIndexLRP)
				createRawActualLRP(otherDomainLRP)
				createRawEvacuatingActualLRP(otherIndexLRP, defaultEvacuationTTL)
			})

			It("should fetch all the instance LRPs for the specified domain", func() {
				Ω(actualLRPs).Should(HaveLen(2))
				Ω(actualLRPs).Should(ConsistOf(baseLRP, yetAnotherIndexLRP))
				Ω(actualLRPs).ShouldNot(ContainElement(otherDomainLRP))
				Ω(actualLRPs).ShouldNot(ContainElement(otherIndexLRP))
			})
		})

		Context("when there are no actual LRPs in the requested domain", func() {
			BeforeEach(func() {
				createRawActualLRP(baseLRP)
				err := bbs.RemoveActualLRP(logger, baseLRPKey, baseLRPContainerKey)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("returns an empty list", func() {
				Ω(actualLRPs).ShouldNot(BeNil())
				Ω(actualLRPs).Should(HaveLen(0))
			})
		})
	})

	Describe("ActualLRPByProcessGuidAndIndex", func() {
		var (
			evacuatingContainerKey models.ActualLRPContainerKey
			evacuatingLRP          models.ActualLRP
			returnedLRP            models.ActualLRP
			returnedErr            error
		)

		BeforeEach(func() {
			evacuatingContainerKey = models.NewActualLRPContainerKey("evacuating-guid", "cell-id")
			evacuatingLRP = models.ActualLRP{
				ActualLRPKey:          baseLRPKey,
				ActualLRPContainerKey: evacuatingContainerKey,
				ActualLRPNetInfo:      netInfo,
				State:                 models.ActualLRPStateRunning,
				Since:                 clock.Now().UnixNano() - 1000,
			}
		})

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
					createRawEvacuatingActualLRP(evacuatingLRP, defaultEvacuationTTL)
				})

				It("returns the /instance entry", func() {
					Ω(returnedErr).ShouldNot(HaveOccurred())
					Ω(returnedLRP).Should(Equal(baseLRP))
				})
			})
		})

		Context("when there is only an /evacuating entry", func() {
			BeforeEach(func() {
				createRawEvacuatingActualLRP(evacuatingLRP, defaultEvacuationTTL)
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

	Describe("EvacuatingActualLRPsByCellID", func() {
		JustBeforeEach(func() {
			var err error
			actualLRPs, err = bbs.EvacuatingActualLRPsByCellID(cellID)
			Ω(err).ShouldNot(HaveOccurred())
		})

		Context("when there are both instance and evacuating LRPs on the requested cell", func() {
			BeforeEach(func() {
				createRawEvacuatingActualLRP(baseLRP, defaultEvacuationTTL)
				createRawEvacuatingActualLRP(yetAnotherIndexLRP, defaultEvacuationTTL)
				createRawActualLRP(otherIndexLRP)
				createRawEvacuatingActualLRP(otherCellIDLRP, defaultEvacuationTTL)
			})

			It("returns only the evacuating LRPs", func() {
				Ω(actualLRPs).Should(HaveLen(2))
				Ω(actualLRPs).Should(ConsistOf(baseLRP, yetAnotherIndexLRP))
			})
		})

		Context("when there are no LRPs", func() {
			BeforeEach(func() {
				// leave some intermediate directories in the store
				createRawEvacuatingActualLRP(baseLRP, defaultEvacuationTTL)
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
