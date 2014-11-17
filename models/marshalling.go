package models

import "encoding/json"

type Validator interface {
	Validate() error
}

func FromJSON(payload []byte, i interface{}) error {
	err := json.Unmarshal(payload, i)
	if err != nil {
		return err
	}

	if v, ok := i.(Validator); ok {
		return v.Validate()
	}

	return nil
}

func ToJSON(i interface{}) ([]byte, error) {
	return json.Marshal(i)
}
