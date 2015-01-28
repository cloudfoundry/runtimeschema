package cc_messages

const CF_ROUTER = "cf-router"

type Routes []Route

type Route struct {
	Port  uint16   `json:"port"`
	Hosts []string `json:"hosts"`
}
