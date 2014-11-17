package models

import "encoding/json"

type CellPresence struct {
	CellID string `json:"cell_id"`
	Stack  string `json:"stack"`
}

func NewCellPresenceFromJSON(payload []byte) (CellPresence, error) {
	var task CellPresence

	err := json.Unmarshal(payload, &task)
	if err != nil {
		return CellPresence{}, err
	}

	return task, nil
}

func (presence CellPresence) ToJSON() []byte {
	bytes, err := json.Marshal(presence)
	if err != nil {
		panic(err)
	}

	return bytes
}
