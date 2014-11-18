package models

import "bytes"

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

type ValidationError []error

func (ve ValidationError) Error() string {
	var buffer bytes.Buffer

	for i, err := range ve {
		if i > 0 {
			buffer.WriteString(", ")
		}
		buffer.WriteString(err.Error())
	}

	return buffer.String()
}
