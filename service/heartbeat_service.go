package service

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs"
	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/storeadapter"
	"path"
	"time"
)

// StopHandler implements the standard termination policy for all services.
type heart struct {
	serviceType  string
	serviceId    string
	serviceValue string
	store        storeadapter.StoreAdapter
	interval     time.Duration
	logger       *gosteno.Logger
	presence     bbs.Presence
	stopChan     chan struct{}
	shutdown     chan struct{}
}

func NewHeartbeatService(store storeadapter.StoreAdapter, logger *gosteno.Logger, interval time.Duration, serviceType, serviceId, serviceValue string) Service {
	key := serviceSchemaPath(serviceType, serviceId)

	return &heart{
		serviceId:    serviceId,
		serviceType:  serviceType,
		serviceValue: serviceValue,
		store:        store,
		interval:     interval,
		logger:       logger,
		stopChan:     make(chan struct{}),
		shutdown:     make(chan struct{}),
		presence:     bbs.NewPresence(store, key, []byte(serviceValue)),
	}
}

func (h *heart) Start(onPrematureStop func()) error {
	statusChan, err := h.presence.Maintain(h.interval)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-h.stopChan:
				// stopped by external
				return
			case status, ok := <-statusChan:
				if !ok {
					// duplicate presense
					onPrematureStop()
					return
				}

				if status {
					h.logger.Infof("maintaining-presence.started.%s.%s", h.serviceType, h.serviceId)
				} else {
					// heartbeat failure
					h.logger.Errorf("maintaining-presence.failed.%s.%s", h.serviceType, h.serviceId)
				}
			}
		}
	}()

	return nil
}

func (h *heart) Stop() error {
	h.stopChan <- struct{}{}
	h.presence.Remove()
	h.shutdown <- struct{}{}
	return nil
}

func (h *heart) AwaitShutdown() {
	<-h.shutdown
}

func serviceSchemaPath(serviceType, serviceId string) string {
	return path.Join(bbs.SchemaRoot, serviceType, serviceId)
}
