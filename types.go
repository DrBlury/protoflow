package protoflow

// Metadata represents the headers carried alongside an event.
type Metadata map[string]string

func (m Metadata) clone() Metadata {
	return m.cloneWithExtra(0)
}

func (m Metadata) cloneWithExtra(extra int) Metadata {
	size := len(m) + extra
	if size <= 0 {
		return Metadata{}
	}

	cloned := make(Metadata, size)
	for k, v := range m {
		cloned[k] = v
	}
	return cloned
}

// With returns a cloned metadata map containing the provided key/value pair.
func (m Metadata) With(key, value string) Metadata {
	cloned := m.cloneWithExtra(1)
	cloned[key] = value
	return cloned
}

// WithAll returns a cloned metadata map containing the supplied entries.
func (m Metadata) WithAll(entries Metadata) Metadata {
	cloned := m.cloneWithExtra(len(entries))
	for k, v := range entries {
		cloned[k] = v
	}
	return cloned
}

// NewMetadata constructs a Metadata map from alternating key/value pairs.
func NewMetadata(pairs ...string) Metadata {
	md := make(Metadata, len(pairs)/2)
	for i := 0; i < len(pairs)-1; i += 2 {
		md[pairs[i]] = pairs[i+1]
	}
	return md
}
