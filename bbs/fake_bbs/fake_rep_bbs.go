// This file was generated by counterfeiter
package fake_bbs

import (
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/ifrit"
)

type FakeRepBBS struct {
	NewCellHeartbeatStub        func(cellPresence models.CellPresence, interval time.Duration) ifrit.Runner
	newCellHeartbeatMutex       sync.RWMutex
	newCellHeartbeatArgsForCall []struct {
		cellPresence models.CellPresence
		interval     time.Duration
	}
	newCellHeartbeatReturns struct {
		result1 ifrit.Runner
	}
	StartTaskStub        func(logger lager.Logger, taskGuid string, cellID string) (bool, error)
	startTaskMutex       sync.RWMutex
	startTaskArgsForCall []struct {
		logger   lager.Logger
		taskGuid string
		cellID   string
	}
	startTaskReturns struct {
		result1 bool
		result2 error
	}
	TaskByGuidStub        func(taskGuid string) (models.Task, error)
	taskByGuidMutex       sync.RWMutex
	taskByGuidArgsForCall []struct {
		taskGuid string
	}
	taskByGuidReturns struct {
		result1 models.Task
		result2 error
	}
	TasksByCellIDStub        func(logger lager.Logger, cellID string) ([]models.Task, error)
	tasksByCellIDMutex       sync.RWMutex
	tasksByCellIDArgsForCall []struct {
		logger lager.Logger
		cellID string
	}
	tasksByCellIDReturns struct {
		result1 []models.Task
		result2 error
	}
	FailTaskStub        func(logger lager.Logger, taskGuid string, failureReason string) error
	failTaskMutex       sync.RWMutex
	failTaskArgsForCall []struct {
		logger        lager.Logger
		taskGuid      string
		failureReason string
	}
	failTaskReturns struct {
		result1 error
	}
	CompleteTaskStub        func(logger lager.Logger, taskGuid string, cellID string, failed bool, failureReason string, result string) error
	completeTaskMutex       sync.RWMutex
	completeTaskArgsForCall []struct {
		logger        lager.Logger
		taskGuid      string
		cellID        string
		failed        bool
		failureReason string
		result        string
	}
	completeTaskReturns struct {
		result1 error
	}
	ActualLRPsByCellIDStub        func(cellID string) ([]models.ActualLRP, error)
	actualLRPsByCellIDMutex       sync.RWMutex
	actualLRPsByCellIDArgsForCall []struct {
		cellID string
	}
	actualLRPsByCellIDReturns struct {
		result1 []models.ActualLRP
		result2 error
	}
	ClaimActualLRPStub        func(models.ActualLRPKey, models.ActualLRPContainerKey, lager.Logger) error
	claimActualLRPMutex       sync.RWMutex
	claimActualLRPArgsForCall []struct {
		arg1 models.ActualLRPKey
		arg2 models.ActualLRPContainerKey
		arg3 lager.Logger
	}
	claimActualLRPReturns struct {
		result1 error
	}
	StartActualLRPStub        func(models.ActualLRPKey, models.ActualLRPContainerKey, models.ActualLRPNetInfo, lager.Logger) error
	startActualLRPMutex       sync.RWMutex
	startActualLRPArgsForCall []struct {
		arg1 models.ActualLRPKey
		arg2 models.ActualLRPContainerKey
		arg3 models.ActualLRPNetInfo
		arg4 lager.Logger
	}
	startActualLRPReturns struct {
		result1 error
	}
	RemoveActualLRPStub        func(models.ActualLRPKey, models.ActualLRPContainerKey, lager.Logger) error
	removeActualLRPMutex       sync.RWMutex
	removeActualLRPArgsForCall []struct {
		arg1 models.ActualLRPKey
		arg2 models.ActualLRPContainerKey
		arg3 lager.Logger
	}
	removeActualLRPReturns struct {
		result1 error
	}
	CrashActualLRPStub        func(key models.ActualLRPKey, containerKey models.ActualLRPContainerKey, logger lager.Logger) error
	crashActualLRPMutex       sync.RWMutex
	crashActualLRPArgsForCall []struct {
		key          models.ActualLRPKey
		containerKey models.ActualLRPContainerKey
		logger       lager.Logger
	}
	crashActualLRPReturns struct {
		result1 error
	}
	FailLRPStub        func(lager.Logger, models.ActualLRPKey, string) error
	failLRPMutex       sync.RWMutex
	failLRPArgsForCall []struct {
		arg1 lager.Logger
		arg2 models.ActualLRPKey
		arg3 string
	}
	failLRPReturns struct {
		result1 error
	}
	EvacuateClaimedActualLRPStub        func(lager.Logger, models.ActualLRPKey, models.ActualLRPContainerKey) error
	evacuateClaimedActualLRPMutex       sync.RWMutex
	evacuateClaimedActualLRPArgsForCall []struct {
		arg1 lager.Logger
		arg2 models.ActualLRPKey
		arg3 models.ActualLRPContainerKey
	}
	evacuateClaimedActualLRPReturns struct {
		result1 error
	}
	EvacuateRunningActualLRPStub        func(lager.Logger, models.ActualLRPKey, models.ActualLRPContainerKey, models.ActualLRPNetInfo, uint64) error
	evacuateRunningActualLRPMutex       sync.RWMutex
	evacuateRunningActualLRPArgsForCall []struct {
		arg1 lager.Logger
		arg2 models.ActualLRPKey
		arg3 models.ActualLRPContainerKey
		arg4 models.ActualLRPNetInfo
		arg5 uint64
	}
	evacuateRunningActualLRPReturns struct {
		result1 error
	}
	EvacuateStoppedActualLRPStub        func(lager.Logger, models.ActualLRPKey, models.ActualLRPContainerKey) error
	evacuateStoppedActualLRPMutex       sync.RWMutex
	evacuateStoppedActualLRPArgsForCall []struct {
		arg1 lager.Logger
		arg2 models.ActualLRPKey
		arg3 models.ActualLRPContainerKey
	}
	evacuateStoppedActualLRPReturns struct {
		result1 error
	}
	EvacuateCrashedActualLRPStub        func(lager.Logger, models.ActualLRPKey, models.ActualLRPContainerKey) error
	evacuateCrashedActualLRPMutex       sync.RWMutex
	evacuateCrashedActualLRPArgsForCall []struct {
		arg1 lager.Logger
		arg2 models.ActualLRPKey
		arg3 models.ActualLRPContainerKey
	}
	evacuateCrashedActualLRPReturns struct {
		result1 error
	}
}

func (fake *FakeRepBBS) NewCellHeartbeat(cellPresence models.CellPresence, interval time.Duration) ifrit.Runner {
	fake.newCellHeartbeatMutex.Lock()
	fake.newCellHeartbeatArgsForCall = append(fake.newCellHeartbeatArgsForCall, struct {
		cellPresence models.CellPresence
		interval     time.Duration
	}{cellPresence, interval})
	fake.newCellHeartbeatMutex.Unlock()
	if fake.NewCellHeartbeatStub != nil {
		return fake.NewCellHeartbeatStub(cellPresence, interval)
	} else {
		return fake.newCellHeartbeatReturns.result1
	}
}

func (fake *FakeRepBBS) NewCellHeartbeatCallCount() int {
	fake.newCellHeartbeatMutex.RLock()
	defer fake.newCellHeartbeatMutex.RUnlock()
	return len(fake.newCellHeartbeatArgsForCall)
}

func (fake *FakeRepBBS) NewCellHeartbeatArgsForCall(i int) (models.CellPresence, time.Duration) {
	fake.newCellHeartbeatMutex.RLock()
	defer fake.newCellHeartbeatMutex.RUnlock()
	return fake.newCellHeartbeatArgsForCall[i].cellPresence, fake.newCellHeartbeatArgsForCall[i].interval
}

func (fake *FakeRepBBS) NewCellHeartbeatReturns(result1 ifrit.Runner) {
	fake.NewCellHeartbeatStub = nil
	fake.newCellHeartbeatReturns = struct {
		result1 ifrit.Runner
	}{result1}
}

func (fake *FakeRepBBS) StartTask(logger lager.Logger, taskGuid string, cellID string) (bool, error) {
	fake.startTaskMutex.Lock()
	fake.startTaskArgsForCall = append(fake.startTaskArgsForCall, struct {
		logger   lager.Logger
		taskGuid string
		cellID   string
	}{logger, taskGuid, cellID})
	fake.startTaskMutex.Unlock()
	if fake.StartTaskStub != nil {
		return fake.StartTaskStub(logger, taskGuid, cellID)
	} else {
		return fake.startTaskReturns.result1, fake.startTaskReturns.result2
	}
}

func (fake *FakeRepBBS) StartTaskCallCount() int {
	fake.startTaskMutex.RLock()
	defer fake.startTaskMutex.RUnlock()
	return len(fake.startTaskArgsForCall)
}

func (fake *FakeRepBBS) StartTaskArgsForCall(i int) (lager.Logger, string, string) {
	fake.startTaskMutex.RLock()
	defer fake.startTaskMutex.RUnlock()
	return fake.startTaskArgsForCall[i].logger, fake.startTaskArgsForCall[i].taskGuid, fake.startTaskArgsForCall[i].cellID
}

func (fake *FakeRepBBS) StartTaskReturns(result1 bool, result2 error) {
	fake.StartTaskStub = nil
	fake.startTaskReturns = struct {
		result1 bool
		result2 error
	}{result1, result2}
}

func (fake *FakeRepBBS) TaskByGuid(taskGuid string) (models.Task, error) {
	fake.taskByGuidMutex.Lock()
	fake.taskByGuidArgsForCall = append(fake.taskByGuidArgsForCall, struct {
		taskGuid string
	}{taskGuid})
	fake.taskByGuidMutex.Unlock()
	if fake.TaskByGuidStub != nil {
		return fake.TaskByGuidStub(taskGuid)
	} else {
		return fake.taskByGuidReturns.result1, fake.taskByGuidReturns.result2
	}
}

func (fake *FakeRepBBS) TaskByGuidCallCount() int {
	fake.taskByGuidMutex.RLock()
	defer fake.taskByGuidMutex.RUnlock()
	return len(fake.taskByGuidArgsForCall)
}

func (fake *FakeRepBBS) TaskByGuidArgsForCall(i int) string {
	fake.taskByGuidMutex.RLock()
	defer fake.taskByGuidMutex.RUnlock()
	return fake.taskByGuidArgsForCall[i].taskGuid
}

func (fake *FakeRepBBS) TaskByGuidReturns(result1 models.Task, result2 error) {
	fake.TaskByGuidStub = nil
	fake.taskByGuidReturns = struct {
		result1 models.Task
		result2 error
	}{result1, result2}
}

func (fake *FakeRepBBS) TasksByCellID(logger lager.Logger, cellID string) ([]models.Task, error) {
	fake.tasksByCellIDMutex.Lock()
	fake.tasksByCellIDArgsForCall = append(fake.tasksByCellIDArgsForCall, struct {
		logger lager.Logger
		cellID string
	}{logger, cellID})
	fake.tasksByCellIDMutex.Unlock()
	if fake.TasksByCellIDStub != nil {
		return fake.TasksByCellIDStub(logger, cellID)
	} else {
		return fake.tasksByCellIDReturns.result1, fake.tasksByCellIDReturns.result2
	}
}

func (fake *FakeRepBBS) TasksByCellIDCallCount() int {
	fake.tasksByCellIDMutex.RLock()
	defer fake.tasksByCellIDMutex.RUnlock()
	return len(fake.tasksByCellIDArgsForCall)
}

func (fake *FakeRepBBS) TasksByCellIDArgsForCall(i int) (lager.Logger, string) {
	fake.tasksByCellIDMutex.RLock()
	defer fake.tasksByCellIDMutex.RUnlock()
	return fake.tasksByCellIDArgsForCall[i].logger, fake.tasksByCellIDArgsForCall[i].cellID
}

func (fake *FakeRepBBS) TasksByCellIDReturns(result1 []models.Task, result2 error) {
	fake.TasksByCellIDStub = nil
	fake.tasksByCellIDReturns = struct {
		result1 []models.Task
		result2 error
	}{result1, result2}
}

func (fake *FakeRepBBS) FailTask(logger lager.Logger, taskGuid string, failureReason string) error {
	fake.failTaskMutex.Lock()
	fake.failTaskArgsForCall = append(fake.failTaskArgsForCall, struct {
		logger        lager.Logger
		taskGuid      string
		failureReason string
	}{logger, taskGuid, failureReason})
	fake.failTaskMutex.Unlock()
	if fake.FailTaskStub != nil {
		return fake.FailTaskStub(logger, taskGuid, failureReason)
	} else {
		return fake.failTaskReturns.result1
	}
}

func (fake *FakeRepBBS) FailTaskCallCount() int {
	fake.failTaskMutex.RLock()
	defer fake.failTaskMutex.RUnlock()
	return len(fake.failTaskArgsForCall)
}

func (fake *FakeRepBBS) FailTaskArgsForCall(i int) (lager.Logger, string, string) {
	fake.failTaskMutex.RLock()
	defer fake.failTaskMutex.RUnlock()
	return fake.failTaskArgsForCall[i].logger, fake.failTaskArgsForCall[i].taskGuid, fake.failTaskArgsForCall[i].failureReason
}

func (fake *FakeRepBBS) FailTaskReturns(result1 error) {
	fake.FailTaskStub = nil
	fake.failTaskReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeRepBBS) CompleteTask(logger lager.Logger, taskGuid string, cellID string, failed bool, failureReason string, result string) error {
	fake.completeTaskMutex.Lock()
	fake.completeTaskArgsForCall = append(fake.completeTaskArgsForCall, struct {
		logger        lager.Logger
		taskGuid      string
		cellID        string
		failed        bool
		failureReason string
		result        string
	}{logger, taskGuid, cellID, failed, failureReason, result})
	fake.completeTaskMutex.Unlock()
	if fake.CompleteTaskStub != nil {
		return fake.CompleteTaskStub(logger, taskGuid, cellID, failed, failureReason, result)
	} else {
		return fake.completeTaskReturns.result1
	}
}

func (fake *FakeRepBBS) CompleteTaskCallCount() int {
	fake.completeTaskMutex.RLock()
	defer fake.completeTaskMutex.RUnlock()
	return len(fake.completeTaskArgsForCall)
}

func (fake *FakeRepBBS) CompleteTaskArgsForCall(i int) (lager.Logger, string, string, bool, string, string) {
	fake.completeTaskMutex.RLock()
	defer fake.completeTaskMutex.RUnlock()
	return fake.completeTaskArgsForCall[i].logger, fake.completeTaskArgsForCall[i].taskGuid, fake.completeTaskArgsForCall[i].cellID, fake.completeTaskArgsForCall[i].failed, fake.completeTaskArgsForCall[i].failureReason, fake.completeTaskArgsForCall[i].result
}

func (fake *FakeRepBBS) CompleteTaskReturns(result1 error) {
	fake.CompleteTaskStub = nil
	fake.completeTaskReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeRepBBS) ActualLRPsByCellID(cellID string) ([]models.ActualLRP, error) {
	fake.actualLRPsByCellIDMutex.Lock()
	fake.actualLRPsByCellIDArgsForCall = append(fake.actualLRPsByCellIDArgsForCall, struct {
		cellID string
	}{cellID})
	fake.actualLRPsByCellIDMutex.Unlock()
	if fake.ActualLRPsByCellIDStub != nil {
		return fake.ActualLRPsByCellIDStub(cellID)
	} else {
		return fake.actualLRPsByCellIDReturns.result1, fake.actualLRPsByCellIDReturns.result2
	}
}

func (fake *FakeRepBBS) ActualLRPsByCellIDCallCount() int {
	fake.actualLRPsByCellIDMutex.RLock()
	defer fake.actualLRPsByCellIDMutex.RUnlock()
	return len(fake.actualLRPsByCellIDArgsForCall)
}

func (fake *FakeRepBBS) ActualLRPsByCellIDArgsForCall(i int) string {
	fake.actualLRPsByCellIDMutex.RLock()
	defer fake.actualLRPsByCellIDMutex.RUnlock()
	return fake.actualLRPsByCellIDArgsForCall[i].cellID
}

func (fake *FakeRepBBS) ActualLRPsByCellIDReturns(result1 []models.ActualLRP, result2 error) {
	fake.ActualLRPsByCellIDStub = nil
	fake.actualLRPsByCellIDReturns = struct {
		result1 []models.ActualLRP
		result2 error
	}{result1, result2}
}

func (fake *FakeRepBBS) ClaimActualLRP(arg1 models.ActualLRPKey, arg2 models.ActualLRPContainerKey, arg3 lager.Logger) error {
	fake.claimActualLRPMutex.Lock()
	fake.claimActualLRPArgsForCall = append(fake.claimActualLRPArgsForCall, struct {
		arg1 models.ActualLRPKey
		arg2 models.ActualLRPContainerKey
		arg3 lager.Logger
	}{arg1, arg2, arg3})
	fake.claimActualLRPMutex.Unlock()
	if fake.ClaimActualLRPStub != nil {
		return fake.ClaimActualLRPStub(arg1, arg2, arg3)
	} else {
		return fake.claimActualLRPReturns.result1
	}
}

func (fake *FakeRepBBS) ClaimActualLRPCallCount() int {
	fake.claimActualLRPMutex.RLock()
	defer fake.claimActualLRPMutex.RUnlock()
	return len(fake.claimActualLRPArgsForCall)
}

func (fake *FakeRepBBS) ClaimActualLRPArgsForCall(i int) (models.ActualLRPKey, models.ActualLRPContainerKey, lager.Logger) {
	fake.claimActualLRPMutex.RLock()
	defer fake.claimActualLRPMutex.RUnlock()
	return fake.claimActualLRPArgsForCall[i].arg1, fake.claimActualLRPArgsForCall[i].arg2, fake.claimActualLRPArgsForCall[i].arg3
}

func (fake *FakeRepBBS) ClaimActualLRPReturns(result1 error) {
	fake.ClaimActualLRPStub = nil
	fake.claimActualLRPReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeRepBBS) StartActualLRP(arg1 models.ActualLRPKey, arg2 models.ActualLRPContainerKey, arg3 models.ActualLRPNetInfo, arg4 lager.Logger) error {
	fake.startActualLRPMutex.Lock()
	fake.startActualLRPArgsForCall = append(fake.startActualLRPArgsForCall, struct {
		arg1 models.ActualLRPKey
		arg2 models.ActualLRPContainerKey
		arg3 models.ActualLRPNetInfo
		arg4 lager.Logger
	}{arg1, arg2, arg3, arg4})
	fake.startActualLRPMutex.Unlock()
	if fake.StartActualLRPStub != nil {
		return fake.StartActualLRPStub(arg1, arg2, arg3, arg4)
	} else {
		return fake.startActualLRPReturns.result1
	}
}

func (fake *FakeRepBBS) StartActualLRPCallCount() int {
	fake.startActualLRPMutex.RLock()
	defer fake.startActualLRPMutex.RUnlock()
	return len(fake.startActualLRPArgsForCall)
}

func (fake *FakeRepBBS) StartActualLRPArgsForCall(i int) (models.ActualLRPKey, models.ActualLRPContainerKey, models.ActualLRPNetInfo, lager.Logger) {
	fake.startActualLRPMutex.RLock()
	defer fake.startActualLRPMutex.RUnlock()
	return fake.startActualLRPArgsForCall[i].arg1, fake.startActualLRPArgsForCall[i].arg2, fake.startActualLRPArgsForCall[i].arg3, fake.startActualLRPArgsForCall[i].arg4
}

func (fake *FakeRepBBS) StartActualLRPReturns(result1 error) {
	fake.StartActualLRPStub = nil
	fake.startActualLRPReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeRepBBS) RemoveActualLRP(arg1 models.ActualLRPKey, arg2 models.ActualLRPContainerKey, arg3 lager.Logger) error {
	fake.removeActualLRPMutex.Lock()
	fake.removeActualLRPArgsForCall = append(fake.removeActualLRPArgsForCall, struct {
		arg1 models.ActualLRPKey
		arg2 models.ActualLRPContainerKey
		arg3 lager.Logger
	}{arg1, arg2, arg3})
	fake.removeActualLRPMutex.Unlock()
	if fake.RemoveActualLRPStub != nil {
		return fake.RemoveActualLRPStub(arg1, arg2, arg3)
	} else {
		return fake.removeActualLRPReturns.result1
	}
}

func (fake *FakeRepBBS) RemoveActualLRPCallCount() int {
	fake.removeActualLRPMutex.RLock()
	defer fake.removeActualLRPMutex.RUnlock()
	return len(fake.removeActualLRPArgsForCall)
}

func (fake *FakeRepBBS) RemoveActualLRPArgsForCall(i int) (models.ActualLRPKey, models.ActualLRPContainerKey, lager.Logger) {
	fake.removeActualLRPMutex.RLock()
	defer fake.removeActualLRPMutex.RUnlock()
	return fake.removeActualLRPArgsForCall[i].arg1, fake.removeActualLRPArgsForCall[i].arg2, fake.removeActualLRPArgsForCall[i].arg3
}

func (fake *FakeRepBBS) RemoveActualLRPReturns(result1 error) {
	fake.RemoveActualLRPStub = nil
	fake.removeActualLRPReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeRepBBS) CrashActualLRP(key models.ActualLRPKey, containerKey models.ActualLRPContainerKey, logger lager.Logger) error {
	fake.crashActualLRPMutex.Lock()
	fake.crashActualLRPArgsForCall = append(fake.crashActualLRPArgsForCall, struct {
		key          models.ActualLRPKey
		containerKey models.ActualLRPContainerKey
		logger       lager.Logger
	}{key, containerKey, logger})
	fake.crashActualLRPMutex.Unlock()
	if fake.CrashActualLRPStub != nil {
		return fake.CrashActualLRPStub(key, containerKey, logger)
	} else {
		return fake.crashActualLRPReturns.result1
	}
}

func (fake *FakeRepBBS) CrashActualLRPCallCount() int {
	fake.crashActualLRPMutex.RLock()
	defer fake.crashActualLRPMutex.RUnlock()
	return len(fake.crashActualLRPArgsForCall)
}

func (fake *FakeRepBBS) CrashActualLRPArgsForCall(i int) (models.ActualLRPKey, models.ActualLRPContainerKey, lager.Logger) {
	fake.crashActualLRPMutex.RLock()
	defer fake.crashActualLRPMutex.RUnlock()
	return fake.crashActualLRPArgsForCall[i].key, fake.crashActualLRPArgsForCall[i].containerKey, fake.crashActualLRPArgsForCall[i].logger
}

func (fake *FakeRepBBS) CrashActualLRPReturns(result1 error) {
	fake.CrashActualLRPStub = nil
	fake.crashActualLRPReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeRepBBS) FailLRP(arg1 lager.Logger, arg2 models.ActualLRPKey, arg3 string) error {
	fake.failLRPMutex.Lock()
	fake.failLRPArgsForCall = append(fake.failLRPArgsForCall, struct {
		arg1 lager.Logger
		arg2 models.ActualLRPKey
		arg3 string
	}{arg1, arg2, arg3})
	fake.failLRPMutex.Unlock()
	if fake.FailLRPStub != nil {
		return fake.FailLRPStub(arg1, arg2, arg3)
	} else {
		return fake.failLRPReturns.result1
	}
}

func (fake *FakeRepBBS) FailLRPCallCount() int {
	fake.failLRPMutex.RLock()
	defer fake.failLRPMutex.RUnlock()
	return len(fake.failLRPArgsForCall)
}

func (fake *FakeRepBBS) FailLRPArgsForCall(i int) (lager.Logger, models.ActualLRPKey, string) {
	fake.failLRPMutex.RLock()
	defer fake.failLRPMutex.RUnlock()
	return fake.failLRPArgsForCall[i].arg1, fake.failLRPArgsForCall[i].arg2, fake.failLRPArgsForCall[i].arg3
}

func (fake *FakeRepBBS) FailLRPReturns(result1 error) {
	fake.FailLRPStub = nil
	fake.failLRPReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeRepBBS) EvacuateClaimedActualLRP(arg1 lager.Logger, arg2 models.ActualLRPKey, arg3 models.ActualLRPContainerKey) error {
	fake.evacuateClaimedActualLRPMutex.Lock()
	fake.evacuateClaimedActualLRPArgsForCall = append(fake.evacuateClaimedActualLRPArgsForCall, struct {
		arg1 lager.Logger
		arg2 models.ActualLRPKey
		arg3 models.ActualLRPContainerKey
	}{arg1, arg2, arg3})
	fake.evacuateClaimedActualLRPMutex.Unlock()
	if fake.EvacuateClaimedActualLRPStub != nil {
		return fake.EvacuateClaimedActualLRPStub(arg1, arg2, arg3)
	} else {
		return fake.evacuateClaimedActualLRPReturns.result1
	}
}

func (fake *FakeRepBBS) EvacuateClaimedActualLRPCallCount() int {
	fake.evacuateClaimedActualLRPMutex.RLock()
	defer fake.evacuateClaimedActualLRPMutex.RUnlock()
	return len(fake.evacuateClaimedActualLRPArgsForCall)
}

func (fake *FakeRepBBS) EvacuateClaimedActualLRPArgsForCall(i int) (lager.Logger, models.ActualLRPKey, models.ActualLRPContainerKey) {
	fake.evacuateClaimedActualLRPMutex.RLock()
	defer fake.evacuateClaimedActualLRPMutex.RUnlock()
	return fake.evacuateClaimedActualLRPArgsForCall[i].arg1, fake.evacuateClaimedActualLRPArgsForCall[i].arg2, fake.evacuateClaimedActualLRPArgsForCall[i].arg3
}

func (fake *FakeRepBBS) EvacuateClaimedActualLRPReturns(result1 error) {
	fake.EvacuateClaimedActualLRPStub = nil
	fake.evacuateClaimedActualLRPReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeRepBBS) EvacuateRunningActualLRP(arg1 lager.Logger, arg2 models.ActualLRPKey, arg3 models.ActualLRPContainerKey, arg4 models.ActualLRPNetInfo, arg5 uint64) error {
	fake.evacuateRunningActualLRPMutex.Lock()
	fake.evacuateRunningActualLRPArgsForCall = append(fake.evacuateRunningActualLRPArgsForCall, struct {
		arg1 lager.Logger
		arg2 models.ActualLRPKey
		arg3 models.ActualLRPContainerKey
		arg4 models.ActualLRPNetInfo
		arg5 uint64
	}{arg1, arg2, arg3, arg4, arg5})
	fake.evacuateRunningActualLRPMutex.Unlock()
	if fake.EvacuateRunningActualLRPStub != nil {
		return fake.EvacuateRunningActualLRPStub(arg1, arg2, arg3, arg4, arg5)
	} else {
		return fake.evacuateRunningActualLRPReturns.result1
	}
}

func (fake *FakeRepBBS) EvacuateRunningActualLRPCallCount() int {
	fake.evacuateRunningActualLRPMutex.RLock()
	defer fake.evacuateRunningActualLRPMutex.RUnlock()
	return len(fake.evacuateRunningActualLRPArgsForCall)
}

func (fake *FakeRepBBS) EvacuateRunningActualLRPArgsForCall(i int) (lager.Logger, models.ActualLRPKey, models.ActualLRPContainerKey, models.ActualLRPNetInfo, uint64) {
	fake.evacuateRunningActualLRPMutex.RLock()
	defer fake.evacuateRunningActualLRPMutex.RUnlock()
	return fake.evacuateRunningActualLRPArgsForCall[i].arg1, fake.evacuateRunningActualLRPArgsForCall[i].arg2, fake.evacuateRunningActualLRPArgsForCall[i].arg3, fake.evacuateRunningActualLRPArgsForCall[i].arg4, fake.evacuateRunningActualLRPArgsForCall[i].arg5
}

func (fake *FakeRepBBS) EvacuateRunningActualLRPReturns(result1 error) {
	fake.EvacuateRunningActualLRPStub = nil
	fake.evacuateRunningActualLRPReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeRepBBS) EvacuateStoppedActualLRP(arg1 lager.Logger, arg2 models.ActualLRPKey, arg3 models.ActualLRPContainerKey) error {
	fake.evacuateStoppedActualLRPMutex.Lock()
	fake.evacuateStoppedActualLRPArgsForCall = append(fake.evacuateStoppedActualLRPArgsForCall, struct {
		arg1 lager.Logger
		arg2 models.ActualLRPKey
		arg3 models.ActualLRPContainerKey
	}{arg1, arg2, arg3})
	fake.evacuateStoppedActualLRPMutex.Unlock()
	if fake.EvacuateStoppedActualLRPStub != nil {
		return fake.EvacuateStoppedActualLRPStub(arg1, arg2, arg3)
	} else {
		return fake.evacuateStoppedActualLRPReturns.result1
	}
}

func (fake *FakeRepBBS) EvacuateStoppedActualLRPCallCount() int {
	fake.evacuateStoppedActualLRPMutex.RLock()
	defer fake.evacuateStoppedActualLRPMutex.RUnlock()
	return len(fake.evacuateStoppedActualLRPArgsForCall)
}

func (fake *FakeRepBBS) EvacuateStoppedActualLRPArgsForCall(i int) (lager.Logger, models.ActualLRPKey, models.ActualLRPContainerKey) {
	fake.evacuateStoppedActualLRPMutex.RLock()
	defer fake.evacuateStoppedActualLRPMutex.RUnlock()
	return fake.evacuateStoppedActualLRPArgsForCall[i].arg1, fake.evacuateStoppedActualLRPArgsForCall[i].arg2, fake.evacuateStoppedActualLRPArgsForCall[i].arg3
}

func (fake *FakeRepBBS) EvacuateStoppedActualLRPReturns(result1 error) {
	fake.EvacuateStoppedActualLRPStub = nil
	fake.evacuateStoppedActualLRPReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeRepBBS) EvacuateCrashedActualLRP(arg1 lager.Logger, arg2 models.ActualLRPKey, arg3 models.ActualLRPContainerKey) error {
	fake.evacuateCrashedActualLRPMutex.Lock()
	fake.evacuateCrashedActualLRPArgsForCall = append(fake.evacuateCrashedActualLRPArgsForCall, struct {
		arg1 lager.Logger
		arg2 models.ActualLRPKey
		arg3 models.ActualLRPContainerKey
	}{arg1, arg2, arg3})
	fake.evacuateCrashedActualLRPMutex.Unlock()
	if fake.EvacuateCrashedActualLRPStub != nil {
		return fake.EvacuateCrashedActualLRPStub(arg1, arg2, arg3)
	} else {
		return fake.evacuateCrashedActualLRPReturns.result1
	}
}

func (fake *FakeRepBBS) EvacuateCrashedActualLRPCallCount() int {
	fake.evacuateCrashedActualLRPMutex.RLock()
	defer fake.evacuateCrashedActualLRPMutex.RUnlock()
	return len(fake.evacuateCrashedActualLRPArgsForCall)
}

func (fake *FakeRepBBS) EvacuateCrashedActualLRPArgsForCall(i int) (lager.Logger, models.ActualLRPKey, models.ActualLRPContainerKey) {
	fake.evacuateCrashedActualLRPMutex.RLock()
	defer fake.evacuateCrashedActualLRPMutex.RUnlock()
	return fake.evacuateCrashedActualLRPArgsForCall[i].arg1, fake.evacuateCrashedActualLRPArgsForCall[i].arg2, fake.evacuateCrashedActualLRPArgsForCall[i].arg3
}

func (fake *FakeRepBBS) EvacuateCrashedActualLRPReturns(result1 error) {
	fake.EvacuateCrashedActualLRPStub = nil
	fake.evacuateCrashedActualLRPReturns = struct {
		result1 error
	}{result1}
}

var _ bbs.RepBBS = new(FakeRepBBS)
