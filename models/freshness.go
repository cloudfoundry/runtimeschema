package models

type Freshness struct {
	Domain       string `json:"domain"`
	TTLInSeconds int    `json:"ttl_in_seconds"`
}

func (freshness Freshness) Validate() error {
	var validationError ValidationError

	if freshness.Domain == "" {
		validationError = append(validationError, ErrInvalidField{"domain"})
	}

	if freshness.TTLInSeconds < 0 {
		validationError = append(validationError, ErrInvalidField{"ttl_in_seconds"})
	}

	if len(validationError) > 0 {
		return validationError
	}

	return nil
}
