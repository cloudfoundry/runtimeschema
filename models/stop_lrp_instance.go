package models

type StopLRPInstance struct {
	ProcessGuid  string `json:"process_guid"`
	InstanceGuid string `json:"instance_guid"`
	Index        int    `json:"index"`
}

func (stop StopLRPInstance) Validate() error {
	var validationError ValidationError

	if stop.ProcessGuid == "" {
		validationError = append(validationError, ErrInvalidJSONMessage{"process_guid"})
	}

	if stop.InstanceGuid == "" {
		validationError = append(validationError, ErrInvalidJSONMessage{"instance_guid"})
	}

	if len(validationError) > 0 {
		return validationError
	}

	return nil
}
