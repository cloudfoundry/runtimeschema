package lrp_bbs_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("StopLRPInstance", func() {
	var actualLRP models.ActualLRP
	var cellPresence models.CellPresence

	BeforeEach(func() {
		cellPresence = models.NewCellPresence("the-cell-id", "the-stack", "cell.example.com", "az1", models.NewCellCapacity(128, 1024, 6))
		desiredLRP := models.DesiredLRP{
			ProcessGuid: "some-process-guid",
			Domain:      "domain",
			Instances:   1,
		}
		registerCell(cellPresence)

		index := 0
		createAndClaim(
			desiredLRP,
			index,
			models.NewActualLRPContainerKey("some-instance-guid", cellPresence.CellID),
			logger,
		)

		var err error
		actualLRP, err = bbs.ActualLRPByProcessGuidAndIndex(desiredLRP.ProcessGuid, index)
		Ω(err).ShouldNot(HaveOccurred())
	})

	Describe("RequestStopLRPInstance", func() {
		Context("When the request is successful", func() {
			It("makes a stop instance request to the correct cell", func() {
				err := bbs.RequestStopLRPInstance(actualLRP.ActualLRPKey, actualLRP.ActualLRPContainerKey)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(fakeCellClient.StopLRPInstanceCallCount()).Should(Equal(1))

				addr1, key1, cnrKey1 := fakeCellClient.StopLRPInstanceArgsForCall(0)
				Ω(addr1).Should(Equal(cellPresence.RepAddress))
				Ω(key1).Should(Equal(actualLRP.ActualLRPKey))
				Ω(cnrKey1).Should(Equal(actualLRP.ActualLRPContainerKey))
			})
		})

		Context("When the cell returns an error", func() {
			var expectedError = errors.New("cell go boom")
			BeforeEach(func() {
				fakeCellClient.StopLRPInstanceReturns(expectedError)
			})

			It("returns the error", func() {
				err := bbs.RequestStopLRPInstance(actualLRP.ActualLRPKey, actualLRP.ActualLRPContainerKey)
				Ω(err).Should(Equal(expectedError))
			})
		})
	})
})
