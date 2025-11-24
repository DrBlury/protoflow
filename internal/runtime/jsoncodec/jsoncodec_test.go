package jsoncodec

import (
	"bytes"
	"strings"
	"testing"
)

type testPayload struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func TestMarshalAndUnmarshal(t *testing.T) {
	in := testPayload{ID: 42, Name: "protoflow"}
	data, err := Marshal(in)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var out testPayload
	if err := Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if out != in {
		t.Fatalf("expected round trip to match, got %#v", out)
	}

	indented, err := MarshalIndent(in, "", "  ")
	if err != nil {
		t.Fatalf("marshal indent failed: %v", err)
	}
	if !strings.Contains(string(indented), "\n  \"id\"") {
		t.Fatalf("expected indented output, got %s", string(indented))
	}
}

func TestEncodeAndDecode(t *testing.T) {
	buf := &bytes.Buffer{}
	payload := testPayload{ID: 7, Name: "stream"}

	if err := Encode(buf, payload); err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var decoded testPayload
	if err := Decode(buf, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded != payload {
		t.Fatalf("expected decoded payload to match, got %#v", decoded)
	}
}
