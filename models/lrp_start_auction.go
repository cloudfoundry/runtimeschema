package models

import "encoding/json"

type LRPStartAuctionState int

const (
	LRPStartAuctionStateInvalid LRPStartAuctionState = iota
	LRPStartAuctionStatePending
	LRPStartAuctionStateClaimed
)

type LRPStartAuction struct {
	DesiredLRP DesiredLRP `json:"desired_lrp"`

	InstanceGuid string `json:"instance_guid"`
	Index        int    `json:"index"`

	State     LRPStartAuctionState `json:"state"`
	UpdatedAt int64                `json:"updated_at"`
}

func NewLRPStartAuctionFromJSON(payload []byte) (LRPStartAuction, error) {
	auction := LRPStartAuction{}

	err := json.Unmarshal(payload, &auction)
	if err != nil {
		return LRPStartAuction{}, err
	}

	return auction, auction.Validate()
}

func (auction LRPStartAuction) Validate() error {
	if auction.InstanceGuid == "" {
		return ErrInvalidJSONMessage{"instance_guid"}
	}
	return auction.DesiredLRP.Validate()
}

func (auction LRPStartAuction) ToJSON() []byte {
	bytes, err := json.Marshal(auction)
	if err != nil {
		panic(err)
	}

	return bytes
}
