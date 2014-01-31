package models_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry-incubator/runtime-schema/models"
)

var _ = Describe("RunOnce", func() {
	var runOnce RunOnce
	var action ExecutorAction

	runOncePayload := `{"guid":"some-guid","reply_to":"some-requester","stack":"some-stack","executor_id":"executor","actions":[{"name":"copy","args":{"compress":true,"extract":true,"from":"old_location","to":"new_location"}}],"container_handle":"17fgsafdfcvc","failed":true,"failure_reason":"because i said so"}`

	BeforeEach(func() {
		action = NewCopyAction("old_location", "new_location", true, true)

		runOnce = RunOnce{
			Guid:            "some-guid",
			ReplyTo:         "some-requester",
			Stack:           "some-stack",
			Actions:         []ExecutorAction{action},
			ExecutorID:      "executor",
			ContainerHandle: "17fgsafdfcvc",
			Failed:          true,
			FailureReason:   "because i said so",
		}
	})

	Describe("ToJSON", func() {
		It("should JSONify", func() {
			json := runOnce.ToJSON()
			Ω(string(json)).Should(Equal(runOncePayload))
		})
	})

	Describe("NewRunOnceFromJSON", func() {
		It("returns a RunOnce with correct fields", func() {
			decodedRunOnce, err := NewRunOnceFromJSON([]byte(runOncePayload))
			Ω(err).ShouldNot(HaveOccurred())

			Ω(decodedRunOnce).Should(Equal(runOnce))
		})

		Context("with an invalid payload", func() {
			It("returns the error", func() {
				decodedRunOnce, err := NewRunOnceFromJSON([]byte("butts lol"))
				Ω(err).Should(HaveOccurred())

				Ω(decodedRunOnce).Should(BeZero())
			})
		})
	})
})
