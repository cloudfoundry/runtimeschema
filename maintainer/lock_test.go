package maintainer_test

import (
	"os"
	"time"

	"github.com/cloudfoundry-incubator/consuladapter"
	"github.com/cloudfoundry-incubator/runtime-schema/maintainer"
	"github.com/hashicorp/consul/consul/structs"

	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	lockKey   = "some-key"
	lockValue = []byte("some-value")
	ttl       = structs.SessionTTLMin
)

var _ = Describe("Lock", func() {
	var (
		consulSession *consuladapter.Session

		lockRunner    ifrit.Runner
		retryInterval time.Duration
		logger        lager.Logger
	)

	getLockValue := func() ([]byte, error) {
		return consulSession.GetAcquiredValue(lockKey)
	}

	BeforeEach(func() {
		consulSession = consulRunner.NewSession("a-session")

		retryInterval = 500 * time.Millisecond
		clock := clock.NewClock()
		logger = lagertest.NewTestLogger("maintainer")

		lockRunner = maintainer.NewLock(consulSession, lockKey, lockValue, clock, retryInterval, logger)
	})

	AfterEach(func() {
		consulSession.Destroy()
	})

	Describe("Maintaining Locks", func() {
		Context("when the node does not exist", func() {
			var lockProcess ifrit.Process

			BeforeEach(func() {
				lockProcess = ifrit.Invoke(lockRunner)
			})

			AfterEach(func() {
				ginkgomon.Kill(lockProcess)
			})

			It("has a value in the store", func() {
				Consistently(getLockValue).Should(Equal(lockValue))
			})
		})

		Context("when the node is deleted after we have aquired a lock", func() {
			var lockProcess ifrit.Process

			BeforeEach(func() {
				lockProcess = ifrit.Invoke(lockRunner)

				kv := consulRunner.NewClient().KV()
				pair, _, err := kv.Get(lockKey, nil)
				Ω(err).ShouldNot(HaveOccurred())
				kv.Release(pair, nil)

				It("exits", func() {
					Eventually(lockProcess.Wait()).Should(Receive(Equal(maintainer.ErrLockLost)))
				})
			})
		})

		Describe("Shut Down", func() {
			var lockProcess ifrit.Process

			Context("when we are holding a lock in the store", func() {
				BeforeEach(func() {
					lockProcess = ifrit.Invoke(lockRunner)
					Eventually(getLockValue).Should(Equal(lockValue))
				})

				AfterEach(func() {
					ginkgomon.Kill(lockProcess)
				})

				It("deletes the node from the store", func() {
					lockProcess.Signal(os.Interrupt)
					Eventually(lockProcess.Wait()).Should(Receive(BeNil()))

					_, err := getLockValue()
					Ω(err).Should(Equal(consuladapter.NewKeyNotFoundError(lockKey)))
				})
			})

			Context("when another maintainer is holding the lock we are trying to hold", func() {
				var otherSession *consuladapter.Session

				BeforeEach(func() {
					otherSession = consulRunner.NewSession("otherSession")

					err := otherSession.AcquireLock(lockKey, []byte("doppel-value"))
					Ω(err).ShouldNot(HaveOccurred())
				})

				JustBeforeEach(func() {
					lockProcess = ifrit.Background(lockRunner)
				})

				AfterEach(func() {
					ginkgomon.Interrupt(lockProcess)
				})

				It("does not delete the original node from the store", func() {
					ginkgomon.Interrupt(lockProcess)

					Consistently(getLockValue).Should(Equal([]byte("doppel-value")))
				})

				It("never signals ready", func() {
					Consistently(lockProcess.Ready()).ShouldNot(Receive())
				})
			})

			Context("when we have lost connection to the store", func() {
				BeforeEach(func() {
					lockProcess = ifrit.Invoke(lockRunner)
					Eventually(lockProcess.Ready()).Should(BeClosed())

					consulRunner.Stop()
				})

				AfterEach(func() {
					consulRunner.Start()
				})

				It("exits", func() {
					Eventually(lockProcess.Wait()).Should(Receive(Equal(maintainer.ErrLockLost)))
				})
			})
		})

		Describe("Lock Contention", func() {
			Context("when someone else tries to acquire the lock after us", func() {
				var lockProcess ifrit.Process

				BeforeEach(func() {
					lockProcess = ifrit.Invoke(lockRunner)
					Eventually(getLockValue).Should(Equal(lockValue))
				})

				AfterEach(func() {
					ginkgomon.Kill(lockProcess)
				})

				It("retains our original value", func() {
					otherSession := consulRunner.NewSession("some-session")
					go func() {
						otherSession.AcquireLock(lockKey, []byte("doppel-value"))
					}()

					Consistently(getLockValue).Should(Equal(lockValue))

					otherSession.Destroy()
					Eventually(otherSession.Err()).Should(Receive(BeNil()))
				})
			})

			Context("when someone else already has the lock first", func() {
				var lockProcess ifrit.Process
				var otherSession *consuladapter.Session

				BeforeEach(func() {
					otherSession = consulRunner.NewSession("other-session")

					err := otherSession.AcquireLock(lockKey, []byte("doppel-value"))
					Ω(err).ShouldNot(HaveOccurred())
				})

				JustBeforeEach(func() {
					lockProcess = ifrit.Background(lockRunner)
				})

				AfterEach(func() {
					otherSession.Destroy()
					ginkgomon.Kill(lockProcess)
				})

				Context("and the other maintainer does not go away", func() {
					It("does not overwrite the existing value", func() {
						Consistently(getLockValue).Should(Equal([]byte("doppel-value")))
					})
				})

				Context("and the other maintainer goes away", func() {
					BeforeEach(func() {
						otherSession.Destroy()
					})

					It("acquires the lock", func() {
						Eventually(getLockValue).Should(Equal(lockValue))
					})
				})
			})
		})

		Describe("Losing connections", func() {
			Context("when we cannot initially connect to the store", func() {
				var lockProcess ifrit.Process

				BeforeEach(func() {
					consulRunner.Stop()
					lockProcess = ifrit.Background(lockRunner)
				})

				AfterEach(func() {
					ginkgomon.Kill(lockProcess)
					consulRunner.Start()
				})

				It("does not acquire the lock", func() {
					Consistently(lockProcess.Ready()).ShouldNot(BeClosed())
				})

				Context("when we are eventually able to connect to the store", func() {
					BeforeEach(func() {
						consulRunner.Start()
					})

					AfterEach(func() {
						ginkgomon.Kill(lockProcess)
					})

					It("acquires the lock", func() {
						Eventually(lockProcess.Ready(), 10).Should(BeClosed())
					})
				})
			})
		})
	})
})
