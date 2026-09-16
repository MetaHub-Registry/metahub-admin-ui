package server

import (
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func testAssets() fs.FS {
	return fstest.MapFS{
		"index.html":    &fstest.MapFile{Data: []byte("<html><body>metahub-admin-ui spa</body></html>")},
		"assets/app.js": &fstest.MapFile{Data: []byte("console.log('spa')")},
	}
}

func TestHealthEndpoint(t *testing.T) {
	srv := httptest.NewServer(New(Options{Assets: testAssets(), Version: "test"}))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/health")
	if err != nil {
		t.Fatalf("GET /api/v1/health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status = %q, want ok", body["status"])
	}
	if body["version"] != "test" {
		t.Errorf("version = %q, want test", body["version"])
	}
}

func TestResolveStubReturnsNoMatch(t *testing.T) {
	// Skeleton ships an empty inventory: every well-formed lookup must be the
	// spec-compliant no-match (404 {}) — never a 5xx.
	srv := httptest.NewServer(New(Options{Assets: testAssets()}))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/resolve?ip=10.10.1.42")
	if err != nil {
		t.Fatalf("GET /api/v1/resolve: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(body) != 0 {
		t.Errorf("body = %v, want empty object", body)
	}
}

func TestResolveRejectsMissingAndInvalidIP(t *testing.T) {
	srv := httptest.NewServer(New(Options{Assets: testAssets()}))
	defer srv.Close()

	for _, tc := range []struct {
		query  string
		status int
	}{
		{query: "", status: http.StatusBadRequest},
		{query: "?ip=", status: http.StatusBadRequest},
		{query: "?ip=not-an-ip", status: http.StatusBadRequest},
		{query: "?ip=999.1.1.1", status: http.StatusBadRequest},
	} {
		resp, err := http.Get(srv.URL + "/api/v1/resolve" + tc.query)
		if err != nil {
			t.Fatalf("GET /api/v1/resolve%s: %v", tc.query, err)
		}
		resp.Body.Close()
		if resp.StatusCode != tc.status {
			t.Errorf("query %q: status = %d, want %d", tc.query, resp.StatusCode, tc.status)
		}
	}
}

func TestSPAServesFilesAndFallsBackToIndex(t *testing.T) {
	srv := httptest.NewServer(New(Options{Assets: testAssets()}))
	defer srv.Close()

	// Real file is served.
	resp, err := http.Get(srv.URL + "/assets/app.js")
	if err != nil {
		t.Fatalf("GET /assets/app.js: %v", err)
	}
	got, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(got), "console.log") {
		t.Fatalf("asset: status = %d, body = %q", resp.StatusCode, got)
	}

	// Unknown routes fall back to index.html (client-side routing).
	for _, path := range []string{"/", "/lookup", "/deeply/nested/route"} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		got, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK || !strings.Contains(string(got), "metahub-admin-ui spa") {
			t.Errorf("%s: status = %d, body = %q — want index.html fallback", path, resp.StatusCode, got)
		}
	}
}

func TestSPAMissingAssetsReturns404Not5xx(t *testing.T) {
	// Nil assets ⇒ no SPA mount; an empty FS with no index.html must not
	// produce a 5xx either.
	srv := httptest.NewServer(New(Options{Assets: fstest.MapFS{}}))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/whatever")
	if err != nil {
		t.Fatalf("GET /whatever: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode >= 500 {
		t.Fatalf("status = %d, want < 500", resp.StatusCode)
	}
}
