package runtime

import (
	"context"

	"github.com/ThreeDotsLabs/watermill"

	transportpkg "github.com/drblury/protoflow/internal/runtime/transport"
)

type Transport = transportpkg.Transport

type TransportFactory interface {
	Build(ctx context.Context, conf *Config, logger watermill.LoggerAdapter) (Transport, error)
}

func defaultTransportFactory() TransportFactory {
	return transportpkg.DefaultFactory()
}
