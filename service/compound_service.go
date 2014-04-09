package service

type compoundService struct {
	services []Service
}

func NewCompoundService(services ...Service) Service {
	return &compoundService{
		services: services,
	}
}

func (c *compoundService) Start(onPrematureStop func()) error {
	panic("UNIMPLEMENTED METHOD")
	return nil
}

func (c *compoundService) Stop() error {
	panic("UNIMPLEMENTED METHOD")
	return nil
}

func (c *compoundService) AwaitShutdown() {
	panic("UNIMPLEMENTED METHOD")
}
