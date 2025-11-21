package runtime

import errspkg "github.com/drblury/protoflow/internal/runtime/errors"

var (
	ErrServiceRequired             = errspkg.ErrServiceRequired
	ErrHandlerRequired             = errspkg.ErrHandlerRequired
	ErrConsumeQueueRequired        = errspkg.ErrConsumeQueueRequired
	ErrHandlerNameRequired         = errspkg.ErrHandlerNameRequired
	ErrConsumeMessageTypeRequired  = errspkg.ErrConsumeMessageTypeRequired
	ErrConsumeMessagePointerNeeded = errspkg.ErrConsumeMessagePointerNeeded
	ErrPublisherRequired           = errspkg.ErrPublisherRequired
	ErrTopicRequired               = errspkg.ErrTopicRequired
)
