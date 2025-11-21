package protoflow

import (
	runtimepkg "github.com/drblury/protoflow/internal/runtime"
	errspkg "github.com/drblury/protoflow/internal/runtime/errors"
)

type UnprocessableEventError = runtimepkg.UnprocessableEventError

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
