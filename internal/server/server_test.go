package server

import (
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/metahub-registry/metahub-admin-ui/internal/inventory"
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
	// Engine without sources: every well-formed lookup must be the
	// spec-compliant no-match (404 {}) — never a 5xx.
	srv := httptest.NewServer(New(Options{Assets: testAssets(), Engine: emptyEngine(t)}))
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

func emptyEngine(t *testing.T) *inventory.Engine {
	t.Helper()
	e, err := inventory.Build(inventory.Config{}, func(context.Context, string) ([]string, error) {
		return nil, &net.DNSError{Err: "no such host", IsNotFound: true}
	})
	if err != nil {
		t.Fatalf("build engine: %v", err)
	}
	return e
}

func testEngine(t *testing.T) *inventory.Engine {
	t.Helper()
	cfg := inventory.Config{
		Entries: []inventory.Entry{
			{IP: "10.10.1.42", Profile: "gpu/a100", Hostname: "gpu-h100-042.a100.cluster.example.com"},
			{CIDR: "10.0.1.0/24", Profile: "zen4"},
		},
		Hostname: inventory.HostnameConfig{LookupTTL: time.Minute, Rules: []inventory.Rule{
			{Pattern: `\.a100\.cluster\.example\.com$`, Profile: "gpu/a100"},
		}},
	}
	e, err := inventory.Build(cfg, func(context.Context, string) ([]string, error) {
		return nil, &net.DNSError{Err: "no such host", IsNotFound: true}
	})
	if err != nil {
		t.Fatalf("build engine: %v", err)
	}
	return e
}

func TestResolveHitsExactIPEntry(t *testing.T) {
	srv := httptest.NewServer(New(Options{Assets: testAssets(), Engine: testEngine(t)}))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/resolve?ip=10.10.1.42")
	if err != nil {
		t.Fatalf("GET /api/v1/resolve: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body struct {
		IP       string `json:"ip"`
		Profile  string `json:"profile"`
		Hostname string `json:"hostname"`
		Source   string `json:"source"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Profile != "gpu/a100" || body.Source != "ip:0" {
		t.Errorf("body = %+v, want gpu/a100 @ ip:0", body)
	}
	if body.Hostname != "gpu-h100-042.a100.cluster.example.com" {
		t.Errorf("hostname = %q", body.Hostname)
	}
}

func TestResolveCIDROmitsHostname(t *testing.T) {
	srv := httptest.NewServer(New(Options{Assets: testAssets(), Engine: testEngine(t)}))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/resolve?ip=10.0.1.7")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(raw), `"source":"cidr:1"`) {
		t.Fatalf("status = %d body = %s", resp.StatusCode, raw)
	}
	if strings.Contains(string(raw), "hostname") {
		t.Errorf("CIDR response must omit hostname: %s", raw)
	}
}

func TestResolveBearerTokenOptional(t *testing.T) {
	newSrv := func() *httptest.Server {
		return httptest.NewServer(New(Options{
			Assets: testAssets(), Engine: emptyEngine(t), InventoryToken: "sekrit",
		}))
	}

	// Health must stay open (k8s probes) while resolve is protected.
	srv := newSrv()
	defer srv.Close()
	if resp, err := http.Get(srv.URL + "/api/v1/health"); err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("health: status=%d err=%v — must not require a token", resp.StatusCode, err)
	}

	// No token / wrong token ⇒ 401; correct bearer ⇒ passes auth (then 404 no-match).
	for _, tc := range []struct {
		name   string
		auth   string
		status int
	}{
		{name: "no token", auth: "", status: http.StatusUnauthorized},
		{name: "wrong token", auth: "Bearer nope", status: http.StatusUnauthorized},
		{name: "correct token", auth: "Bearer sekrit", status: http.StatusNotFound},
	} {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/resolve?ip=10.0.0.1", nil)
		if tc.auth != "" {
			req.Header.Set("Authorization", tc.auth)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		resp.Body.Close()
		if resp.StatusCode != tc.status {
			t.Errorf("%s: status = %d, want %d", tc.name, resp.StatusCode, tc.status)
		}
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
