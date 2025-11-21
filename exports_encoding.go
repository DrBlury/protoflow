package protoflow

import jsoncodec "github.com/drblury/protoflow/internal/runtime/jsoncodec"

var (
	Marshal       = jsoncodec.Marshal
	MarshalIndent = jsoncodec.MarshalIndent
	Unmarshal     = jsoncodec.Unmarshal
	Encode        = jsoncodec.Encode
	Decode        = jsoncodec.Decode
)
