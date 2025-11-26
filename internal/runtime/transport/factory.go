package transport

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/drblury/protoflow/internal/runtime/config"
	newtransport "github.com/drblury/protoflow/transport"

	// Import all transport packages to register them.
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

// Transport combines a publisher and subscriber pair produced by a factory.
type Transport struct {
	Publisher  message.Publisher
	Subscriber message.Subscriber
}

// Factory abstracts how Protoflow initialises message transports.
type Factory interface {
	Build(ctx context.Context, conf *config.Config, logger watermill.LoggerAdapter) (Transport, error)
}

// DefaultFactory returns the built-in transport factory that uses the
// modular transport registry.
func DefaultFactory() Factory {
	return defaultFactory{}
}

type defaultFactory struct{}

func (defaultFactory) Build(ctx context.Context, conf *config.Config, logger watermill.LoggerAdapter) (Transport, error) {
	if conf == nil {
		return Transport{}, fmt.Errorf("config is required")
	}

	// Use the new transport registry to build the transport.
	t, err := newtransport.Build(ctx, conf, logger)
	if err != nil {
		return Transport{}, err
	}

	return Transport{
		Publisher:  t.Publisher,
		Subscriber: t.Subscriber,
	}, nil
}
