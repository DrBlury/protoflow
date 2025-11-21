package protoflow

import (
	runtimepkg "github.com/drblury/protoflow/internal/runtime"
	handlerpkg "github.com/drblury/protoflow/internal/runtime/handlers"
	"google.golang.org/protobuf/proto"
)

type (
	MessageHandlerRegistration                = runtimepkg.MessageHandlerRegistration
	JSONHandlerRegistration[T any, O any]     = handlerpkg.JSONHandlerRegistration[T, O]
	JSONMessageContext[T any]                 = handlerpkg.JSONMessageContext[T]
	JSONMessageOutput[T any]                  = handlerpkg.JSONMessageOutput[T]
	JSONMessageHandler[T any, O any]          = handlerpkg.JSONMessageHandler[T, O]
	ProtoHandlerRegistration[T proto.Message] = handlerpkg.ProtoHandlerRegistration[T]
	ProtoHandlerOption                        = handlerpkg.ProtoHandlerOption
	ProtoMessageContext[T proto.Message]      = handlerpkg.ProtoMessageContext[T]
	ProtoMessageOutput                        = handlerpkg.ProtoMessageOutput
	ProtoMessageHandler[T proto.Message]      = handlerpkg.ProtoMessageHandler[T]
)

var (
	RegisterMessageHandler  = runtimepkg.RegisterMessageHandler
	WithPublishMessageTypes = handlerpkg.WithPublishMessageTypes
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
