package runtime

import (
	"context"
	"fmt"
	"sync"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/plugin"
	"google.golang.org/protobuf/proto"

	configpkg "github.com/drblury/protoflow/internal/runtime/config"
	loggingpkg "github.com/drblury/protoflow/internal/runtime/logging"
	transportpkg "github.com/drblury/protoflow/internal/runtime/transport"
)

var routerRun = func(router *message.Router, ctx context.Context) error {
	return router.Run(ctx)
}

// ProtoValidator validates unmarshalled payloads. Implementations typically
// forward to protovalidate or a custom struct validator.
type ProtoValidator interface {
	Validate(value any) error
}

// OutboxStore persists processed messages so they can be forwarded reliably.
type OutboxStore interface {
	StoreOutgoingMessage(ctx context.Context, eventType, uuid, payload string) error
}

// ServiceDependencies holds the optional collaborators that the Service can use.
// Leave fields nil to skip the related middleware.
type ServiceDependencies struct {
	Outbox                    OutboxStore
	Validator                 ProtoValidator
	Middlewares               []MiddlewareRegistration // Appended after the default middleware chain.
	DisableDefaultMiddlewares bool                     // Skips registering the default middleware chain when true.
	TransportFactory          transportpkg.Factory
}

// Service wires a Watermill router, publisher, subscriber, and middleware chain.
type Service struct {
	Conf   *configpkg.Config
	Logger loggingpkg.ServiceLogger

	publisher  message.Publisher
	subscriber message.Subscriber
	router     *message.Router

	validator ProtoValidator
	outbox    OutboxStore

	protoRegistry   map[string]func() proto.Message
	protoRegistryMu sync.RWMutex
}

// NewService constructs a Service for the supplied configuration. Register handlers
// on the returned Service before calling Start.
func NewService(conf *configpkg.Config, log loggingpkg.ServiceLogger, ctx context.Context, deps ServiceDependencies) *Service {
	wmLogger := loggingpkg.NewWatermillAdapter(log)
	log.Info("Creating event service",
		loggingpkg.LogFields{
			"pubsub_system": conf.PubSubSystem,
			"config":        conf,
		})

	s := &Service{
		Conf:          conf,
		Logger:        log,
		validator:     deps.Validator,
		outbox:        deps.Outbox,
		protoRegistry: make(map[string]func() proto.Message),
	}

	factory := deps.TransportFactory
	if factory == nil {
		factory = transportpkg.DefaultFactory()
	}
	transport, err := factory.Build(ctx, conf, wmLogger)
	if err != nil {
		panic(err)
	}

	s.publisher = transport.Publisher
	s.subscriber = transport.Subscriber

	router, err := message.NewRouter(message.RouterConfig{}, wmLogger)
	if err != nil {
		panic(err)
	}

	s.router = router
	s.router.AddPlugin(plugin.SignalsHandler)

	s.registerConfiguredMiddlewares(deps)

	return s
}

// Start runs the underlying Watermill router until the provided context is cancelled.
func (s *Service) Start(ctx context.Context) error {
	return routerRun(s.router, ctx)
}

func (s *Service) registerConfiguredMiddlewares(deps ServiceDependencies) {
	var defaults []MiddlewareRegistration
	if !deps.DisableDefaultMiddlewares {
		defaults = DefaultMiddlewares()
	}
	registrations := make([]MiddlewareRegistration, 0, len(defaults)+len(deps.Middlewares))
	registrations = append(registrations, defaults...)
	registrations = append(registrations, deps.Middlewares...)

	for _, reg := range registrations {
		if err := s.RegisterMiddleware(reg); err != nil {
			name := reg.Name
			if name == "" {
				name = "anonymous_middleware"
			}
			panic(fmt.Sprintf("failed to register middleware %s: %v", name, err))
		}
	}
}
