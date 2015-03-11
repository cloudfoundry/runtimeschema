package repositories

import (
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/go-gorp/gorp"
)

type task struct {
	Id    int64 `db:"id"`
	Index int64 `db:"index"`

	CreatedAt int64  `db:"created_at"`
	UpdatedAt int64  `db:"updated_at"`
	TaskGuid  string `db:"task_guid"`
	Domain    string `db:"domain"`
	CellID    string `db:"cell_id"`

	Payload string `db:"payload"`
}

//go:generate counterfeiter -o fake_repositories/fake_task_repository.go . TaskRepository
type TaskRepository interface {
	Create(sql gorp.SqlExecutor, modelTask models.Task) (models.Task, error)
	GetAll(sql gorp.SqlExecutor) ([]models.Task, error)
	GetAllWithIndex(sql gorp.SqlExecutor) ([]TaskWithIndex, error)
	GetByTaskGuid(sql gorp.SqlExecutor, guid string) (models.Task, int64, error)
	GetAllByDomain(sql gorp.SqlExecutor, domain string) ([]models.Task, error)
	DeleteByTaskGuid(sql gorp.SqlExecutor, guid string) error
	CompareAndSwapByIndex(sql gorp.SqlExecutor, modelTask models.Task, index int64) (models.Task, error)
}

type TaskWithIndex struct {
	Task  models.Task
	Index int64
}

type taskRepository struct {
}

func NewTaskRepository(dbmap *gorp.DbMap) (TaskRepository, error) {
	dbmap.AddTableWithName(task{}, "tasks").SetKeys(true, "Id").SetVersionCol("Index")
	_, err := dbmap.Exec("CREATE TABLE IF NOT EXISTS tasks (" +
		"id bigint(20) NOT NULL AUTO_INCREMENT, PRIMARY KEY (id)," +
		"`index` bigint(20) DEFAULT NULL," +
		"created_at bigint(20) DEFAULT NULL," +
		"updated_at bigint(20) DEFAULT NULL," +
		"task_guid varchar(255) NOT NULL UNIQUE," +
		"domain varchar(255) DEFAULT NULL," +
		"cell_id varchar(255) DEFAULT NULL," +
		"payload longtext DEFAULT NULL" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8")

	return &taskRepository{}, err
}

func (repo *taskRepository) Create(sql gorp.SqlExecutor, model models.Task) (models.Task, error) {
	t, err := taskFromModel(&model)
	if err != nil {
		return models.Task{}, err
	}

	err = sql.Insert(&t)
	if err != nil {
		return models.Task{}, err
	}

	return modelFromTask(&t)
}

func (repo *taskRepository) GetAll(sql gorp.SqlExecutor) ([]models.Task, error) {
	tasks, err := sql.Select(task{}, "select * from tasks")
	if err != nil {
		return nil, err
	}

	return tasksToModels(tasks)
}

func (repo *taskRepository) GetAllWithIndex(sql gorp.SqlExecutor) ([]TaskWithIndex, error) {
	tasks, err := sql.Select(task{}, "select * from tasks")
	if err != nil {
		return nil, err
	}

	return tasksToModelsWithIndex(tasks)
}

func (repo *taskRepository) GetByTaskGuid(sql gorp.SqlExecutor, guid string) (models.Task, int64, error) {
	t := task{}
	err := sql.SelectOne(&t, "select * from tasks where task_guid = ?", guid)
	if err != nil {
		return models.Task{}, 0, err
	}

	model, err := modelFromTask(&t)
	if err != nil {
		return models.Task{}, 0, err
	}

	return model, t.Index, nil
}

func (repo *taskRepository) GetAllByDomain(sql gorp.SqlExecutor, domain string) ([]models.Task, error) {
	tasks, err := sql.Select(task{}, "select * from tasks where domain = ?", domain)
	if err != nil {
		return nil, err
	}

	return tasksToModels(tasks)
}

func (repo *taskRepository) DeleteByTaskGuid(sql gorp.SqlExecutor, guid string) error {
	id, index, err := primaryKeyAndIndexForTaskGuid(sql, guid)
	if err != nil {
		return err
	}

	task := task{Id: id, Index: index}
	_, err = sql.Delete(&task)

	return err
}

func (repo *taskRepository) CompareAndSwapByIndex(sql gorp.SqlExecutor, modelTask models.Task, index int64) (models.Task, error) {
	id, _, err := primaryKeyAndIndexForTaskGuid(sql, modelTask.TaskGuid)
	if err != nil {
		return models.Task{}, err
	}

	task, err := taskFromModel(&modelTask)
	if err != nil {
		return models.Task{}, err
	}

	task.Id = id
	task.Index = index

	_, err = sql.Update(&task)
	if err != nil {
		return models.Task{}, err
	}

	return modelFromTask(&task)
}

func primaryKeyAndIndexForTaskGuid(sql gorp.SqlExecutor, guid string) (int64, int64, error) {
	t := task{}

	err := sql.SelectOne(&t, "select tasks.id, tasks.index from tasks where task_guid = ?", guid)
	if err != nil {
		return 0, 0, err
	}

	return t.Id, t.Index, nil
}

func taskFromModel(model *models.Task) (task, error) {
	payload, err := model.MarshalJSON()
	if err != nil {
		return task{}, err
	}

	return task{
		TaskGuid: model.TaskGuid,
		Domain:   model.Domain,
		CellID:   model.CellID,
		Payload:  string(payload),
	}, nil
}

func modelFromTask(task *task) (models.Task, error) {
	model := models.Task{}

	err := model.UnmarshalJSON([]byte(task.Payload))
	if err != nil {
		return models.Task{}, err
	}

	return model, nil
}

func tasksToModels(tasks []interface{}) ([]models.Task, error) {
	modelTasks := make([]models.Task, 0, len(tasks))
	for _, t := range tasks {
		if task, ok := t.(*task); ok {
			model, err := modelFromTask(task)
			if err != nil {
				return nil, err
			}
			modelTasks = append(modelTasks, model)
		}
	}

	return modelTasks, nil
}

func tasksToModelsWithIndex(tasks []interface{}) ([]TaskWithIndex, error) {
	tasksWithIndex := make([]TaskWithIndex, 0, len(tasks))
	for _, t := range tasks {
		if task, ok := t.(*task); ok {
			model, err := modelFromTask(task)
			if err != nil {
				return nil, err
			}

			taskWithIndex := TaskWithIndex{
				Task:  model,
				Index: task.Index,
			}

			tasksWithIndex = append(tasksWithIndex, taskWithIndex)
		}
	}

	return tasksWithIndex, nil
}

// func (t *task) PreInsert(s gorp.SqlExecutor) error {
// 	now := time.Now().UnixNano()
// 	t.CreatedAt = now
// 	t.UpdatedAt = now
// 	return nil
// }

// func (t *task) PreUpdate(s gorp.SqlExecutor) error {
// 	t.UpdatedAt = time.Now().UnixNano()
// 	return nil
// }
