package shared

import (
	"path"
	"strconv"

	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

const DataSchemaRoot = "/v1/"
const ActualLRPSchemaRoot = DataSchemaRoot + "actual"
const DesiredLRPSchemaRoot = DataSchemaRoot + "desired"
const TaskSchemaRoot = DataSchemaRoot + "task"
const DomainSchemaRoot = DataSchemaRoot + "domain"

const ActualLRPInstanceKey = "instance"
const ActualLRPEvacuatingKey = "evacuating"

func ActualLRPProcessDir(processGuid string) string {
	return path.Join(ActualLRPSchemaRoot, processGuid)
}

func ActualLRPIndexDir(processGuid string, index int) string {
	return path.Join(ActualLRPProcessDir(processGuid), strconv.Itoa(index))
}

func ActualLRPSchemaPath(processGuid string, index int) string {
	return path.Join(ActualLRPIndexDir(processGuid, index), ActualLRPInstanceKey)
}

func EvacuatingActualLRPSchemaPath(processGuid string, index int) string {
	return path.Join(ActualLRPIndexDir(processGuid, index), ActualLRPEvacuatingKey)
}

func DesiredLRPSchemaPath(lrp models.DesiredLRP) string {
	return DesiredLRPSchemaPathByProcessGuid(lrp.ProcessGuid)
}

func DesiredLRPSchemaPathByProcessGuid(processGuid string) string {
	return path.Join(DesiredLRPSchemaRoot, processGuid)
}

func TaskSchemaPath(taskGuid string) string {
	return path.Join(TaskSchemaRoot, taskGuid)
}

func DomainSchemaPath(domain string) string {
	return path.Join(DomainSchemaRoot, domain)
}

const LockSchemaRoot = "v1/locks"
const CellSchemaRoot = LockSchemaRoot + "/cell"
const ReceptorSchemaRoot = LockSchemaRoot + "/receptor"

func LockSchemaPath(lockName string) string {
	return path.Join(LockSchemaRoot, lockName)
}

func CellSchemaPath(cellID string) string {
	return path.Join(CellSchemaRoot, cellID)
}

func ReceptorSchemaPath(receptorID string) string {
	return path.Join(ReceptorSchemaRoot, receptorID)
}
