package runtime

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	configpkg "github.com/drblury/protoflow/internal/runtime/config"
)

func TestHandleGetHandlersReturnsJSON(t *testing.T) {
	svc := &Service{
		Conf: &configpkg.Config{
			WebUICORSAllowedOrigins: []string{"*"},
		},
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
	req.Header.Set("Origin", "http://localhost:3000")
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

func TestHandleGetHandlersPreflightRequest(t *testing.T) {
	svc := &Service{
		Conf: &configpkg.Config{
			WebUICORSAllowedOrigins: []string{"http://localhost:3000"},
		},
		handlers: []*HandlerInfo{},
	}

	req := httptest.NewRequest(http.MethodOptions, "/api/handlers", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()

	svc.handleGetHandlers(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content for preflight, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("expected matching origin in CORS header, got %s", got)
	}
}

func TestHandleGetHandlersNoCORSConfig(t *testing.T) {
	svc := &Service{
		Conf:     &configpkg.Config{},
		handlers: []*HandlerInfo{},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/handlers", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()

	svc.handleGetHandlers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
	// No CORS header should be set when no origins configured
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no CORS header when not configured, got %s", got)
	}
}

func TestHandleGetHandlersOriginNotAllowed(t *testing.T) {
	svc := &Service{
		Conf: &configpkg.Config{
			WebUICORSAllowedOrigins: []string{"http://allowed.com"},
		},
		handlers: []*HandlerInfo{},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/handlers", nil)
	req.Header.Set("Origin", "http://notallowed.com")
	rec := httptest.NewRecorder()

	svc.handleGetHandlers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
	// No CORS header should be set for disallowed origin
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no CORS header for disallowed origin, got %s", got)
	}
}

func TestGetAllowedCORSOrigin(t *testing.T) {
	tests := []struct {
		name      string
		config    *configpkg.Config
		origin    string
		wantAllow string
	}{
		{
			name:      "nil config returns empty",
			config:    nil,
			origin:    "http://test.com",
			wantAllow: "",
		},
		{
			name:      "wildcard allows any origin",
			config:    &configpkg.Config{WebUICORSAllowedOrigins: []string{"*"}},
			origin:    "http://any.com",
			wantAllow: "*",
		},
		{
			name:      "exact match returns origin",
			config:    &configpkg.Config{WebUICORSAllowedOrigins: []string{"http://test.com"}},
			origin:    "http://test.com",
			wantAllow: "http://test.com",
		},
		{
			name:      "case insensitive match",
			config:    &configpkg.Config{WebUICORSAllowedOrigins: []string{"HTTP://TEST.COM"}},
			origin:    "http://test.com",
			wantAllow: "http://test.com",
		},
		{
			name:      "no match returns empty",
			config:    &configpkg.Config{WebUICORSAllowedOrigins: []string{"http://allowed.com"}},
			origin:    "http://notallowed.com",
			wantAllow: "",
		},
		{
			name:      "empty origins list returns empty",
			config:    &configpkg.Config{WebUICORSAllowedOrigins: []string{}},
			origin:    "http://test.com",
			wantAllow: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &Service{Conf: tt.config}
			got := svc.getAllowedCORSOrigin(tt.origin)
			if got != tt.wantAllow {
				t.Errorf("getAllowedCORSOrigin() = %q, want %q", got, tt.wantAllow)
			}
		})
	}
}

func TestStartWebUIServerDisabled(t *testing.T) {
	svc := &Service{
		Conf: &configpkg.Config{
			WebUIEnabled: false,
		},
	}

	// Should not panic and should not register handlers
	svc.StartWebUIServer()

	if len(svc.httpServers) > 0 {
		t.Fatal("expected no HTTP servers when WebUI is disabled")
	}
}

func TestStartWebUIServerDefaultPort(t *testing.T) {
	svc := &Service{
		Conf: &configpkg.Config{
			WebUIEnabled: true,
			WebUIPort:    0, // Should default to 8081
		},
	}

	svc.StartWebUIServer()

	if svc.httpServers == nil {
		t.Fatal("expected HTTP servers map to be initialized")
	}
	if _, ok := svc.httpServers[8081]; !ok {
		t.Fatal("expected handler to be registered on default port 8081")
	}
}

func TestStartWebUIServerCustomPort(t *testing.T) {
	svc := &Service{
		Conf: &configpkg.Config{
			WebUIEnabled: true,
			WebUIPort:    9000,
		},
	}

	svc.StartWebUIServer()

	if svc.httpServers == nil {
		t.Fatal("expected HTTP servers map to be initialized")
	}
	if _, ok := svc.httpServers[9000]; !ok {
		t.Fatal("expected handler to be registered on custom port 9000")
	}
}
