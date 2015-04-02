package heartbeater

import (
	"errors"
	"os"
	"time"

	"github.com/cloudfoundry-incubator/consuladapter"
	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"
)

var (
	ErrLockFailed       = errors.New("failed to compare and swap")
	ErrStoreUnavailable = errors.New("failed to connect to store")
	ErrLockLost         = errors.New("lock lost")
)

type Heartbeater struct {
	consul *consuladapter.Adapter
	key    string
	value  []byte
	ttl    time.Duration

	clock         clock.Clock
	retryInterval time.Duration

	logger lager.Logger
}

func New(
	consul *consuladapter.Adapter,
	heartbeatKey string,
	heartbeatValue []byte,
	ttl time.Duration,
	clock clock.Clock,
	retryInterval time.Duration,
	logger lager.Logger,
) Heartbeater {
	return Heartbeater{
		consul: consul,
		key:    heartbeatKey,
		value:  heartbeatValue,
		ttl:    ttl,

		clock:         clock,
		retryInterval: retryInterval,

		logger: logger,
	}
}

func (h Heartbeater) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	logger := h.logger.Session("heartbeat", lager.Data{"key": h.key, "value": string(h.value)})
	logger.Info("starting")
	defer logger.Info("done")

	cancelChan := make(chan struct{})
	errChan := make(chan error, 1)
	lostChanChan := make(chan (<-chan struct{}))
	acquire := func() {
		logger.Info("acquiring-lock")
		lostChan, err := h.consul.AcquireAndMaintainLock(h.key, h.value, h.ttl, cancelChan)
		if err != nil {
			logger.Info("received-error", lager.Data{"error": err.Error()})
			errChan <- err
		} else {
			lostChanChan <- lostChan
		}
	}

	var lostChan <-chan struct{}
	var c <-chan time.Time

	go acquire()

	for {
		select {
		case sig := <-signals:
			logger.Info("shutting-down", lager.Data{"received-signal": sig})

			close(cancelChan)

			logger.Debug("releasing-lock")
			err := h.consul.ReleaseAndDeleteLock(h.key)
			if err != nil {
				logger.Debug("failed-releasing-lock", lager.Data{"error-message": err.Error()})
			} else {
				logger.Debug("succeeded-releasing-lock")
			}

			return nil
		case <-lostChan:
			logger.Info("lost-lock")

			lostChan = nil
			return ErrLockLost
		case lostChan = <-lostChanChan:
			logger.Info("succeeded-acquiring-lock")

			if ready != nil {
				close(ready)
				logger.Info("started")
				ready = nil
			}
		case err := <-errChan:
			logger.Error("failed-acquiring-lock", err)

			c = h.clock.NewTimer(h.retryInterval).C()
		case <-c:
			logger.Info("retrying-acquiring-lock")

			c = nil
			go acquire()
		}
	}
}
