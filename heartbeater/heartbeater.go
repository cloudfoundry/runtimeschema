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
	client storeadapter.StoreAdapter
	key    string
	value  string

	keyCreateRetryInterval time.Duration
	keyHeartbeatInterval   time.Duration
	keyTTL                 uint64
	clock                  clock.Clock

	logger lager.Logger
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
		client: etcdClient,
		key:    heartbeatKey,
		value:  heartbeatValue,

		keyCreateRetryInterval: heartbeatInterval,
		keyHeartbeatInterval:   heartbeatInterval,
		keyTTL:                 uint64(math.Ceil((heartbeatInterval * 2).Seconds())),
		clock:                  clock,

		logger: logger,
	}
}

func (h Heartbeater) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	logger := h.logger.Session("heartbeat", lager.Data{"key": h.key, "value": h.value})

	node := storeadapter.StoreNode{
		Key:   h.key,
		Value: []byte(h.value),
		TTL:   h.keyTTL,
	}

	if h.acquireHeartbeat(logger, node, signals) {
		close(ready)

		return h.maintainHeartbeat(logger, node, signals)
	}

	return nil
}

func (h Heartbeater) acquireHeartbeat(logger lager.Logger, node storeadapter.StoreNode, signals <-chan os.Signal) bool {
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

			intervalTimer.Reset(h.keyCreateRetryInterval)
		}
	}

	logger.Info("started")
	return true
}

func (h Heartbeater) maintainHeartbeat(logger lager.Logger, node storeadapter.StoreNode, signals <-chan os.Signal) error {
	var connectionTimer clock.Timer
	var connectionTimeout <-chan time.Time

	intervalTimer := h.clock.NewTimer(h.keyHeartbeatInterval)
	defer intervalTimer.Stop()

	for {
		select {
		case sig := <-signals:
			logger.Info("received-shutdown-signal")
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
			logger.Debug("compare-and-swapping")
			err := h.client.CompareAndSwap(node, node)
			if err != nil {
				logger.Error("failed-compare-and-swapping", err)
			} else {
				logger.Debug("succeeded-compare-and-swapping")
			}
			switch err {
			case storeadapter.ErrorTimeout:
				if connectionTimeout == nil {
					connectionTimer = h.clock.NewTimer(time.Duration(h.keyTTL) * time.Second)
					connectionTimeout = connectionTimer.C()
				}
			case storeadapter.ErrorKeyNotFound:
				logger.Debug("re-creating-node")
				err = h.client.Create(node)
				if err != nil {
					logger.Error("failed-re-creating-node", err)
				} else {
					logger.Debug("succeeded-re-creating-node")
				}
				if err != nil && connectionTimeout == nil {
					connectionTimer = h.clock.NewTimer(time.Duration(h.keyTTL) * time.Second)
					connectionTimeout = connectionTimer.C()
				}
			case nil:
				if connectionTimeout != nil {
					connectionTimer.Stop()
					connectionTimeout = nil
				}
			default:
				return ErrLockFailed
			}
			intervalTimer.Reset(h.keyHeartbeatInterval)
		}
	}
}
