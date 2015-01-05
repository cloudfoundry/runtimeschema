package routes

import "github.com/tedsuo/rata"

const (
	StopLRPInstance = "StopLRPInstance"
)

var StopLRPRoutes = rata.Routes{
	{Name: StopLRPInstance, Method: "POST", Path: "/lrps/:process_guid/instances/:instance_guid/stop"},
}
