package service

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/pivotal-golang/lager"
)

// StopHandler implements the standard termination policy for all services.
type interrupter struct {
	stopSignal chan os.Signal
	logger     lager.Logger
	shutdown   chan struct{}
}

func NewInterrupterService(logger lager.Logger) Service {
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
			i.logger.Info("interrupter.received-stop-signal", lager.Data{
				"signal": sig.String(),
			})

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
