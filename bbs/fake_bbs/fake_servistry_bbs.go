package fake_bbs

import (
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"time"
)

type RegisterCCInputs struct {
	Registration models.CCRegistrationMessage
	Ttl          time.Duration
}

type UnregisterCCInputs struct {
	Registration models.CCRegistrationMessage
}

type FakeServistryBBS struct {
	RegisterCCInputs  []RegisterCCInputs
	RegisterCCOutputs struct {
		Err error
	}

	UnregisterCCInputs  []UnregisterCCInputs
	UnregisterCCOutputs struct {
		Err error
	}
}

func NewFakeServistryBBS() *FakeServistryBBS {
	return &FakeServistryBBS{}
}

func (bbs *FakeServistryBBS) RegisterCC(registration models.CCRegistrationMessage, ttl time.Duration) error {
	bbs.RegisterCCInputs = append(bbs.RegisterCCInputs, RegisterCCInputs{
		Registration: registration,
		Ttl:          ttl,
	})
	return bbs.RegisterCCOutputs.Err
}

func (bbs *FakeServistryBBS) UnregisterCC(registration models.CCRegistrationMessage) error {
	bbs.UnregisterCCInputs = append(bbs.UnregisterCCInputs, UnregisterCCInputs{
		Registration: registration,
	})
	return bbs.UnregisterCCOutputs.Err
}
