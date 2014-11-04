package models

type ErrInvalidJSONMessage struct {
	MissingField string
}

func (err ErrInvalidJSONMessage) Error() string {
	return "JSON has missing/invalid field: " + err.MissingField
}

type ErrInvalidModification struct {
	InvalidField string
}

func (err ErrInvalidModification) Error() string {
	return "attempt to make invalid change to field: " + err.InvalidField
}
