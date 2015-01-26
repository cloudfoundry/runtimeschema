package heartbeater_test

import (
	"fmt"
	"os"
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/heartbeater"
	"github.com/cloudfoundry/gunk/workpool"
	"github.com/cloudfoundry/storeadapter"
	"github.com/cloudfoundry/storeadapter/etcdstoreadapter"
	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/clock/fakeclock"

	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"
	"github.com/tedsuo/ifrit"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	heartbeatKey   = "/some-key"
	heartbeatValue = "some-value"
)

var etcdClient storeadapter.StoreAdapter

var _ = Describe("Heartbeater", func() {
	var (
		etcdProxy ifrit.Process
		heart     ifrit.Runner
		logger    lager.Logger

		heartbeatClock    clock.Clock
		heartbeatInterval time.Duration

		expectedHeartbeatNode storeadapter.StoreNode
	)

	BeforeEach(func() {
		heartbeatClock = clock.NewClock()

		etcdRunner.Stop()
		etcdRunner.Start()
		etcdProxy = ifrit.Invoke(proxyRunner)

		etcdClient = etcdstoreadapter.NewETCDStoreAdapter([]string{proxyUrl}, workpool.NewWorkPool(10))
		etcdClient.Connect()

		logger = lagertest.NewTestLogger("test")
		heartbeatInterval = 100 * time.Millisecond

		expectedHeartbeatNode = storeadapter.StoreNode{
			Key:   heartbeatKey,
			Value: []byte(heartbeatValue),
			TTL:   1,
		}

		heart = heartbeater.New(etcdClient, heartbeatClock, heartbeatKey, heartbeatValue, heartbeatInterval, logger)
	})

	AfterEach(func() {
		etcdProxy.Signal(os.Kill)
		Eventually(etcdProxy.Wait(), 3*time.Second).Should(Receive(BeNil()))
	})

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
			Consistently(matchHeartbeatNode(expectedHeartbeatNode), heartbeatInterval*4).ShouldNot(HaveOccurred())
		})

		Context("and it is sent a signal", func() {
			BeforeEach(func() {
				heartBeat.Signal(os.Interrupt)
			})

			It("exits and deletes the node from the store", func() {
				Eventually(heartBeat.Wait()).Should(Receive(BeNil()))
				_, err := etcdClient.Get(expectedHeartbeatNode.Key)
				Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))
			})
		})

		Context("and it is sent the kill signal", func() {
			BeforeEach(func() {
				heartBeat.Signal(os.Kill)
			})

			It("exits and does not delete the node from the store", func() {
				Eventually(heartBeat.Wait()).Should(Receive(BeNil()))
				Ω(matchHeartbeatNode(expectedHeartbeatNode)()).ShouldNot(HaveOccurred())
			})
		})
	})

	Context("when the node is deleted after we have aquired a lock", func() {
		var heartBeat ifrit.Process

		BeforeEach(func() {
			heartBeat = ifrit.Invoke(heart)
			err := etcdClient.Delete(heartbeatKey)
			Ω(err).ShouldNot(HaveOccurred())
		})

		AfterEach(func() {
			heartBeat.Signal(os.Kill)
			Eventually(heartBeat.Wait()).Should(Receive())
		})

		It("re-creates the node", func() {
			Eventually(matchHeartbeatNode(expectedHeartbeatNode)).Should(BeNil())
		})

		Describe("when there is a connection error", func() {
			BeforeEach(func() {
				etcdProxy.Signal(os.Kill)
				Eventually(etcdProxy.Wait(), 3*time.Second).Should(Receive(BeNil()))
			})

			It("retries until it succeeds", func() {
				restarted := make(chan struct{})
				go func() {
					time.Sleep(500 * time.Millisecond)
					etcdProxy = ifrit.Invoke(proxyRunner)
					close(restarted)
				}()

				Eventually(matchHeartbeatNode(expectedHeartbeatNode)).Should(BeNil())
				<-restarted
			})

			Describe("when the TTL expires", func() {
				It("exits with an error", func() {
					Eventually(heartBeat.Wait(), 5*time.Second).Should(Receive(Equal(heartbeater.ErrStoreUnavailable)))
				})
			})
		})

		Describe("when someone else acquires the lock after us", func() {
			var doppelNode storeadapter.StoreNode

			BeforeEach(func() {
				doppelNode = storeadapter.StoreNode{
					Key:   heartbeatKey,
					Value: []byte("doppel-value"),
				}

				err := etcdClient.Create(doppelNode)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("does not write to the node", func() {
				Consistently(matchHeartbeatNode(doppelNode), heartbeatInterval*2).Should(BeNil())
			})

			It("exits with an error", func() {
				Eventually(heartBeat.Wait(), 5*time.Second).Should(Receive(Equal(heartbeater.ErrLockFailed)))
			})
		})
	})

	Context("when we already have the lock", func() {
		var sigChan chan os.Signal
		var readyChan chan struct{}
		var doneChan chan struct{}

		BeforeEach(func() {
			sigChan = make(chan os.Signal, 1)
			readyChan = make(chan struct{})
			doneChan = make(chan struct{})

			heartBeat := ifrit.Invoke(heart)
			heartBeat.Signal(os.Kill)
			Eventually(heartBeat.Wait()).Should(Receive(BeNil()))

			go func() {
				heart.Run(sigChan, readyChan)
				close(doneChan)
			}()
		})

		AfterEach(func() {
			sigChan <- os.Kill
			Eventually(doneChan).Should(BeClosed())
		})

		It("becomes ready immediately", func() {
			select {
			case <-readyChan:
			case <-time.After(1 * time.Second):
				Fail("TTL expired before heartbeater became ready")
			}
		})

		It("continuously writes the given node to the store", func() {
			<-readyChan
			Consistently(matchHeartbeatNode(expectedHeartbeatNode), heartbeatInterval*4).ShouldNot(HaveOccurred())
		})
	})

	Context("when someone else already has the lock first", func() {
		var doppelNode storeadapter.StoreNode
		var heartbeat ifrit.Process

		BeforeEach(func() {
			doppelNode = storeadapter.StoreNode{
				Key:   heartbeatKey,
				Value: []byte("doppel-value"),
			}

			err := etcdClient.Create(doppelNode)
			Ω(err).ShouldNot(HaveOccurred())
		})

		JustBeforeEach(func() {
			heartbeat = ifrit.Background(heart)
		})

		AfterEach(func() {
			err := etcdClient.Delete(heartbeatKey)
			Ω(err).ShouldNot(HaveOccurred())

			heartbeat.Signal(os.Kill)
			Eventually(heartbeat.Wait()).Should(Receive(BeNil()))
		})

		Context("and the other maintainer does not go away", func() {
			It("does not overwrite the existing value", func() {
				Consistently(matchHeartbeatNode(doppelNode), 2*heartbeatInterval).Should(BeNil())
			})

			It("exits when signalled", func() {
				heartbeat.Signal(os.Kill)
				Eventually(heartbeat.Wait()).Should(Receive())
			})

			It("never signals ready", func() {
				Consistently(heartbeat.Ready()).ShouldNot(Receive())
			})
		})

		Context("and the other maintainer goes away", func() {
			var fakeClock *fakeclock.FakeClock
			var done chan struct{}

			BeforeEach(func() {
				fakeClock = fakeclock.NewFakeClock(time.Now())
				heartbeatClock = fakeClock

				heartbeatInterval = time.Minute
				expectedHeartbeatNode.TTL = 120

				heart = heartbeater.New(etcdClient, heartbeatClock, heartbeatKey, heartbeatValue, heartbeatInterval, logger)

				done = make(chan struct{})
				go func(etcdClient storeadapter.StoreAdapter) {
					defer GinkgoRecover()
					defer close(done)

					time.Sleep(time.Second)
					err := etcdClient.Delete(heartbeatKey)
					Ω(err).ShouldNot(HaveOccurred())
				}(etcdClient)
			})

			It("registers an interval timer as a fallback", func() {
				<-done
				Eventually(fakeClock.WatcherCount).Should(Equal(1))

				fakeClock.Increment(heartbeatInterval)
				Eventually(matchHeartbeatNode(expectedHeartbeatNode)).Should(BeNil())
			})

			It("starts heartbeating once it disappears", func() {
				Eventually(matchHeartbeatNode(expectedHeartbeatNode), 2*time.Second).Should(BeNil())
			})
		})
	})

	Context("when we cannot connect to etcd", func() {
		var heartbeatChan chan ifrit.Process

		BeforeEach(func() {
			etcdProxy.Signal(os.Kill)
			Eventually(etcdProxy.Wait()).Should(Receive(BeNil()))

			heartbeatChan = make(chan ifrit.Process)
			go func() {
				heartbeatChan <- ifrit.Invoke(heart)
			}()
		})

		Context("and etcd eventually comes back", func() {
			var etcdProxyChan chan ifrit.Process

			BeforeEach(func() {
				etcdProxyChan = make(chan ifrit.Process)
				go func() {
					time.Sleep(heartbeatInterval)
					etcdProxyChan <- ifrit.Invoke(proxyRunner)
				}()
			})

			AfterEach(func() {
				etcdProxy = <-etcdProxyChan
				heartBeat := <-heartbeatChan
				heartBeat.Signal(os.Kill)
				Eventually(heartBeat.Wait()).Should(Receive(BeNil()))
			})

			It("begins heartbeating", func() {
				Eventually(matchHeartbeatNode(expectedHeartbeatNode)).Should(BeNil())
			})
		})

		Context("and etcd never comes back", func() {
			AfterEach(func() {
				// we have to start etcd to ensure the blocked heartbeat does not pollute other tests
				etcdProxy = ifrit.Invoke(proxyRunner)
				var heartBeat ifrit.Process
				Eventually(heartbeatChan).Should(Receive(&heartBeat))
				heartBeat.Signal(os.Kill)
			})

			It("blocks on envoke forever", func() {
				Consistently(heartbeatChan, 2*heartbeatInterval).ShouldNot(Receive())
			})
		})
	})

	Context("when we lose our existing connection to etcd", func() {
		var heartBeat ifrit.Process

		BeforeEach(func() {
			heartBeat = ifrit.Invoke(heart)

			// simulate network partiion
			etcdProxy.Signal(os.Kill)
			Eventually(etcdProxy.Wait()).Should(Receive(BeNil()))
		})

		Context("and etcd comes back before the ttl expires", func() {
			BeforeEach(func() {
				time.Sleep(heartbeatInterval)
				etcdProxy = ifrit.Invoke(proxyRunner)
			})

			AfterEach(func() {
				heartBeat.Signal(os.Kill)
				Eventually(heartBeat.Wait()).Should(Receive(BeNil()))
			})

			It("resumes heartbeating", func() {
				Eventually(matchHeartbeatNode(expectedHeartbeatNode), 2*time.Second).Should(BeNil())
			})
		})

		Context("and etcd does not come back before the ttl expires", func() {
			It("exits with an error", func() {
				Eventually(heartBeat.Wait(), 4*time.Second).Should(Receive(Equal(heartbeater.ErrStoreUnavailable)))
			})
		})
	})
})

func matchHeartbeatNode(expectedNode storeadapter.StoreNode) func() error {
	return func() error {
		node, err := etcdClient.Get(expectedNode.Key)

		if err != nil {
			return err
		}

		if node.Key != expectedNode.Key {
			return fmt.Errorf("Key not matching: %s : %s", node.Key, expectedNode.Key)
		}

		if string(node.Value) != string(expectedNode.Value) {
			return fmt.Errorf("Value not matching: %s : %s", node.Value, expectedNode.Value)
		}

		if node.TTL != expectedNode.TTL {
			return fmt.Errorf("TTL not matching: %d : %d", node.TTL, expectedNode.TTL)
		}

		return nil
	}
}
