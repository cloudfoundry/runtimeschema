package routes

import "github.com/tedsuo/rata"

const (
	StopLRPInstance = "StopLRPInstance"
)

var StopLRPRoutes = rata.Routes{
	{Name: StopLRPInstance, Method: "POST", Path: "/lrps/:process_guid/instance/:instance_guid/index/:index/stop"},
}
