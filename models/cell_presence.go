package models

type CellPresence struct {
	CellID     string `json:"cell_id"`
	Stack      string `json:"stack"`
	RepAddress string `json:"rep_address"`
}
