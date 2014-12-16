package models

type LRPStartAuctionState int

type LRPStartAuction struct {
	DesiredLRP DesiredLRP `json:"desired_lrp"`

	InstanceGuid string `json:"instance_guid"`
	Index        int    `json:"index"`
}

func (auction LRPStartAuction) Validate() error {
	var validationError ValidationError

	if auction.InstanceGuid == "" {
		validationError = append(validationError, ErrInvalidField{"instance_guid"})
	}

	err := auction.DesiredLRP.Validate()
	if err != nil {
		validationError = append(validationError, err)
	}

	if len(validationError) > 0 {
		return validationError
	}

	return nil
}
