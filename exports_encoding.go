package protoflow

import runtimepkg "github.com/drblury/protoflow/internal/runtime"

var (
	Marshal       = runtimepkg.Marshal
	MarshalIndent = runtimepkg.MarshalIndent
	Unmarshal     = runtimepkg.Unmarshal
	Encode        = runtimepkg.Encode
	Decode        = runtimepkg.Decode
)
