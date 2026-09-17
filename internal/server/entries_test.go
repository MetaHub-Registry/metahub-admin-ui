package server

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/metahub-registry/metahub-admin-ui/internal/inventory"
)

func engineForEntries(t *testing.T) *inventory.Engine {
	t.Helper()
	e, err := inventory.Build(inventory.Config{
		Entries: []inventory.Entry{
			{IP: "10.10.1.42", Profile: "gpu/a100"},
			{CIDR: "10.10.1.0/24", Profile: "zen4"},
		},
	}, func(context.Context, string) ([]string, error) {
		return nil, &net.DNSError{Err: "no such host", IsNotFound: true}
	})
	if err != nil {
		t.Fatalf("build engine: %v", err)
	}
	return e
}

func TestAddEntryRoundTrip(t *testing.T) {
	srv := httptest.NewServer(New(Options{Assets: testAssets(), Engine: engineForEntries(t)}))
	defer srv.Close()

	// Register the agent IP → 201.
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/entries",
		strings.NewReader(`{"ip":"192.168.5.115","profile":"e2e/42","hostname":"frame.hvk"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /api/v1/entries: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}

	// Resolve → runtime source + profile.
	res, err := http.Get(srv.URL + "/api/v1/resolve?ip=192.168.5.115")
	if err != nil {
		t.Fatalf("GET resolve: %v", err)
	}
	defer res.Body.Close()
	var body struct {
		Profile string `json:"profile"`
		Source  string `json:"source"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Profile != "e2e/42" || body.Source != "runtime-ip:192.168.5.115" {
		t.Fatalf("resolved %+v", body)
	}

	// Duplicate → 409.
	req, _ = http.NewRequest(http.MethodPost, srv.URL+"/api/v1/entries",
		strings.NewReader(`{"ip":"192.168.5.115","profile":"other"}`))
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("duplicate POST: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate status = %d, want 409", resp.StatusCode)
	}

	// Snapshot flags the runtime entry.
	snap, err := http.Get(srv.URL + "/api/v1/inventory")
	if err != nil {
		t.Fatalf("GET inventory: %v", err)
	}
	defer snap.Body.Close()
	var snapshot struct {
		Entries []struct {
			IP      string `json:"ip"`
			Runtime bool   `json:"runtime"`
		} `json:"entries"`
	}
	if err := json.NewDecoder(snap.Body).Decode(&snapshot); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	runtimeFlags := 0
	for _, e := range snapshot.Entries {
		if e.Runtime {
			runtimeFlags++
			if e.IP != "192.168.5.115" {
				t.Errorf("flagged entry = %+v", e)
			}
		}
	}
	if runtimeFlags != 1 {
		t.Errorf("runtime flags = %d, want 1", runtimeFlags)
	}

	// Deregister → 204, then static fallback.
	req, _ = http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/entries?ip=192.168.5.115", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE status = %d, want 204", resp.StatusCode)
	}
	res, err = http.Get(srv.URL + "/api/v1/resolve?ip=192.168.5.115")
	if err != nil {
		t.Fatalf("GET resolve after removal: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("after removal status = %d, want 404 (IP not in static config)", res.StatusCode)
	}
}

func TestAddEntryValidation(t *testing.T) {
	srv := httptest.NewServer(New(Options{Assets: testAssets(), Engine: engineForEntries(t)}))
	defer srv.Close()

	for _, tc := range []struct {
		name   string
		body   string
		status int
	}{
		{"empty profile", `{"ip":"10.9.9.9"}`, http.StatusBadRequest},
		{"both ip and cidr", `{"ip":"10.9.9.9","cidr":"10.0.0.0/8","profile":"x"}`, http.StatusBadRequest},
		{"neither ip nor cidr", `{"profile":"x"}`, http.StatusBadRequest},
		{"invalid ip", `{"ip":"nope","profile":"x"}`, http.StatusBadRequest},
		{"invalid cidr", `{"cidr":"10.0.0.0/99","profile":"x"}`, http.StatusBadRequest},
		{"malformed json", `not json`, http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/entries", strings.NewReader(tc.body))
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("POST: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tc.status {
				var body map[string]any
				_ = json.NewDecoder(resp.Body).Decode(&body)
				t.Fatalf("status = %d, want %d (body %v)", resp.StatusCode, tc.status, body)
			}
		})
	}
}

func TestDeleteEntrySemantics(t *testing.T) {
	srv := httptest.NewServer(New(Options{Assets: testAssets(), Engine: engineForEntries(t)}))
	defer srv.Close()

	// Both params → 400.
	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/entries?ip=1.2.3.4&cidr=10.0.0.0/8", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE both params: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("both params status = %d, want 400", resp.StatusCode)
	}

	// No params → 400.
	req, _ = http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/entries", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE no params: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("no params status = %d, want 400", resp.StatusCode)
	}

	// Static entry → 404 with the not-removable hint.
	req, _ = http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/entries?ip=10.10.1.42", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE static entry: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("static entry status = %d, want 404", resp.StatusCode)
	}
	var body map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if !strings.Contains(body["error"], "static YAML entry") {
		t.Errorf("hint = %q", body["error"])
	}

	// Unknown entry → plain 404.
	req, _ = http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/entries?ip=172.16.1.1", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE unknown entry: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown entry status = %d, want 404", resp.StatusCode)
	}
}

func TestEntriesBearerProtected(t *testing.T) {
	srv := httptest.NewServer(New(Options{
		Assets: testAssets(), Engine: engineForEntries(t), InventoryToken: "sekrit",
	}))
	defer srv.Close()

	// POST without token → 401.
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/entries",
		strings.NewReader(`{"ip":"10.9.9.9","profile":"x"}`))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("POST without token status = %d, want 401", resp.StatusCode)
	}

	// DELETE without token → 401.
	req, _ = http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/entries?ip=10.9.9.9", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("DELETE without token status = %d, want 401", resp.StatusCode)
	}

	// With the token → passes auth.
	req, _ = http.NewRequest(http.MethodPost, srv.URL+"/api/v1/entries",
		strings.NewReader(`{"ip":"10.9.9.9","profile":"x"}`))
	req.Header.Set("Authorization", "Bearer sekrit")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST with token: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST with token status = %d, want 201", resp.StatusCode)
	}
}
