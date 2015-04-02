package heartbeater_test

import (
	"os"
	"time"

	"github.com/cloudfoundry-incubator/consuladapter"
	"github.com/cloudfoundry-incubator/runtime-schema/heartbeater"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/consul/structs"

	"github.com/pivotal-golang/clock/fakeclock"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"
	"github.com/tedsuo/ifrit"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	heartbeatKey   = "some-key"
	heartbeatValue = []byte("some-value")
	ttl            = structs.SessionTTLMin
)
var consulAdapter *consuladapter.Adapter

var newConsulAdapter = func() *consuladapter.Adapter {
	adapter, err := consuladapter.NewAdapter([]string{proxyAddress}, defaultScheme)
	Ω(err).ShouldNot(HaveOccurred())
	return adapter
}

var _ = Describe("Heartbeater", func() {

	var getValueForHeartbeatKey = func() ([]byte, error) {
		return consulAdapter.GetValue(heartbeatKey)
	}

	var (
		consulProxy   ifrit.Process
		heart         ifrit.Runner
		retryInterval time.Duration
		clock         *fakeclock.FakeClock
		logger        lager.Logger
	)

	BeforeEach(func() {
		consulProxy = ifrit.Invoke(proxyRunner)

		consulAdapter = newConsulAdapter()

		retryInterval = 100 * time.Millisecond
		clock = fakeclock.NewFakeClock(time.Now())
		logger = lagertest.NewTestLogger("heartbeater")

		heart = heartbeater.New(consulAdapter, heartbeatKey, heartbeatValue, ttl, clock, retryInterval, logger)
	})

	AfterEach(func() {
		consulProxy.Signal(os.Kill)
		Eventually(consulProxy.Wait()).Should(Receive(BeNil()))
	})

	Describe("Maintaining Locks", func() {
		Context("when the node does not exist", func() {
			var heartBeat ifrit.Process

			BeforeEach(func() {
				heartBeat = ifrit.Invoke(heart)
			})

			AfterEach(func() {
				heartBeat.Signal(os.Kill)
				Eventually(heartBeat.Wait()).Should(Receive(BeNil()))
			})

			It("continuously writes the given node to the store", func() {
				Consistently(getValueForHeartbeatKey, ttl+time.Second).Should(Equal(heartbeatValue))
			})
		})

		Context("when the node is deleted after we have aquired a lock", func() {
			var heartBeat ifrit.Process

			BeforeEach(func() {
				heartBeat = ifrit.Invoke(heart)
				Eventually(getValueForHeartbeatKey).Should(Equal(heartbeatValue))

				Eventually(func() error {
					err := consulAdapter.ReleaseAndDeleteLock(heartbeatKey)
					Ω(err).ShouldNot(HaveOccurred())
					_, err = getValueForHeartbeatKey()
					return err
				}).Should(Equal(consuladapter.NewKeyNotFoundError(heartbeatKey)))
			})

			It("exits", func() {
				Eventually(heartBeat.Wait()).Should(Receive(Equal(heartbeater.ErrLockLost)))
			})
		})
	})

	Describe("Shut Down", func() {
		var heartBeat ifrit.Process

		Context("when we are holding a lock in the store", func() {
			BeforeEach(func() {
				heartBeat = ifrit.Invoke(heart)
				Eventually(getValueForHeartbeatKey).Should(Equal(heartbeatValue))
			})

			AfterEach(func() {
				heartBeat.Signal(os.Kill)
				Eventually(heartBeat.Wait()).Should(Receive(BeNil()))
			})

			It("deletes the node from the store", func() {
				heartBeat.Signal(os.Interrupt)
				Eventually(heartBeat.Wait()).Should(Receive(BeNil()))

				_, err := getValueForHeartbeatKey()
				Ω(err).Should(Equal(consuladapter.NewKeyNotFoundError(heartbeatKey)))
			})
		})

		Context("when another maintainer is holding the lock we are trying to hold", func() {
			var cancelChan chan struct{}
			var otherAdapter *consuladapter.Adapter

			BeforeEach(func() {
				otherAdapter = newConsulAdapter()

				cancelChan = make(chan struct{})
				_, err := otherAdapter.AcquireAndMaintainLock(heartbeatKey, []byte("doppel-value"), ttl, cancelChan)
				Ω(err).ShouldNot(HaveOccurred())
			})

			JustBeforeEach(func() {
				heartBeat = ifrit.Background(heart)
			})

			AfterEach(func() {
				close(cancelChan)
				heartBeat.Signal(os.Kill)
				Eventually(heartBeat.Wait()).Should(Receive(BeNil()))
			})

			It("does not delete the original node from the store", func() {
				heartBeat.Signal(os.Interrupt)
				Eventually(heartBeat.Wait()).Should(Receive(BeNil()))

				Consistently(getValueForHeartbeatKey).Should(Equal([]byte("doppel-value")))
			})

			It("never signals ready", func() {
				Consistently(heartBeat.Ready()).ShouldNot(Receive())
			})
		})

		Context("when we have lost connection to the store", func() {
			BeforeEach(func() {
				heartBeat = ifrit.Invoke(heart)
				Eventually(getValueForHeartbeatKey).Should(Equal(heartbeatValue))

				// simulate network partiion
				consulProxy.Signal(os.Kill)
				Eventually(consulProxy.Wait()).Should(Receive(BeNil()))
			})

			It("exits", func() {
				Eventually(heartBeat.Wait()).Should(Receive(Equal(heartbeater.ErrLockLost)))
			})
		})
	})

	Describe("Lock Contention", func() {
		Context("when someone else tries to acquire the lock after us", func() {
			var heartBeat ifrit.Process

			BeforeEach(func() {
				heartBeat = ifrit.Invoke(heart)
				Eventually(getValueForHeartbeatKey).Should(Equal(heartbeatValue))
			})

			AfterEach(func() {
				heartBeat.Signal(os.Kill)
				Eventually(heartBeat.Wait()).Should(Receive())
			})

			It("retains our original value", func() {
				errChan := make(chan error)
				cancelChan := make(chan struct{})

				go func() {
					_, err := newConsulAdapter().AcquireAndMaintainLock(heartbeatKey, []byte("doppel-value"), ttl, cancelChan)
					errChan <- err
				}()

				time.Sleep(api.DefaultLockWaitTime + time.Second)
				close(cancelChan)
				Eventually(errChan, api.DefaultLockWaitTime).Should(Receive(Equal(consuladapter.NewCancelledLockAttemptError(heartbeatKey))))

				value, err := getValueForHeartbeatKey()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(value).Should(Equal(heartbeatValue))
			})
		})

		Context("when someone else already has the lock first", func() {
			var heartbeat ifrit.Process
			var cancelChan chan struct{}
			var otherAdapter *consuladapter.Adapter

			BeforeEach(func() {
				otherAdapter = newConsulAdapter()

				cancelChan = make(chan struct{})
				_, err := otherAdapter.AcquireAndMaintainLock(heartbeatKey, []byte("doppel-value"), ttl, cancelChan)
				Ω(err).ShouldNot(HaveOccurred())
			})

			JustBeforeEach(func() {
				heartbeat = ifrit.Background(heart)
			})

			AfterEach(func() {
				_ = otherAdapter.ReleaseAndDeleteLock(heartbeatKey)
				close(cancelChan)

				heartbeat.Signal(os.Kill)
				Eventually(heartbeat.Wait()).Should(Receive(BeNil()))
			})

			Context("and the other maintainer does not go away", func() {
				It("does not overwrite the existing value", func() {
					Consistently(getValueForHeartbeatKey).Should(Equal([]byte("doppel-value")))
				})
			})

			Context("and the other maintainer goes away", func() {
				BeforeEach(func() {
					heart = heartbeater.New(consulAdapter, heartbeatKey, heartbeatValue, ttl, clock, retryInterval, logger)

					go func(adapter *consuladapter.Adapter) {
						defer GinkgoRecover()

						err := adapter.ReleaseAndDeleteLock(heartbeatKey)
						Ω(err).ShouldNot(HaveOccurred())
					}(otherAdapter)
				})

				It("starts heartbeating once it disappears", func() {
					clock.Increment(2 * retryInterval)
					Eventually(getValueForHeartbeatKey).Should(Equal(heartbeatValue))
				})
			})
		})
	})

	Describe("Losing connections", func() {
		Context("when we cannot initially connect to the store", func() {
			var heartBeat ifrit.Process

			BeforeEach(func() {
				consulProxy.Signal(os.Kill)
				Eventually(consulProxy.Wait()).Should(Receive(BeNil()))

				heartBeat = ifrit.Background(heart)
			})

			Context("when we are eventually able to connect to the store", func() {
				var consulProxyChan chan ifrit.Process

				BeforeEach(func() {
					consulProxyChan = make(chan ifrit.Process)
					go func() {
						consulProxyChan <- ifrit.Invoke(proxyRunner)
					}()

					clock.Increment(2 * retryInterval)
				})

				AfterEach(func() {
					consulProxy = <-consulProxyChan
					heartBeat.Signal(os.Kill)
					Eventually(heartBeat.Wait()).Should(Receive(BeNil()))
				})

				It("begins heartbeating", func() {
					Eventually(getValueForHeartbeatKey).Should(Equal(heartbeatValue))
				})
			})

			Context("when we consistently are unable to connect to the store", func() {
				BeforeEach(func() {
					clock.Increment(2 * retryInterval)
				})

				AfterEach(func() {
					// we have to start the store to ensure the blocked heartbeat does not pollute other tests
					heartBeat.Signal(os.Kill)
					Eventually(heartBeat.Wait()).Should(Receive(BeNil()))
				})

				It("blocks on invoke forever", func() {
					Consistently(heartBeat.Ready(), ttl+time.Second).ShouldNot(Receive())
				})
			})
		})
	})
})
