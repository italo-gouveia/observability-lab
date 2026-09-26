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

	// The handler injects a ~10% synthetic error, so either outcome is valid.
	if rec.Code != http.StatusOK && rec.Code != http.StatusInternalServerError {
		t.Fatalf("work: got %d, want 200 or 500", rec.Code)
	}
	if rec.Body.Len() == 0 {
		t.Fatal("work: expected a non-empty body")
	}
}
