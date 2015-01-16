package task_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

func (bbs *TaskBBS) Tasks(logger lager.Logger) ([]models.Task, error) {
	node, err := bbs.store.ListRecursively(shared.TaskSchemaRoot)
	if err == storeadapter.ErrorKeyNotFound {
		return []models.Task{}, nil
	} else if err != nil {
		return []models.Task{}, shared.ConvertStoreError(err)
	}

	tasks := []models.Task{}
	for _, node := range node.ChildNodes {
		var task models.Task
		err := models.FromJSON(node.Value, &task)
		if err != nil {
			logger.Error("failed-to-unmarshal-task", err, lager.Data{
				"key":   node.Key,
				"value": node.Value,
			})
		} else {
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

func (bbs *TaskBBS) TaskByGuid(guid string) (models.Task, error) {
	task, _, err := bbs.getTask(guid)
	return task, err
}

func (bbs *TaskBBS) PendingTasks(logger lager.Logger) ([]models.Task, error) {
	all, err := bbs.Tasks(logger)
	return filterTasksByState(all, models.TaskStatePending), err
}

func (bbs *TaskBBS) RunningTasks(logger lager.Logger) ([]models.Task, error) {
	all, err := bbs.Tasks(logger)
	return filterTasksByState(all, models.TaskStateRunning), err
}

func (bbs *TaskBBS) CompletedTasks(logger lager.Logger) ([]models.Task, error) {
	all, err := bbs.Tasks(logger)
	return filterTasksByState(all, models.TaskStateCompleted), err
}

func (bbs *TaskBBS) ResolvingTasks(logger lager.Logger) ([]models.Task, error) {
	all, err := bbs.Tasks(logger)
	return filterTasksByState(all, models.TaskStateResolving), err
}

func (bbs *TaskBBS) TasksByDomain(logger lager.Logger, domain string) ([]models.Task, error) {
	all, err := bbs.Tasks(logger)
	return filterTasks(all, func(task models.Task) bool {
		return task.Domain == domain
	}), err
}

func (bbs *TaskBBS) TasksByCellID(logger lager.Logger, cellId string) ([]models.Task, error) {
	all, err := bbs.Tasks(logger)
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
