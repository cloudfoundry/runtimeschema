package models

type ReceptorPresence struct {
	ReceptorID  string `json:"id"`
	ReceptorURL string `json:"address"`
}

func (r ReceptorPresence) Validate() error {
	var validationError ValidationError
	if r.ReceptorID == "" {
		validationError = validationError.Append(ErrInvalidField{"id"})
	}

	if r.ReceptorURL == "" {
		validationError = validationError.Append(ErrInvalidField{"address"})
	}
	if len(validationError) > 0 {
		return validationError
	}
	return nil
}
