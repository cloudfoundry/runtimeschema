package lrp_bbs_test

import (
	"errors"
	"sync"

	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("StopLRPInstance", func() {
	var actualLRP *models.ActualLRP
	var cellPresence models.CellPresence

	BeforeEach(func() {
		cellPresence = models.CellPresence{
			CellID:     "the-cell-id",
			Stack:      "the-stack",
			RepAddress: "cell.example.com",
		}
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
				err := bbs.RequestStopLRPInstance(*actualLRP)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(fakeCellClient.StopLRPInstanceCallCount()).Should(Equal(1))

				addr1, stop1 := fakeCellClient.StopLRPInstanceArgsForCall(0)
				Ω(addr1).Should(Equal(cellPresence.RepAddress))
				Ω(stop1).Should(Equal(*actualLRP))
			})
		})

		Context("When the cell returns an error", func() {
			var expectedError = errors.New("cell go boom")
			BeforeEach(func() {
				fakeCellClient.StopLRPInstanceReturns(expectedError)
			})

			It("returns the error", func() {
				err := bbs.RequestStopLRPInstance(*actualLRP)
				Ω(err).Should(Equal(expectedError))
			})
		})

		Context("when the store is out of commission", func() {
			itRetriesUntilStoreComesBack(func() error {
				return bbs.RequestStopLRPInstance(*actualLRP)
			})
		})
	})

	Describe("RequestStopLRPInstances", func() {
		var anotherActualLRP *models.ActualLRP

		BeforeEach(func() {
			anotherDesiredLRP := models.DesiredLRP{
				ProcessGuid: "some-other-process-guid",
				Domain:      "domain",
				Instances:   1,
			}

			index := 0
			createAndClaim(
				anotherDesiredLRP,
				index,
				models.NewActualLRPContainerKey("some-other-instance-guid", cellPresence.CellID),
				logger,
			)

			var err error
			anotherActualLRP, err = bbs.ActualLRPByProcessGuidAndIndex(anotherDesiredLRP.ProcessGuid, index)
			Ω(err).ShouldNot(HaveOccurred())

			wg := new(sync.WaitGroup)
			wg.Add(2)

			fakeCellClient.StopLRPInstanceStub = func(string, models.ActualLRP) error {
				wg.Done()
				wg.Wait()
				return nil
			}
		})

		It("stops the LRP instances on the correct cell, in parallel", func() {
			err := bbs.RequestStopLRPInstances([]models.ActualLRP{*actualLRP, *anotherActualLRP})
			Ω(err).ShouldNot(HaveOccurred())

			Ω(fakeCellClient.StopLRPInstanceCallCount()).Should(Equal(2))

			addr1, stop1 := fakeCellClient.StopLRPInstanceArgsForCall(0)
			Ω(addr1).Should(Equal(cellPresence.RepAddress))

			addr2, stop2 := fakeCellClient.StopLRPInstanceArgsForCall(1)
			Ω(addr2).Should(Equal(cellPresence.RepAddress))

			Ω([]models.ActualLRP{stop1, stop2}).Should(ConsistOf(
				*actualLRP,
				*anotherActualLRP,
			))
		})

		Context("when the store is out of commission", func() {
			itRetriesUntilStoreComesBack(func() error {
				return bbs.RequestStopLRPInstances([]models.ActualLRP{*actualLRP, *anotherActualLRP})
			})
		})
	})
})
