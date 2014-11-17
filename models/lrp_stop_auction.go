package models

type LRPStopAuctionState int

const (
	LRPStopAuctionStateInvalid LRPStopAuctionState = iota
	LRPStopAuctionStatePending
	LRPStopAuctionStateClaimed
)

type LRPStopAuction struct {
	ProcessGuid string `json:"process_guid"`
	Index       int    `json:"index"`

	State     LRPStopAuctionState `json:"state"`
	UpdatedAt int64               `json:"updated_at"`
}

func (auction LRPStopAuction) Validate() error {
	if auction.ProcessGuid == "" {
		return ValidationError{ErrInvalidJSONMessage{"process_guid"}}
	}

	return nil
}
