package main

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCorsMiddleware(t *testing.T) {
	// Create a mock next handler
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Wrap it with the middleware
	handler := corsMiddleware(nextHandler)

	t.Run("GET request adds headers and calls next", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com/foo", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Check body to ensure next handler was called
		assert.Equal(t, "OK", w.Body.String())

		// Check headers
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Origin, Content-Type, Accept, Connect-Protocol-Version, Connect-Timeout-Ms", resp.Header.Get("Access-Control-Allow-Headers"))
		assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", resp.Header.Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "Connect-Protocol-Version, Connect-Timeout-Ms", resp.Header.Get("Access-Control-Expose-Headers"))
	})

	t.Run("OPTIONS request handles preflight", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "http://example.com/foo", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Check headers
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Origin, Content-Type, Accept, Connect-Protocol-Version, Connect-Timeout-Ms", resp.Header.Get("Access-Control-Allow-Headers"))
		assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", resp.Header.Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "Connect-Protocol-Version, Connect-Timeout-Ms", resp.Header.Get("Access-Control-Expose-Headers"))

		// Next handler should NOT be called for OPTIONS (based on current implementation return)
		// If next handler was called, body would be "OK".
		// Note: The implementation has a `return` after WriteHeader(StatusOK) for OPTIONS.
		assert.Empty(t, w.Body.String())
	})
}

func TestLoggingMiddleware(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(nil) // Restore default logger (stderr usually)
	}()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	handler := loggingMiddleware(nextHandler)

	req := httptest.NewRequest("GET", "/test-uri", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusTeapot, resp.StatusCode)

	// Verify log output contains expected fields
	logOutput := buf.String()
	assert.True(t, strings.Contains(logOutput, "method=GET"))
	assert.True(t, strings.Contains(logOutput, "uri=/test-uri"))
	assert.True(t, strings.Contains(logOutput, "remote_ip=1.2.3.4:1234"))
}
