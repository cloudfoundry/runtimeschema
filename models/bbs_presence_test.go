package models_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

var _ = Describe("BBSPresence", func() {
	var bbsPresence models.BBSPresence

	var payload string

	BeforeEach(func() {
		bbsPresence = models.NewBBSPresence("some-id", "https://some-host/")

		payload = `{
			"id": "some-id",
			"url": "https://some-host/"
		}`
	})

	Describe("ToJSON", func() {
		It("should JSONify", func() {
			json, err := models.ToJSON(&bbsPresence)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(json)).To(MatchJSON(payload))
		})
	})

	Describe("FromJSON", func() {
		It("returns an BBSPresence with correct fields", func() {
			decodedBBSPresence := &models.BBSPresence{}
			err := models.FromJSON([]byte(payload), decodedBBSPresence)
			Expect(err).NotTo(HaveOccurred())

			Expect(decodedBBSPresence).To(Equal(&bbsPresence))
		})

		Context("with an invalid payload", func() {
			It("returns the error", func() {
				payload = "aliens are here on earth"
				decodedBBSPresence := &models.BBSPresence{}
				err := models.FromJSON([]byte(payload), decodedBBSPresence)

				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Validate", func() {
		Context("when bbs presense is valid", func() {
			BeforeEach(func() {
				bbsPresence = models.NewBBSPresence("some-id", "https://some-host/")
			})

			It("returns no error", func() {
				Expect(bbsPresence.Validate()).NotTo(HaveOccurred())
			})
		})

		Context("when ID of bbs presense is invalid", func() {
			BeforeEach(func() {
				bbsPresence = models.NewBBSPresence("", "https://some-host/")
			})

			It("returns an error", func() {
				err := bbsPresence.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err).To(ContainElement(models.ErrInvalidField{Field: "id"}))
			})
		})

		Context("when url of bbs presense is missing", func() {
			BeforeEach(func() {
				bbsPresence = models.NewBBSPresence("some-id", "")
			})

			It("returns an error", func() {
				err := bbsPresence.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err).To(ContainElement(models.ErrInvalidField{Field: "url"}))
			})
		})

		Context("when url of bbs presense isn't absolute", func() {
			BeforeEach(func() {
				bbsPresence = models.NewBBSPresence("some-id", "//some-host/")
			})

			It("returns an error", func() {
				err := bbsPresence.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err).To(ContainElement(models.ErrInvalidField{Field: "url"}))
			})
		})
	})
})
