package runtime

import (
	"github.com/ThreeDotsLabs/watermill/message"

	metadatapkg "github.com/drblury/protoflow/internal/runtime/metadata"
)

func metadataFromWatermill(md message.Metadata) Metadata {
	return metadatapkg.FromWatermill(md)
}

func metadataToWatermill(metadata Metadata) message.Metadata {
	return metadatapkg.ToWatermill(metadata)
}
