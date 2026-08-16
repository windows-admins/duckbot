package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAllowDuckPointsCORS(t *testing.T) {
	handler := allowDuckPointsCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	tests := []struct {
		name           string
		origin         string
		expectedOrigin string
	}{
		{
			name:           "primary domain",
			origin:         "https://duckpoints.com",
			expectedOrigin: "https://duckpoints.com",
		},
		{
			name:           "www domain",
			origin:         "https://www.duckpoints.com",
			expectedOrigin: "https://www.duckpoints.com",
		},
		{
			name:   "unapproved domain",
			origin: "https://example.com",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/guild/618712310185197588/things", nil)
			request.Header.Set("Origin", test.origin)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if actual := response.Header().Get("Access-Control-Allow-Origin"); actual != test.expectedOrigin {
				t.Fatalf("expected Access-Control-Allow-Origin %q, got %q", test.expectedOrigin, actual)
			}
		})
	}
}
