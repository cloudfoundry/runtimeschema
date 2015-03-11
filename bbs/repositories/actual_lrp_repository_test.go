package repositories_test

import (
	"database/sql"
	"log"
	"os"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/repositories"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/go-gorp/gorp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ActualLRPRepository", func() {

	var (
		repository repositories.ActualLRPRepository

		dbmap *gorp.DbMap
		db    *sql.DB

		actual       models.ActualLRP
		lrpKey       models.ActualLRPKey
		containerKey models.ActualLRPContainerKey
		netInfo      models.ActualLRPNetInfo
	)

	BeforeEach(func() {
		var err error
		db, err = sql.Open("mysql", "root:password@tcp(127.0.0.1:3306)/bbs")
		Ω(err).ShouldNot(HaveOccurred())

		dbmap = &gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{
			Engine:   "InnoDB",
			Encoding: "UTF8",
		}}

		dbmap.TraceOn("[DB] ", log.New(os.Stdout, "", 0))

		_, err = dbmap.Exec("drop table if exists actual_lrps")
		Ω(err).ShouldNot(HaveOccurred())

		repository, err = repositories.NewActualLRPRepository(dbmap)
		Ω(err).ShouldNot(HaveOccurred())

		lrpKey = models.NewActualLRPKey("some-guid", 2, "some-domain")
		containerKey = models.NewActualLRPContainerKey("some-instance-guid", "some-cell-id")
		netInfo = models.NewActualLRPNetInfo("1.2.3.4", []models.PortMapping{
			{ContainerPort: 8080},
			{ContainerPort: 8081, HostPort: 1234},
		})

		actual = models.ActualLRP{
			ActualLRPKey:          lrpKey,
			ActualLRPContainerKey: containerKey,
			ActualLRPNetInfo:      netInfo,
			CrashCount:            1,
			State:                 models.ActualLRPStateRunning,
			Since:                 1138,
			ModificationTag: models.ModificationTag{
				Epoch: "some-guid",
				Index: 50,
			},
		}
	})

	Describe("Create", func() {
		It("inserts a row into the database", func() {
			result, err := repository.Create(dbmap, actual)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(result).Should(Equal(actual))

			foundActual, _, err := repository.GetByProcessGuid(dbmap, actual.ProcessGuid)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(foundActual).Should(Equal(result))
		})
	})

	Describe("GetByProcessGuid", func() {

		BeforeEach(func() {
			_, err := repository.Create(dbmap, actual)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("returns the with the requested process guid", func() {
			lrp, _, err := repository.GetByProcessGuid(dbmap, actual.ProcessGuid)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(lrp).Should(Equal(actual))
		})

		It("returns the version index", func() {
			_, idx, err := repository.GetByProcessGuid(dbmap, actual.ProcessGuid)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(idx).Should(BeEquivalentTo(1))
		})

		Context("when the process doesn't exist", func() {
			It("returns a resource not found error", func() {
				_, _, err := repository.GetByProcessGuid(dbmap, "a-non-existent-guid")
				Ω(err).Should(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		})
	})
})
