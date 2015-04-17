package cc_messages

type LRPInstanceState string

const (
	LRPInstanceStateStarting LRPInstanceState = "STARTING"
	LRPInstanceStateRunning  LRPInstanceState = "RUNNING"
	LRPInstanceStateCrashed  LRPInstanceState = "CRASHED"
	LRPInstanceStateUnknown  LRPInstanceState = "UNKNOWN"
)

type LRPInstance struct {
	ProcessGuid  string            `json:"process_guid"`
	InstanceGuid string            `json:"instance_guid"`
	Index        uint              `json:"index"`
	State        LRPInstanceState  `json:"state"`
	Details      string            `json:"details,omitempty"`
	Since        int64             `json:"since_in_ns"`
	Stats        *LRPInstanceStats `json:"stats,omitempty"`
}

type LRPInstanceStats struct {
	CpuPercentage float64 `json:"cpu"`
	MemoryBytes   uint64  `json:"mem"`
	DiskBytes     uint64  `json:"disk"`
}
