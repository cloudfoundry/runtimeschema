package models

import "regexp"

type ProtocolName string

const (
	TCPProtocol  ProtocolName = "tcp"
	UDPProtocol  ProtocolName = "udp"
	ICMPProtocol ProtocolName = "icmp"
	AllProtocol  ProtocolName = "all"

	maxPort int = 65535
)

type PortRange struct {
	Start uint `json:"start"`
	End   uint `json:"end"`
}

type CIDR struct {
	NetworkAddress string `json:"network_address"`
	PrefixLength   uint8  `json:"prefix_length"`
}

type SecurityGroupRule struct {
	Protocol    ProtocolName `json:"protocol"`
	PortRange   PortRange    `json:"port_range"`
	Destination CIDR         `json:"destination"`
}

var destinationRegex = regexp.MustCompile(`^(([0-9][1-9]{0,2})\.){3}([0-9][1-9]{0,2})`)

func (rule SecurityGroupRule) Validate() error {
	var validationError ValidationError

	switch rule.Protocol {
	case TCPProtocol:
	case UDPProtocol:
	case ICMPProtocol:
	case AllProtocol:
	default:
		validationError = validationError.Append(ErrInvalidField{"protocol"})
	}

	if rule.PortRange.Start < 1 || rule.PortRange.Start > 65535 {
		validationError = validationError.Append(ErrInvalidField{"port_range"})
	}
	if rule.PortRange.End < 1 || rule.PortRange.End > 65535 {
		validationError = validationError.Append(ErrInvalidField{"port_range"})
	}
	if rule.PortRange.Start > rule.PortRange.End {
		validationError = validationError.Append(ErrInvalidField{"port_range"})
	}

	if rule.Destination.NetworkAddress == "" {
		validationError = validationError.Append(ErrInvalidField{"destination"})
	}

	if rule.Destination.PrefixLength > 32 {
		validationError = validationError.Append(ErrInvalidField{"destination"})
	}

	if !validationError.Empty() {
		return validationError
	}

	return nil
}
