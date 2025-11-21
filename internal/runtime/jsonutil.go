package runtime

import (
	"io"

	jsoncodec "github.com/drblury/protoflow/internal/runtime/jsoncodec"
)

func Marshal(v any) ([]byte, error) {
	return jsoncodec.Marshal(v)
}

func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	return jsoncodec.MarshalIndent(v, prefix, indent)
}

func Unmarshal(data []byte, v any) error {
	return jsoncodec.Unmarshal(data, v)
}

func Encode(w io.Writer, v any) error {
	return jsoncodec.Encode(w, v)
}

func Decode(r io.Reader, v any) error {
	return jsoncodec.Decode(r, v)
}
