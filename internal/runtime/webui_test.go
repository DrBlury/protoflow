package runtime

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandleGetHandlersReturnsJSON(t *testing.T) {
	svc := &Service{
		handlers: []*HandlerInfo{
			{
				Name:         "orders",
				ConsumeQueue: "orders.created",
				PublishQueue: "orders.audit",
				Stats: &HandlerStats{
					MessagesProcessed:   3,
					MessagesFailed:      1,
					TotalProcessingTime: int64(time.Millisecond),
					LastProcessedAt:     time.Now().UTC().Round(time.Millisecond),
				},
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/handlers", nil)
	rec := httptest.NewRecorder()

	svc.handleGetHandlers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected application/json content type, got %s", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected Access-Control-Allow-Origin header to be '*', got %s", got)
	}

	var payload []HandlerInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unexpected error decoding response: %v", err)
	}
	if len(payload) != 1 || payload[0].Name != "orders" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if payload[0].Stats == nil {
		t.Fatalf("expected stats to be present in payload")
	}
}
