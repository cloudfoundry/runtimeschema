package task_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

func (bbs *TaskBBS) GetAllTasks() ([]models.Task, error) {
	node, err := bbs.store.ListRecursively(shared.TaskSchemaRoot)
	if err == storeadapter.ErrorKeyNotFound {
		return []models.Task{}, nil
	}

	if err != nil {
		return []models.Task{}, err
	}

	tasks := []models.Task{}
	for _, node := range node.ChildNodes {
		task, err := models.NewTaskFromJSON(node.Value)
		if err != nil {
			bbs.logger.Error("failed-to-unmarshal-task", err, lager.Data{
				"key":   node.Key,
				"value": node.Value,
			})
		} else {
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

func (bbs *TaskBBS) GetTaskByGuid(guid string) (models.Task, error) {
	task, _, err := bbs.getTask(guid)
	return task, err
}

func (bbs *TaskBBS) GetAllPendingTasks() ([]models.Task, error) {
	all, err := bbs.GetAllTasks()
	return filterTasksByState(all, models.TaskStatePending), err
}

func (bbs *TaskBBS) GetAllClaimedTasks() ([]models.Task, error) {
	all, err := bbs.GetAllTasks()
	return filterTasksByState(all, models.TaskStateClaimed), err
}

func (bbs *TaskBBS) GetAllRunningTasks() ([]models.Task, error) {
	all, err := bbs.GetAllTasks()
	return filterTasksByState(all, models.TaskStateRunning), err
}

func (bbs *TaskBBS) GetAllCompletedTasks() ([]models.Task, error) {
	all, err := bbs.GetAllTasks()
	return filterTasksByState(all, models.TaskStateCompleted), err
}

func (bbs *TaskBBS) GetAllResolvingTasks() ([]models.Task, error) {
	all, err := bbs.GetAllTasks()
	return filterTasksByState(all, models.TaskStateResolving), err
}

func (bbs *TaskBBS) GetAllTasksByDomain(domain string) ([]models.Task, error) {
	all, err := bbs.GetAllTasks()
	return filterTasks(all, func(task models.Task) bool {
		return task.Domain == domain
	}), err
}

func (bbs *TaskBBS) GetAllTasksByCellID(cellId string) ([]models.Task, error) {
	all, err := bbs.GetAllTasks()
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
	var node storeadapter.StoreNode
	err := shared.RetryIndefinitelyOnStoreTimeout(func() error {
		var err error
		node, err = bbs.store.Get(shared.TaskSchemaPath(taskGuid))
		return err
	})

	if err != nil {
		return models.Task{}, 0, err
	}

	task, err := models.NewTaskFromJSON(node.Value)

	return task, node.Index, err
}
