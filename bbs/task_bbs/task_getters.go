package task_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

func (bbs *TaskBBS) OldTasks(logger lager.Logger) ([]models.Task, error) {
	logger.Info("fetching-tasks-from-store")
	node, err := bbs.store.ListRecursively(shared.TaskSchemaRoot)
	if err == storeadapter.ErrorKeyNotFound {
		logger.Info("no-tasks-to-fetch")
		return []models.Task{}, nil
	} else if err != nil {
		logger.Error("failed-fetching-tasks-from-store", err)
		return []models.Task{}, shared.ConvertStoreError(err)
	}
	logger.Info("succeeded-fetching-tasks-from-store")

	logger.Debug("unmarshalling-tasks")
	tasks := []models.Task{}
	for _, node := range node.ChildNodes {
		var task models.Task
		err := models.FromJSON(node.Value, &task)
		if err != nil {
			logger.Error("failed-unmarshalling-task", err, lager.Data{
				"key":   node.Key,
				"value": node.Value,
			})
		} else {
			tasks = append(tasks, task)
		}
	}
	logger.Debug("succeeded-unmarshalling-tasks")

	return tasks, nil
}

func (bbs *TaskBBS) OldTaskByGuid(logger lager.Logger, guid string) (models.Task, error) {
	logger = logger.WithData(lager.Data{"guid": guid})

	logger.Debug("getting-task")
	task, _, err := bbs.getTask(guid)
	if err != nil {
		logger.Error("failed-getting-task", err)
		return models.Task{}, err
	} else {
		logger.Debug("succeeded-getting-task")
		return task, nil
	}
}

func (bbs *TaskBBS) PendingTasks(logger lager.Logger) ([]models.Task, error) {
	all, err := bbs.OldTasks(logger)
	return filterTasksByState(all, models.TaskStatePending), err
}

func (bbs *TaskBBS) RunningTasks(logger lager.Logger) ([]models.Task, error) {
	all, err := bbs.OldTasks(logger)
	return filterTasksByState(all, models.TaskStateRunning), err
}

func (bbs *TaskBBS) CompletedTasks(logger lager.Logger) ([]models.Task, error) {
	all, err := bbs.OldTasks(logger)
	return filterTasksByState(all, models.TaskStateCompleted), err
}

func (bbs *TaskBBS) FailedTasks(logger lager.Logger) ([]models.Task, error) {
	all, err := bbs.OldTasks(logger)
	return filterTasks(all, func(task models.Task) bool {
		return task.State == models.TaskStateCompleted && task.Failed
	}), err
}

func (bbs *TaskBBS) ResolvingTasks(logger lager.Logger) ([]models.Task, error) {
	all, err := bbs.OldTasks(logger)
	return filterTasksByState(all, models.TaskStateResolving), err
}

func (bbs *TaskBBS) tasksByDomain(logger lager.Logger, domain string) ([]models.Task, error) {
	all, err := bbs.OldTasks(logger)
	return filterTasks(all, func(task models.Task) bool {
		return task.Domain == domain
	}), err
}

func (bbs *TaskBBS) tasksByCellID(logger lager.Logger, cellId string) ([]models.Task, error) {
	all, err := bbs.OldTasks(logger)
	return filterTasks(all, func(task models.Task) bool {
		return task.CellID == cellId
	}), err
}

func filterTasks(tasks []models.Task, filterFunc func(models.Task) bool) []models.Task {
	result := make([]models.Task, 0)
	for _, task := range tasks {
		if filterFunc(task) {
			result = append(result, task)
		}
	}
	return result
}

func filterTasksByState(tasks []models.Task, state models.TaskState) []models.Task {
	return filterTasks(tasks, func(task models.Task) bool {
		return task.State == state
	})
}

func (bbs *TaskBBS) getTask(taskGuid string) (models.Task, uint64, error) {
	node, err := bbs.store.Get(shared.TaskSchemaPath(taskGuid))
	if err != nil {
		return models.Task{}, 0, shared.ConvertStoreError(err)
	}

	var task models.Task
	err = models.FromJSON(node.Value, &task)

	return task, node.Index, err
}
