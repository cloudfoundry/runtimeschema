package cc_messages

import (
	"encoding/json"

	"github.com/cloudfoundry-incubator/receptor"
)

const CF_ROUTER = "cf-router"

type Routes []Route

type Route struct {
	Hostnames []string `json:"routes"`
	Port      uint16   `json:"port"`
}

type receptorRoute struct {
	Hostnames []string `json:"hostnames"`
	Port      uint16   `json:"port"`
}

func NewRoutingInfo(hostnames []string, port uint16) *receptor.RoutingInfo {
	routingData, err := json.Marshal([]receptorRoute{
		{Hostnames: hostnames, Port: port},
	})
	if err != nil {
		panic("unexpected failure to marshal route")
	}

	routingInfo := json.RawMessage(routingData)

	return &receptor.RoutingInfo{
		CF_ROUTER: &routingInfo,
	}
}

func RouteFromRoutingInfo(routingInfo *receptor.RoutingInfo) Route {
	route := Route{}
	if routingInfo != nil {
		if message, found := (*routingInfo)[CF_ROUTER]; found {
			receptorRoutes := []receptorRoute{}

			err := json.Unmarshal([]byte(*message), &receptorRoutes)
			if err != nil {
				panic("unexpected failure to marshal route")
			}

			if len(receptorRoutes) > 0 {
				route.Hostnames = receptorRoutes[0].Hostnames
				route.Port = receptorRoutes[0].Port
			}
		}
	}

	return route
}
