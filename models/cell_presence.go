package models

type CellPresence struct {
	CellID     string `json:"cell_id"`
	Stack      string `json:"stack"`
	RepAddress string `json:"rep_address"`
}

func (c CellPresence) Validate() error {
	var validationError ValidationError
	if c.CellID == "" {
		validationError = validationError.Append(ErrInvalidField{"cell_id"})
	}

	if c.Stack == "" {
		validationError = validationError.Append(ErrInvalidField{"stack"})
	}
	if c.RepAddress == "" {
		validationError = validationError.Append(ErrInvalidField{"rep_address"})
	}
	if len(validationError) > 0 {
		return validationError
	}
	return nil

	return nil
}
