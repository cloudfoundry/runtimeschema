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
    "instance_guid": "some-instance-guid",
    "index": 2,
    "state": 1,
    "updated_at": 1138
  }`

		startAuction = LRPStartAuction{
			Index:        2,
			InstanceGuid: "some-instance-guid",

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

			State:     LRPStartAuctionStatePending,
			UpdatedAt: 1138,
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

		for field, payload := range map[string]string{
			"instance_guid": `{"process_guid": "process-guid", "index": 0}`,
		} {
			missingField := field
			jsonPayload := payload

			Context("when the json is missing a "+missingField, func() {
				BeforeEach(func() {
					startAuctionPayload = jsonPayload
				})

				It("returns an error indicating so", func() {
					Ω(err).Should(HaveOccurred())
					Ω(err.Error()).Should(ContainSubstring("Invalid field: " + missingField))
				})
			})
		}
	})
})
