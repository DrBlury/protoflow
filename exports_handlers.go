package protoflow

import (
	runtimepkg "github.com/drblury/protoflow/internal/runtime"
	"google.golang.org/protobuf/proto"
)

type (
	MessageHandlerRegistration                = runtimepkg.MessageHandlerRegistration
	JSONHandlerRegistration[T any, O any]     = runtimepkg.JSONHandlerRegistration[T, O]
	JSONMessageContext[T any]                 = runtimepkg.JSONMessageContext[T]
	JSONMessageOutput[T any]                  = runtimepkg.JSONMessageOutput[T]
	JSONMessageHandler[T any, O any]          = runtimepkg.JSONMessageHandler[T, O]
	ProtoHandlerRegistration[T proto.Message] = runtimepkg.ProtoHandlerRegistration[T]
	ProtoHandlerOption                        = runtimepkg.ProtoHandlerOption
	ProtoMessageContext[T proto.Message]      = runtimepkg.ProtoMessageContext[T]
	ProtoMessageOutput                        = runtimepkg.ProtoMessageOutput
	ProtoMessageHandler[T proto.Message]      = runtimepkg.ProtoMessageHandler[T]
)

var (
	RegisterMessageHandler  = runtimepkg.RegisterMessageHandler
	WithPublishMessageTypes = runtimepkg.WithPublishMessageTypes
)

func RegisterJSONHandler[T any, O any](svc *Service, cfg JSONHandlerRegistration[T, O]) error {
	return runtimepkg.RegisterJSONHandler[T, O](svc, cfg)
}

func RegisterProtoHandler[T proto.Message](svc *Service, cfg ProtoHandlerRegistration[T]) error {
	return runtimepkg.RegisterProtoHandler[T](svc, cfg)
}

func NewProtoMessage[T proto.Message]() (T, error) {
	return runtimepkg.NewProtoMessage[T]()
}

func MustProtoMessage[T proto.Message]() T {
	return runtimepkg.MustProtoMessage[T]()
}
