package runtime

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/plugin"
	"google.golang.org/protobuf/proto"

	configpkg "github.com/drblury/protoflow/internal/runtime/config"
	errspkg "github.com/drblury/protoflow/internal/runtime/errors"
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
	ErrorClassifier           ErrorClassifier
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

	handlers   []*HandlerInfo
	handlersMu sync.RWMutex

	httpServers    map[int]*http.ServeMux
	runningServers []*http.Server
	httpServersMu  sync.Mutex
	httpCtx        context.Context
	httpCancel     context.CancelFunc

	errorClassifier ErrorClassifier
	resourceTracker *resourceTracker
}

// NewService constructs a Service for the supplied configuration. Register handlers
// on the returned Service before calling Start.
// Panics if configuration is invalid or transport cannot be built.
// Use TryNewService for error-returning variant.
func NewService(conf *configpkg.Config, log loggingpkg.ServiceLogger, ctx context.Context, deps ServiceDependencies) *Service {
	svc, err := TryNewService(conf, log, ctx, deps)
	if err != nil {
		panic(err)
	}
	return svc
}

// TryNewService constructs a Service for the supplied configuration.
// Returns an error instead of panicking if configuration is invalid or transport cannot be built.
func TryNewService(conf *configpkg.Config, log loggingpkg.ServiceLogger, ctx context.Context, deps ServiceDependencies) (*Service, error) {
	if conf == nil {
		return nil, errspkg.ErrConfigRequired
	}
	if log == nil {
		return nil, errspkg.ErrLoggerRequired
	}

	// Validate configuration before proceeding
	if err := conf.Validate(); err != nil {
		return nil, errspkg.NewConfigValidationError(err)
	}

	wmLogger := loggingpkg.NewWatermillAdapter(log)
	log.Info("Creating event service",
		loggingpkg.LogFields{
			"pubsub_system": conf.PubSubSystem,
			"config":        conf,
		})

	s := &Service{
		Conf:            conf,
		Logger:          log,
		validator:       deps.Validator,
		outbox:          deps.Outbox,
		protoRegistry:   make(map[string]func() proto.Message),
		resourceTracker: newResourceTracker(),
	}

	if deps.ErrorClassifier != nil {
		s.errorClassifier = deps.ErrorClassifier
	} else {
		s.errorClassifier = defaultErrorClassifier
	}

	factory := deps.TransportFactory
	if factory == nil {
		factory = transportpkg.DefaultFactory()
	}
	transport, err := factory.Build(ctx, conf, wmLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to build transport: %w", err)
	}

	s.publisher = transport.Publisher
	s.subscriber = transport.Subscriber

	router, err := message.NewRouter(message.RouterConfig{}, wmLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create router: %w", err)
	}

	s.router = router
	s.router.AddPlugin(plugin.SignalsHandler)

	s.registerConfiguredMiddlewares(deps)

	return s, nil
}

// Start runs the underlying Watermill router until the provided context is cancelled.
func (s *Service) Start(ctx context.Context) error {
	s.httpCtx, s.httpCancel = context.WithCancel(ctx)
	s.StartWebUIServer()
	s.startHTTPServers()

	// Start shutdown watcher
	go func() {
		<-ctx.Done()
		s.stopHTTPServers()
	}()

	return routerRun(s.router, ctx)
}

// Stop gracefully shuts down HTTP servers. Called automatically when the context passed to Start is cancelled.
func (s *Service) Stop() {
	if s.httpCancel != nil {
		s.httpCancel()
	}
	s.stopHTTPServers()
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

func (s *Service) getErrorClassifier() ErrorClassifier {
	if s.errorClassifier == nil {
		return defaultErrorClassifier
	}
	return s.errorClassifier
}

func (s *Service) getResourceTracker() *resourceTracker {
	if s.resourceTracker == nil {
		s.resourceTracker = newResourceTracker()
	}
	return s.resourceTracker
}

// Publish sends a raw Watermill message to the specified topic.
// Use PublishProto for type-safe proto message publishing.
func (s *Service) Publish(ctx context.Context, topic string, msgs ...*message.Message) error {
	if s == nil {
		return errspkg.ErrServiceRequired
	}
	if s.publisher == nil {
		return errspkg.ErrPublisherRequired
	}
	if topic == "" {
		return errspkg.ErrTopicRequired
	}
	for _, msg := range msgs {
		if ctx != nil {
			msg.SetContext(ctx)
		}
	}
	return s.publisher.Publish(topic, msgs...)
}

func (s *Service) RegisterHTTPHandler(port int, pattern string, handler http.Handler) {
	s.httpServersMu.Lock()
	defer s.httpServersMu.Unlock()

	if s.httpServers == nil {
		s.httpServers = make(map[int]*http.ServeMux)
	}

	mux, ok := s.httpServers[port]
	if !ok {
		mux = http.NewServeMux()
		s.httpServers[port] = mux
	}

	mux.Handle(pattern, handler)
}

func (s *Service) startHTTPServers() {
	s.httpServersMu.Lock()
	defer s.httpServersMu.Unlock()

	for port, mux := range s.httpServers {
		addr := fmt.Sprintf(":%d", port)
		s.Logger.Info("Starting HTTP server", loggingpkg.LogFields{"address": addr})

		// Configure server with security timeouts to prevent slowloris attacks
		server := &http.Server{
			Addr:         addr,
			Handler:      mux,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		}
		s.runningServers = append(s.runningServers, server)

		go func(srv *http.Server) {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				s.Logger.Error("Failed to start HTTP server", err, loggingpkg.LogFields{"address": srv.Addr})
			}
		}(server)
	}
}

func (s *Service) stopHTTPServers() {
	s.httpServersMu.Lock()
	defer s.httpServersMu.Unlock()

	for _, srv := range s.runningServers {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := srv.Shutdown(ctx); err != nil {
			s.Logger.Error("Failed to gracefully shutdown HTTP server", err, loggingpkg.LogFields{"address": srv.Addr})
		}
		cancel()
	}
	s.runningServers = nil
}
