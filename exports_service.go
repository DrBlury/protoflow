package protoflow

import runtimepkg "github.com/drblury/protoflow/internal/runtime"

type (
	Config              = runtimepkg.Config
	Service             = runtimepkg.Service
	ServiceDependencies = runtimepkg.ServiceDependencies
	ProtoValidator      = runtimepkg.ProtoValidator
	OutboxStore         = runtimepkg.OutboxStore
)

var (
	NewService = runtimepkg.NewService
)
