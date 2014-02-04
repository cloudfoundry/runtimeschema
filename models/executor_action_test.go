package models_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry-incubator/runtime-schema/models"

	"encoding/json"
)

var _ = Describe("ExecutorAction", func() {
	var action *ExecutorAction

	Describe("With an invalid action", func() {
		It("should fail to marshal", func() {
			invalidAction := []string{"butts", "from", "mars"}
			payload, err := json.Marshal(&ExecutorAction{Action: invalidAction})
			Ω(payload).Should(BeZero())
			Ω(err.(*json.MarshalerError).Err).Should(Equal(InvalidActionConversion))
		})

		It("should fail to unmarshal", func() {
			var unmarshalledAction *ExecutorAction
			actionPayload := `{"action":"buttz","args":{"from":"space"}}`
			err := json.Unmarshal([]byte(actionPayload), &unmarshalledAction)
			Ω(err).Should(Equal(InvalidActionConversion))
		})
	})

	Describe("Copy", func() {
		actionPayload := `{"action":"copy","args":{"from":"old_location","to":"new_location","extract":true,"compress":true}}`

		BeforeEach(func() {
			action = &ExecutorAction{
				Action: CopyAction{
					From:     "old_location",
					To:       "new_location",
					Extract:  true,
					Compress: true,
				},
			}
		})

		Describe("Converting to JSON", func() {
			It("creates a json representation of the object", func() {
				json, err := json.Marshal(action)
				Ω(err).Should(BeNil())
				Ω(string(json)).Should(Equal(actionPayload))
			})
		})

		Describe("Converting from JSON", func() {
			It("constructs an object from the json string", func() {
				var unmarshalledAction *ExecutorAction
				err := json.Unmarshal([]byte(actionPayload), &unmarshalledAction)
				Ω(err).Should(BeNil())
				Ω(unmarshalledAction).Should(Equal(action))
			})
		})
	})
})
