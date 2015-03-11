package volume_bbs

import (
	"errors"
	"fmt"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs/bbserrors"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/services_bbs"
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/cb"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/storeadapter"
	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"
)

type VolumeBBS struct {
	store            storeadapter.StoreAdapter
	clock            clock.Clock
	auctioneerClient cb.AuctioneerClient
	services         *services_bbs.ServicesBBS
}

func New(
	store storeadapter.StoreAdapter,
	clock clock.Clock,
	auctioneerClient cb.AuctioneerClient,
	services *services_bbs.ServicesBBS,
) *VolumeBBS {
	return &VolumeBBS{
		store:            store,
		clock:            clock,
		auctioneerClient: auctioneerClient,
		services:         services,
	}
}

func (bbs *VolumeBBS) DesireVolumeSet(logger lager.Logger, vol models.VolumeSet) error {
	logger = logger.Session("desire-volume-set", lager.Data{"volume-set-guid": vol.VolumeSetGuid})
	logger.Info("starting")
	defer logger.Info("complete")

	vol.Since = bbs.clock.Now().UnixNano()

	value, err := models.ToJSON(vol)
	if err != nil {
		return err
	}

	err = bbs.store.Create(storeadapter.StoreNode{
		Key:   shared.VolumeSetSchemaPath(vol.VolumeSetGuid),
		Value: value,
	})
	if err != nil {
		logger.Error("failed-to-create-volume-set", err)
		return shared.ConvertStoreError(err)
	}

	err = bbs.createAndStartInstanceRange(logger, 0, vol.Instances, vol)
	if err != nil {
		logger.Error("failed-to-request-volume-auction", err)
		return err
	}
	return nil
}

func (bbs *VolumeBBS) volumeWithIndex(logger lager.Logger, volumeSetGuid string, index int) (*models.Volume, uint64, error) {
	node, err := bbs.store.Get(shared.VolumeSchemaPath(volumeSetGuid, index))
	if err != nil {
		if err != storeadapter.ErrorKeyNotFound {
			logger.Error("failed-to-get-volume", err)
		}
		return nil, 0, shared.ConvertStoreError(err)
	}

	vol := &models.Volume{}
	err = models.FromJSON(node.Value, vol)

	if err != nil {
		logger.Error("failed-to-unmarshal-volume", err)
		return nil, 0, err
	}

	return vol, node.Index, err
}

func (bbs *VolumeBBS) VolumesByVolumeSetGuid(logger lager.Logger, volSetGuid string) ([]models.Volume, error) {
	vols := []models.Volume{}

	if volSetGuid == "" {
		return vols, bbserrors.ErrNoVolumeSetGuid
	}

	node, err := bbs.store.ListRecursively(shared.VolumeDir(volSetGuid))
	if err == storeadapter.ErrorKeyNotFound {
		return vols, nil
	} else if err != nil {
		return vols, shared.ConvertStoreError(err)
	}

	for _, instanceNode := range node.ChildNodes {
		var vol models.Volume
		err = models.FromJSON(instanceNode.Value, &vol)
		if err != nil {
			return vols, fmt.Errorf("cannot parse lrp JSON for key %s: %s", instanceNode.Key, err.Error())
		} else {
			vols = append(vols, vol)
		}
	}

	return vols, nil
}

func (bbs *VolumeBBS) RunVolume(logger lager.Logger, volSetGuid string, index int, volGuid, cellID string) error {
	vol, storeIndex, err := bbs.volumeWithIndex(logger, volSetGuid, index)
	if err != nil {
		logger.Error("failed-to-get-volume", err, lager.Data{"volume-set-guid": volSetGuid, "index": index})
		return shared.ConvertStoreError(err)
	}
	if vol.State != models.VolumeStatePending {
		stateErr := errors.New("can only transition from Pending")
		logger.Error("invalid-existing-volume-state", stateErr, lager.Data{"state": vol.State})
		return stateErr
	}

	vol.VolumeGuid = volGuid
	vol.CellID = cellID
	vol.State = models.VolumeStateRunning
	vol.Since = bbs.clock.Now().UnixNano()
	value, err := models.ToJSON(vol)
	if err != nil {
		logger.Error("failed-to-marshal-volume", err, lager.Data{"volume": vol})
		return err
	}

	err = bbs.store.CompareAndSwapByIndex(storeIndex, storeadapter.StoreNode{
		Key:   shared.VolumeSchemaPath(vol.VolumeSetGuid, vol.Index),
		Value: value,
	})

	if err != nil {
		logger.Error("failed-to-update-volume", err, lager.Data{"volume": vol})
		return shared.ConvertStoreError(err)
	}

	return nil
}

func (bbs *VolumeBBS) FailVolume(logger lager.Logger, volSetGuid string, index int, placementError string) error {
	vol, storeIndex, err := bbs.volumeWithIndex(logger, volSetGuid, index)
	if err != nil {
		logger.Error("failed-to-get-volume", err, lager.Data{"volume-set-guid": volSetGuid, "index": index})
		return shared.ConvertStoreError(err)
	}
	vol.State = models.VolumeStateFailed
	vol.Since = bbs.clock.Now().UnixNano()
	vol.PlacementError = placementError
	value, err := models.ToJSON(vol)
	if err != nil {
		logger.Error("failed-to-marshal-volume", err, lager.Data{"volume": vol})
		return err
	}

	err = bbs.store.CompareAndSwapByIndex(storeIndex, storeadapter.StoreNode{
		Key:   shared.VolumeSchemaPath(vol.VolumeSetGuid, vol.Index),
		Value: value,
	})

	if err != nil {
		logger.Error("failed-to-update-volume", err, lager.Data{"volume": vol})
		return shared.ConvertStoreError(err)
	}

	return nil
}

func (bbs *VolumeBBS) createVolume(logger lager.Logger, volSet models.VolumeSet, index int) error {
	logger = logger.Session("create-volume")

	volume := models.Volume{
		VolumeSetGuid:    volSet.VolumeSetGuid,
		SizeMB:           volSet.SizeMB,
		ReservedMemoryMB: volSet.ReservedMemoryMB,
		Index:            index,
		State:            models.VolumeStatePending,
		Since:            bbs.clock.Now().UnixNano(),
	}

	value, err := models.ToJSON(volume)
	if err != nil {
		logger.Error("failed-to-marshal-volume", err, lager.Data{"volume": volume})
		return err
	}

	err = bbs.store.Create(storeadapter.StoreNode{
		Key:   shared.VolumeSchemaPath(volume.VolumeSetGuid, volume.Index),
		Value: value,
	})

	if err != nil {
		logger.Error("failed-to-create-volume", err, lager.Data{"volume": volume})
		return shared.ConvertStoreError(err)
	}

	return nil
}

func (bbs *VolumeBBS) createAndStartInstanceRange(logger lager.Logger, lower, upper int, volSet models.VolumeSet) error {
	logger = logger.Session("start-instance-range", lager.Data{"lower": lower, "upper": upper})
	logger.Info("starting")
	defer logger.Info("complete")

	indices := make([]uint, upper-lower)
	i := 0
	for actualIndex := lower; actualIndex < upper; actualIndex++ {
		err := bbs.createVolume(logger, volSet, actualIndex)
		if err != nil {
			return err
		}
		indices[i] = uint(actualIndex)
		i++
	}

	startRequest := models.VolumeStartRequest{
		VolumeSet: volSet,
		Indices:   indices,
	}

	return bbs.requestVolumeAuctions([]models.VolumeStartRequest{startRequest})
}

func (bbs *VolumeBBS) requestVolumeAuctions(volStarts []models.VolumeStartRequest) error {
	auctioneerAddress, err := bbs.services.AuctioneerAddress()
	if err != nil {
		return err
	}

	return bbs.auctioneerClient.RequestVolumeAuctions(auctioneerAddress, volStarts)
}
