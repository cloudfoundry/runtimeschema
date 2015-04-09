package maintainer

import (
	"errors"
	"os"
	"time"

	"github.com/cloudfoundry-incubator/consuladapter"
	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"
)

var (
	ErrLockLost = errors.New("lock lost")
)

type Lock struct {
	consul *consuladapter.Session
	key    string
	value  []byte

	clock         clock.Clock
	retryInterval time.Duration

	logger lager.Logger
}

func NewLock(
	consul *consuladapter.Session,
	lockKey string,
	lockValue []byte,
	clock clock.Clock,
	retryInterval time.Duration,
	logger lager.Logger,
) Lock {
	return Lock{
		consul: consul,
		key:    lockKey,
		value:  lockValue,

		clock:         clock,
		retryInterval: retryInterval,

		logger: logger,
	}
}

func (l Lock) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	logger := l.logger.Session("lock", lager.Data{"key": l.key, "value": string(l.value)})
	logger.Info("starting")

	defer func() {
		l.consul.Destroy()
		logger.Info("done")
	}()

	errChan := make(chan error, 1)

	acquire := func(session *consuladapter.Session) {
		logger.Info("acquiring-lock")
		errChan <- session.AcquireLock(l.key, l.value)
	}

	var c <-chan time.Time

	go acquire(l.consul)

	for {
		select {
		case sig := <-signals:
			logger.Info("shutting-down", lager.Data{"received-signal": sig})

			logger.Debug("releasing-lock")
			return nil
		case err := <-l.consul.Err():
			if err == consuladapter.LostLockError(l.key) {
				logger.Info("lost-lock")
				return ErrLockLost
			}

		case err := <-errChan:
			if err == nil {
				logger.Info("acquire-lock-succeeded")

				if ready != nil {
					close(ready)
					logger.Info("started")
					ready = nil
				} else {
					logger.Info("lost-lock")
					return ErrLockLost
				}
			} else {
				logger.Error("acquire-lock-failed", err)
				c = l.clock.NewTimer(l.retryInterval).C()
			}
		case <-c:
			logger.Info("retrying-acquiring-lock")

			newSession, err := l.consul.Recreate()
			if err != nil {
				c = l.clock.NewTimer(l.retryInterval).C()
			} else {
				l.consul = newSession
				c = nil
				go acquire(newSession)
			}
		}
	}
}
