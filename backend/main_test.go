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
	// Tester explicitement la compatibilité sans frontend configuré.
	t.Setenv("FRONTEND_URL", "")
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

func TestConfiguredFrontendOrigin(t *testing.T) {
	origin := "https://demo-angular-go-pg-front.valadeploy.internal"
	t.Setenv("FRONTEND_URL", origin)

	for _, tc := range []struct {
		method string
		path   string
		status int
	}{
		{http.MethodGet, "/api/v1/health", http.StatusOK},
		{http.MethodOptions, "/api/v1/tasks", http.StatusNoContent},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.Header.Set("Origin", origin)

		if tc.method == http.MethodOptions {
			req.Header.Set("Access-Control-Request-Method", "POST")
			req.Header.Set("Access-Control-Request-Headers", "content-type")
		}

		rec := httptest.NewRecorder()
		routes(nil).ServeHTTP(rec, req)

		if rec.Code != tc.status {
			t.Fatalf("%s: expected HTTP %d, got %d",
				tc.method, tc.status, rec.Code)
		}

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != origin {
			t.Fatalf("%s: unexpected allowed origin %q", tc.method, got)
		}
	}
}

func TestConfiguredFrontendRejectsOtherStacks(t *testing.T) {
	t.Setenv(
		"FRONTEND_URL",
		"https://demo-angular-go-pg-front.valadeploy.internal",
	)

	for _, origin := range []string{
		"https://autre-projet-front.valadeploy.internal",
		"https://memoire-demo-frontend.valadeploy.internal",
		"https://evil.example.com",
	} {
		req := httptest.NewRequest(http.MethodOptions, "/api/v1/tasks", nil)
		req.Header.Set("Origin", origin)

		rec := httptest.NewRecorder()
		routes(nil).ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s: expected 403, got %d", origin, rec.Code)
		}

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Fatalf("%s: unexpected CORS header %q", origin, got)
		}
	}
}

func TestConfiguredFrontendAllowsLocalDevelopment(t *testing.T) {
	t.Setenv(
		"FRONTEND_URL",
		"https://demo-angular-go-pg-front.valadeploy.internal",
	)

	origin := "http://localhost:4200"

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/tasks", nil)
	req.Header.Set("Origin", origin)

	rec := httptest.NewRecorder()
	routes(nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != origin {
		t.Fatalf("unexpected allowed origin %q", got)
	}
}
