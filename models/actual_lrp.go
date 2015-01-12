package models

import (
	"errors"
	"time"
)

type ActualLRPsByIndex map[int]ActualLRP

type ActualLRPsByProcessGuidAndIndex map[string]ActualLRPsByIndex

func (set ActualLRPsByProcessGuidAndIndex) Add(actual ActualLRP) {
	actuals, found := set[actual.ProcessGuid]
	if !found {
		actuals = ActualLRPsByIndex{}
		set[actual.ProcessGuid] = actuals
	}

	actuals[actual.Index] = actual
}

func (set ActualLRPsByProcessGuidAndIndex) Each(predicate func(actual ActualLRP)) {
	for _, indexSet := range set {
		for _, actual := range indexSet {
			predicate(actual)
		}
	}
}

type ActualLRPState string

const (
	ActualLRPStateUnclaimed ActualLRPState = "UNCLAIMED"
	ActualLRPStateClaimed   ActualLRPState = "CLAIMED"
	ActualLRPStateRunning   ActualLRPState = "RUNNING"
	ActualLRPStateCrashed   ActualLRPState = "CRASHED"
)

var ActualLRPStates = []ActualLRPState{
	ActualLRPStateUnclaimed,
	ActualLRPStateClaimed,
	ActualLRPStateRunning,
	ActualLRPStateCrashed,
}

type ActualLRPKey struct {
	ProcessGuid string `json:"process_guid"`
	Index       int    `json:"index"`
	Domain      string `json:"domain"`
}

func NewActualLRPKey(processGuid string, index int, domain string) ActualLRPKey {
	return ActualLRPKey{
		ProcessGuid: processGuid,
		Index:       index,
		Domain:      domain,
	}
}

func (key ActualLRPKey) Validate() error {
	var validationError ValidationError

	if key.ProcessGuid == "" {
		validationError = validationError.Append(ErrInvalidField{"process_guid"})
	}

	if key.Index < 0 {
		validationError = validationError.Append(ErrInvalidField{"index"})
	}

	if key.Domain == "" {
		validationError = validationError.Append(ErrInvalidField{"domain"})
	}

	if !validationError.Empty() {
		return validationError
	}

	return nil
}

type ActualLRPContainerKey struct {
	InstanceGuid string `json:"instance_guid"`
	CellID       string `json:"cell_id"`
}

var emptyActualLRPContainerKey = ActualLRPContainerKey{}

func (key *ActualLRPContainerKey) Empty() bool {
	return *key == emptyActualLRPContainerKey
}

func NewActualLRPContainerKey(instanceGuid string, cellID string) ActualLRPContainerKey {
	return ActualLRPContainerKey{
		InstanceGuid: instanceGuid,
		CellID:       cellID,
	}
}

func (key ActualLRPContainerKey) Validate() error {
	var validationError ValidationError

	if key.CellID == "" {
		validationError = validationError.Append(ErrInvalidField{"cell_id"})
	}

	if key.InstanceGuid == "" {
		validationError = validationError.Append(ErrInvalidField{"instance_guid"})
	}

	if !validationError.Empty() {
		return validationError
	}

	return nil
}

type ActualLRPNetInfo struct {
	Address string        `json:"address"`
	Ports   []PortMapping `json:"ports"`
}

func NewActualLRPNetInfo(address string, ports []PortMapping) ActualLRPNetInfo {
	return ActualLRPNetInfo{
		Address: address,
		Ports:   ports,
	}
}

func (info *ActualLRPNetInfo) Empty() bool {
	return info.Address == "" && len(info.Ports) == 0
}

func (key ActualLRPNetInfo) Validate() error {
	var validationError ValidationError

	if key.Address == "" {
		return validationError.Append(ErrInvalidField{"address"})
	}

	return nil
}

const MaxCrashBackoff = 16 * time.Minute

type ActualLRPCrashInfo struct {
	CrashCount    int   `json:"crash_count"`
	LastCrashedAt int64 `json:"last_crashed_at"`
}

func NewActualLRPCrashInfo(crashCount int, lastCrashedAt int64) ActualLRPCrashInfo {
	return ActualLRPCrashInfo{
		CrashCount:    crashCount,
		LastCrashedAt: lastCrashedAt,
	}
}

func (crashInfo ActualLRPCrashInfo) Crashed() bool {
	return crashInfo.CrashCount > 3
}

func powerOfTwo(pow int) int64 {
	if pow < 0 {
		panic("pow cannot be negative")
	}
	return 1 << uint(pow)
}

type ActualLRP struct {
	ActualLRPKey
	ActualLRPContainerKey
	ActualLRPNetInfo
	ActualLRPCrashInfo
	State ActualLRPState `json:"state"`
	Since int64          `json:"since"`
}

type ActualLRPChange struct {
	Before ActualLRP
	After  ActualLRP
}

func (actual ActualLRP) ShouldStartUnclaimed(timestamp int64) bool {
	if actual.State != ActualLRPStateUnclaimed {
		return false
	}

	if actual.Since < timestamp {
		return true
	}
	return false
}

func (actual ActualLRP) ShouldRestartCrash(timestamp int64) bool {
	if actual.State != ActualLRPStateCrashed {
		return false
	}

	lastCrashedAt := actual.ActualLRPCrashInfo.LastCrashedAt
	count := actual.ActualLRPCrashInfo.CrashCount
	switch {
	case count < 3:
		return true

	case count < 8:
		threshhold := lastCrashedAt + (30*time.Second.Nanoseconds())*powerOfTwo(count-3)
		if threshhold <= timestamp {
			return true
		}

	case count < 200:
		threshhold := lastCrashedAt + MaxCrashBackoff.Nanoseconds()
		if threshhold <= timestamp {
			return true
		}
	}

	return false
}

func (before ActualLRP) AllowsTransitionTo(lrpKey ActualLRPKey, containerKey ActualLRPContainerKey, newState ActualLRPState) bool {
	if before.ActualLRPKey != lrpKey {
		return false
	}

	if before.State == ActualLRPStateClaimed && newState == ActualLRPStateRunning {
		return true
	}

	if (before.State == ActualLRPStateClaimed || before.State == ActualLRPStateRunning) &&
		(newState == ActualLRPStateClaimed || newState == ActualLRPStateRunning) &&
		(before.ActualLRPContainerKey != containerKey) {
		return false
	}

	return true
}

func (actual ActualLRP) Validate() error {
	var validationError ValidationError

	err := actual.ActualLRPKey.Validate()
	if err != nil {
		validationError = validationError.Append(err)
	}

	if actual.Since == 0 {
		validationError = validationError.Append(ErrInvalidField{"since"})
	}

	switch actual.State {
	case ActualLRPStateUnclaimed:
		if !actual.ActualLRPContainerKey.Empty() {
			validationError = validationError.Append(errors.New("container key cannot be set when state is unclaimed"))
		}
		if !actual.ActualLRPNetInfo.Empty() {
			validationError = validationError.Append(errors.New("net info cannot be set when state is unclaimed"))
		}

	case ActualLRPStateClaimed:
		if err := actual.ActualLRPContainerKey.Validate(); err != nil {
			validationError = validationError.Append(err)
		}
		if !actual.ActualLRPNetInfo.Empty() {
			validationError = validationError.Append(errors.New("net info cannot be set when state is claimed"))
		}

	case ActualLRPStateRunning:
		if err := actual.ActualLRPContainerKey.Validate(); err != nil {
			validationError = validationError.Append(err)
		}
		if err := actual.ActualLRPNetInfo.Validate(); err != nil {
			validationError = validationError.Append(err)
		}

	case ActualLRPStateCrashed:
		if !actual.ActualLRPContainerKey.Empty() {
			validationError = validationError.Append(errors.New("container key cannot be set when state is crashed"))
		}
		if !actual.ActualLRPNetInfo.Empty() {
			validationError = validationError.Append(errors.New("net info cannot be set when state is crashed"))
		}

	default:
		validationError = validationError.Append(ErrInvalidField{"state"})
	}

	if !validationError.Empty() {
		return validationError
	}

	return nil
}
