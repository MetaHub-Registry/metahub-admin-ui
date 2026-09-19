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

// learnEngine: static CIDR default so learned hits are distinguishable.
func engineForCIDR(t *testing.T) *inventory.Engine {
	t.Helper()
	e, err := inventory.Build(inventory.Config{
		Entries: []inventory.Entry{
			{CIDR: "10.0.0.0/8", Profile: "subnet-default"},
		},
	}, func(context.Context, string) ([]string, error) {
		return nil, &net.DNSError{Err: "no such host", IsNotFound: true}
	})
	if err != nil {
		t.Fatalf("build engine: %v", err)
	}
	return e
}

func TestLearnRoundTrip(t *testing.T) {
	srv2 := httptest.NewServer(New(Options{Assets: testAssets(), Engine: engineForCIDR(t)}))
	defer srv2.Close()

	// First learn → 201.
	resp, err := http.Post(srv2.URL+"/api/v1/entries", "application/json",
		strings.NewReader(`{"ip":"10.9.9.9","profile":"node/aws","learned":true}`))
	if err != nil {
		t.Fatalf("learn POST: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("first learn status = %d, want 201", resp.StatusCode)
	}

	// Resolve → learned source.
	res, err := http.Get(srv2.URL + "/api/v1/resolve?ip=10.9.9.9")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	defer res.Body.Close()
	var body struct {
		Profile string `json:"profile"`
		Source  string `json:"source"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Profile != "node/aws" || body.Source != "learned:10.9.9.9" {
		t.Fatalf("resolved %+v", body)
	}

	// Re-learn (last-write-wins) → 200.
	resp, err = http.Post(srv2.URL+"/api/v1/entries", "application/json",
		strings.NewReader(`{"ip":"10.9.9.9","profile":"other/role","learned":true}`))
	if err != nil {
		t.Fatalf("re-learn POST: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("re-learn status = %d, want 200", resp.StatusCode)
	}

	// DELETE removes the learned entry → CIDR default visible again.
	req, _ := http.NewRequest(http.MethodDelete, srv2.URL+"/api/v1/entries?ip=10.9.9.9", nil)
	del, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	del.Body.Close()
	if del.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE status = %d, want 204", del.StatusCode)
	}
	res, err = http.Get(srv2.URL + "/api/v1/resolve?ip=10.9.9.9")
	if err != nil {
		t.Fatalf("resolve after removal: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("after removal status = %d, want 200 (CIDR hit)", res.StatusCode)
	}
}

func TestLearnValidation(t *testing.T) {
	srv := httptest.NewServer(New(Options{Assets: testAssets(), Engine: engineForCIDR(t)}))
	defer srv.Close()

	for _, tc := range []struct {
		name   string
		body   string
		status int
	}{
		{"cidr+learned rejected", `{"cidr":"10.0.0.0/8","profile":"x","learned":true}`, http.StatusBadRequest},
		{"empty profile", `{"ip":"10.9.9.9","learned":true}`, http.StatusBadRequest},
		{"invalid ip", `{"ip":"nope","profile":"x","learned":true}`, http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Post(srv.URL+"/api/v1/entries", "application/json",
				strings.NewReader(tc.body))
			if err != nil {
				t.Fatalf("POST: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tc.status {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tc.status)
			}
		})
	}
}

func TestLearnSnapshotFlag(t *testing.T) {
	srv := httptest.NewServer(New(Options{Assets: testAssets(), Engine: engineForCIDR(t)}))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/entries", "application/json",
		strings.NewReader(`{"ip":"10.9.9.9","profile":"node/aws","learned":true}`))
	if err != nil {
		t.Fatalf("learn POST: %v", err)
	}
	resp.Body.Close()

	snap, err := http.Get(srv.URL + "/api/v1/inventory")
	if err != nil {
		t.Fatalf("GET inventory: %v", err)
	}
	defer snap.Body.Close()
	var snapshot struct {
		Entries []struct {
			IP      string `json:"ip"`
			Runtime bool   `json:"runtime"`
			Learned bool   `json:"learned"`
		} `json:"entries"`
	}
	if err := json.NewDecoder(snap.Body).Decode(&snapshot); err != nil {
		t.Fatalf("decode: %v", err)
	}
	found := false
	for _, e := range snapshot.Entries {
		if e.Learned {
			found = true
			if e.IP != "10.9.9.9" || !e.Runtime {
				t.Errorf("learned snapshot entry = %+v", e)
			}
		}
	}
	if !found {
		t.Fatal("no learned-flagged entry in snapshot")
	}
}
