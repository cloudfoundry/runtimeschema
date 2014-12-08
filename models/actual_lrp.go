package models

type ActualLRPState int

const (
	ActualLRPStateInvalid ActualLRPState = iota
	ActualLRPStateStarting
	ActualLRPStateRunning
)

type ActualLRPChange struct {
	Before *ActualLRP
	After  *ActualLRP
}

type ActualLRP struct {
	ProcessGuid  string `json:"process_guid"`
	InstanceGuid string `json:"instance_guid"`
	CellID       string `json:"cell_id"`
	Domain       string `json:"domain"`

	Index int `json:"index"`

	Host  string        `json:"host"`
	Ports []PortMapping `json:"ports"`

	State ActualLRPState `json:"state"`
	Since int64          `json:"since"`
}

type ActualLRPsByIndex map[int]ActualLRP

func NewActualLRP(
	processGuid string,
	instanceGuid string,
	cellID string,
	domain string,
	index int,
) (ActualLRP, error) {

	lrp := ActualLRP{
		ProcessGuid:  processGuid,
		InstanceGuid: instanceGuid,
		CellID:       cellID,
		Domain:       domain,

		Index: index,
	}

	err := lrp.Validate()
	if err != nil {
		return ActualLRP{}, err
	}

	return lrp, nil
}

func (actual ActualLRP) Validate() error {
	var validationError ValidationError

	if actual.ProcessGuid == "" {
		validationError = append(validationError, ErrInvalidField{"process_guid"})
	}

	if actual.InstanceGuid == "" {
		validationError = append(validationError, ErrInvalidField{"instance_guid"})
	}

	if actual.CellID == "" {
		validationError = append(validationError, ErrInvalidField{"cell_id"})
	}

	if actual.Domain == "" {
		validationError = append(validationError, ErrInvalidField{"domain"})
	}

	if len(validationError) > 0 {
		return validationError
	}

	return nil
}
