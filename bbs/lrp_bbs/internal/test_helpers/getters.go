package test_helpers

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"

	. "github.com/onsi/gomega"
)

func (t *TestHelper) GetInstanceActualLRP(lrpKey models.ActualLRPKey) (models.ActualLRP, error) {
	node, err := t.etcdClient.Get(shared.ActualLRPSchemaPath(lrpKey.ProcessGuid, lrpKey.Index))
	if err == storeadapter.ErrorKeyNotFound {
		return models.ActualLRP{}, bbserrors.ErrStoreResourceNotFound
	}
	Expect(err).NotTo(HaveOccurred())

	var lrp models.ActualLRP
	err = models.FromJSON(node.Value, &lrp)
	Expect(err).NotTo(HaveOccurred())

	return lrp, nil
}
