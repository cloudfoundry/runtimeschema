package heartbeater

import (
	"errors"
	"math"
	"os"
	"time"

	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"
)

var (
	ErrLockFailed       = errors.New("failed to compare and swap")
	ErrStoreUnavailable = errors.New("failed to connect to etcd")
)

type Heartbeater struct {
	client   storeadapter.StoreAdapter
	key      string
	value    string
	interval time.Duration
	logger   lager.Logger

	clock clock.Clock
}

func New(
	etcdClient storeadapter.StoreAdapter,
	clock clock.Clock,
	heartbeatKey string,
	heartbeatValue string,
	heartbeatInterval time.Duration,
	logger lager.Logger,
) Heartbeater {
	return Heartbeater{
		client:   etcdClient,
		clock:    clock,
		key:      heartbeatKey,
		value:    heartbeatValue,
		interval: heartbeatInterval,
		logger:   logger,
	}
}

func (h Heartbeater) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	logger := h.logger.Session("heartbeat", lager.Data{"key": h.key, "value": h.value})

	ttl := uint64(math.Ceil((h.interval * 2).Seconds()))

	node := storeadapter.StoreNode{
		Key:   h.key,
		Value: []byte(h.value),
		TTL:   ttl,
	}

	if h.acquireHeartbeat(logger, node, ttl, signals) {
		close(ready)

		return h.maintainHeartbeat(logger, node, ttl, signals)
	}

	return nil
}

func (h Heartbeater) acquireHeartbeat(logger lager.Logger, node storeadapter.StoreNode, ttl uint64, signals <-chan os.Signal) bool {
	logger.Info("starting")

	err := h.client.CompareAndSwap(node, node)
	if err != nil {
		var stopWatch chan<- bool
		var watchEvents <-chan storeadapter.WatchEvent
		var watchErrors <-chan error

		watchEvents, stopWatch, watchErrors = h.client.Watch(h.key)
		defer close(stopWatch)

		intervalTimer := h.clock.NewTimer(0)
		defer intervalTimer.Stop()

	WATCH:
		for {
			select {
			case event := <-watchEvents:
				if !(event.Type == storeadapter.DeleteEvent || event.Type == storeadapter.ExpireEvent) {
					continue WATCH
				}
			case <-intervalTimer.C():
			case <-watchErrors:
				watchEvents, stopWatch, watchErrors = h.client.Watch(h.key)
				continue WATCH
			case <-signals:
				return false
			}

			err := h.client.Create(node)
			if err == nil {
				logger.Info("created-node")
				break
			}

			intervalTimer.Reset(h.interval)
		}
	}

	logger.Info("started")
	return true
}

func (h Heartbeater) maintainHeartbeat(logger lager.Logger, node storeadapter.StoreNode, ttl uint64, signals <-chan os.Signal) error {
	var connectionTimer clock.Timer
	var connectionTimeout <-chan time.Time

	intervalTimer := h.clock.NewTimer(h.interval)
	defer intervalTimer.Stop()

	for {
		select {
		case sig := <-signals:
			switch sig {
			case os.Kill:
				return nil
			default:
				h.client.CompareAndDelete(node)
				return nil
			}

		case <-connectionTimeout:
			logger.Info("connection-timed-out")
			return ErrStoreUnavailable

		case <-intervalTimer.C():
			err := h.client.CompareAndSwap(node, node)
			switch err {
			case storeadapter.ErrorTimeout:
				logger.Error("store-timeout", err)
				if connectionTimeout == nil {
					connectionTimer = h.clock.NewTimer(time.Duration(ttl) * time.Second)
					connectionTimeout = connectionTimer.C()
				}
			case storeadapter.ErrorKeyNotFound:
				err = h.client.Create(node)
				if err != nil && connectionTimeout == nil {
					connectionTimer = h.clock.NewTimer(time.Duration(ttl) * time.Second)
					connectionTimeout = connectionTimer.C()
				}
			case nil:
				if connectionTimeout != nil {
					connectionTimer.Stop()
					connectionTimeout = nil
				}
			default:
				logger.Error("compare-and-swap-failed", err)
				return ErrLockFailed
			}
			intervalTimer.Reset(h.interval)
		}
	}
}
