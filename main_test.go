package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewHTTPServerConfiguresCompatibilityServiceEntry(t *testing.T) {
	server := newHTTPServer(":9090")
	if server.Addr != ":9090" {
		t.Fatalf("Addr = %q, want :9090", server.Addr)
	}
	if server.Handler == nil {
		t.Fatal("Handler = nil")
	}
	if server.ReadHeaderTimeout < time.Second {
		t.Fatalf("ReadHeaderTimeout = %v, want a bounded timeout", server.ReadHeaderTimeout)
	}

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()
	server.Handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("GET /healthz status = %d, want %d", response.Code, http.StatusOK)
	}
}
