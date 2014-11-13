package models

type Freshness struct {
	Domain       string `json:"domain"`
	TTLInSeconds int    `json:"ttl_in_seconds"`
}

func (freshness Freshness) Validate() error {
	if freshness.Domain == "" {
		return ErrInvalidJSONMessage{"domain"}
	}

	if freshness.TTLInSeconds < 0 {
		return ErrInvalidJSONMessage{"ttl_in_seconds"}
	}

	return nil
}
