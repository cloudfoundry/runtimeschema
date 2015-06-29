package task_bbs

import (
	"time"

	"github.com/cloudfoundry-incubator/consuladapter"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/services_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/metric"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/workpool"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

const throttlerSize = 20

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

func (bbs *TaskBBS) ConvergeTasks(
	logger lager.Logger,
	kickTaskDuration, expirePendingTaskDuration, expireCompletedTaskDuration time.Duration,
	cellsLoader *services_bbs.CellsLoader,
) {
	taskLog := logger.Session("converge-tasks")
	taskLog.Info("starting-convergence")
	defer taskLog.Info("finished-convergence")

	convergeTaskRunsCounter.Increment()

	convergeStart := bbs.clock.Now()

	defer func() {
		convergeTaskDuration.Send(time.Since(convergeStart))
	}()

	logger.Debug("listing-tasks")
	taskState, err := bbs.store.ListRecursively(shared.TaskSchemaRoot)
	if err != nil {
		logger.Debug("failed-listing-task")
		return
	}
	logger.Debug("succeeded-listing-task")

	logger.Debug("listing-cells")
	cellSet, err := cellsLoader.Cells()
	if err != nil {
		switch err.(type) {
		case consuladapter.PrefixNotFoundError:
			cellSet = models.CellSet{}
		default:
			logger.Debug("failed-listing-cells")
			return
		}
	}
	logger.Debug("succeeded-listing-cells")

	logError := func(task models.Task, message string) {
		taskLog.Error(message, nil, lager.Data{
			"task-guid": task.TaskGuid,
		})
	}

	tasksToComplete := []models.Task{}
	scheduleForCompletion := func(task models.Task) {
		if task.CompletionCallbackURL == nil {
			return
		}
		tasksToComplete = append(tasksToComplete, task)
	}

	keysToDelete := []string{}

	tasksToCAS := []compareAndSwappableTask{}
	scheduleForCASByIndex := func(index uint64, newTask models.Task) {
		tasksToCAS = append(tasksToCAS, compareAndSwappableTask{
			OldIndex: index,
			NewTask:  newTask,
		})
	}

	tasksToAuction := []models.Task{}

	var tasksKicked uint64 = 0

	logger.Debug("determining-convergence-work", lager.Data{"num-tasks": len(taskState.ChildNodes)})
	for _, node := range taskState.ChildNodes {
		var task models.Task
		err = models.FromJSON(node.Value, &task)
		if err != nil {
			taskLog.Error("failed-to-unmarshal-task-json", err, lager.Data{
				"key":   node.Key,
				"value": node.Value,
			})

			keysToDelete = append(keysToDelete, node.Key)
			continue
		}

		shouldKickTask := bbs.durationSinceTaskUpdated(task) >= kickTaskDuration

		switch task.State {
		case models.TaskStatePending:
			shouldMarkAsFailed := bbs.durationSinceTaskCreated(task) >= expirePendingTaskDuration
			if shouldMarkAsFailed {
				logError(task, "failed-to-start-in-time")
				scheduleForCASByIndex(node.Index, bbs.markTaskFailed(task, "not started within time limit"))
				tasksKicked++
			} else if shouldKickTask {
				taskLog.Info("requesting-auction-for-pending-task", lager.Data{"task-guid": task.TaskGuid})
				tasksToAuction = append(tasksToAuction, task)
				tasksKicked++
			}
		case models.TaskStateRunning:
			cellIsAlive := cellSet.HasCellID(task.CellID)
			if !cellIsAlive {
				logError(task, "cell-disappeared")
				scheduleForCASByIndex(node.Index, bbs.markTaskFailed(task, "cell disappeared before completion"))
				tasksKicked++
			}
		case models.TaskStateCompleted:
			shouldDeleteTask := bbs.durationSinceTaskFirstCompleted(task) >= expireCompletedTaskDuration
			if shouldDeleteTask {
				logError(task, "failed-to-start-resolving-in-time")
				keysToDelete = append(keysToDelete, node.Key)
			} else if shouldKickTask {
				taskLog.Info("kicking-completed-task", lager.Data{"task-guid": task.TaskGuid})
				scheduleForCompletion(task)
				tasksKicked++
			}
		case models.TaskStateResolving:
			shouldDeleteTask := bbs.durationSinceTaskFirstCompleted(task) >= expireCompletedTaskDuration
			if shouldDeleteTask {
				logError(task, "failed-to-resolve-in-time")
				keysToDelete = append(keysToDelete, node.Key)
			} else if shouldKickTask {
				taskLog.Info("demoting-resolving-to-completed", lager.Data{"task-guid": task.TaskGuid})
				demoted := demoteToCompleted(task)
				scheduleForCASByIndex(node.Index, demoted)
				scheduleForCompletion(demoted)
				tasksKicked++
			}
		}
	}
	logger.Debug("done-determining-convergence-work", lager.Data{
		"num-tasks-to-auction":  len(tasksToAuction),
		"num-tasks-to-cas":      len(tasksToCAS),
		"num-tasks-to-complete": len(tasksToComplete),
		"num-keys-to-delete":    len(keysToDelete),
	})

	if len(tasksToAuction) > 0 {
		logger.Debug("requesting-task-auctions", lager.Data{"num-tasks-to-auction": len(tasksToAuction)})
		if err := bbs.requestTaskAuctions(taskLog, tasksToAuction); err != nil {
			taskGuids := make([]string, len(tasksToAuction))
			for i, task := range tasksToAuction {
				taskGuids[i] = task.TaskGuid
			}
			taskLog.Error("failed-to-request-auctions-for-pending-tasks", err,
				lager.Data{"task-guids": taskGuids})
		}
		logger.Debug("done-requesting-task-auctions", lager.Data{"num-tasks-to-auction": len(tasksToAuction)})
	}

	tasksKickedCounter.Add(tasksKicked)
	logger.Debug("compare-and-swapping-tasks", lager.Data{"num-tasks-to-cas": len(tasksToCAS)})
	err = bbs.batchCompareAndSwapTasks(tasksToCAS, taskLog)
	if err != nil {
		return
	}
	logger.Debug("done-compare-and-swapping-tasks", lager.Data{"num-tasks-to-cas": len(tasksToCAS)})

	logger.Debug("marking-tasks-completed", lager.Data{"num-tasks-to-complete": len(tasksToComplete)})
	bbs.completeTasks(tasksToComplete, taskLog)
	logger.Debug("done-marking-tasks-completed", lager.Data{"num-tasks-to-complete": len(tasksToComplete)})

	tasksPrunedCounter.Add(uint64(len(keysToDelete)))
	logger.Debug("deleting-keys", lager.Data{"num-keys-to-delete": len(keysToDelete)})
	bbs.store.Delete(keysToDelete...)
	logger.Debug("done-deleting-keys", lager.Data{"num-keys-to-delete": len(keysToDelete)})
}

func (bbs *TaskBBS) durationSinceTaskCreated(task models.Task) time.Duration {
	return bbs.clock.Now().Sub(time.Unix(0, task.CreatedAt))
}

func (bbs *TaskBBS) durationSinceTaskUpdated(task models.Task) time.Duration {
	return bbs.clock.Now().Sub(time.Unix(0, task.UpdatedAt))
}

func (bbs *TaskBBS) durationSinceTaskFirstCompleted(task models.Task) time.Duration {
	if task.FirstCompletedAt == 0 {
		return 0
	}
	return bbs.clock.Now().Sub(time.Unix(0, task.FirstCompletedAt))
}

func (bbs *TaskBBS) markTaskFailed(task models.Task, reason string) models.Task {
	return bbs.markTaskCompleted(task, true, reason, "")
}

func (bbs *TaskBBS) markTaskCompleted(task models.Task, failed bool, failureReason string, result string) models.Task {
	task.UpdatedAt = bbs.clock.Now().UnixNano()
	task.FirstCompletedAt = bbs.clock.Now().UnixNano()
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

func (bbs *TaskBBS) batchCompareAndSwapTasks(tasksToCAS []compareAndSwappableTask, taskLog lager.Logger) error {
	if len(tasksToCAS) == 0 {
		return nil
	}

	works := []func(){}

	for _, taskToCAS := range tasksToCAS {
		task := taskToCAS.NewTask
		task.UpdatedAt = bbs.clock.Now().UnixNano()
		value, err := models.ToJSON(task)
		if err != nil {
			taskLog.Error("failed-to-marshal", err, lager.Data{
				"task-guid": task.TaskGuid,
			})
			continue
		}

		newStoreNode := storeadapter.StoreNode{
			Key:   shared.TaskSchemaPath(task.TaskGuid),
			Value: value,
		}
		index := taskToCAS.OldIndex
		works = append(works, func() {
			err := bbs.store.CompareAndSwapByIndex(index, newStoreNode)
			if err != nil {
				taskLog.Error("failed-to-compare-and-swap", err, lager.Data{
					"task-guid": task.TaskGuid,
				})
			}
		})
	}

	throttler, err := workpool.NewThrottler(throttlerSize, works)
	if err != nil {
		return err
	}

	throttler.Work()
	return nil
}

func (bbs *TaskBBS) completeTasks(tasksToComplete []models.Task, taskLog lager.Logger) {
	if len(tasksToComplete) == 0 {
		return
	}

	err := bbs.taskClient.CompleteTasks(bbs.receptorTaskHandlerURL, tasksToComplete)
	if err != nil {
		taskGuids := make([]string, len(tasksToComplete))
		for i, task := range tasksToComplete {
			taskGuids[i] = task.TaskGuid
		}
		taskLog.Error("failed-to-complete-tasks", err, lager.Data{
			"task-guids": taskGuids,
		})
	}
}
