package protoflow

import "github.com/ThreeDotsLabs/watermill/message"

func metadataFromWatermill(md message.Metadata) Metadata {
	if len(md) == 0 {
		return Metadata{}
	}

	result := make(Metadata, len(md))
	for k, v := range md {
		result[k] = v
	}
	return result
}
