package routes

import "github.com/tedsuo/rata"

const (
	StopLRPInstance = "StopLRPInstance"
)

var StopLRPRoutes = rata.Routes{
	{Name: StopLRPInstance, Method: "DELETE", Path: "/lrp/:process_guid/index/:index/instance/:instance_guid"},
}
