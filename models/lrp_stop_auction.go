package models

import "encoding/json"

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

func NewLRPStopAuctionFromJSON(payload []byte) (LRPStopAuction, error) {
	var stopAuction LRPStopAuction

	err := json.Unmarshal(payload, &stopAuction)
	if err != nil {
		return LRPStopAuction{}, err
	}

	err = stopAuction.Validate()
	if err != nil {
		return LRPStopAuction{}, err
	}

	return stopAuction, nil
}

func (auction LRPStopAuction) ToJSON() []byte {
	bytes, err := json.Marshal(auction)
	if err != nil {
		panic(err)
	}

	return bytes
}

func (auction LRPStopAuction) Validate() error {
	if auction.ProcessGuid == "" {
		return ValidationError{ErrInvalidJSONMessage{"process_guid"}}
	}

	return nil
}
