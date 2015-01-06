package task_bbs

import (
	"fmt"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/lager"
)

// The stager calls this when it wants to desire a payload
// stagerTaskBBS will retry this repeatedly if it gets a StoreTimeout error (up to N seconds?)
// If this fails, the stager should bail and run its "this-failed-to-stage" routine
func (bbs *TaskBBS) DesireTask(logger lager.Logger, task models.Task) error {
	err := task.Validate()
	if err != nil {
		return err
	}
	task.State = models.TaskStatePending

	err = shared.RetryIndefinitelyOnStoreTimeout(func() error {
		if task.CreatedAt == 0 {
			task.CreatedAt = bbs.timeProvider.Now().UnixNano()
		}
		task.UpdatedAt = bbs.timeProvider.Now().UnixNano()
		value, err := models.ToJSON(task)
		if err != nil {
			return err
		}
		return bbs.store.Create(storeadapter.StoreNode{
			Key:   shared.TaskSchemaPath(task.TaskGuid),
			Value: value,
		})
	})

	if err != nil {
		return err
	}

	err = bbs.requestTaskAuctions([]models.Task{task})
	if err != nil {
		bbs.logger.Error("failed-sending-task-auction", err, lager.Data{"task": task})
		// The creation succeeded, the auction request error can be dropped
	}

	return nil
}

// The cell calls this when it is about to run the task in the allocated container
// stagerTaskBBS will retry this repeatedly if it gets a StoreTimeout error (up to N seconds?)
// If this fails, the cell should assume that someone else will run it and should clean up and bail
func (bbs *TaskBBS) StartTask(logger lager.Logger, taskGuid string, cellID string) error {
	task, index, err := bbs.getTask(taskGuid)

	if err != nil {
		return fmt.Errorf("cannot start non-existing task: %s", err.Error())
	}

	if task.State == models.TaskStateRunning && task.CellID == cellID {
		return nil
	}

	err = validateStateTransition(task.State, models.TaskStateRunning)
	if err != nil {
		return err
	}

	task.UpdatedAt = bbs.timeProvider.Now().UnixNano()
	task.State = models.TaskStateRunning
	task.CellID = cellID

	value, err := models.ToJSON(task)
	if err != nil {
		return err
	}

	return shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return bbs.store.CompareAndSwapByIndex(index, storeadapter.StoreNode{
			Key:   shared.TaskSchemaPath(taskGuid),
			Value: value,
		})
	})
}

// The cell calls this when the user requested to cancel the task
// stagerTaskBBS will retry this repeatedly if it gets a StoreTimeout error (up to N seconds?)
// Will fail if the task has already been cancelled or completed normally
func (bbs *TaskBBS) CancelTask(logger lager.Logger, taskGuid string) error {
	task, index, err := bbs.getTask(taskGuid)
	if err != nil {
		return err
	}

	err = validateStateTransition(task.State, models.TaskStateCompleted)
	if err != nil {
		return err
	}

	task = bbs.markTaskCompleted(task, true, "task was cancelled", "")

	value, err := models.ToJSON(task)
	if err != nil {
		return err
	}

	return shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return bbs.store.CompareAndSwapByIndex(index, storeadapter.StoreNode{
			Key:   shared.TaskSchemaPath(taskGuid),
			Value: value,
		})
	})
}

// The cell calls this when it has finished running the task (be it success or failure)
// stagerTaskBBS will retry this repeatedly if it gets a StoreTimeout error (up to N seconds?)
// This really really shouldn't fail.  If it does, blog about it and walk away. If it failed in a
// consistent way (i.e. key already exists), there's probably a flaw in our design.
func (bbs *TaskBBS) CompleteTask(logger lager.Logger, taskGuid string, cellID string, failed bool, failureReason string, result string) error {
	task, index, err := bbs.getTask(taskGuid)
	if err != nil {
		return err
	}

	if task.State == models.TaskStateRunning && task.CellID != cellID {
		return bbserrors.ErrTaskRunningOnDifferentCell
	}

	err = validateStateTransition(task.State, models.TaskStateCompleted)
	if err != nil {
		return err
	}

	task = bbs.markTaskCompleted(task, failed, failureReason, result)

	value, err := models.ToJSON(task)
	if err != nil {
		return err
	}

	return shared.RetryIndefinitelyOnStoreTimeout(func() error {
		err := bbs.store.CompareAndSwapByIndex(index, storeadapter.StoreNode{
			Key:   shared.TaskSchemaPath(taskGuid),
			Value: value,
		})
		if err != nil {
			return err
		}

		if task.CompletionCallbackURL == nil {
			return nil
		}

		receptorPresence, err := bbs.services.Receptor()
		if err != nil {
			bbs.logger.Error("could-not-fetch-receptors", err)
			return nil
		}

		err = bbs.taskClient.CompleteTasks(receptorPresence.ReceptorURL, []models.Task{task})
		if err != nil {
			bbs.logger.Error("failed-to-complete-task", err)
			return nil
		}

		return nil
	})
}

// The stager calls this when it wants to claim a completed task.  This ensures that only one
// stager ever attempts to handle a completed task
func (bbs *TaskBBS) ResolvingTask(logger lager.Logger, taskGuid string) error {
	task, index, err := bbs.getTask(taskGuid)
	if err != nil {
		return err
	}

	err = validateStateTransition(task.State, models.TaskStateResolving)
	if err != nil {
		return err
	}

	task.UpdatedAt = bbs.timeProvider.Now().UnixNano()
	task.State = models.TaskStateResolving

	value, err := models.ToJSON(task)
	if err != nil {
		return err
	}

	return shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return bbs.store.CompareAndSwapByIndex(index, storeadapter.StoreNode{
			Key:   shared.TaskSchemaPath(taskGuid),
			Value: value,
		})
	})
}

// The stager calls this when it wants to signal that it has received a completion and is handling it
// stagerTaskBBS will retry this repeatedly if it gets a StoreTimeout error (up to N seconds?)
// If this fails, the stager should assume that someone else is handling the completion and should bail
func (bbs *TaskBBS) ResolveTask(logger lager.Logger, taskGuid string) error {
	task, _, err := bbs.getTask(taskGuid)

	if err != nil {
		return fmt.Errorf("cannot resolve non-existing task: %s", err.Error())
	}

	err = validateCanDelete(task.State)
	if err != nil {
		return err
	}

	return shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return bbs.store.Delete(shared.TaskSchemaPath(taskGuid))
	})
}

func validateStateTransition(from, to models.TaskState) error {
	if (from != models.TaskStatePending && to == models.TaskStateRunning) ||
		((from != models.TaskStatePending && from != models.TaskStateRunning) && to == models.TaskStateCompleted) ||
		(from != models.TaskStateCompleted && to == models.TaskStateResolving) {
		return bbserrors.NewTaskStateTransitionError(from, to)
	} else {
		return nil
	}
}

func validateCanDelete(from models.TaskState) error {
	if from != models.TaskStateResolving {
		return bbserrors.NewTaskCannotBeResolvedError(from)
	} else {
		return nil
	}
}

func (bbs *TaskBBS) requestTaskAuctions(tasks []models.Task) error {
	auctioneerAddress, err := bbs.services.AuctioneerAddress()
	if err != nil {
		return err
	}

	err = bbs.auctioneerClient.RequestTaskAuctions(auctioneerAddress, tasks)
	if err != nil {
		return err
	}

	return nil
}
