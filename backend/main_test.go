package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	routes(nil).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var result Response
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "online" {
		t.Fatalf("unexpected status: %s", result.Status)
	}
}

func TestMessage(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/message", nil)
	rec := httptest.NewRecorder()
	routes(nil).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var result Response
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Message == "" {
		t.Fatal("empty message")
	}
}

func TestCORS(t *testing.T) {
	for _, origin := range []string{
		"https://memoire-demo-frontend.valadeploy.internal",
		"http://127.0.0.1:14200",
	} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
		req.Header.Set("Origin", origin)
		rec := httptest.NewRecorder()
		routes(nil).ServeHTTP(rec, req)
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != origin {
			t.Fatalf("origin %s: got %q", origin, got)
		}
	}
}

func TestDisallowOtherOrigins(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/tasks", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rec := httptest.NewRecorder()
	routes(nil).ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestDBNotConfigured(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	rec := httptest.NewRecorder()
	routes(nil).ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}
