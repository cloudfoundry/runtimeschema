package shared

import (
	"path"
	"strconv"

	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

const SchemaRoot = "/v1/"
const CellSchemaRoot = SchemaRoot + "cell"
const LRPStartAuctionSchemaRoot = SchemaRoot + "start"
const LRPStopAuctionSchemaRoot = SchemaRoot + "stop"
const ActualLRPSchemaRoot = SchemaRoot + "actual"
const DesiredLRPSchemaRoot = SchemaRoot + "desired"
const TaskSchemaRoot = SchemaRoot + "task"
const LockSchemaRoot = SchemaRoot + "locks"
const FreshnessSchemaRoot = SchemaRoot + "freshness"

func CellSchemaPath(cellID string) string {
	return path.Join(CellSchemaRoot, cellID)
}

func LRPStartAuctionProcessDir(lrp models.LRPStartAuction) string {
	return path.Join(LRPStartAuctionSchemaRoot, lrp.DesiredLRP.ProcessGuid)
}

func LRPStartAuctionSchemaPath(lrp models.LRPStartAuction) string {
	return path.Join(LRPStartAuctionProcessDir(lrp), strconv.Itoa(lrp.Index))
}

func LRPStopAuctionProcessDir(lrp models.LRPStopAuction) string {
	return path.Join(LRPStopAuctionSchemaRoot, lrp.ProcessGuid)
}

func LRPStopAuctionSchemaPath(lrp models.LRPStopAuction) string {
	return path.Join(LRPStopAuctionProcessDir(lrp), strconv.Itoa(lrp.Index))
}

func ActualLRPProcessDir(processGuid string) string {
	return path.Join(ActualLRPSchemaRoot, processGuid)
}

func ActualLRPIndexDir(processGuid string, index int) string {
	return path.Join(ActualLRPProcessDir(processGuid), strconv.Itoa(index))
}

func ActualLRPSchemaPath(processGuid string, index int, instanceGuid string) string {
	return path.Join(ActualLRPIndexDir(processGuid, index), instanceGuid)
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

func LockSchemaPath(lockName string) string {
	return path.Join(LockSchemaRoot, lockName)
}

func FreshnessSchemaPath(domain string) string {
	return path.Join(FreshnessSchemaRoot, domain)
}
