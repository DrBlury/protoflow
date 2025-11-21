package jsoncodec

import (
	"io"

	"github.com/bytedance/sonic"
)

var defaultConfig = sonic.ConfigStd

func Marshal(v any) ([]byte, error) {
	return defaultConfig.Marshal(v)
}

func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	return defaultConfig.MarshalIndent(v, prefix, indent)
}

func Unmarshal(data []byte, v any) error {
	return defaultConfig.Unmarshal(data, v)
}

func Encode(w io.Writer, v any) error {
	enc := defaultConfig.NewEncoder(w)
	return enc.Encode(v)
}

func Decode(r io.Reader, v any) error {
	dec := defaultConfig.NewDecoder(r)
	return dec.Decode(v)
}
