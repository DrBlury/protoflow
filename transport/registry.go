package transport

import (
	"context"
	"fmt"
	"sync"

	"github.com/ThreeDotsLabs/watermill"
)

// Registry maintains a mapping of transport names to their builders and capabilities.
// Transport packages should register themselves using Register.
type Registry struct {
	mu           sync.RWMutex
	builders     map[string]Builder
	capabilities map[string]Capabilities
}

// DefaultRegistry is the global transport registry.
var DefaultRegistry = NewRegistry()

// NewRegistry creates a new transport registry.
func NewRegistry() *Registry {
	return &Registry{
		builders:     make(map[string]Builder),
		capabilities: make(map[string]Capabilities),
	}
}

// Register adds a transport builder to the registry.
// The name should match the PubSubSystem config value (e.g., "kafka", "rabbitmq").
func (r *Registry) Register(name string, builder Builder) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.builders[name] = builder
}

// RegisterWithCapabilities adds a transport builder and its capabilities to the registry.
func (r *Registry) RegisterWithCapabilities(name string, builder Builder, caps Capabilities) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.builders[name] = builder
	r.capabilities[name] = caps
}

// GetCapabilities returns the capabilities for a registered transport.
// Returns a zero Capabilities struct if the transport is unknown.
func (r *Registry) GetCapabilities(name string) Capabilities {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if caps, ok := r.capabilities[name]; ok {
		return caps
	}
	return Capabilities{Name: name}
}

// Build creates a transport using the registered builder for the config's PubSubSystem.
func (r *Registry) Build(ctx context.Context, cfg Config, logger watermill.LoggerAdapter) (Transport, error) {
	if cfg == nil {
		return Transport{}, fmt.Errorf("config is required")
	}

	name := cfg.GetPubSubSystem()

	r.mu.RLock()
	builder, ok := r.builders[name]
	r.mu.RUnlock()

	if !ok {
		return Transport{}, fmt.Errorf("unknown transport: %q (registered: %v)", name, r.Names())
	}

	return builder(ctx, cfg, logger)
}

// Names returns the list of registered transport names.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.builders))
	for name := range r.builders {
		names = append(names, name)
	}
	return names
}

// Has returns true if a transport is registered with the given name.
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.builders[name]
	return ok
}

// Register adds a transport builder to the default registry.
func Register(name string, builder Builder) {
	DefaultRegistry.Register(name, builder)
}

// RegisterWithCapabilities adds a transport builder and its capabilities to the default registry.
func RegisterWithCapabilities(name string, builder Builder, caps Capabilities) {
	DefaultRegistry.RegisterWithCapabilities(name, builder, caps)
}

// Build creates a transport using the default registry.
func Build(ctx context.Context, cfg Config, logger watermill.LoggerAdapter) (Transport, error) {
	return DefaultRegistry.Build(ctx, cfg, logger)
}
