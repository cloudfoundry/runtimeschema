package routes

import "github.com/tedsuo/rata"

const (
	CompleteTasks = "CompleteTasks"
)

var CompleteTasksRoutes = rata.Routes{
	{Name: CompleteTasks, Method: "PUT", Path: "/internal/tasks/complete"},
}
