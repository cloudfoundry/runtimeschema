package models

type LRPStart struct {
	DesiredLRP DesiredLRP `json:"desired_lrp"`
	Index      int        `json:"index"`
}

func (lrpstart LRPStart) Validate() error {
	var validationError ValidationError

	err := lrpstart.DesiredLRP.Validate()
	if err != nil {
		validationError = append(validationError, err)
	}

	if len(validationError) > 0 {
		return validationError
	}

	return nil
}
