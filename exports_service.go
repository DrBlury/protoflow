package protoflow

import (
	runtimepkg "github.com/drblury/protoflow/internal/runtime"
	configpkg "github.com/drblury/protoflow/internal/runtime/config"
	transportpkg "github.com/drblury/protoflow/internal/runtime/transport"
)

type (
	Config              = configpkg.Config
	Service             = runtimepkg.Service
	ServiceDependencies = runtimepkg.ServiceDependencies
	ProtoValidator      = runtimepkg.ProtoValidator
	OutboxStore         = runtimepkg.OutboxStore
	Transport           = transportpkg.Transport
	TransportFactory    = transportpkg.Factory
)

var (
	NewService = runtimepkg.NewService
)
