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
	PortRange   *PortRange   `json:"port_range,omitempty"`
	IcmpInfo    *ICMPInfo    `json:"icmp_info,omitempty"`
}

type ICMPInfo struct {
	Type int32 `json:"type"`
	Code int32 `json:"code"`
}

func (rule SecurityGroupRule) Validate() error {
	var validationError ValidationError

	switch rule.Protocol {
	case TCPProtocol, UDPProtocol:
		validationError = rule.validatePortRange()
		if rule.IcmpInfo != nil {
			validationError = validationError.Append(ErrInvalidField{"icmp_info"})
		}
	case ICMPProtocol:
		if rule.PortRange != nil {
			validationError = validationError.Append(ErrInvalidField{"port_range"})
		}
		if rule.IcmpInfo == nil {
			validationError = validationError.Append(ErrInvalidField{"icmp_info"})
		}
	case AllProtocol:
		if rule.PortRange != nil {
			validationError = validationError.Append(ErrInvalidField{"port_range"})
		}
		if rule.IcmpInfo != nil {
			validationError = validationError.Append(ErrInvalidField{"icmp_info"})
		}
	default:
		validationError = validationError.Append(ErrInvalidField{"protocol"})
	}

	_, _, err := net.ParseCIDR(rule.Destination)
	if err != nil {
		validationError = validationError.Append(ErrInvalidField{"destination"})
	}

	if !validationError.Empty() {
		return validationError
	}

	return nil
}

func (rule SecurityGroupRule) validatePortRange() ValidationError {
	var validationError ValidationError

	if rule.PortRange == nil {
		validationError = validationError.Append(ErrInvalidField{"port_range"})
	} else {
		if rule.PortRange.Start < 1 || rule.PortRange.Start > 65535 {
			validationError = validationError.Append(ErrInvalidField{"port_range"})
		}
		if rule.PortRange.End < 1 || rule.PortRange.End > 65535 {
			validationError = validationError.Append(ErrInvalidField{"port_range"})
		}
		if rule.PortRange.Start > rule.PortRange.End {
			validationError = validationError.Append(ErrInvalidField{"port_range"})
		}
	}

	return validationError
}
