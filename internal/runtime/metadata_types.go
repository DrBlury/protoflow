package runtime

// Metadata represents the headers carried alongside an event.
import metadatapkg "github.com/drblury/protoflow/internal/runtime/metadata"

// Metadata re-exports the metadata type so existing runtime code continues to compile.
type Metadata = metadatapkg.Metadata

// NewMetadata constructs a metadata map from alternating key/value pairs.
var NewMetadata = metadatapkg.New
