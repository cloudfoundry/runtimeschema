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
		var (
			domainALRP models.ActualLRP
			domainBLRP models.ActualLRP
			domainCLRP models.ActualLRP

			netInfo models.ActualLRPNetInfo
		)

		BeforeEach(func() {
			netInfo = models.NewActualLRPNetInfo("127.0.0.1", []models.PortMapping{{8080, 80}})

			domainALRP = models.ActualLRP{
				ActualLRPKey:          models.NewActualLRPKey("guidA", 1, "domain-a"),
				ActualLRPContainerKey: models.NewActualLRPContainerKey("some-instance-guid-1", "cell-id"),
				ActualLRPNetInfo:      netInfo,
				State:                 models.ActualLRPStateRunning,
				Since:                 clock.Now().UnixNano(),
			}

			err := bbs.StartActualLRP(domainALRP.ActualLRPKey, domainALRP.ActualLRPContainerKey, domainALRP.ActualLRPNetInfo, logger)
			Ω(err).ShouldNot(HaveOccurred())

			domainBLRP = models.ActualLRP{
				ActualLRPKey:          models.NewActualLRPKey("guidB", 2, "domain-b"),
				ActualLRPContainerKey: models.NewActualLRPContainerKey("some-instance-guid-2", "cell-id"),
				ActualLRPNetInfo:      netInfo,
				State:                 models.ActualLRPStateRunning,
				Since:                 clock.Now().UnixNano(),
			}

			err = bbs.StartActualLRP(domainBLRP.ActualLRPKey, domainBLRP.ActualLRPContainerKey, domainBLRP.ActualLRPNetInfo, logger)
			Ω(err).ShouldNot(HaveOccurred())

			createAndClaim(
				models.DesiredLRP{ProcessGuid: "guidD", Domain: "domain-c", Instances: 1},
				0,
				models.NewActualLRPContainerKey("some-instance-guid", "cell-id"),
				logger,
			)

			domainCLRP = getInstanceActualLRP(models.NewActualLRPKey("guidD", 0, "domain-c"))
		})

		Describe("ActualLRPs", func() {
			It("returns all actual long running processes", func() {
				all, err := bbs.ActualLRPs()
				Ω(err).ShouldNot(HaveOccurred())

				Ω(all).Should(HaveLen(3))
				Ω(all).Should(ContainElement(domainALRP))
				Ω(all).Should(ContainElement(domainBLRP))
				Ω(all).Should(ContainElement(domainCLRP))
			})
		})

		Describe("ActualLRPsByCellID", func() {
			BeforeEach(func() {
				createAndClaim(
					models.DesiredLRP{ProcessGuid: "some-other-process", Domain: "some-other-domain", Instances: 1},
					0,
					models.NewActualLRPContainerKey("some-other-instance", "some-other-cell"),
					logger,
				)
			})

			It("returns actual long running processes belongs to 'cell-id'", func() {
				actualLrpsForMainCell, err := bbs.ActualLRPsByCellID("cell-id")
				Ω(err).ShouldNot(HaveOccurred())

				Ω(actualLrpsForMainCell).Should(ConsistOf(domainALRP, domainCLRP, domainBLRP))

				actualLrpsForOtherCell, err := bbs.ActualLRPsByCellID("some-other-cell")
				Ω(err).ShouldNot(HaveOccurred())

				Ω(actualLrpsForOtherCell).Should(HaveLen(1))
			})
		})

		Describe("ActualLRPsByProcessGuid", func() {
			It("should fetch all LRPs for the specified guid", func() {
				lrps, err := bbs.ActualLRPsByProcessGuid("guidD")
				Ω(err).ShouldNot(HaveOccurred())
				Ω(lrps).Should(HaveLen(1))
				Ω(lrps).Should(ContainElement(domainCLRP))
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

			Context("when there is an UNCLAIMED /instance entry", func() {
				BeforeEach(func() {
					instanceLRP = models.ActualLRP{
						ActualLRPKey: lrpKey,
						State:        models.ActualLRPStateUnclaimed,
						Since:        1138,
					}
					createRawActualLRP(instanceLRP)
				})

				It("returns the /instance entry", func() {
					Ω(returnedErr).ShouldNot(HaveOccurred())
					Ω(returnedLRP).Should(Equal(instanceLRP))
				})

				Context("when there is also an /evacuating entry", func() {
					BeforeEach(func() {
						createRawEvacuatingActualLRP(evacuatingLRP)
					})

					It("returns the /evacuating entry", func() {
						Ω(returnedErr).ShouldNot(HaveOccurred())
						Ω(returnedLRP).Should(Equal(evacuatingLRP))
					})
				})
			})

			Context("when there is a CLAIMED /instance entry", func() {
				BeforeEach(func() {
					instanceLRP = models.ActualLRP{
						ActualLRPKey:          lrpKey,
						ActualLRPContainerKey: instanceContainerKey,
						State: models.ActualLRPStateClaimed,
						Since: 1138,
					}
					createRawActualLRP(instanceLRP)
				})

				It("returns the /instance entry", func() {
					Ω(returnedErr).ShouldNot(HaveOccurred())
					Ω(returnedLRP).Should(Equal(instanceLRP))
				})

				Context("when there is also an /evacuating entry", func() {
					BeforeEach(func() {
						createRawEvacuatingActualLRP(evacuatingLRP)
					})

					It("returns the /evacuating entry", func() {
						Ω(returnedErr).ShouldNot(HaveOccurred())
						Ω(returnedLRP).Should(Equal(evacuatingLRP))
					})
				})
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
						createRawEvacuatingActualLRP(evacuatingLRP)
					})

					It("returns the /instance entry", func() {
						Ω(returnedErr).ShouldNot(HaveOccurred())
						Ω(returnedLRP).Should(Equal(instanceLRP))
					})
				})
			})

			Context("when there is a CRASHED /instance entry", func() {
				BeforeEach(func() {
					instanceLRP = models.ActualLRP{
						ActualLRPKey: lrpKey,
						State:        models.ActualLRPStateCrashed,
						Since:        1138,
					}
					createRawActualLRP(instanceLRP)
				})

				It("returns the /instance entry", func() {
					Ω(returnedErr).ShouldNot(HaveOccurred())
					Ω(returnedLRP).Should(Equal(instanceLRP))
				})

				Context("when there is also an /evacuating entry", func() {
					BeforeEach(func() {
						createRawEvacuatingActualLRP(evacuatingLRP)
					})

					It("returns the /instance entry", func() {
						Ω(returnedErr).ShouldNot(HaveOccurred())
						Ω(returnedLRP).Should(Equal(instanceLRP))
					})
				})
			})

			Context("when there is only an /evacuating entry", func() {
				BeforeEach(func() {
					createRawEvacuatingActualLRP(evacuatingLRP)
				})

				It("returns the /evacuating entry", func() {
					Ω(returnedErr).ShouldNot(HaveOccurred())
					Ω(returnedLRP).Should(Equal(evacuatingLRP))
				})
			})

			Context("when there are no entries", func() {
				It("returns an ErrStoreResourceNotFound", func() {
					Ω(returnedErr).Should(Equal(bbserrors.ErrStoreResourceNotFound))
				})
			})
		})

		Describe("ActualLRPsByDomain", func() {
			var runningLrp3 models.ActualLRP

			BeforeEach(func() {
				runningLrp3 = models.ActualLRP{
					ActualLRPKey:          models.NewActualLRPKey("guidC", 3, "domain-b"),
					ActualLRPContainerKey: models.NewActualLRPContainerKey("some-instance-guid-3", "cell-id"),
					ActualLRPNetInfo:      netInfo,
					State:                 models.ActualLRPStateRunning,
					Since:                 clock.Now().UnixNano(),
				}
				err := bbs.StartActualLRP(runningLrp3.ActualLRPKey, runningLrp3.ActualLRPContainerKey, runningLrp3.ActualLRPNetInfo, logger)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("should fetch all LRPs for the specified guid", func() {
				lrps, err := bbs.ActualLRPsByDomain("domain-b")
				Ω(err).ShouldNot(HaveOccurred())

				Ω(lrps).Should(HaveLen(2))
				Ω(lrps).ShouldNot(ContainElement(domainALRP))
				Ω(lrps).Should(ContainElement(domainBLRP))
				Ω(lrps).Should(ContainElement(runningLrp3))
			})

			Context("when there are no actual LRPs in the requested domain", func() {
				It("returns an empty list", func() {
					lrps, err := bbs.ActualLRPsByDomain("bogus-domain")
					Ω(err).ShouldNot(HaveOccurred())

					Ω(lrps).ShouldNot(BeNil())
					Ω(lrps).Should(HaveLen(0))
				})
			})
		})
	})
})
