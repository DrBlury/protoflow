// Package transport provides transport types and interfaces for the internal runtime.
// Transport implementations are now in github.com/drblury/protoflow/transport/*.
package transport

import (
	newtransport "github.com/drblury/protoflow/transport"
)

// Capabilities is an alias for the modular transport Capabilities.
type Capabilities = newtransport.Capabilities

// CapabilitiesProvider is an alias for the modular transport CapabilitiesProvider.
type CapabilitiesProvider = newtransport.CapabilitiesProvider

// Predefined capability sets - aliased from the new transport package.
var (
	ChannelCapabilities       = newtransport.ChannelCapabilities
	KafkaCapabilities         = newtransport.KafkaCapabilities
	RabbitMQCapabilities      = newtransport.RabbitMQCapabilities
	NATSCapabilities          = newtransport.NATSCapabilities
	NATSJetStreamCapabilities = newtransport.NATSJetStreamCapabilities
	AWSCapabilities           = newtransport.AWSCapabilities
	SQLiteCapabilities        = newtransport.SQLiteCapabilities
	PostgresCapabilities      = newtransport.PostgresCapabilities
	HTTPCapabilities          = newtransport.HTTPCapabilities
	IOCapabilities            = newtransport.IOCapabilities
)

// GetCapabilities returns the capabilities for a transport by name.
func GetCapabilities(transportName string) Capabilities {
	return newtransport.GetCapabilities(transportName)
}
