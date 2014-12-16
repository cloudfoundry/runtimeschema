package models

type LRPStartAuctionState int

type LRPStartAuction struct {
	DesiredLRP DesiredLRP `json:"desired_lrp"`
	Index      int        `json:"index"`
}

func (auction LRPStartAuction) Validate() error {
	var validationError ValidationError

	err := auction.DesiredLRP.Validate()
	if err != nil {
		validationError = append(validationError, err)
	}

	if len(validationError) > 0 {
		return validationError
	}

	return nil
}
