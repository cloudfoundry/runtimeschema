package fake_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	"time"
)

type FakeExecutorBBS struct {
	callsToConverge int

	maintainConvergeInterval      time.Duration
	maintainConvergeExecutorID    string
	maintainConvergeStatusChannel <-chan bool
	maintainConvergeStopChannel   chan<- chan bool
	maintainConvergeLockError     error

	maintainingPresenceHeartbeatInterval time.Duration
	maintainingPresenceExecutorID        string
	maintainingPresencePresence          *FakePresence
	maintainingPresenceError             error

	claimedRunOnce  *models.RunOnce
	claimRunOnceErr error

	startedRunOnce  *models.RunOnce
	startRunOnceErr error

	completedRunOnce           *models.RunOnce
	completeRunOnceErr         error
	convergeRunOnceTimeToClaim time.Duration
}

func NewFakeExecutorBBS() *FakeExecutorBBS {
	return &FakeExecutorBBS{}
}

func (fakeBBS *FakeExecutorBBS) MaintainExecutorPresence(heartbeatInterval time.Duration, executorID string) (bbs.Presence, <-chan bool, error) {
	fakeBBS.maintainingPresenceHeartbeatInterval = heartbeatInterval
	fakeBBS.maintainingPresenceExecutorID = executorID
	fakeBBS.maintainingPresencePresence = &FakePresence{}
	status, _ := fakeBBS.maintainingPresencePresence.Maintain(heartbeatInterval)

	return fakeBBS.maintainingPresencePresence, status, fakeBBS.maintainingPresenceError
}

func (fakeBBS *FakeExecutorBBS) WatchForDesiredRunOnce() (<-chan *models.RunOnce, chan<- bool, <-chan error) {
	return nil, nil, nil
}

func (fakeBBS *FakeExecutorBBS) ClaimRunOnce(runOnce *models.RunOnce, executorID string) error {
	runOnce.ExecutorID = executorID
	fakeBBS.claimedRunOnce = runOnce
	return fakeBBS.claimRunOnceErr
}

func (fakeBBS *FakeExecutorBBS) StartRunOnce(runOnce *models.RunOnce, containerHandle string) error {
	runOnce.ContainerHandle = containerHandle
	fakeBBS.startedRunOnce = runOnce
	return fakeBBS.startRunOnceErr
}

func (fakeBBS *FakeExecutorBBS) CompleteRunOnce(runOnce *models.RunOnce, failed bool, failureReason string, result string) error {
	runOnce.Failed = failed
	runOnce.FailureReason = failureReason
	runOnce.Result = result
	fakeBBS.completedRunOnce = runOnce
	return fakeBBS.completeRunOnceErr
}

func (fakeBBS *FakeExecutorBBS) ConvergeRunOnce(timeToClaim time.Duration) {
	fakeBBS.convergeRunOnceTimeToClaim = timeToClaim
	fakeBBS.callsToConverge++
}

func (fakeBBS *FakeExecutorBBS) MaintainConvergeLock(interval time.Duration, executorID string) (<-chan bool, chan<- chan bool, error) {
	fakeBBS.maintainConvergeInterval = interval
	fakeBBS.maintainConvergeExecutorID = executorID
	status := make(chan bool)
	fakeBBS.maintainConvergeStatusChannel = status
	stop := make(chan chan bool)
	fakeBBS.maintainConvergeStopChannel = stop

	ticker := time.NewTicker(interval)
	go func() {
		status <- true
		for {
			select {
			case <-ticker.C:
				status <- true
			case release := <-stop:
				ticker.Stop()
				close(status)
				if release != nil {
					close(release)
				}

				return
			}
		}
	}()

	return fakeBBS.maintainConvergeStatusChannel, fakeBBS.maintainConvergeStopChannel, fakeBBS.maintainConvergeLockError
}

func (fakeBBS *FakeExecutorBBS) Stop() {
	if fakeBBS.maintainingPresencePresence != nil {
		fakeBBS.maintainingPresencePresence.Remove()
	}

	if fakeBBS.maintainConvergeStopChannel != nil {
		fakeBBS.maintainConvergeStopChannel <- nil
	}
}
