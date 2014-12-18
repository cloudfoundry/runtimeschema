package models

type LRPStart struct {
	DesiredLRP DesiredLRP `json:"desired_lrp"`
	Index      int        `json:"index"`
}

func (lrpstart LRPStart) Validate() error {
	var validationError ValidationError

	err := lrpstart.DesiredLRP.Validate()
	if err != nil {
		validationError = validationError.Append(err)
	}

	if lrpstart.Index < 0 {
		validationError = validationError.Append(ErrInvalidField{"index"})
	}

	if len(validationError) > 0 {
		return validationError
	}

	return nil
}
