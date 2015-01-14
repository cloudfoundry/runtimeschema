package models

import "net"

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

type SecurityGroupRule struct {
	Protocol    ProtocolName `json:"protocol"`
	Destination string       `json:"destination"`
	PortRange   PortRange    `json:"port_range"`
}

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

	_, _, err := net.ParseCIDR(rule.Destination)
	if err != nil {
		validationError = validationError.Append(ErrInvalidField{"destination"})
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

	if !validationError.Empty() {
		return validationError
	}

	return nil
}
