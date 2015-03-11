package repositories_test

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/repositories"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/go-gorp/gorp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DesiredLRPRepository", func() {
	var (
		repository repositories.DesiredLRPRepository

		dbmap *gorp.DbMap
		db    *sql.DB

		desired models.DesiredLRP
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

		_, err = dbmap.Exec("drop table if exists desired_lrps")
		Ω(err).ShouldNot(HaveOccurred())

		repository, err = repositories.NewDesiredLRPRepository(dbmap)
		Ω(err).ShouldNot(HaveOccurred())

		rawMessage := json.RawMessage([]byte(`{"port":8080,"hosts":["route-1","route-2"]}`))
		desired = models.DesiredLRP{
			Domain:      "some-domain",
			ProcessGuid: "some-guid",

			Instances:  1,
			Stack:      "some-stack",
			RootFSPath: "docker:///docker.com/docker",
			MemoryMB:   1024,
			DiskMB:     512,
			CPUWeight:  42,
			Privileged: true,
			Routes: map[string]*json.RawMessage{
				"router": &rawMessage,
			},
			Annotation: "some-annotation",
			Ports: []uint16{
				5678,
			},
			LogGuid:     "log-guid",
			LogSource:   "the cloud",
			MetricsGuid: "metrics-guid",
			ModificationTag: models.ModificationTag{
				Epoch: "some-epoch",
				Index: 50,
			},
			EnvironmentVariables: []models.EnvironmentVariable{
				{
					Name:  "ENV_VAR_NAME",
					Value: "some environment variable value",
				},
			},
			Setup: &models.DownloadAction{
				From: "http://example.com",
				To:   "/tmp/internet",
			},
			Action: &models.RunAction{
				Path: "ls",
			},
			Monitor: &models.RunAction{
				Path: "reboot",
			},
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
		}
	})

	Describe("Create", func() {
		It("inserts a row into the database", func() {
			result, err := repository.Create(dbmap, desired)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(result).Should(Equal(desired))

			foundDesired, _, err := repository.GetByProcessGuid(dbmap, desired.ProcessGuid)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(foundDesired).Should(Equal(result))
		})
	})

	Describe("GetAll", func() {
		var anotherDesiredLRP models.DesiredLRP

		BeforeEach(func() {
			anotherDesiredLRP = desired
			anotherDesiredLRP.ProcessGuid = "another-process-guid"

			_, err := repository.Create(dbmap, desired)
			Ω(err).ShouldNot(HaveOccurred())

			_, err = repository.Create(dbmap, anotherDesiredLRP)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("gets all of the desired LRPs", func() {
			lrps, err := repository.GetAll(dbmap)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(lrps).ShouldNot(BeNil())
			Ω(lrps).Should(ConsistOf(desired, anotherDesiredLRP))
		})
	})

	Describe("GetByProcessGuid", func() {

		BeforeEach(func() {
			_, err := repository.Create(dbmap, desired)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("returns the with the requested process guid", func() {
			lrp, _, err := repository.GetByProcessGuid(dbmap, desired.ProcessGuid)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(lrp).Should(Equal(desired))
		})

		It("returns the version index", func() {
			_, idx, err := repository.GetByProcessGuid(dbmap, desired.ProcessGuid)
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

	Describe("GetAllByDomain", func() {
		var desired2, desired3 models.DesiredLRP

		BeforeEach(func() {
			desired2 = desired
			desired2.ProcessGuid = "desired-process-guid-2"

			desired3 = desired
			desired3.ProcessGuid = "desired-process-guid-3"
			desired3.Domain = "some-other-domain"

			_, err := repository.Create(dbmap, desired)
			Ω(err).ShouldNot(HaveOccurred())

			_, err = repository.Create(dbmap, desired2)
			Ω(err).ShouldNot(HaveOccurred())

			_, err = repository.Create(dbmap, desired3)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("returns all lrps in the requested domain", func() {
			lrps, err := repository.GetAllByDomain(dbmap, "some-domain")
			Ω(err).ShouldNot(HaveOccurred())

			Ω(lrps).Should(ConsistOf(desired, desired2))
		})

		It("does not return lrps that are not in the requested domain", func() {
			lrps, err := repository.GetAllByDomain(dbmap, "some-domain")
			Ω(err).ShouldNot(HaveOccurred())

			Ω(lrps).ShouldNot(ContainElement(desired3))
		})

		Context("when no lrps are in the requested domain", func() {
			It("returns an empty list", func() {
				lrps, err := repository.GetAllByDomain(dbmap, "a-non-existent-guid")
				Ω(err).ShouldNot(HaveOccurred())
				Ω(lrps).Should(BeEmpty())
			})
		})
	})

	Describe("DeleteByProcessGuid", func() {
		BeforeEach(func() {
			_, err := repository.Create(dbmap, desired)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("deletes the row associated with the requested process guid", func() {
			err := repository.DeleteByProcessGuid(dbmap, desired.ProcessGuid)
			Ω(err).ShouldNot(HaveOccurred())
		})

		Context("when the requested guid does not exist", func() {
			It("returns a resource not found error", func() {
				err := repository.DeleteByProcessGuid(dbmap, "a-non-existent-guid")
				Ω(err).Should(Equal(bbserrors.ErrStoreResourceNotFound))
			})
		})
	})

	Describe("UpdateDesiredLRP", func() {
		var updateRequest models.DesiredLRPUpdate

		BeforeEach(func() {
			_, err := repository.Create(dbmap, desired)
			Ω(err).ShouldNot(HaveOccurred())

			instances := 3
			annotation := "a brand new annotation"
			routes := json.RawMessage([]byte(`"routing info"`))

			updateRequest.Instances = &instances
			updateRequest.Annotation = &annotation
			updateRequest.Routes = map[string]*json.RawMessage{
				"routes": &routes,
			}
		})

		It("updates the desired LRP with the requested process guid", func() {
			updated, err := repository.UpdateDesiredLRP(dbmap, desired.ProcessGuid, updateRequest)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(updated.Instances).Should(Equal(*updateRequest.Instances))
			Ω(updated.Annotation).Should(Equal(*updateRequest.Annotation))
			Ω(updated.Routes).Should(Equal(updateRequest.Routes))

			foundDesired, _, err := repository.GetByProcessGuid(dbmap, desired.ProcessGuid)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(updated).Should(Equal(foundDesired))
		})
	})
})
