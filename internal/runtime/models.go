package runtime

// UnprocessableEventError wraps payloads that failed validation or unmarshalling.
type UnprocessableEventError struct {
	eventMessage string
	err          error
}

func (e *UnprocessableEventError) Error() string {
	return "unprocessable event: " + e.eventMessage + " error: " + e.err.Error()
}
