package protoflow

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/plugin"
	"google.golang.org/protobuf/proto"
)

var logLevelMapping = map[slog.Level]slog.Level{
	slog.LevelDebug: slog.LevelDebug,
	slog.LevelInfo:  slog.LevelInfo,
	slog.LevelWarn:  slog.LevelWarn,
	slog.LevelError: slog.LevelError,
}

var routerRun = func(router *message.Router, ctx context.Context) error {
	return router.Run(ctx)
}

// ProtoValidator validates protobuf messages after they are unmarshalled.
type ProtoValidator interface {
	Validate(proto.Message) error
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
}

// Service wires a Watermill router, publisher, subscriber, and middleware chain.
type Service struct {
	Conf       *Config
	publisher  message.Publisher
	subscriber message.Subscriber
	router     *message.Router
	Logger     watermill.LoggerAdapter

	validator ProtoValidator
	outbox    OutboxStore

	protoRegistry   map[string]func() proto.Message
	protoRegistryMu sync.RWMutex
}

// NewService constructs a Service for the supplied configuration. Register handlers
// on the returned Service before calling Start.
func NewService(conf *Config, log *slog.Logger, ctx context.Context, deps ServiceDependencies) *Service {
	logger := watermill.NewSlogLoggerWithLevelMapping(log, logLevelMapping)
	logger.Info("Creating event service",
		watermill.LogFields{
			"pubsub_system": conf.PubSubSystem,
			"config":        conf,
		})

	s := &Service{
		Conf:          conf,
		Logger:        logger,
		validator:     deps.Validator,
		outbox:        deps.Outbox,
		protoRegistry: make(map[string]func() proto.Message),
	}

	setupPubSub(s, conf, logger, ctx)

	router, err := message.NewRouter(message.RouterConfig{}, logger)
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

func setupPubSub(s *Service, conf *Config, logger watermill.LoggerAdapter, ctx context.Context) {
	switch conf.PubSubSystem {
	case "kafka":
		s.createKafkaPublisher(conf.KafkaBrokers, logger)
		s.createKafkaSubscriber(conf.KafkaConsumerGroup, conf.KafkaBrokers, logger)
		return
	case "rabbitmq":
		ampqConn, ampqConfig := s.setupAmpq(conf, logger)
		s.createRabbitMQPublisher(ampqConfig, ampqConn, logger)
		s.createRabbitMQSubscriber(ampqConfig, ampqConn, logger)
		return
	case "aws":
		cfg := s.createAWSConfig(ctx)
		logger.Info("Created AWS config",
			watermill.LogFields{
				"AWSConfig": cfg,
			},
		)
		s.createAwsPublisher(logger, cfg)
		s.createAwsSubscriber(logger, cfg)
		return
	default:
		panic("unsupported PubSubSystem, must be 'kafka', 'aws' or 'rabbitmq'")
	}
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
