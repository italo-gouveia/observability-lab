package main

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newServer wires the real routes; with no OTel providers registered the global
// tracer/meter are no-ops, so these tests need no collector.

func TestHealthzReturnsOK(t *testing.T) {
	srv := newServer(slog.Default())
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("healthz: got %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestWorkReturnsHandledStatusWithBody(t *testing.T) {
	srv := newServer(slog.Default())
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/work", nil))

	// No java-orders in a unit test, so the downstream call fails and the handler
	// maps it to 502. With a reachable downstream it would be 200.
	if rec.Code != http.StatusOK && rec.Code != http.StatusBadGateway {
		t.Fatalf("work: got %d, want 200 or 502", rec.Code)
	}
	if rec.Body.Len() == 0 {
		t.Fatal("work: expected a non-empty body")
	}
}
