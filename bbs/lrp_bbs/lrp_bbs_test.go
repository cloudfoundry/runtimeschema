package lrp_bbs_test

import (
	. "github.com/cloudfoundry-incubator/runtime-schema/bbs/lrp_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LRP", func() {
	var bbs *LongRunningProcessBBS

	BeforeEach(func() {
		bbs = New(etcdClient)
	})

	Describe("DesireLongRunningProcess", func() {
		var lrp models.DesiredLRP

		BeforeEach(func() {
			lrp = models.DesiredLRP{
				ProcessGuid: "some-process-guid",
				Instances:   5,
				Stack:       "some-stack",
				MemoryMB:    1024,
				DiskMB:      512,
				Routes:      []string{"route-1", "route-2"},
			}
		})

		It("creates /v1/desired/<process-guid>/<index>", func() {
			err := bbs.DesireLongRunningProcess(lrp)
			Ω(err).ShouldNot(HaveOccurred())

			node, err := etcdClient.Get("/v1/desired/some-process-guid")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(node.Value).Should(Equal(lrp.ToJSON()))
		})

		Context("when the store is out of commission", func() {
			itRetriesUntilStoreComesBack(func() error {
				return bbs.DesireLongRunningProcess(lrp)
			})
		})
	})

	Describe("Adding and removing actual LRPs", func() {
		var lrp models.LRP

		BeforeEach(func() {
			lrp = models.LRP{
				ProcessGuid:  "some-process-guid",
				InstanceGuid: "some-instance-guid",
				Index:        1,

				Host: "1.2.3.4",
				Ports: []models.PortMapping{
					{ContainerPort: 8080, HostPort: 65100},
					{ContainerPort: 8081, HostPort: 65101},
				},
			}
		})

		Describe("ReportActualLongRunningProcessAsStarting", func() {
			It("creates /v1/actual/<process-guid>/<index>/<instance-guid>", func() {
				err := bbs.ReportActualLongRunningProcessAsStarting(lrp)
				Ω(err).ShouldNot(HaveOccurred())

				node, err := etcdClient.Get("/v1/actual/some-process-guid/1/some-instance-guid")
				Ω(err).ShouldNot(HaveOccurred())

				expectedLRP := lrp
				expectedLRP.State = models.LRPStateStarting
				Ω(node.Value).Should(MatchJSON(expectedLRP.ToJSON()))
			})

			Context("when the store is out of commission", func() {
				itRetriesUntilStoreComesBack(func() error {
					return bbs.ReportActualLongRunningProcessAsStarting(lrp)
				})
			})
		})

		Describe("ReportActualLongRunningProcessAsRunning", func() {
			It("creates /v1/actual/<process-guid>/<index>/<instance-guid>", func() {
				err := bbs.ReportActualLongRunningProcessAsRunning(lrp)
				Ω(err).ShouldNot(HaveOccurred())

				node, err := etcdClient.Get("/v1/actual/some-process-guid/1/some-instance-guid")
				Ω(err).ShouldNot(HaveOccurred())

				expectedLRP := lrp
				expectedLRP.State = models.LRPStateRunning
				Ω(node.Value).Should(MatchJSON(expectedLRP.ToJSON()))
			})

			Context("when the store is out of commission", func() {
				itRetriesUntilStoreComesBack(func() error {
					return bbs.ReportActualLongRunningProcessAsRunning(lrp)
				})
			})
		})

		Describe("RemoveActualLongRunningProcess", func() {
			BeforeEach(func() {
				bbs.ReportActualLongRunningProcessAsStarting(lrp)
			})

			It("should remove the LRP", func() {
				err := bbs.RemoveActualLongRunningProcess(lrp)
				Ω(err).ShouldNot(HaveOccurred())

				_, err = etcdClient.Get("/v1/actual/some-process-guid/1/some-instance-guid")
				Ω(err).Should(MatchError(storeadapter.ErrorKeyNotFound))
			})

			Context("when the store is out of commission", func() {
				itRetriesUntilStoreComesBack(func() error {
					return bbs.RemoveActualLongRunningProcess(lrp)
				})
			})
		})
	})

})
