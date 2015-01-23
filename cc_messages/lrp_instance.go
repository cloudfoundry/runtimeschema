package cc_messages

type LRPInstanceState string

const (
	LRPInstanceStateStarting LRPInstanceState = "STARTING"
	LRPInstanceStateRunning  LRPInstanceState = "RUNNING"
	LRPInstanceStateFlapping LRPInstanceState = "FLAPPING"
	LRPInstanceStateUnknown  LRPInstanceState = "UNKNOWN"
)

type LRPInstance struct {
	ProcessGuid  string           `json:"process_guid"`
	InstanceGuid string           `json:"instance_guid"`
	Index        uint             `json:"index"`
	State        LRPInstanceState `json:"state"`
	Since        int64            `json:"since_in_ns"`
}
