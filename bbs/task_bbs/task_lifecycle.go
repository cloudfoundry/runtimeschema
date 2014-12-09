package task_bbs

import (
	"fmt"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
)

// The stager calls this when it wants to desire a payload
// stagerTaskBBS will retry this repeatedly if it gets a StoreTimeout error (up to N seconds?)
// If this fails, the stager should bail and run its "this-failed-to-stage" routine
func (s *TaskBBS) DesireTask(task models.Task) error {
	err := task.Validate()
	if err != nil {
		return err
	}

	err = shared.RetryIndefinitelyOnStoreTimeout(func() error {
		if task.CreatedAt == 0 {
			task.CreatedAt = s.timeProvider.Now().UnixNano()
		}
		task.UpdatedAt = s.timeProvider.Now().UnixNano()
		task.State = models.TaskStatePending
		value, err := models.ToJSON(task)
		if err != nil {
			return err
		}
		return s.store.Create(storeadapter.StoreNode{
			Key:   shared.TaskSchemaPath(task.TaskGuid),
			Value: value,
		})
	})
	return err
}

// The cell calls this when it is about to run the task in the allocated container
// stagerTaskBBS will retry this repeatedly if it gets a StoreTimeout error (up to N seconds?)
// If this fails, the cell should assume that someone else will run it and should clean up and bail
func (bbs *TaskBBS) StartTask(taskGuid string, cellID string) error {
	task, index, err := bbs.getTask(taskGuid)

	if err != nil {
		return fmt.Errorf("cannot start non-existing task: %s", err.Error())
	}

	if task.State != models.TaskStatePending {
		return bbserrors.ErrTaskCannotBeStarted
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
func (bbs *TaskBBS) CancelTask(taskGuid string) error {
	task, index, err := bbs.getTask(taskGuid)

	if err == bbserrors.ErrStoreResourceNotFound {
		return bbserrors.ErrTaskNotFound
	} else if err != nil {
		return err
	} else if task == nil {
		return bbserrors.ErrTaskNotFound
	}

	if task.State != models.TaskStatePending && task.State != models.TaskStateRunning {
		return bbserrors.ErrTaskCannotBeCancelled
	}

	*task = bbs.markTaskCompleted(*task, true, "task was cancelled", "")

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
func (bbs *TaskBBS) CompleteTask(taskGuid string, failed bool, failureReason string, result string) error {
	task, index, err := bbs.getTask(taskGuid)

	if err != nil || task == nil {
		return bbserrors.ErrTaskNotFound
	}

	if task.State != models.TaskStateRunning {
		return bbserrors.ErrTaskCannotBeCompleted
	}

	*task = bbs.markTaskCompleted(*task, failed, failureReason, result)

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

		err = bbs.taskClient.CompleteTask(receptorPresence.ReceptorURL, task)
		if err != nil {
			bbs.logger.Error("failed-to-complete-task", err)
			return nil
		}

		return nil
	})
}

// The stager calls this when it wants to claim a completed task.  This ensures that only one
// stager ever attempts to handle a completed task
func (bbs *TaskBBS) ResolvingTask(taskGuid string) error {
	task, index, err := bbs.getTask(taskGuid)

	if err != nil {
		return bbserrors.ErrTaskNotFound
	}

	if task.State != models.TaskStateCompleted {
		return bbserrors.ErrTaskCannotBeMarkedAsResolving
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
func (bbs *TaskBBS) ResolveTask(taskGuid string) error {
	task, _, err := bbs.getTask(taskGuid)

	if err != nil {
		return fmt.Errorf("cannot resolve non-existing task: %s", err.Error())
	}

	if task.State != models.TaskStateResolving {
		return bbserrors.ErrTaskCannotBeResolved
	}

	return shared.RetryIndefinitelyOnStoreTimeout(func() error {
		return bbs.store.Delete(shared.TaskSchemaPath(taskGuid))
	})
}
