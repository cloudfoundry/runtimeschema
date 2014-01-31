package models_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry-incubator/runtime-schema/models"

	"encoding/json"
)

var _ = Describe("ExecutorAction", func() {
	var action ExecutorAction

	actionPayload := `{"name":"copy","args":{"compress":true,"extract":true,"from":"old_location","to":"new_location"}}`

	BeforeEach(func() {
		action = ExecutorAction{
			Name: "copy",
			Args: Arguments{
				"from":     "old_location",
				"to":       "new_location",
				"extract":  true,
				"compress": true,
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
			var unmarshalledAction ExecutorAction
			err := json.Unmarshal([]byte(actionPayload), &unmarshalledAction)
			Ω(err).Should(BeNil())
			Ω(unmarshalledAction).Should(Equal(action))
		})
	})

	Describe("Factories", func() {
		It("makes a copy object", func() {
			newCopy := NewCopyAction("http://from-location.com/myapp", "to-location", true, true)
			Ω(newCopy).ShouldNot(BeNil())

			Ω(newCopy.Name).Should(Equal("copy"))
			Ω(newCopy.Args).Should(Equal(Arguments{
				"from":     "http://from-location.com/myapp",
				"to":       "to-location",
				"extract":  true,
				"compress": true,
			}))
		})
	})
})
