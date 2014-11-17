package lrp_bbs_test

import (
	"fmt"

	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LRP", func() {
	const cellID = "some-cell-id"

	var lrp models.DesiredLRP

	BeforeEach(func() {
		lrp = models.DesiredLRP{
			Domain:      "tests",
			ProcessGuid: "some-process-guid",
			Instances:   5,
			Stack:       "some-stack",
			MemoryMB:    1024,
			DiskMB:      512,
			Routes:      []string{"route-1", "route-2"},
			Action: &models.DownloadAction{
				From: "http://example.com",
				To:   "/tmp/internet",
			},
		}
	})

	Describe("Adding and removing DesireLRP", func() {
		Context("when the LRP does not yet exist", func() {
			It("creates /v1/desired/<process-guid>/<index>", func() {
				err := bbs.DesireLRP(lrp)
				Ω(err).ShouldNot(HaveOccurred())

				node, err := etcdClient.Get("/v1/desired/some-process-guid")
				Ω(err).ShouldNot(HaveOccurred())
				expected, err := models.ToJSON(lrp)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(node.Value).Should(Equal(expected))
			})
		})

		Context("when deleting the DesiredLRP", func() {
			BeforeEach(func() {
				err := bbs.DesireLRP(lrp)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("should delete it", func() {
				err := bbs.RemoveDesiredLRPByProcessGuid(lrp.ProcessGuid)
				Ω(err).ShouldNot(HaveOccurred())

				_, err = etcdClient.Get("/v1/desired/some-process-guid")
				Ω(err).Should(MatchError(storeadapter.ErrorKeyNotFound))
			})

			Context("when the desired LRP does not exist", func() {
				It("returns an ErrorKeyNotFound", func() {
					err := bbs.RemoveDesiredLRPByProcessGuid("monkey")
					Ω(err).Should(MatchError(storeadapter.ErrorKeyNotFound))
				})
			})
		})

		Context("when the store is out of commission", func() {
			itRetriesUntilStoreComesBack(func() error {
				return bbs.DesireLRP(lrp)
			})
		})

		Context("with an invalid LRP", func() {
			var desireError error

			BeforeEach(func() {
				lrp.Domain = ""
				desireError = bbs.DesireLRP(lrp)
			})

			It("returns an error", func() {
				Ω(desireError).Should(HaveOccurred())
				Ω(desireError).Should(BeAssignableToTypeOf(*new(models.ValidationError)))
			})
		})
	})

	Describe("Adding and removing actual LRPs", func() {
		var processGuid, instanceGuid, domain string
		var index int

		BeforeEach(func() {
			processGuid = "some-process-guid"
			instanceGuid = "some-instance-guid"
			domain = "some-domain"
			index = 1
		})

		Describe("ReportActualLRPAsStarting", func() {
			It("creates /v1/actual/<process-guid>/<index>/<instance-guid>", func() {
				lrp, err := bbs.ReportActualLRPAsStarting(processGuid, instanceGuid, cellID, domain, index)
				Ω(err).ShouldNot(HaveOccurred())

				node, err := etcdClient.Get(fmt.Sprintf("/v1/actual/%s/%d/%s", processGuid, index, instanceGuid))
				Ω(err).ShouldNot(HaveOccurred())

				expectedLRP := lrp
				expectedLRP.State = models.ActualLRPStateStarting
				expectedLRP.Since = timeProvider.Time().UnixNano()
				expectedLRP.CellID = cellID

				expectedJSON, err := models.ToJSON(expectedLRP)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(node.Value).Should(MatchJSON(expectedJSON))
			})

			Context("when the store is out of commission", func() {
				itRetriesUntilStoreComesBack(func() error {
					_, err := bbs.ReportActualLRPAsStarting(processGuid, instanceGuid, cellID, domain, index)
					return err
				})
			})
		})

		Describe("ReportActualLRPAsRunning", func() {
			var startedLRP models.ActualLRP

			BeforeEach(func() {
				startedLRP = models.ActualLRP{
					InstanceGuid: instanceGuid,
					ProcessGuid:  processGuid,
					CellID:       cellID,
					Domain:       domain,
					Index:        index,
					Since:        timeProvider.Time().UnixNano(),
				}
			})

			It("creates /v1/actual/<process-guid>/<index>/<instance-guid>", func() {
				err := bbs.ReportActualLRPAsRunning(startedLRP, cellID)
				Ω(err).ShouldNot(HaveOccurred())

				node, err := etcdClient.Get(fmt.Sprintf("/v1/actual/%s/%d/%s", processGuid, index, instanceGuid))
				Ω(err).ShouldNot(HaveOccurred())

				expectedLRP := startedLRP
				expectedLRP.State = models.ActualLRPStateRunning
				expectedLRP.Since = timeProvider.Time().UnixNano()
				expectedLRP.CellID = cellID

				expectedJSON, err := models.ToJSON(expectedLRP)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(node.Value).Should(MatchJSON(expectedJSON))
			})

			Context("when the store is out of commission", func() {
				itRetriesUntilStoreComesBack(func() error {
					return bbs.ReportActualLRPAsRunning(startedLRP, cellID)
				})
			})
		})

		Describe("RemoveActualLRP", func() {
			var processGuid, instanceGuid, domain string
			var index int
			var lrp models.ActualLRP

			BeforeEach(func() {
				processGuid = "some-process-guid"
				instanceGuid = "some-instance-guid"
				domain = "some-domain"
				index = 1
				var err error
				lrp, err = bbs.ReportActualLRPAsStarting(processGuid, instanceGuid, cellID, domain, index)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("should remove the LRP", func() {
				err := bbs.RemoveActualLRP(lrp)
				Ω(err).ShouldNot(HaveOccurred())

				_, err = etcdClient.Get(fmt.Sprintf("/v1/actual/%s/%d/%s", processGuid, index, instanceGuid))
				Ω(err).Should(MatchError(storeadapter.ErrorKeyNotFound))
			})

			Context("when the store is out of commission", func() {
				itRetriesUntilStoreComesBack(func() error {
					return bbs.RemoveActualLRP(lrp)
				})
			})
		})

		Describe("RemoveActualLRPForIndex", func() {
			var processGuid, instanceGuid, domain string
			var index int
			var lrp models.ActualLRP

			BeforeEach(func() {
				processGuid = "some-process-guid"
				instanceGuid = "some-instance-guid"
				domain = "some-domain"
				index = 1
				var err error
				lrp, err = bbs.ReportActualLRPAsStarting(processGuid, instanceGuid, cellID, domain, index)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("should remove the LRP", func() {
				err := bbs.RemoveActualLRPForIndex(lrp.ProcessGuid, lrp.Index, lrp.InstanceGuid)
				Ω(err).ShouldNot(HaveOccurred())

				_, err = etcdClient.Get(fmt.Sprintf("/v1/actual/%s/%d/%s", processGuid, index, instanceGuid))
				Ω(err).Should(MatchError(storeadapter.ErrorKeyNotFound))
			})

			Context("when the store is out of commission", func() {
				itRetriesUntilStoreComesBack(func() error {
					return bbs.RemoveActualLRPForIndex(lrp.ProcessGuid, lrp.Index, lrp.InstanceGuid)
				})
			})
		})
	})

	Describe("upsert desired LRPs", func() {
		value := models.DesiredLRP{
			Domain:      "tests",
			ProcessGuid: "some-guid",
			Stack:       "some-stack",
			Instances:   1,
			Action: &models.DownloadAction{
				From: "http://example.com",
				To:   "/tmp/internet",
			},
		}

		BeforeEach(func() {
			err := bbs.DesireLRP(value)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("updates the desired lrp if the modifications are valid", func() {
			newValue := value
			newValue.Instances = 2
			newValue.Routes = []string{"example.com", "foobar"}

			err := bbs.DesireLRP(newValue)
			Ω(err).ShouldNot(HaveOccurred())

			current, err := bbs.DesiredLRPByProcessGuid("some-guid")
			Ω(err).ShouldNot(HaveOccurred())

			Ω(current).Should(Equal(newValue))
		})

		It("fails to update the desired lrp if the modifications are invalid", func() {
			newValue := value
			newValue.Stack = "foo"

			err := bbs.DesireLRP(newValue)
			Ω(err).Should(HaveOccurred())

			current, err := bbs.DesiredLRPByProcessGuid("some-guid")
			Ω(err).ShouldNot(HaveOccurred())

			Ω(current).Should(Equal(value))
		})
	})

	Describe("Changing desired LRPs", func() {
		var changeErr error

		prevValue := models.DesiredLRP{
			Domain:      "tests",
			ProcessGuid: "some-guid",
			Stack:       "some-stack",
			Instances:   1,
			Action: &models.DownloadAction{
				From: "http://example.com",
				To:   "/tmp/internet",
			},
		}

		Context("with a before and after", func() {
			var before models.DesiredLRP
			var after models.DesiredLRP

			JustBeforeEach(func() {
				changeErr = bbs.ChangeDesiredLRP(models.DesiredLRPChange{
					Before: &before,
					After:  &after,
				})
			})

			BeforeEach(func() {
				err := bbs.DesireLRP(prevValue)
				Ω(err).ShouldNot(HaveOccurred())

				before = prevValue
				after = prevValue

				after.MemoryMB = 1024
			})

			Context("when the current value matches", func() {
				It("does not return an error", func() {
					Ω(changeErr).ShouldNot(HaveOccurred())
				})

				It("updates the value in the store", func() {
					current, err := bbs.DesiredLRPByProcessGuid("some-guid")
					Ω(err).ShouldNot(HaveOccurred())

					Ω(current).Should(Equal(after))
				})
			})

			Context("when the current value does not match", func() {
				BeforeEach(func() {
					before.Instances++
				})

				It("returns an error", func() {
					Ω(changeErr).Should(HaveOccurred())
				})

				It("does not update the value in the store", func() {
					current, err := bbs.DesiredLRPByProcessGuid("some-guid")
					Ω(err).ShouldNot(HaveOccurred())

					Ω(current).Should(Equal(prevValue))
				})
			})
		})

		Context("with a before but no after", func() {
			var before models.DesiredLRP

			JustBeforeEach(func() {
				changeErr = bbs.ChangeDesiredLRP(models.DesiredLRPChange{
					Before: &before,
				})
			})

			BeforeEach(func() {
				before = prevValue

				err := bbs.DesireLRP(before)
				Ω(err).ShouldNot(HaveOccurred())
			})

			Context("when the current value matches", func() {
				It("deletes the desired state", func() {
					_, err := bbs.DesiredLRPByProcessGuid("some-guid")
					Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))
				})
			})

			Context("when the current value does not match", func() {
				BeforeEach(func() {
					before.Instances++
				})

				It("returns an error", func() {
					Ω(changeErr).Should(HaveOccurred())
				})

				It("does not remove the value from the store", func() {
					current, err := bbs.DesiredLRPByProcessGuid("some-guid")
					Ω(err).ShouldNot(HaveOccurred())

					Ω(current).Should(Equal(prevValue))
				})
			})
		})

		Context("with no before, but an after", func() {
			var after models.DesiredLRP

			JustBeforeEach(func() {
				changeErr = bbs.ChangeDesiredLRP(models.DesiredLRPChange{
					After: &after,
				})
			})

			BeforeEach(func() {
				after = prevValue
				after.MemoryMB = 1024
			})

			Context("when the current value does not exist", func() {
				It("creates the value at the given key", func() {
					current, err := bbs.DesiredLRPByProcessGuid("some-guid")
					Ω(err).ShouldNot(HaveOccurred())

					Ω(current).Should(Equal(after))
				})
			})

			Context("when a value already exists at the key", func() {
				BeforeEach(func() {
					err := bbs.DesireLRP(prevValue)
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("returns an error", func() {
					Ω(changeErr).Should(HaveOccurred())
				})

				It("does not change the value in the store", func() {
					current, err := bbs.DesiredLRPByProcessGuid("some-guid")
					Ω(err).ShouldNot(HaveOccurred())

					Ω(current).Should(Equal(prevValue))
				})
			})
		})
	})

	Describe("Updating DesireLRP", func() {
		BeforeEach(func() {
			err := bbs.DesireLRP(lrp)
			Ω(err).ShouldNot(HaveOccurred())
		})

		Context("When the updates are valid", func() {
			It("updates an existing DesireLRP", func() {
				routes := []string{"new-route-1", "new-route-2"}
				annotation := "new-annotation"
				instances := 16
				update := models.DesiredLRPUpdate{
					Routes:     routes,
					Annotation: &annotation,
					Instances:  &instances,
				}

				err := bbs.UpdateDesiredLRP(lrp.ProcessGuid, update)
				Ω(err).ShouldNot(HaveOccurred())

				updated, err := bbs.DesiredLRPByProcessGuid(lrp.ProcessGuid)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(updated.Routes).Should(Equal(routes))
				Ω(updated.Annotation).Should(Equal(annotation))
				Ω(updated.Instances).Should(Equal(instances))
			})
		})

		Context("When the updates are invalid", func() {
			It("instances cannot be set to zero", func() {
				instances := 0

				update := models.DesiredLRPUpdate{
					Instances: &instances,
				}

				err := bbs.UpdateDesiredLRP(lrp.ProcessGuid, update)
				Ω(err).Should(HaveOccurred())
				Ω(err.Error()).Should(ContainSubstring("instances"))

				updated, err := bbs.DesiredLRPByProcessGuid(lrp.ProcessGuid)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(updated).Should(Equal(lrp))
			})
		})

		Context("When the LRP does not exist", func() {
			It("returns an ErrorKeyNotFound", func() {
				instances := 0

				err := bbs.UpdateDesiredLRP("garbage-guid", models.DesiredLRPUpdate{
					Instances: &instances,
				})
				Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))
			})
		})
	})
})
