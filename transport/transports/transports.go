// Package transports imports all built-in transports for auto-registration.
// Import this package to have all transports registered with the default registry.
package transports

import (
	"github.com/drblury/protoflow/transport/aws"
	"github.com/drblury/protoflow/transport/channel"
	"github.com/drblury/protoflow/transport/http"
	"github.com/drblury/protoflow/transport/io"
	"github.com/drblury/protoflow/transport/jetstream"
	"github.com/drblury/protoflow/transport/kafka"
	"github.com/drblury/protoflow/transport/nats"
	"github.com/drblury/protoflow/transport/postgres"
	"github.com/drblury/protoflow/transport/rabbitmq"
	"github.com/drblury/protoflow/transport/sqlite"
)

// Register registers all built-in transports with the default registry.
// This is called automatically via init() when this package is imported.
func Register() {
	aws.Register()
	channel.Register()
	http.Register()
	io.Register()
	jetstream.Register()
	kafka.Register()
	nats.Register()
	postgres.Register()
	rabbitmq.Register()
	sqlite.Register()
}

//nolint:gochecknoinits // This is the single entry point for transport registration
func init() {
	Register()
}
