package protoflow

import (
	"errors"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"google.golang.org/protobuf/proto"
)

type handlerRegistration struct {
	Name               string
	ConsumeQueue       string
	Subscriber         message.Subscriber
	PublishQueue       string
	Publisher          message.Publisher
	Handler            message.HandlerFunc
	consumeMessageType proto.Message
}

func (s *Service) registerHandler(cfg handlerRegistration) error {
	if cfg.Handler == nil {
		return errors.New("handler function is required")
	}
	if cfg.ConsumeQueue == "" {
		return errors.New("consume queue is required")
	}
	if cfg.Subscriber == nil {
		cfg.Subscriber = s.subscriber
	}
	if cfg.Publisher == nil {
		cfg.Publisher = s.publisher
	}
	if cfg.consumeMessageType != nil {
		s.registerProtoType(cfg.consumeMessageType)
		if cfg.Name == "" {
			cfg.Name = fmt.Sprintf("%T-Handler", cfg.consumeMessageType)
		}
	}
	if cfg.Name == "" {
		return errors.New("handler name is required")
	}

	s.router.AddHandler(
		cfg.Name,
		cfg.ConsumeQueue,
		cfg.Subscriber,
		cfg.PublishQueue,
		cfg.Publisher,
		cfg.Handler,
	)

	return nil
}

// RegisterProtoMessage exposes a proto message type for validation without registering a handler.
func (s *Service) RegisterProtoMessage(msg proto.Message) {
	s.registerProtoType(msg)
}

func (s *Service) registerProtoType(msg proto.Message) {
	if msg == nil {
		return
	}

	typeName := fmt.Sprintf("%T", msg)

	s.protoRegistryMu.Lock()
	s.protoRegistry[typeName] = func() proto.Message {
		clone := proto.Clone(msg)
		proto.Reset(clone)
		return clone
	}
	s.protoRegistryMu.Unlock()
}
