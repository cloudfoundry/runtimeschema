package maintainer

import (
	"os"
	"time"

	"github.com/cloudfoundry-incubator/consuladapter"
	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"
)

type Presence struct {
	consul *consuladapter.Session
	key    string
	value  []byte

	clock         clock.Clock
	retryInterval time.Duration

	logger lager.Logger
}

func NewPresence(
	consul *consuladapter.Session,
	lockKey string,
	lockValue []byte,
	clock clock.Clock,
	retryInterval time.Duration,
	logger lager.Logger,
) Presence {
	return Presence{
		consul: consul,
		key:    lockKey,
		value:  lockValue,

		clock:         clock,
		retryInterval: retryInterval,

		logger: logger,
	}
}

func (p Presence) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	logger := p.logger.Session("presence", lager.Data{"key": p.key, "value": string(p.value)})
	logger.Info("starting")

	defer func() {
		p.consul.Destroy()
		logger.Info("done")
	}()

	type presenceResult struct {
		presenceLost <-chan string
		err          error
	}

	presenceCh := make(chan presenceResult, 1)
	setPresence := func(session *consuladapter.Session) {
		logger.Info("setting-presence")
		presenceLost, err := session.SetPresence(p.key, p.value)
		presenceCh <- presenceResult{presenceLost, err}
	}

	var c <-chan time.Time
	var presenceLost <-chan string

	go setPresence(p.consul)

	close(ready)
	logger.Info("started")

	for {
		select {
		case sig := <-signals:
			logger.Info("shutting-down", lager.Data{"received-signal": sig})

			logger.Debug("removing-presence")
			return nil
		case err := <-p.consul.Err():
			var data lager.Data
			if err != nil {
				data = lager.Data{"err": err.Error()}
			}
			logger.Info("consul-error", data)

			presenceLost = nil
			c = p.clock.NewTimer(p.retryInterval).C()
		case result := <-presenceCh:
			if result.err == nil {
				logger.Info("set-presence-succeeded")
				presenceLost = result.presenceLost
			} else {
				logger.Error("set-presence-failed", result.err)
				c = p.clock.NewTimer(p.retryInterval).C()
			}
		case <-presenceLost:
			logger.Info("presence-lost")
			presenceLost = nil

			c = p.clock.NewTimer(p.retryInterval).C()
		case <-c:
			logger.Info("retrying-set-presence")

			presenceLost = nil
			newSession, err := p.consul.Recreate()
			if err != nil {
				c = p.clock.NewTimer(p.retryInterval).C()
			} else {
				p.consul = newSession
				c = nil
				go setPresence(newSession)
			}
		}
	}
}
