package heartbeater

import (
	"errors"
	"math"
	"os"
	"time"

	"github.com/cloudfoundry/gunk/timeprovider"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

var (
	ErrLockFailed       = errors.New("failed to compare and swap")
	ErrStoreUnavailable = errors.New("failed to connect to etcd")
)

type Heartbeater struct {
	Client   storeadapter.StoreAdapter
	Key      string
	Value    string
	Interval time.Duration
	Logger   lager.Logger

	timeProvider timeprovider.TimeProvider
}

func New(
	etcdClient storeadapter.StoreAdapter,
	timeProvider timeprovider.TimeProvider,
	heartbeatKey string,
	heartbeatValue string,
	heartbeatInterval time.Duration,
	logger lager.Logger,
) Heartbeater {
	return Heartbeater{
		Client:       etcdClient,
		timeProvider: timeProvider,
		Key:          heartbeatKey,
		Value:        heartbeatValue,
		Interval:     heartbeatInterval,
		Logger:       logger,
	}
}

func (h Heartbeater) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	logger := h.Logger.Session("heartbeat", lager.Data{"key": h.Key, "value": h.Value})

	ttl := uint64(math.Ceil((h.Interval * 2).Seconds()))

	node := storeadapter.StoreNode{
		Key:   h.Key,
		Value: []byte(h.Value),
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

	err := h.Client.CompareAndSwap(node, node)
	if err != nil {
		var stopWatch chan<- bool
		var watchEvents <-chan storeadapter.WatchEvent
		var watchErrors <-chan error

		watchEvents, stopWatch, watchErrors = h.Client.Watch(h.Key)
		defer close(stopWatch)

		intervalTimer := h.timeProvider.NewTimer(0)
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
				watchEvents, stopWatch, watchErrors = h.Client.Watch(h.Key)
				continue WATCH
			case <-signals:
				return false
			}

			err := h.Client.Create(node)
			if err == nil {
				logger.Info("created-node")
				break
			}

			intervalTimer.Reset(h.Interval)
		}
	}

	logger.Info("started")
	return true
}

func (h Heartbeater) maintainHeartbeat(logger lager.Logger, node storeadapter.StoreNode, ttl uint64, signals <-chan os.Signal) error {
	var connectionTimer timeprovider.Timer
	var connectionTimeout <-chan time.Time

	intervalTimer := h.timeProvider.NewTimer(h.Interval)
	defer intervalTimer.Stop()

	for {
		select {
		case sig := <-signals:
			switch sig {
			case os.Kill:
				return nil
			default:
				h.Client.CompareAndDelete(node)
				return nil
			}

		case <-connectionTimeout:
			logger.Info("connection-timed-out")
			return ErrStoreUnavailable

		case <-intervalTimer.C():
			err := h.Client.CompareAndSwap(node, node)
			switch err {
			case storeadapter.ErrorTimeout:
				logger.Error("store-timeout", err)
				if connectionTimeout == nil {
					connectionTimer = h.timeProvider.NewTimer(time.Duration(ttl) * time.Second)
					connectionTimeout = connectionTimer.C()
				}
			case storeadapter.ErrorKeyNotFound:
				err = h.Client.Create(node)
				if err != nil && connectionTimeout == nil {
					connectionTimer = h.timeProvider.NewTimer(time.Duration(ttl) * time.Second)
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
			intervalTimer.Reset(h.Interval)
		}
	}
}
