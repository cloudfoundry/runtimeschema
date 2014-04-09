package service

import (
	"github.com/cloudfoundry/gosteno"
	"os"
	"os/signal"
	"syscall"
)

// StopHandler implements the standard termination policy for all services.
type interrupter struct {
	stopSignal chan os.Signal
	logger     *gosteno.Logger
	shutdown   chan struct{}
}

func NewInterrupterService(logger *gosteno.Logger) Service {
	return &interrupter{
		stopSignal: make(chan os.Signal, 1),
		logger:     logger,
		shutdown:   make(chan struct{}),
	}
}

func (i *interrupter) Start(onPrematureStop func()) error {
	signal.Notify(i.stopSignal, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig, ok := <-i.stopSignal
		if ok {
			i.logger.Infof("interrupter.stop_signal.recieved.%s", sig)
			i.Stop()
			onPrematureStop()
		}
	}()

	return nil
}

func (i *interrupter) Stop() error {
	signal.Stop(i.stopSignal)
	close(i.stopSignal)
	i.shutdown <- struct{}{}
	return nil
}

func (i *interrupter) AwaitShutdown() {
	<-i.shutdown
}
