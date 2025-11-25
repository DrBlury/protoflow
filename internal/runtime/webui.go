package runtime

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (s *Service) StartWebUIServer() {
	if !s.Conf.WebUIEnabled {
		return
	}

	port := s.Conf.WebUIPort
	if port == 0 {
		port = 8081
	}

	s.RegisterHTTPHandler(port, "/api/handlers", http.HandlerFunc(s.handleGetHandlers))
}

func (s *Service) handleGetHandlers(w http.ResponseWriter, r *http.Request) {
	s.handlersMu.RLock()
	defer s.handlersMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")

	// Set CORS headers based on configuration
	if s.Conf != nil && len(s.Conf.WebUICORSAllowedOrigins) > 0 {
		origin := r.Header.Get("Origin")
		allowedOrigin := s.getAllowedCORSOrigin(origin)
		if allowedOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
	}

	// Handle preflight requests
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := json.NewEncoder(w).Encode(s.handlers); err != nil {
		s.Logger.Error("Failed to encode handlers", err, nil)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// getAllowedCORSOrigin checks if the request origin is allowed and returns the appropriate
// Access-Control-Allow-Origin value.
func (s *Service) getAllowedCORSOrigin(requestOrigin string) string {
	if s.Conf == nil {
		return ""
	}
	for _, allowed := range s.Conf.WebUICORSAllowedOrigins {
		if allowed == "*" {
			return "*"
		}
		if strings.EqualFold(allowed, requestOrigin) {
			return requestOrigin
		}
	}
	return ""
}
