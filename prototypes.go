package protoflow

import "google.golang.org/protobuf/proto"

// NewProtoMessage instantiates a zero-value protobuf message for the provided generic type.
func NewProtoMessage[T proto.Message]() (T, error) {
	var zero T
	return ensureProtoPrototype(zero)
}

// MustProtoMessage instantiates the protobuf message and panics if the type cannot be created.
func MustProtoMessage[T proto.Message]() T {
	msg, err := NewProtoMessage[T]()
	if err != nil {
		panic(err)
	}
	return msg
}
