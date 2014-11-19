package models

import "bytes"

type ErrInvalidField struct {
	Field string
}

func (err ErrInvalidField) Error() string {
	return "Invalid field: " + err.Field
}

type ErrInvalidModification struct {
	InvalidField string
}

func (err ErrInvalidModification) Error() string {
	return "attempt to make invalid change to field: " + err.InvalidField
}

type Validator interface {
	Validate() error
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
