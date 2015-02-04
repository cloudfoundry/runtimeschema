package lrp_bbs_test

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/lrp_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LrpGetters", func() {
	Context("DesiredLRPs", func() {
		var (
			desiredLrp1 models.DesiredLRP
			desiredLrp2 models.DesiredLRP
			desiredLrp3 models.DesiredLRP
		)

		BeforeEach(func() {
			desiredLrp1 = models.DesiredLRP{
				Domain:      "tests",
				ProcessGuid: "guidA",
				Stack:       "stack",
				Instances:   1,
				Action: &models.DownloadAction{
					From: "http://example.com",
					To:   "/tmp/internet",
				},
			}

			desiredLrp2 = models.DesiredLRP{
				Domain:      "tests",
				ProcessGuid: "guidB",
				Stack:       "stack",
				Instances:   1,
				Action: &models.DownloadAction{
					From: "http://example.com",
					To:   "/tmp/internet",
				},
			}

			desiredLrp3 = models.DesiredLRP{
				Domain:      "tests",
				ProcessGuid: "guidC",
				Stack:       "stack",
				Instances:   1,
				Action: &models.DownloadAction{
					From: "http://example.com",
					To:   "/tmp/internet",
				},
			}
		})

		JustBeforeEach(func() {
			err := bbs.DesireLRP(logger, desiredLrp1)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.DesireLRP(logger, desiredLrp2)
			Ω(err).ShouldNot(HaveOccurred())

			err = bbs.DesireLRP(logger, desiredLrp3)
			Ω(err).ShouldNot(HaveOccurred())
		})

		Describe("DesiredLRPs", func() {
			It("returns all desired long running processes", func() {
				all, err := bbs.DesiredLRPs()
				Ω(err).ShouldNot(HaveOccurred())

				Ω(all).Should(HaveLen(3))
				Ω(all).Should(ContainElement(desiredLrp1))
				Ω(all).Should(ContainElement(desiredLrp2))
				Ω(all).Should(ContainElement(desiredLrp3))
			})
		})

		Describe("DesiredLRPsByDomain", func() {
			BeforeEach(func() {
				desiredLrp1.Domain = "domain-1"
				desiredLrp2.Domain = "domain-1"
				desiredLrp3.Domain = "domain-2"
			})

			It("returns all desired long running processes for the given domain", func() {
				byDomain, err := bbs.DesiredLRPsByDomain("domain-1")
				Ω(err).ShouldNot(HaveOccurred())
				Ω(byDomain).Should(ConsistOf([]models.DesiredLRP{desiredLrp1, desiredLrp2}))

				byDomain, err = bbs.DesiredLRPsByDomain("domain-2")
				Ω(err).ShouldNot(HaveOccurred())
				Ω(byDomain).Should(ConsistOf([]models.DesiredLRP{desiredLrp3}))
			})

			It("blows up with an empty string domain", func() {
				_, err := bbs.DesiredLRPsByDomain("")
				Ω(err).Should(Equal(lrp_bbs.ErrNoDomain))
			})
		})

		Describe("DesiredLRPByProcessGuid", func() {
			It("returns all desired long running processes", func() {
				desiredLrp, err := bbs.DesiredLRPByProcessGuid("guidA")
				Ω(err).ShouldNot(HaveOccurred())

				Ω(desiredLrp).Should(Equal(desiredLrp1))
			})

			Context("when the Desired LRP does not exist", func() {
				It("returns ErrStoreResourceNotFound", func() {
					_, err := bbs.DesiredLRPByProcessGuid("non-existent")
					Ω(err).Should(Equal(bbserrors.ErrStoreResourceNotFound))
				})
			})
		})
	})

	Context("ActualLRPs", func() {
		const defaultEvacuationTTL = 0

		var (
			domainALRP           models.ActualLRP
			domainBLRP           models.ActualLRP
			domainCLRP           models.ActualLRP
			evacuatingDomainCLRP models.ActualLRP

			netInfo models.ActualLRPNetInfo

			actualLRPs []models.ActualLRP

			requestedProcessGuid string
			requestedCellID      string
			requestedDomain      string
		)

		BeforeEach(func() {
			netInfo = models.NewActualLRPNetInfo("127.0.0.1", []models.PortMapping{{8080, 80}})

			domainALRP = models.ActualLRP{
				ActualLRPKey:          models.NewActualLRPKey("guid-a", 1, "domain-a"),
				ActualLRPContainerKey: models.NewActualLRPContainerKey("some-instance-guid-1", "cell-id"),
				ActualLRPNetInfo:      netInfo,
				State:                 models.ActualLRPStateRunning,
				Since:                 clock.Now().UnixNano(),
			}

			domainBLRP = models.ActualLRP{
				ActualLRPKey:          models.NewActualLRPKey("guid-b", 2, "domain-b"),
				ActualLRPContainerKey: models.NewActualLRPContainerKey("some-instance-guid-2", "cell-id"),
				ActualLRPNetInfo:      netInfo,
				State:                 models.ActualLRPStateRunning,
				Since:                 clock.Now().UnixNano(),
			}

			domainCLRP = models.ActualLRP{
				ActualLRPKey:          models.NewActualLRPKey("guid-c", 0, "domain-c"),
				ActualLRPContainerKey: models.NewActualLRPContainerKey("some-instance-guid", "cell-id"),
				State: models.ActualLRPStateClaimed,
				Since: clock.Now().UnixNano(),
			}

			evacuatingDomainCLRP = models.ActualLRP{
				ActualLRPKey:          models.NewActualLRPKey("guid-c", 0, "domain-c"),
				ActualLRPContainerKey: models.NewActualLRPContainerKey("evacuating-instance-guid", "cell-id"),
				ActualLRPNetInfo:      netInfo,
				State:                 models.ActualLRPStateRunning,
				Since:                 clock.Now().UnixNano(),
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
					createRawActualLRP(domainALRP)
					createRawActualLRP(domainBLRP)
					createRawActualLRP(domainCLRP)
					createRawEvacuatingActualLRP(evacuatingDomainCLRP, defaultEvacuationTTL)
				})

				It("returns all the /instance LRPs and no /evacuating LRPs", func() {
					Ω(actualLRPs).Should(HaveLen(3))
					Ω(actualLRPs).Should(ContainElement(domainALRP))
					Ω(actualLRPs).Should(ContainElement(domainBLRP))
					Ω(actualLRPs).Should(ContainElement(domainCLRP))
					Ω(actualLRPs).ShouldNot(ContainElement(evacuatingDomainCLRP))
				})
			})

			Context("when there are no LRPs", func() {
				BeforeEach(func() {
					// leave some intermediate directories in the store
					createRawActualLRP(domainALRP)
					err := bbs.RemoveActualLRP(domainALRP.ActualLRPKey, domainALRP.ActualLRPContainerKey, logger)
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
				actualLRPsByIndex, err = bbs.ActualLRPsByProcessGuid(requestedProcessGuid)
				Ω(err).ShouldNot(HaveOccurred())
			})

			Context("when there are only /instance LRPs", func() {
				var (
					sharedProcessGuidLRP1          models.ActualLRP
					sharedProcessGuidLRP2          models.ActualLRP
					sharedProcessGuidEvacuatingLRP models.ActualLRP
				)

				BeforeEach(func() {
					requestedProcessGuid = "actual-lrps-by-process-guid"
					sharedProcessGuidLRP1 = models.ActualLRP{
						ActualLRPKey:          models.NewActualLRPKey(requestedProcessGuid, 0, "domain-c"),
						ActualLRPContainerKey: models.NewActualLRPContainerKey("some-instance-guid", "cell-id"),
						State: models.ActualLRPStateClaimed,
						Since: clock.Now().UnixNano(),
					}

					createRawActualLRP(sharedProcessGuidLRP1)

					sharedProcessGuidLRP2 = models.ActualLRP{
						ActualLRPKey:          models.NewActualLRPKey(requestedProcessGuid, 1, "domain-c"),
						ActualLRPContainerKey: models.NewActualLRPContainerKey("some-instance-guid", "cell-id"),
						State: models.ActualLRPStateClaimed,
						Since: clock.Now().UnixNano(),
					}

					createRawActualLRP(sharedProcessGuidLRP2)

					sharedProcessGuidEvacuatingLRP = models.ActualLRP{
						ActualLRPKey:          models.NewActualLRPKey(requestedProcessGuid, 2, "domain-c"),
						ActualLRPContainerKey: models.NewActualLRPContainerKey("evacuating-instance-guid", "cell-id"),
						ActualLRPNetInfo:      netInfo,
						State:                 models.ActualLRPStateRunning,
						Since:                 clock.Now().UnixNano(),
					}

					createRawEvacuatingActualLRP(sharedProcessGuidEvacuatingLRP, defaultEvacuationTTL)

					createRawActualLRP(domainALRP)
				})

				It("returns only the /instance LRPs for the requested process guid", func() {
					Ω(actualLRPsByIndex).Should(HaveLen(2))
					Ω(actualLRPsByIndex[0]).Should(Equal(sharedProcessGuidLRP1))
					Ω(actualLRPsByIndex[1]).Should(Equal(sharedProcessGuidLRP2))

					actualLRPs = actualLRPsByIndex.Slice()
					Ω(actualLRPs).ShouldNot(ContainElement(sharedProcessGuidEvacuatingLRP))
					Ω(actualLRPs).ShouldNot(ContainElement(domainALRP))
				})
			})

			Context("when there are no LRPs", func() {
				BeforeEach(func() {
					// leave some intermediate directories in the store
					createRawActualLRP(domainALRP)
					err := bbs.RemoveActualLRP(domainALRP.ActualLRPKey, domainALRP.ActualLRPContainerKey, logger)
					Ω(err).ShouldNot(HaveOccurred())
					requestedProcessGuid = domainALRP.ProcessGuid
				})

				It("returns an empty map", func() {
					Ω(actualLRPsByIndex).ShouldNot(BeNil())
					Ω(actualLRPsByIndex).Should(BeEmpty())
				})
			})
		})

		Describe("ActualLRPsByCellID", func() {
			BeforeEach(func() {
				requestedCellID = "cell-id"
			})

			JustBeforeEach(func() {
				var err error
				actualLRPs, err = bbs.ActualLRPsByCellID(requestedCellID)
				Ω(err).ShouldNot(HaveOccurred())
			})

			Context("when there are /instance and /evacuating LRPs", func() {
				var otherLRP models.ActualLRP

				BeforeEach(func() {
					createRawActualLRP(domainALRP)
					createRawActualLRP(domainBLRP)
					createRawActualLRP(domainCLRP)
					createRawEvacuatingActualLRP(evacuatingDomainCLRP, defaultEvacuationTTL)

					otherLRP := models.ActualLRP{
						ActualLRPKey:          models.NewActualLRPKey("some-other-process", 0, "some-other-domain"),
						ActualLRPContainerKey: models.NewActualLRPContainerKey("some-other-instance-guid", "some-other-cell"),
						State: models.ActualLRPStateClaimed,
						Since: clock.Now().UnixNano(),
					}
					createRawActualLRP(otherLRP)
				})

				It("returns the /instance actual lrps belonging to the requested cell id", func() {
					Ω(actualLRPs).Should(ConsistOf(domainALRP, domainCLRP, domainBLRP))
					Ω(actualLRPs).ShouldNot(ContainElement(evacuatingDomainCLRP))
					Ω(actualLRPs).ShouldNot(ContainElement(otherLRP))
				})
			})

			Context("when there are no LRPs", func() {
				BeforeEach(func() {
					// leave some intermediate directories in the store
					createRawActualLRP(domainALRP)
					err := bbs.RemoveActualLRP(domainALRP.ActualLRPKey, domainALRP.ActualLRPContainerKey, logger)
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("returns an empty list", func() {
					Ω(actualLRPs).ShouldNot(BeNil())
					Ω(actualLRPs).Should(BeEmpty())
				})
			})
		})

		Describe("ActualLRPsByDomain", func() {
			BeforeEach(func() {
				requestedDomain = "domain-c"
			})

			JustBeforeEach(func() {
				var err error
				actualLRPs, err = bbs.ActualLRPsByDomain(requestedDomain)
				Ω(err).ShouldNot(HaveOccurred())
			})

			Context("when there are both /instance and /evacuating LRPs in the domain", func() {
				BeforeEach(func() {
					createRawActualLRP(domainBLRP)
					createRawActualLRP(domainCLRP)
					createRawEvacuatingActualLRP(evacuatingDomainCLRP, defaultEvacuationTTL)
				})

				It("should fetch all the instance LRPs for the specified domain", func() {
					Ω(actualLRPs).Should(HaveLen(1))
					Ω(actualLRPs).Should(ConsistOf(domainCLRP))
					Ω(actualLRPs).ShouldNot(ContainElement(domainBLRP))
					Ω(actualLRPs).ShouldNot(ContainElement(evacuatingDomainCLRP))
				})
			})

			Context("when there are no actual LRPs in the requested domain", func() {
				BeforeEach(func() {
					createRawActualLRP(domainCLRP)
					err := bbs.RemoveActualLRP(domainCLRP.ActualLRPKey, domainCLRP.ActualLRPContainerKey, logger)
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
				lrpKey                 models.ActualLRPKey
				instanceContainerKey   models.ActualLRPContainerKey
				evacuatingContainerKey models.ActualLRPContainerKey
				netInfo                models.ActualLRPNetInfo
				evacuatingLRP          models.ActualLRP
				returnedLRP            models.ActualLRP
				returnedErr            error
				instanceLRP            models.ActualLRP
			)

			BeforeEach(func() {
				lrpKey = models.NewActualLRPKey("process-guid", 0, "domain")
				instanceContainerKey = models.NewActualLRPContainerKey("instance-guid", "cell-id")
				evacuatingContainerKey = models.NewActualLRPContainerKey("evacuating-guid", "cell-id")
				netInfo = models.NewActualLRPNetInfo("address", []models.PortMapping{{ContainerPort: 1234, HostPort: 5678}})
				evacuatingLRP = models.ActualLRP{
					ActualLRPKey:          lrpKey,
					ActualLRPContainerKey: evacuatingContainerKey,
					ActualLRPNetInfo:      netInfo,
					State:                 models.ActualLRPStateRunning,
					Since:                 3417,
				}
			})

			JustBeforeEach(func() {
				returnedLRP, returnedErr = bbs.ActualLRPByProcessGuidAndIndex("process-guid", 0)
			})

			Context("when there is a RUNNING /instance entry", func() {
				BeforeEach(func() {
					instanceLRP = models.ActualLRP{
						ActualLRPKey:          lrpKey,
						ActualLRPContainerKey: instanceContainerKey,
						ActualLRPNetInfo:      netInfo,
						State:                 models.ActualLRPStateRunning,
						Since:                 1138,
					}
					createRawActualLRP(instanceLRP)
				})

				It("returns the /instance entry", func() {
					Ω(returnedErr).ShouldNot(HaveOccurred())
					Ω(returnedLRP).Should(Equal(instanceLRP))
				})

				Context("when there is also an /evacuating entry", func() {
					BeforeEach(func() {
						createRawEvacuatingActualLRP(evacuatingLRP, defaultEvacuationTTL)
					})

					It("returns the /instance entry", func() {
						Ω(returnedErr).ShouldNot(HaveOccurred())
						Ω(returnedLRP).Should(Equal(instanceLRP))
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
	})
})
