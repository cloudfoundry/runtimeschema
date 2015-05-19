package test_helpers

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/gomega"
)

func (t *TestHelper) RegisterCell(cell models.CellPresence) {
	jsonBytes, err := models.ToJSON(cell)
	Expect(err).NotTo(HaveOccurred())

	err = t.consulSession.AcquireLock(shared.CellSchemaPath(cell.CellID), jsonBytes)
	Expect(err).NotTo(HaveOccurred())
}

func (t *TestHelper) RegisterAuctioneer(auctioneer models.AuctioneerPresence) {
	jsonBytes, err := models.ToJSON(auctioneer)
	Expect(err).NotTo(HaveOccurred())

	err = t.consulSession.AcquireLock(shared.LockSchemaPath("auctioneer_lock"), jsonBytes)
	Expect(err).NotTo(HaveOccurred())
}
