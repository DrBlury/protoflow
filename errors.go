package protoflow

import "errors"

var (
	ErrServiceRequired             = errors.New("protoflow: event service is required")
	ErrHandlerRequired             = errors.New("protoflow: handler function is required")
	ErrConsumeQueueRequired        = errors.New("protoflow: consume queue is required")
	ErrHandlerNameRequired         = errors.New("protoflow: handler name is required")
	ErrConsumeMessageTypeRequired  = errors.New("protoflow: consume message type is required")
	ErrConsumeMessagePointerNeeded = errors.New("protoflow: consume message type must be a pointer")
	ErrPublisherRequired           = errors.New("protoflow: publisher is required")
	ErrTopicRequired               = errors.New("protoflow: topic is required")
)
