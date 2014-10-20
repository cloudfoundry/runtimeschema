package task_bbs

import (
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/metric"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

const (
	convergeTaskRunsCounter = metric.Counter("ConvergenceTaskRuns")
	convergeTaskDuration    = metric.Duration("ConvergenceTaskDuration")

	tasksKickedCounter = metric.Counter("ConvergenceTasksKicked")
	tasksPrunedCounter = metric.Counter("ConvergenceTasksPruned")
)

type compareAndSwappableTask struct {
	OldIndex uint64
	NewTask  models.Task
}

// ConvergeTask is run by *one* executor every X seconds (doesn't really matter what X is.. pick something performant)
// Converge will:
// 1. Kick (by setting) any run-onces that are still pending (and have been for > convergence interval)
// 2. Kick (by setting) any run-onces that are completed (and have been for > convergence interval)
// 3. Delete any run-onces that are completed (and have been for > timeToResolve interval)
// 4. Demote to pending any claimed run-onces that have been claimed for > 30s
// 5. Demote to completed any resolving run-onces that have been resolving for > 30s
// 6. Mark as failed any run-onces that have been in the pending state for > timeToClaim
// 7. Mark as failed any claimed or running run-onces whose executor has stopped maintaining presence
func (bbs *TaskBBS) ConvergeTask(timeToClaim, convergenceInterval, timeToResolve time.Duration) {
	convergeTaskRunsCounter.Increment()

	convergeStart := time.Now()

	// make sure to get funcy here otherwise the time will be precomputed
	defer func() {
		convergeTaskDuration.Send(time.Since(convergeStart))
	}()

	taskLog := bbs.logger.Session("converge-tasks")

	taskState, err := bbs.store.ListRecursively(shared.TaskSchemaRoot)
	if err != nil {
		return
	}

	executorState, err := bbs.store.ListRecursively(shared.ExecutorSchemaRoot)
	if err == storeadapter.ErrorKeyNotFound {
		executorState = storeadapter.StoreNode{}
	} else if err != nil {
		return
	}

	logError := func(task models.Task, message string) {
		taskLog.Error(message, nil, lager.Data{
			"task": task,
		})
	}

	keysToDelete := []string{}

	tasksToCAS := []compareAndSwappableTask{}
	scheduleForCASByIndex := func(index uint64, newTask models.Task) {
		tasksToCAS = append(tasksToCAS, compareAndSwappableTask{
			OldIndex: index,
			NewTask:  newTask,
		})
	}

	for _, node := range taskState.ChildNodes {
		task, err := models.NewTaskFromJSON(node.Value)
		if err != nil {
			taskLog.Error("failed-to-unmarshal-task-json", err, lager.Data{
				"key":   node.Key,
				"value": node.Value,
			})

			keysToDelete = append(keysToDelete, node.Key)
			continue
		}

		shouldKickTask := bbs.durationSinceTaskUpdated(task) >= convergenceInterval

		switch task.State {
		case models.TaskStatePending:
			shouldMarkAsFailed := bbs.durationSinceTaskCreated(task) >= timeToClaim
			if shouldMarkAsFailed {
				logError(task, "failed-to-claim")
				scheduleForCASByIndex(node.Index, bbs.markTaskFailed(task, "not claimed within time limit"))
			} else if shouldKickTask {
				scheduleForCASByIndex(node.Index, task)
			}
		case models.TaskStateClaimed:
			_, executorIsAlive := executorState.Lookup(task.ExecutorID)
			if !executorIsAlive {
				logError(task, "executor-disappeared")
				scheduleForCASByIndex(node.Index, bbs.markTaskFailed(task, "executor disappeared before completion"))
			}
		case models.TaskStateRunning:
			_, executorIsAlive := executorState.Lookup(task.ExecutorID)
			if !executorIsAlive {
				logError(task, "executor-disappeared")
				scheduleForCASByIndex(node.Index, bbs.markTaskFailed(task, "executor disappeared before completion"))
			}
		case models.TaskStateCompleted:
			shouldDeleteTask := bbs.durationSinceTaskFirstCompleted(task) >= timeToResolve
			if shouldDeleteTask {
				logError(task, "resolution-expired")
				keysToDelete = append(keysToDelete, node.Key)
			} else if shouldKickTask {
				scheduleForCASByIndex(node.Index, task)
			}
		case models.TaskStateResolving:
			shouldDeleteTask := bbs.durationSinceTaskFirstCompleted(task) >= timeToResolve
			if shouldDeleteTask {
				logError(task, "resolution-expired")
				keysToDelete = append(keysToDelete, node.Key)
			} else if shouldKickTask {
				logError(task, "failed-to-resolve")
				scheduleForCASByIndex(node.Index, demoteToCompleted(task))
			}
		}
	}

	tasksKickedCounter.Add(uint64(len(tasksToCAS)))
	bbs.batchCompareAndSwapTasks(tasksToCAS)

	tasksPrunedCounter.Add(uint64(len(keysToDelete)))
	bbs.store.Delete(keysToDelete...)
}

func (bbs *TaskBBS) durationSinceTaskCreated(task models.Task) time.Duration {
	return bbs.timeProvider.Time().Sub(time.Unix(0, task.CreatedAt))
}

func (bbs *TaskBBS) durationSinceTaskUpdated(task models.Task) time.Duration {
	return bbs.timeProvider.Time().Sub(time.Unix(0, task.UpdatedAt))
}

func (bbs *TaskBBS) durationSinceTaskFirstCompleted(task models.Task) time.Duration {
	if task.FirstCompletedAt == 0 {
		return 0
	}
	return bbs.timeProvider.Time().Sub(time.Unix(0, task.FirstCompletedAt))
}

func (bbs *TaskBBS) markTaskFailed(task models.Task, reason string) models.Task {
	return bbs.markTaskCompleted(task, true, reason, "")
}

func (bbs *TaskBBS) markTaskCompleted(task models.Task, failed bool, failureReason string, result string) models.Task {
	task.UpdatedAt = bbs.timeProvider.Time().UnixNano()
	task.FirstCompletedAt = bbs.timeProvider.Time().UnixNano()
	task.State = models.TaskStateCompleted
	task.Failed = failed
	task.FailureReason = failureReason
	task.Result = result
	return task
}

func demoteToCompleted(task models.Task) models.Task {
	task.State = models.TaskStateCompleted
	return task
}

func (bbs *TaskBBS) batchCompareAndSwapTasks(tasksToCAS []compareAndSwappableTask) {
	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(len(tasksToCAS))
	for _, taskToCAS := range tasksToCAS {
		task := taskToCAS.NewTask
		task.UpdatedAt = bbs.timeProvider.Time().UnixNano()
		newStoreNode := storeadapter.StoreNode{
			Key:   shared.TaskSchemaPath(task.TaskGuid),
			Value: task.ToJSON(),
		}

		go func(taskToCAS compareAndSwappableTask, newStoreNode storeadapter.StoreNode) {
			err := bbs.store.CompareAndSwapByIndex(taskToCAS.OldIndex, newStoreNode)
			if err != nil {
				bbs.logger.Error("failed-to-compare-and-swap", err)
			}

			waitGroup.Done()
		}(taskToCAS, newStoreNode)
	}

	waitGroup.Wait()
}
