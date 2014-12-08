package routes

import "github.com/tedsuo/rata"

const (
	CompleteTask = "CompleteTask"
)

var CompleteTaskRoutes = rata.Routes{
	{Name: CompleteTask, Method: "PUT", Path: "/internal/tasks/complete"},
}
