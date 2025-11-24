package metadata

import (
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
)

func TestCloneDoesNotAlias(t *testing.T) {
	original := Metadata{"a": "1", "b": "2"}
	clone := original.Clone()
	clone["a"] = "changed"

	if original["a"] != "1" {
		t.Fatalf("expected original map to stay untouched, got %q", original["a"])
	}
	if len(clone) != len(original) {
		t.Fatalf("expected clone to have same size")
	}
}

func TestCloneEmpty(t *testing.T) {
	var m Metadata
	cloned := m.Clone()
	if cloned == nil {
		t.Fatal("expected non-nil map")
	}
	if len(cloned) != 0 {
		t.Fatal("expected empty map")
	}
}

func TestWithAndWithAll(t *testing.T) {
	base := Metadata{"foo": "bar"}
	enriched := base.With("baz", "qux")
	if base["baz"] != "" {
		t.Fatalf("expected base map to remain unchanged")
	}
	if enriched["baz"] != "qux" {
		t.Fatalf("expected enriched map to add entry")
	}

	merged := enriched.WithAll(Metadata{"alpha": "beta"})
	if merged["alpha"] != "beta" {
		t.Fatalf("expected merged metadata to include new value")
	}
	if merged["baz"] != "qux" {
		t.Fatalf("expected existing entries to persist")
	}
}

func TestNewPairs(t *testing.T) {
	md := New("key", "value", "another", "entry")
	if md["key"] != "value" {
		t.Fatalf("expected key to be set")
	}
	if md["another"] != "entry" {
		t.Fatalf("expected another entry to be set")
	}
}

func TestToAndFromWatermill(t *testing.T) {
	md := Metadata{"source": "api"}
	wm := ToWatermill(md)
	if wm["source"] != "api" {
		t.Fatalf("expected watermill metadata to copy entries")
	}
	wm["source"] = "mutation"
	if md["source"] != "api" {
		t.Fatalf("expected original metadata to be immutable to watermill changes")
	}

	if len(ToWatermill(nil)) != 0 {
		t.Fatal("expected nil input to return empty metadata")
	}

	roundTrip := FromWatermill(message.Metadata{"event": "order"})
	if roundTrip["event"] != "order" {
		t.Fatalf("expected watermill metadata to convert back")
	}
}

func TestFromWatermillEmpty(t *testing.T) {
	md := FromWatermill(nil)
	if md == nil {
		t.Fatal("expected non-nil map")
	}
	if len(md) != 0 {
		t.Fatal("expected empty map")
	}
}
