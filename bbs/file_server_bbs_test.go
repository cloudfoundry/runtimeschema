package bbs_test

import (
	. "github.com/cloudfoundry-incubator/runtime-schema/bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/models/factories"
	"github.com/cloudfoundry/storeadapter"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Stager BBS", func() {
	var (
		bbs           *BBS
		fileServerURL string
		fileServerId  string
		interval      uint64
		errors        chan error
		stop          chan bool
		err           error
	)

	BeforeEach(func() {
		bbs = New(store)
	})

	Describe("MaintainFileServerPresence", func() {
		BeforeEach(func() {
			fileServerURL = "stubFileServerURL"
			fileServerId = factories.GenerateGuid()
			interval = uint64(1)

			stop, errors, err = bbs.MaintainFileServerPresence(interval, fileServerURL, fileServerId)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("should put /file-server/FILE_SERVER_ID in the store with a TTL", func() {
			node, err := store.Get("/v1/file_server/" + fileServerId)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(node).Should(Equal(storeadapter.StoreNode{
				Key:   "/v1/file_server/" + fileServerId,
				Value: []byte(fileServerURL),
				TTL:   interval, // move to config one day
			}))

			close(stop)
		})

		It("should periodically maintain the TTL", func() {
			time.Sleep(2 * time.Second)

			_, err = store.Get("/v1/file_server/" + fileServerId)
			Ω(err).ShouldNot(HaveOccurred())

			close(stop)
		})

		It("should report an error and stop trying if it fails to update the TTL", func() {
			err = store.Delete("/v1/file_server/" + fileServerId)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(errors, 2).Should(Receive())
			close(stop)
		})

		It("should be possible to stop maintaining presence", func() {
			close(stop)

			time.Sleep(2 * time.Second)

			_, err = store.Get("/v1/file_server/" + fileServerId)
			Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))
		})
	})

	Describe("GetAvailableFileServer", func() {
		Context("when there are available file servers", func() {
			BeforeEach(func() {
				fileServerURL = "http://128.70.3.29:8012"
				fileServerId = factories.GenerateGuid()
				interval = uint64(1)

				stop, errors, err = bbs.MaintainFileServerPresence(interval, fileServerURL, fileServerId)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("should get from /v1/file_server/", func() {
				url, err := bbs.GetAvailableFileServer()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(url).Should(Equal(fileServerURL))
			})
		})

		Context("when there are several available file servers", func() {
			var otherFileServerURL string
			BeforeEach(func() {
				fileServerURL = "http://guy"
				otherFileServerURL = "http://other.guy"

				fileServerId = factories.GenerateGuid()
				otherFileServerId := factories.GenerateGuid()

				interval = uint64(1)

				stop, errors, err = bbs.MaintainFileServerPresence(interval, fileServerURL, fileServerId)
				Ω(err).ShouldNot(HaveOccurred())

				stop, errors, err = bbs.MaintainFileServerPresence(interval, otherFileServerURL, otherFileServerId)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("should pick one at random", func() {
				result := map[string]bool{}

				for i := 0; i < 10; i++ {
					url, err := bbs.GetAvailableFileServer()
					Ω(err).ShouldNot(HaveOccurred())
					result[url] = true
				}

				Ω(result).Should(HaveLen(2))
				Ω(result).Should(HaveKey(fileServerURL))
				Ω(result).Should(HaveKey(otherFileServerURL))
			})
		})

		Context("when there are none", func() {
			It("should error", func() {
				url, err := bbs.GetAvailableFileServer()
				Ω(err).Should(HaveOccurred())
				Ω(url).Should(BeZero())
			})
		})
	})
})
