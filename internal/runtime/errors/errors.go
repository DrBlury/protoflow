package errors

import sterrors "errors"

var (
	ErrServiceRequired             = sterrors.New("protoflow: event service is required")
	ErrHandlerRequired             = sterrors.New("protoflow: handler function is required")
	ErrConsumeQueueRequired        = sterrors.New("protoflow: consume queue is required")
	ErrHandlerNameRequired         = sterrors.New("protoflow: handler name is required")
	ErrConsumeMessageTypeRequired  = sterrors.New("protoflow: consume message type is required")
	ErrConsumeMessagePointerNeeded = sterrors.New("protoflow: consume message type must be a pointer")
	ErrPublisherRequired           = sterrors.New("protoflow: publisher is required")
	ErrTopicRequired               = sterrors.New("protoflow: topic is required")
)
