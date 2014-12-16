package models_test

import (
	. "github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LRPStartAuction", func() {
	var startAuction LRPStartAuction
	var startAuctionPayload string

	BeforeEach(func() {
		startAuctionPayload = `{
    "desired_lrp": {
      "process_guid": "some-guid",
      "domain": "tests",
      "instances": 1,
      "stack": "some-stack",
      "start_timeout": 0,
      "root_fs": "docker:///docker.com/docker",
      "action": {"download": {
          "from": "http://example.com",
          "to": "/tmp/internet",
          "cache_key": ""
        }
      },
      "disk_mb": 512,
      "memory_mb": 1024,
      "cpu_weight": 42,
      "ports": [
        5678
      ],
      "routes": [
        "route-1",
        "route-2"
      ],
      "log_guid": "log-guid",
      "log_source": "the cloud"
    },
    "index": 2
  }`

		startAuction = LRPStartAuction{
			Index: 2,

			DesiredLRP: DesiredLRP{
				Domain:      "tests",
				ProcessGuid: "some-guid",

				RootFSPath: "docker:///docker.com/docker",
				Instances:  1,
				Stack:      "some-stack",
				MemoryMB:   1024,
				DiskMB:     512,
				CPUWeight:  42,
				Routes:     []string{"route-1", "route-2"},
				Ports: []uint32{
					5678,
				},
				LogGuid:   "log-guid",
				LogSource: "the cloud",
				Action: &DownloadAction{
					From: "http://example.com",
					To:   "/tmp/internet",
				},
			},
		}
	})

	Describe("ToJSON", func() {
		It("should JSONify", func() {
			jsonPayload, err := ToJSON(&startAuction)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(string(jsonPayload)).Should(MatchJSON(startAuctionPayload))
		})
	})

	Describe("FromJSON", func() {
		var decodedStartAuction *LRPStartAuction
		var err error

		JustBeforeEach(func() {
			decodedStartAuction = &LRPStartAuction{}
			err = FromJSON([]byte(startAuctionPayload), decodedStartAuction)
		})

		It("returns a LRP with correct fields", func() {
			Ω(err).ShouldNot(HaveOccurred())
			Ω(decodedStartAuction).Should(Equal(&startAuction))
		})

		Context("with an invalid payload", func() {
			BeforeEach(func() {
				startAuctionPayload = "aliens lol"
			})

			It("returns the error", func() {
				Ω(err).Should(HaveOccurred())
			})
		})
	})
})
