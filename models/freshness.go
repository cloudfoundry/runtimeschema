package models

type Freshness struct {
	Domain       string `json:"domain"`
	TTLInSeconds int    `json:"ttl_in_seconds"`
}

func (freshness Freshness) Validate() error {
	var validationError ValidationError

	if freshness.Domain == "" {
		validationError = validationError.Append(ErrInvalidField{"domain"})
	}

	if freshness.TTLInSeconds < 0 {
		validationError = validationError.Append(ErrInvalidField{"ttl_in_seconds"})
	}

	if !validationError.Empty() {
		return validationError
	}

	return nil
}
