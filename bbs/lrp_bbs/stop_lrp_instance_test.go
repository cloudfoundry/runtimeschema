package lrp_bbs_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("StopLRPInstance", func() {
	var stopInstance models.StopLRPInstance
	var cellPresence models.CellPresence

	BeforeEach(func() {
		stopInstance = models.StopLRPInstance{
			ProcessGuid:  "some-process-guid",
			InstanceGuid: "some-instance-guid",
			Index:        5678,
		}

		cellPresence = models.CellPresence{
			CellID:     "the-cell-id",
			Stack:      "the-stack",
			RepAddress: "cell.example.com",
		}
		registerCell(cellPresence)

		_, err := bbs.ReportActualLRPAsStarting(stopInstance.ProcessGuid, stopInstance.InstanceGuid, cellPresence.CellID, "domain", stopInstance.Index)
		Ω(err).ShouldNot(HaveOccurred())
	})

	Describe("RequestStopLRPInstance", func() {
		Context("When the request is successful", func() {
			It("makes a stop instance request to the correct cell", func() {
				err := bbs.RequestStopLRPInstance(stopInstance)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(fakeCellClient.StopLRPInstanceCallCount()).Should(Equal(1))

				addr1, stop1 := fakeCellClient.StopLRPInstanceArgsForCall(0)
				Ω(addr1).Should(Equal(cellPresence.RepAddress))
				Ω(stop1).Should(Equal(stopInstance))
			})
		})

		Context("When the cell returns an error", func() {
			var expectedError = errors.New("cell go boom")
			BeforeEach(func() {
				fakeCellClient.StopLRPInstanceReturns(expectedError)
			})

			It("returns the error", func() {
				err := bbs.RequestStopLRPInstance(stopInstance)
				Ω(err).Should(Equal(expectedError))
			})
		})

		Context("when the store is out of commission", func() {
			itRetriesUntilStoreComesBack(func() error {
				return bbs.RequestStopLRPInstance(stopInstance)
			})
		})
	})

	Describe("RequestStopLRPInstances", func() {
		var anotherStopInstance models.StopLRPInstance

		BeforeEach(func() {
			anotherStopInstance = models.StopLRPInstance{
				ProcessGuid:  "some-other-process-guid",
				InstanceGuid: "some-other-instance-guid",
				Index:        1234,
			}

			_, err := bbs.ReportActualLRPAsStarting(anotherStopInstance.ProcessGuid, anotherStopInstance.InstanceGuid, cellPresence.CellID, "domain", anotherStopInstance.Index)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("stops the LRP instances on the correct cell", func() {
			err := bbs.RequestStopLRPInstances([]models.StopLRPInstance{stopInstance, anotherStopInstance})
			Ω(err).ShouldNot(HaveOccurred())

			Ω(fakeCellClient.StopLRPInstanceCallCount()).Should(Equal(2))

			addr1, stop1 := fakeCellClient.StopLRPInstanceArgsForCall(0)
			Ω(addr1).Should(Equal(cellPresence.RepAddress))
			Ω(stop1).Should(Equal(stopInstance))

			addr2, stop2 := fakeCellClient.StopLRPInstanceArgsForCall(1)
			Ω(addr2).Should(Equal(cellPresence.RepAddress))
			Ω(stop2).Should(Equal(anotherStopInstance))
		})

		Context("when the store is out of commission", func() {
			itRetriesUntilStoreComesBack(func() error {
				return bbs.RequestStopLRPInstances([]models.StopLRPInstance{stopInstance})
			})
		})
	})
})
