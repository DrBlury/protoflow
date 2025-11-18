package protoflow

// Metadata represents the headers carried alongside an event.
type Metadata map[string]string

func (m Metadata) clone() Metadata {
	if len(m) == 0 {
		return Metadata{}
	}

	cloned := make(Metadata, len(m))
	for k, v := range m {
		cloned[k] = v
	}
	return cloned
}
