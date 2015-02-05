package routes

import "github.com/tedsuo/rata"

const (
	CancelTask = "CancelTask"
)

var CancelTaskRoutes = rata.Routes{
	{Name: CancelTask, Method: "POST", Path: "/tasks/:task_guid/cancel"},
}
