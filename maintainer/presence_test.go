package maintainer_test

import (
	"os"
	"time"

	"github.com/cloudfoundry-incubator/consuladapter"
	"github.com/cloudfoundry-incubator/runtime-schema/maintainer"

	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	presenceKey   = "some-presence"
	presenceValue = []byte("some-value")
)

var _ = Describe("Presence", func() {
	var (
		consulSession *consuladapter.Session

		presenceRunner ifrit.Runner
		retryInterval  time.Duration
		logger         lager.Logger
	)

	getPresenceValue := func() ([]byte, error) {
		return consulSession.GetAcquiredValue(presenceKey)
	}

	BeforeEach(func() {
		consulSession = consulRunner.NewSession("a-session")

		retryInterval = 500 * time.Millisecond
		clock := clock.NewClock()
		logger = lagertest.NewTestLogger("maintainer")

		presenceRunner = maintainer.NewPresence(consulSession, presenceKey, presenceValue, clock, retryInterval, logger)
	})

	AfterEach(func() {
		consulSession.Destroy()
	})

	Describe("Maintaining Presence", func() {
		Context("when the node does not exist", func() {
			var presenceProcess ifrit.Process

			BeforeEach(func() {
				presenceProcess = ifrit.Invoke(presenceRunner)
			})

			AfterEach(func() {
				ginkgomon.Kill(presenceProcess)
			})

			It("has a value in the store", func() {
				Eventually(getPresenceValue).Should(Equal(presenceValue))
				Consistently(getPresenceValue).Should(Equal(presenceValue))
			})
		})

		Context("when the node is deleted after we have set presence", func() {
			var presenceProcess ifrit.Process

			BeforeEach(func() {
				presenceProcess = ifrit.Invoke(presenceRunner)

				kv := consulRunner.NewClient().KV()
				pair, _, err := kv.Get(presenceKey, nil)
				Ω(err).ShouldNot(HaveOccurred())
				kv.Release(pair, nil)

				It("exits", func() {
					Eventually(presenceProcess.Wait()).Should(Receive(Equal(maintainer.ErrLockLost)))
				})
			})
		})

		Describe("Shut Down", func() {
			var presenceProcess ifrit.Process

			Context("when we have presence in the store", func() {
				BeforeEach(func() {
					presenceProcess = ifrit.Invoke(presenceRunner)
					Eventually(getPresenceValue).Should(Equal(presenceValue))
				})

				AfterEach(func() {
					ginkgomon.Kill(presenceProcess)
				})

				It("deletes the node from the store", func() {
					presenceProcess.Signal(os.Interrupt)
					Eventually(presenceProcess.Wait()).Should(Receive(BeNil()))

					_, err := getPresenceValue()
					Ω(err).Should(Equal(consuladapter.NewKeyNotFoundError(presenceKey)))
				})
			})

			Context("when another maintainer has presence we are trying to set", func() {
				var otherSession *consuladapter.Session

				BeforeEach(func() {
					otherSession = consulRunner.NewSession("otherSession")

					_, err := otherSession.SetPresence(presenceKey, []byte("doppel-value"))
					Ω(err).ShouldNot(HaveOccurred())
				})

				JustBeforeEach(func() {
					presenceProcess = ifrit.Background(presenceRunner)
				})

				AfterEach(func() {
					ginkgomon.Interrupt(presenceProcess)
				})

				It("does not delete the original node from the store", func() {
					ginkgomon.Interrupt(presenceProcess)

					Consistently(getPresenceValue).Should(Equal([]byte("doppel-value")))
				})

				It("never signals ready", func() {
					Consistently(presenceProcess.Ready()).ShouldNot(Receive())
				})
			})

			Context("when we have lost connection to the store", func() {
				BeforeEach(func() {
					presenceProcess = ifrit.Invoke(presenceRunner)
					Eventually(presenceProcess.Ready()).Should(BeClosed())

					consulRunner.Stop()
				})

				AfterEach(func() {
					consulRunner.Start()
				})

				It("remains up", func() {
					Consistently(presenceProcess.Wait()).ShouldNot(Receive())
				})
			})
		})

		Describe("Lock Contention", func() {
			Context("when someone else tries to gain presence after us", func() {
				var presenceProcess ifrit.Process

				BeforeEach(func() {
					presenceProcess = ifrit.Invoke(presenceRunner)
					Eventually(getPresenceValue).Should(Equal(presenceValue))
				})

				AfterEach(func() {
					ginkgomon.Kill(presenceProcess)
				})

				It("retains our original value", func() {
					otherSession := consulRunner.NewSession("some-session")
					go func() {
						otherSession.SetPresence(presenceKey, []byte("doppel-value"))
					}()

					Consistently(getPresenceValue).Should(Equal(presenceValue))

					otherSession.Destroy()
					Eventually(otherSession.Err()).Should(Receive(BeNil()))
				})
			})

			Context("when someone else already has presence first", func() {
				var presenceProcess ifrit.Process
				var otherSession *consuladapter.Session

				BeforeEach(func() {
					otherSession = consulRunner.NewSession("other-session")

					_, err := otherSession.SetPresence(presenceKey, []byte("doppel-value"))
					Ω(err).ShouldNot(HaveOccurred())
				})

				JustBeforeEach(func() {
					presenceProcess = ifrit.Background(presenceRunner)
				})

				AfterEach(func() {
					otherSession.Destroy()
					ginkgomon.Kill(presenceProcess)
				})

				Context("and the other maintainer does not go away", func() {
					It("does not overwrite the existing value", func() {
						Consistently(getPresenceValue).Should(Equal([]byte("doppel-value")))
					})
				})

				Context("and the other maintainer goes away", func() {
					BeforeEach(func() {
						otherSession.Destroy()
					})

					It("gains presence", func() {
						Eventually(getPresenceValue).Should(Equal(presenceValue))
					})
				})
			})
		})

		Describe("Losing connections", func() {
			Context("when we cannot initially connect to the store", func() {
				var presenceProcess ifrit.Process

				BeforeEach(func() {
					consulRunner.Stop()
					presenceProcess = ifrit.Background(presenceRunner)
				})

				AfterEach(func() {
					ginkgomon.Kill(presenceProcess)
					consulRunner.Start()
				})

				It("is ready but does not exit", func() {
					Eventually(presenceProcess.Ready()).Should(BeClosed())
					Consistently(presenceProcess.Wait()).ShouldNot(BeClosed())
				})

				Context("when we are eventually able to connect to the store", func() {
					BeforeEach(func() {
						consulRunner.Start()
						consulRunner.WaitUntilReady()
					})

					AfterEach(func() {
						ginkgomon.Kill(presenceProcess)
					})

					It("sets presence", func() {
						Eventually(getPresenceValue).Should(Equal(presenceValue))
					})
				})
			})
		})
	})
})
