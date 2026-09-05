package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCORS(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name           string
		allowedOrigins []string
		origin         string
		expectedOrigin string
		expectedVary   string
	}{
		{
			name:           "Wildcard allows everything",
			allowedOrigins: []string{"*"},
			origin:         "http://example.com",
			expectedOrigin: "*",
			expectedVary:   "",
		},
		{
			name:           "Allowed origin match",
			allowedOrigins: []string{"http://example.com", "http://test.com"},
			origin:         "http://example.com",
			expectedOrigin: "http://example.com",
			expectedVary:   "Origin",
		},
		{
			name:           "Disallowed origin no header",
			allowedOrigins: []string{"http://example.com"},
			origin:         "http://malicious.com",
			expectedOrigin: "",
			expectedVary:   "",
		},
		{
			name:           "No origin header in request",
			allowedOrigins: []string{"http://example.com"},
			origin:         "",
			expectedOrigin: "",
			expectedVary:   "",
		},
		{
			name:           "Empty allowed origins",
			allowedOrigins: []string{},
			origin:         "http://example.com",
			expectedOrigin: "",
			expectedVary:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerToTest := CORS(tt.allowedOrigins)(nextHandler)
			req := httptest.NewRequest("GET", "http://server/foo", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rec := httptest.NewRecorder()

			handlerToTest.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedOrigin, rec.Header().Get("Access-Control-Allow-Origin"))
			assert.Equal(t, tt.expectedVary, rec.Header().Get("Vary"))
			assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", rec.Header().Get("Access-Control-Allow-Methods"))
		})
	}

	t.Run("OPTIONS request returns 200", func(t *testing.T) {
		handlerToTest := CORS([]string{"*"})(nextHandler)

		req := httptest.NewRequest("OPTIONS", "http://server/foo", nil)
		rec := httptest.NewRecorder()

		handlerToTest.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
