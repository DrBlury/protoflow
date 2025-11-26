// Package transports imports all built-in transports for auto-registration.
// Import this package to have all transports registered with the default registry.
package transports

import (
	// Import all transports for side-effect registration
	_ "github.com/drblury/protoflow/transport/aws"
	_ "github.com/drblury/protoflow/transport/channel"
	_ "github.com/drblury/protoflow/transport/http"
	_ "github.com/drblury/protoflow/transport/io"
	_ "github.com/drblury/protoflow/transport/jetstream"
	_ "github.com/drblury/protoflow/transport/kafka"
	_ "github.com/drblury/protoflow/transport/nats"
	_ "github.com/drblury/protoflow/transport/postgres"
	_ "github.com/drblury/protoflow/transport/rabbitmq"
	_ "github.com/drblury/protoflow/transport/sqlite"
)
