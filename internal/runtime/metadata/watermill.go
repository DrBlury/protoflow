package metadata

import "github.com/ThreeDotsLabs/watermill/message"

// FromWatermill converts Watermill metadata into Protoflow metadata.
func FromWatermill(md message.Metadata) Metadata {
	if len(md) == 0 {
		return Metadata{}
	}

	result := make(Metadata, len(md))
	for k, v := range md {
		result[k] = v
	}
	return result
}

// ToWatermill converts Protoflow metadata into a Watermill map.
func ToWatermill(metadata Metadata) message.Metadata {
	if len(metadata) == 0 {
		return message.Metadata{}
	}

	wm := make(message.Metadata, len(metadata))
	for k, v := range metadata {
		wm[k] = v
	}
	return wm
}
