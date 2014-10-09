// This file was generated by counterfeiter
package fake_bbs

import (
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/tedsuo/ifrit"
)

type FakeRouteEmitterBBS struct {
	WatchForDesiredLRPChangesStub        func() (<-chan models.DesiredLRPChange, chan<- bool, <-chan error)
	watchForDesiredLRPChangesMutex       sync.RWMutex
	watchForDesiredLRPChangesArgsForCall []struct{}
	watchForDesiredLRPChangesReturns     struct {
		result1 <-chan models.DesiredLRPChange
		result2 chan<- bool
		result3 <-chan error
	}
	WatchForActualLRPChangesStub        func() (<-chan models.ActualLRPChange, chan<- bool, <-chan error)
	watchForActualLRPChangesMutex       sync.RWMutex
	watchForActualLRPChangesArgsForCall []struct{}
	watchForActualLRPChangesReturns     struct {
		result1 <-chan models.ActualLRPChange
		result2 chan<- bool
		result3 <-chan error
	}
	GetAllDesiredLRPsStub        func() ([]models.DesiredLRP, error)
	getAllDesiredLRPsMutex       sync.RWMutex
	getAllDesiredLRPsArgsForCall []struct{}
	getAllDesiredLRPsReturns     struct {
		result1 []models.DesiredLRP
		result2 error
	}
	GetRunningActualLRPsStub        func() ([]models.ActualLRP, error)
	getRunningActualLRPsMutex       sync.RWMutex
	getRunningActualLRPsArgsForCall []struct{}
	getRunningActualLRPsReturns     struct {
		result1 []models.ActualLRP
		result2 error
	}
	GetDesiredLRPByProcessGuidStub        func(processGuid string) (models.DesiredLRP, error)
	getDesiredLRPByProcessGuidMutex       sync.RWMutex
	getDesiredLRPByProcessGuidArgsForCall []struct {
		processGuid string
	}
	getDesiredLRPByProcessGuidReturns struct {
		result1 models.DesiredLRP
		result2 error
	}
	GetRunningActualLRPsByProcessGuidStub        func(processGuid string) ([]models.ActualLRP, error)
	getRunningActualLRPsByProcessGuidMutex       sync.RWMutex
	getRunningActualLRPsByProcessGuidArgsForCall []struct {
		processGuid string
	}
	getRunningActualLRPsByProcessGuidReturns struct {
		result1 []models.ActualLRP
		result2 error
	}
	NewRouteEmitterLockStub        func(emitterID string, interval time.Duration) ifrit.Runner
	newRouteEmitterLockMutex       sync.RWMutex
	newRouteEmitterLockArgsForCall []struct {
		emitterID string
		interval  time.Duration
	}
	newRouteEmitterLockReturns struct {
		result1 ifrit.Runner
	}
}

func (fake *FakeRouteEmitterBBS) WatchForDesiredLRPChanges() (<-chan models.DesiredLRPChange, chan<- bool, <-chan error) {
	fake.watchForDesiredLRPChangesMutex.Lock()
	fake.watchForDesiredLRPChangesArgsForCall = append(fake.watchForDesiredLRPChangesArgsForCall, struct{}{})
	fake.watchForDesiredLRPChangesMutex.Unlock()
	if fake.WatchForDesiredLRPChangesStub != nil {
		return fake.WatchForDesiredLRPChangesStub()
	} else {
		return fake.watchForDesiredLRPChangesReturns.result1, fake.watchForDesiredLRPChangesReturns.result2, fake.watchForDesiredLRPChangesReturns.result3
	}
}

func (fake *FakeRouteEmitterBBS) WatchForDesiredLRPChangesCallCount() int {
	fake.watchForDesiredLRPChangesMutex.RLock()
	defer fake.watchForDesiredLRPChangesMutex.RUnlock()
	return len(fake.watchForDesiredLRPChangesArgsForCall)
}

func (fake *FakeRouteEmitterBBS) WatchForDesiredLRPChangesReturns(result1 <-chan models.DesiredLRPChange, result2 chan<- bool, result3 <-chan error) {
	fake.WatchForDesiredLRPChangesStub = nil
	fake.watchForDesiredLRPChangesReturns = struct {
		result1 <-chan models.DesiredLRPChange
		result2 chan<- bool
		result3 <-chan error
	}{result1, result2, result3}
}

func (fake *FakeRouteEmitterBBS) WatchForActualLRPChanges() (<-chan models.ActualLRPChange, chan<- bool, <-chan error) {
	fake.watchForActualLRPChangesMutex.Lock()
	fake.watchForActualLRPChangesArgsForCall = append(fake.watchForActualLRPChangesArgsForCall, struct{}{})
	fake.watchForActualLRPChangesMutex.Unlock()
	if fake.WatchForActualLRPChangesStub != nil {
		return fake.WatchForActualLRPChangesStub()
	} else {
		return fake.watchForActualLRPChangesReturns.result1, fake.watchForActualLRPChangesReturns.result2, fake.watchForActualLRPChangesReturns.result3
	}
}

func (fake *FakeRouteEmitterBBS) WatchForActualLRPChangesCallCount() int {
	fake.watchForActualLRPChangesMutex.RLock()
	defer fake.watchForActualLRPChangesMutex.RUnlock()
	return len(fake.watchForActualLRPChangesArgsForCall)
}

func (fake *FakeRouteEmitterBBS) WatchForActualLRPChangesReturns(result1 <-chan models.ActualLRPChange, result2 chan<- bool, result3 <-chan error) {
	fake.WatchForActualLRPChangesStub = nil
	fake.watchForActualLRPChangesReturns = struct {
		result1 <-chan models.ActualLRPChange
		result2 chan<- bool
		result3 <-chan error
	}{result1, result2, result3}
}

func (fake *FakeRouteEmitterBBS) GetAllDesiredLRPs() ([]models.DesiredLRP, error) {
	fake.getAllDesiredLRPsMutex.Lock()
	fake.getAllDesiredLRPsArgsForCall = append(fake.getAllDesiredLRPsArgsForCall, struct{}{})
	fake.getAllDesiredLRPsMutex.Unlock()
	if fake.GetAllDesiredLRPsStub != nil {
		return fake.GetAllDesiredLRPsStub()
	} else {
		return fake.getAllDesiredLRPsReturns.result1, fake.getAllDesiredLRPsReturns.result2
	}
}

func (fake *FakeRouteEmitterBBS) GetAllDesiredLRPsCallCount() int {
	fake.getAllDesiredLRPsMutex.RLock()
	defer fake.getAllDesiredLRPsMutex.RUnlock()
	return len(fake.getAllDesiredLRPsArgsForCall)
}

func (fake *FakeRouteEmitterBBS) GetAllDesiredLRPsReturns(result1 []models.DesiredLRP, result2 error) {
	fake.GetAllDesiredLRPsStub = nil
	fake.getAllDesiredLRPsReturns = struct {
		result1 []models.DesiredLRP
		result2 error
	}{result1, result2}
}

func (fake *FakeRouteEmitterBBS) GetRunningActualLRPs() ([]models.ActualLRP, error) {
	fake.getRunningActualLRPsMutex.Lock()
	fake.getRunningActualLRPsArgsForCall = append(fake.getRunningActualLRPsArgsForCall, struct{}{})
	fake.getRunningActualLRPsMutex.Unlock()
	if fake.GetRunningActualLRPsStub != nil {
		return fake.GetRunningActualLRPsStub()
	} else {
		return fake.getRunningActualLRPsReturns.result1, fake.getRunningActualLRPsReturns.result2
	}
}

func (fake *FakeRouteEmitterBBS) GetRunningActualLRPsCallCount() int {
	fake.getRunningActualLRPsMutex.RLock()
	defer fake.getRunningActualLRPsMutex.RUnlock()
	return len(fake.getRunningActualLRPsArgsForCall)
}

func (fake *FakeRouteEmitterBBS) GetRunningActualLRPsReturns(result1 []models.ActualLRP, result2 error) {
	fake.GetRunningActualLRPsStub = nil
	fake.getRunningActualLRPsReturns = struct {
		result1 []models.ActualLRP
		result2 error
	}{result1, result2}
}

func (fake *FakeRouteEmitterBBS) GetDesiredLRPByProcessGuid(processGuid string) (models.DesiredLRP, error) {
	fake.getDesiredLRPByProcessGuidMutex.Lock()
	fake.getDesiredLRPByProcessGuidArgsForCall = append(fake.getDesiredLRPByProcessGuidArgsForCall, struct {
		processGuid string
	}{processGuid})
	fake.getDesiredLRPByProcessGuidMutex.Unlock()
	if fake.GetDesiredLRPByProcessGuidStub != nil {
		return fake.GetDesiredLRPByProcessGuidStub(processGuid)
	} else {
		return fake.getDesiredLRPByProcessGuidReturns.result1, fake.getDesiredLRPByProcessGuidReturns.result2
	}
}

func (fake *FakeRouteEmitterBBS) GetDesiredLRPByProcessGuidCallCount() int {
	fake.getDesiredLRPByProcessGuidMutex.RLock()
	defer fake.getDesiredLRPByProcessGuidMutex.RUnlock()
	return len(fake.getDesiredLRPByProcessGuidArgsForCall)
}

func (fake *FakeRouteEmitterBBS) GetDesiredLRPByProcessGuidArgsForCall(i int) string {
	fake.getDesiredLRPByProcessGuidMutex.RLock()
	defer fake.getDesiredLRPByProcessGuidMutex.RUnlock()
	return fake.getDesiredLRPByProcessGuidArgsForCall[i].processGuid
}

func (fake *FakeRouteEmitterBBS) GetDesiredLRPByProcessGuidReturns(result1 models.DesiredLRP, result2 error) {
	fake.GetDesiredLRPByProcessGuidStub = nil
	fake.getDesiredLRPByProcessGuidReturns = struct {
		result1 models.DesiredLRP
		result2 error
	}{result1, result2}
}

func (fake *FakeRouteEmitterBBS) GetRunningActualLRPsByProcessGuid(processGuid string) ([]models.ActualLRP, error) {
	fake.getRunningActualLRPsByProcessGuidMutex.Lock()
	fake.getRunningActualLRPsByProcessGuidArgsForCall = append(fake.getRunningActualLRPsByProcessGuidArgsForCall, struct {
		processGuid string
	}{processGuid})
	fake.getRunningActualLRPsByProcessGuidMutex.Unlock()
	if fake.GetRunningActualLRPsByProcessGuidStub != nil {
		return fake.GetRunningActualLRPsByProcessGuidStub(processGuid)
	} else {
		return fake.getRunningActualLRPsByProcessGuidReturns.result1, fake.getRunningActualLRPsByProcessGuidReturns.result2
	}
}

func (fake *FakeRouteEmitterBBS) GetRunningActualLRPsByProcessGuidCallCount() int {
	fake.getRunningActualLRPsByProcessGuidMutex.RLock()
	defer fake.getRunningActualLRPsByProcessGuidMutex.RUnlock()
	return len(fake.getRunningActualLRPsByProcessGuidArgsForCall)
}

func (fake *FakeRouteEmitterBBS) GetRunningActualLRPsByProcessGuidArgsForCall(i int) string {
	fake.getRunningActualLRPsByProcessGuidMutex.RLock()
	defer fake.getRunningActualLRPsByProcessGuidMutex.RUnlock()
	return fake.getRunningActualLRPsByProcessGuidArgsForCall[i].processGuid
}

func (fake *FakeRouteEmitterBBS) GetRunningActualLRPsByProcessGuidReturns(result1 []models.ActualLRP, result2 error) {
	fake.GetRunningActualLRPsByProcessGuidStub = nil
	fake.getRunningActualLRPsByProcessGuidReturns = struct {
		result1 []models.ActualLRP
		result2 error
	}{result1, result2}
}

func (fake *FakeRouteEmitterBBS) NewRouteEmitterLock(emitterID string, interval time.Duration) ifrit.Runner {
	fake.newRouteEmitterLockMutex.Lock()
	fake.newRouteEmitterLockArgsForCall = append(fake.newRouteEmitterLockArgsForCall, struct {
		emitterID string
		interval  time.Duration
	}{emitterID, interval})
	fake.newRouteEmitterLockMutex.Unlock()
	if fake.NewRouteEmitterLockStub != nil {
		return fake.NewRouteEmitterLockStub(emitterID, interval)
	} else {
		return fake.newRouteEmitterLockReturns.result1
	}
}

func (fake *FakeRouteEmitterBBS) NewRouteEmitterLockCallCount() int {
	fake.newRouteEmitterLockMutex.RLock()
	defer fake.newRouteEmitterLockMutex.RUnlock()
	return len(fake.newRouteEmitterLockArgsForCall)
}

func (fake *FakeRouteEmitterBBS) NewRouteEmitterLockArgsForCall(i int) (string, time.Duration) {
	fake.newRouteEmitterLockMutex.RLock()
	defer fake.newRouteEmitterLockMutex.RUnlock()
	return fake.newRouteEmitterLockArgsForCall[i].emitterID, fake.newRouteEmitterLockArgsForCall[i].interval
}

func (fake *FakeRouteEmitterBBS) NewRouteEmitterLockReturns(result1 ifrit.Runner) {
	fake.NewRouteEmitterLockStub = nil
	fake.newRouteEmitterLockReturns = struct {
		result1 ifrit.Runner
	}{result1}
}

var _ bbs.RouteEmitterBBS = new(FakeRouteEmitterBBS)
