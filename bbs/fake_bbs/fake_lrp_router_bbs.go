package fake_bbs

import "github.com/cloudfoundry-incubator/runtime-schema/models"

type FakeLRPRouterBBS struct {
	desiredLRPChan     chan models.DesiredLRP
	desiredLRPStopChan chan bool
	desiredLRPErrChan  chan error

	actualLRPChan     chan models.LRP
	actualLRPStopChan chan bool
	actualLRPErrChan  chan error

	AllDesiredLRPs []models.DesiredLRP
	AllActualLRPs  []models.LRP

	DesiredLRP models.DesiredLRP
	ActualLRPs []models.LRP

	WhenGettingAllActualLongRunningProcesses func() ([]models.LRP, error)
	WhenGettingAllDesiredLongRunningProcesses func() ([]models.DesiredLRP, error)
}

func NewFakeLRPRouterBBS() *FakeLRPRouterBBS {
	return &FakeLRPRouterBBS{
		desiredLRPChan:     make(chan models.DesiredLRP, 1),
		desiredLRPStopChan: make(chan bool),
		desiredLRPErrChan:  make(chan error),
		actualLRPChan:      make(chan models.LRP, 1),
		actualLRPStopChan:  make(chan bool),
		actualLRPErrChan:   make(chan error),
	}
}

func (fakeBBS *FakeLRPRouterBBS) WatchForDesiredLongRunningProcesses() (<-chan models.DesiredLRP, chan<- bool, <-chan error) {
	return fakeBBS.desiredLRPChan, fakeBBS.desiredLRPStopChan, fakeBBS.desiredLRPErrChan
}

func (fakeBBS *FakeLRPRouterBBS) WatchForActualLongRunningProcesses() (<-chan models.LRP, chan<- bool, <-chan error) {
	return fakeBBS.actualLRPChan, fakeBBS.actualLRPStopChan, fakeBBS.actualLRPErrChan
}

func (fakeBBS *FakeLRPRouterBBS) GetAllDesiredLongRunningProcesses() ([]models.DesiredLRP, error) {
	if fakeBBS.WhenGettingAllDesiredLongRunningProcesses != nil {
		return fakeBBS.WhenGettingAllDesiredLongRunningProcesses()
	}
	return fakeBBS.AllDesiredLRPs, nil
}

func (fakeBBS *FakeLRPRouterBBS) GetAllActualLongRunningProcesses() ([]models.LRP, error) {
	if fakeBBS.WhenGettingAllActualLongRunningProcesses != nil {
		return fakeBBS.WhenGettingAllActualLongRunningProcesses()
	}
	return fakeBBS.AllActualLRPs, nil
}

func (fakeBBS *FakeLRPRouterBBS) GetDesiredLRP(processGuid string) (models.DesiredLRP, error) {
	return fakeBBS.DesiredLRP, nil
}

func (fakeBBS *FakeLRPRouterBBS) GetActualLRPs(processGuid string) ([]models.LRP, error) {
	return fakeBBS.ActualLRPs, nil
}
