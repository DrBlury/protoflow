package protoflow

import runtimepkg "github.com/drblury/protoflow/internal/runtime"

type UnprocessableEventError = runtimepkg.UnprocessableEventError

var (
	ErrServiceRequired             = runtimepkg.ErrServiceRequired
	ErrHandlerRequired             = runtimepkg.ErrHandlerRequired
	ErrConsumeQueueRequired        = runtimepkg.ErrConsumeQueueRequired
	ErrHandlerNameRequired         = runtimepkg.ErrHandlerNameRequired
	ErrConsumeMessageTypeRequired  = runtimepkg.ErrConsumeMessageTypeRequired
	ErrConsumeMessagePointerNeeded = runtimepkg.ErrConsumeMessagePointerNeeded
	ErrPublisherRequired           = runtimepkg.ErrPublisherRequired
	ErrTopicRequired               = runtimepkg.ErrTopicRequired
)
