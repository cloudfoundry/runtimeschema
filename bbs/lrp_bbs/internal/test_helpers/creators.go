package test_helpers

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"

	. "github.com/onsi/gomega"
)

func (t *TestHelper) SetRawActualLRP(lrp models.ActualLRP) {
	value, err := json.Marshal(lrp) // do NOT use models.ToJSON; don't want validations
	Expect(err).NotTo(HaveOccurred())

	err = t.etcdClient.SetMulti([]storeadapter.StoreNode{
		{
			Key:   shared.ActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index),
			Value: value,
		},
	})

	Expect(err).NotTo(HaveOccurred())
}

func (t *TestHelper) SetRawEvacuatingActualLRP(lrp models.ActualLRP, ttlInSeconds uint64) {
	value, err := json.Marshal(lrp) // do NOT use models.ToJSON; don't want validations
	Expect(err).NotTo(HaveOccurred())

	err = t.etcdClient.SetMulti([]storeadapter.StoreNode{
		{
			Key:   shared.EvacuatingActualLRPSchemaPath(lrp.ProcessGuid, lrp.Index),
			Value: value,
			TTL:   ttlInSeconds,
		},
	})

	Expect(err).NotTo(HaveOccurred())
}

func (t *TestHelper) SetRawDesiredLRP(d models.DesiredLRP) {
	value, err := json.Marshal(d) // do NOT use models.ToJSON; don't want validations
	Expect(err).NotTo(HaveOccurred())

	err = t.etcdClient.SetMulti([]storeadapter.StoreNode{
		{
			Key:   shared.DesiredLRPSchemaPath(d),
			Value: value,
		},
	})

	Expect(err).NotTo(HaveOccurred())
}

func (t *TestHelper) CreateValidDesiredLRP(guid string) {
	t.SetRawDesiredLRP(t.NewValidDesiredLRP(guid))
}

func (t *TestHelper) CreateValidActualLRP(guid string, index int) {
	t.SetRawActualLRP(t.NewValidActualLRP(guid, index))
}

func (t *TestHelper) CreateValidEvacuatingLRP(guid string, index int) {
	t.SetRawEvacuatingActualLRP(t.NewValidActualLRP(guid, index), 100)
}

func (t *TestHelper) CreateMalformedDesiredLRP(guid string) {
	t.createMalformedValueForKey(shared.DesiredLRPSchemaPath(models.DesiredLRP{ProcessGuid: guid}))
}

func (t *TestHelper) CreateMalformedActualLRP(guid string, index int) {
	t.createMalformedValueForKey(shared.ActualLRPSchemaPath(guid, index))
}

func (t *TestHelper) CreateMalformedEvacuatingLRP(guid string, index int) {
	t.createMalformedValueForKey(shared.EvacuatingActualLRPSchemaPath(guid, index))
}

func (t *TestHelper) createMalformedValueForKey(key string) {
	err := t.etcdClient.Create(storeadapter.StoreNode{
		Key:   key,
		Value: []byte("ßßßßßß"),
	})

	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("error occurred at key '%s'", key))
}
