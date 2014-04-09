package service

// The Service interface contains all the behavior a component must implement in
// order to be considered a proper service.
type Service interface {
	Start(onPrematureStop func()) error
	Stop() error
	AwaitShutdown()
}
