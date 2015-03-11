package repositories_test

import (
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/repositories"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/go-gorp/gorp"
	_ "github.com/go-sql-driver/mysql"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("TaskRepository", func() {
	var (
		repository repositories.TaskRepository

		dbmap *gorp.DbMap
		db    *sql.DB
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

		_, err = dbmap.Exec("drop table if exists tasks")
		Ω(err).ShouldNot(HaveOccurred())

		repository, err = repositories.NewTaskRepository(dbmap)
		Ω(err).ShouldNot(HaveOccurred())
	})

	Describe("Create", func() {
		var task models.Task

		BeforeEach(func() {
			task = models.Task{
				TaskGuid:   "my-task-guid",
				Domain:     "my-domain",
				RootFSPath: "docker:///docker.com/docker",
				Stack:      "some-stack",
				EnvironmentVariables: []models.EnvironmentVariable{
					{
						Name:  "ENV_VAR_NAME",
						Value: "an environmment value",
					},
				},
				Action: &models.DownloadAction{
					From:     "old_location",
					To:       "new_location",
					CacheKey: "the-cache-key",
				},
				MemoryMB:         256,
				DiskMB:           1024,
				CPUWeight:        42,
				Privileged:       true,
				LogGuid:          "123",
				LogSource:        "APP",
				MetricsGuid:      "456",
				CreatedAt:        time.Date(2014, time.February, 25, 23, 46, 11, 00, time.UTC).UnixNano(),
				UpdatedAt:        time.Date(2014, time.February, 25, 23, 46, 11, 10, time.UTC).UnixNano(),
				FirstCompletedAt: time.Date(2014, time.February, 25, 23, 46, 11, 30, time.UTC).UnixNano(),
				ResultFile:       "some-file.txt",
				State:            models.TaskStatePending,
				CellID:           "cell",

				Result:        "turboencabulated",
				Failed:        true,
				FailureReason: "because i said so",

				EgressRules: []models.SecurityGroupRule{
					{
						Protocol:     "tcp",
						Destinations: []string{"0.0.0.0/0"},
						PortRange: &models.PortRange{
							Start: 1,
							End:   1024,
						},
						Log: true,
					},
					{
						Protocol:     "udp",
						Destinations: []string{"8.8.0.0/16"},
						Ports:        []uint16{53},
					},
				},

				Annotation: `[{"anything": "you want!"}]... dude`,
			}
		})

		It("inserts a row into the database", func() {
			result, err := repository.Create(dbmap, task)
			Ω(err).ShouldNot(HaveOccurred())

			foundTask, _, err := repository.GetByTaskGuid(dbmap, "my-task-guid")
			Ω(err).ShouldNot(HaveOccurred())

			Ω(foundTask).Should(Equal(result))
		})

		Context("when inserting a task with an existing guid", func() {
			BeforeEach(func() {
				_, err := repository.Create(dbmap, task)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("raises an error", func() {
				_, err := repository.Create(dbmap, task)
				Ω(err).Should(HaveOccurred())
			})
		})
	})

})
