package inventory

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}

const validYAML = `
inventory:
  entries:
    - ip: "10.10.1.42"
      profile: "gpu/a100"
      hostname: "gpu-h100-042.a100.cluster.example.com"
    - cidr: "10.0.1.0/24"
      profile: "zen4"
  hostname:
    lookup_ttl: 90s
    rules:
      - pattern: '\.a100\.cluster\.example\.com$'
        profile: "gpu/a100"
      - pattern: '\.zen4\.cpu\.example\.com$'
        profile: "arch/zen4"
`

func TestLoadConfigValid(t *testing.T) {
	cfg, err := LoadConfig(writeTemp(t, validYAML))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if len(cfg.Entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(cfg.Entries))
	}
	if cfg.Entries[0].Profile != "gpu/a100" || cfg.Entries[0].Hostname == "" {
		t.Errorf("entry[0] = %+v", cfg.Entries[0])
	}
	if cfg.Hostname.LookupTTL != 90*time.Second {
		t.Errorf("lookup_ttl = %v, want 90s", cfg.Hostname.LookupTTL)
	}
	if len(cfg.Hostname.Rules) != 2 {
		t.Fatalf("rules = %d, want 2", len(cfg.Hostname.Rules))
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	if _, err := LoadConfig(filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestValidateEmptyConfig(t *testing.T) {
	// An empty inventory is valid — the service answers 404 no-match.
	if err := (Config{}).Validate(); err != nil {
		t.Fatalf("empty config must be valid, got: %v", err)
	}
}

func TestValidateFailsNamingTheEntry(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Config
		wantSub string // substring the error must contain
	}{
		{
			name:    "invalid regex blocks startup",
			cfg:     Config{Hostname: HostnameConfig{Rules: []Rule{{Pattern: `\.a100[`, Profile: "x"}}}},
			wantSub: `hostname.rules[0]: invalid pattern '\.a100['`,
		},
		{
			name: "invalid CIDR blocks startup",
			cfg: Config{Entries: []Entry{
				{CIDR: "10.0.1.0/64", Profile: "zen4"},
			}},
			wantSub: `entries[0]: invalid cidr "10.0.1.0/64"`,
		},
		{
			name: "invalid IP blocks startup",
			cfg: Config{Entries: []Entry{
				{IP: "999.1.1.1", Profile: "x"},
			}},
			wantSub: `entries[0]: invalid ip "999.1.1.1"`,
		},
		{
			name: "empty profile blocks startup",
			cfg: Config{Entries: []Entry{
				{IP: "10.0.0.1"},
			}},
			wantSub: `entries[0]: empty profile`,
		},
		{
			name: "empty rule profile blocks startup",
			cfg: Config{Hostname: HostnameConfig{Rules: []Rule{
				{Pattern: `\.x$`},
			}}},
			wantSub: `hostname.rules[0]: empty profile`,
		},
		{
			name: "empty rule pattern blocks startup",
			cfg: Config{Hostname: HostnameConfig{Rules: []Rule{
				{Profile: "x"},
			}}},
			wantSub: `hostname.rules[0]: empty pattern`,
		},
		{
			name: "entry without ip and cidr blocks startup",
			cfg: Config{Entries: []Entry{
				{Profile: "x"},
			}},
			wantSub: `entries[0]: neither ip nor cidr set`,
		},
		{
			name: "entry with both ip and cidr blocks startup",
			cfg: Config{Entries: []Entry{
				{IP: "10.0.0.1", CIDR: "10.0.0.0/8", Profile: "x"},
			}},
			wantSub: `entries[0]: ip and cidr are mutually exclusive`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !contains(err.Error(), tc.wantSub) {
				t.Errorf("error = %q, want substring %q", err.Error(), tc.wantSub)
			}
		})
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
