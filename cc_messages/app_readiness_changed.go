package cc_messages

type AppReadinessChangedRequest struct {
	Instance string `json:"instance"`
	Index    int    `json:"index"`
	CellID   string `json:"cell_id"`
	Ready    bool   `json:"ready"`
}
