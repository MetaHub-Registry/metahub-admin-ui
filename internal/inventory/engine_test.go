package inventory

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

// countingPTR is a stub resolver that answers from a fixed map and counts
// real lookups (cache assertions).
type countingPTR struct {
	answers map[string][]string // nil entry ⇒ NXDOMAIN (negative)
	fail    map[string]error    // transient failures
	calls   map[string]int
}

func newPTR() *countingPTR {
	return &countingPTR{
		answers: map[string][]string{},
		fail:    map[string]error{},
		calls:   map[string]int{},
	}
}

func (p *countingPTR) lookup(_ context.Context, ip string) ([]string, error) {
	p.calls[ip]++
	if err, ok := p.fail[ip]; ok {
		return nil, err
	}
	names, ok := p.answers[ip]
	if !ok {
		return nil, &net.DNSError{Err: "no such host", Name: ip, IsNotFound: true}
	}
	return names, nil
}

func (p *countingPTR) engine(t *testing.T, cfg Config) *Engine {
	t.Helper()
	e, err := Build(cfg, p.lookup)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return e
}

func testEngine(t *testing.T, cfg Config, ptr PTRLookup) *Engine {
	t.Helper()
	e, err := Build(cfg, ptr)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return e
}

func ip(t *testing.T, s string) net.IP {
	t.Helper()
	p := net.ParseIP(s)
	if p == nil {
		t.Fatalf("bad test ip %q", s)
	}
	return p
}

// --- spec scenarios ---------------------------------------------------------

func TestResolveExactIPEntry(t *testing.T) {
	e := testEngine(t, Config{Entries: []Entry{
		{IP: "10.10.1.42", Profile: "gpu/a100", Hostname: "gpu-h100-042.a100.cluster.example.com"},
	}}, newPTR().lookup)

	res, ok := e.Resolve(context.Background(), ip(t, "10.10.1.42"))
	if !ok {
		t.Fatal("want match")
	}
	if res.Profile != "gpu/a100" {
		t.Errorf("profile = %q", res.Profile)
	}
	if res.Source != "ip:0" {
		t.Errorf("source = %q, want ip:0", res.Source)
	}
	if res.Hostname != "gpu-h100-042.a100.cluster.example.com" {
		t.Errorf("hostname = %q", res.Hostname)
	}
}

func TestResolveHostnamePatternBeatsCIDR(t *testing.T) {
	ptr := newPTR()
	ptr.answers["10.10.1.42"] = []string{"gpu-h100-042.a100.cluster.example.com."}
	e := testEngine(t, Config{
		Entries: []Entry{{CIDR: "10.10.1.0/24", Profile: "zen4"}},
		Hostname: HostnameConfig{Rules: []Rule{
			{Pattern: `\.a100\.cluster\.example\.com$`, Profile: "gpu/a100"},
		}},
	}, ptr.lookup)

	res, ok := e.Resolve(context.Background(), ip(t, "10.10.1.42"))
	if !ok {
		t.Fatal("want match")
	}
	if res.Profile != "gpu/a100" {
		t.Errorf("profile = %q, want gpu/a100 (hostname beats cidr)", res.Profile)
	}
	if res.Source != "hostname:0" {
		t.Errorf("source = %q", res.Source)
	}
	if res.Hostname != "gpu-h100-042.a100.cluster.example.com" {
		t.Errorf("hostname = %q (trailing dot must be stripped)", res.Hostname)
	}
}

func TestResolveLongestPrefixWins(t *testing.T) {
	e := testEngine(t, Config{Entries: []Entry{
		{CIDR: "10.0.0.0/8", Profile: "generic"},
		{CIDR: "10.10.1.0/24", Profile: "zen4"},
	}}, newPTR().lookup)

	res, ok := e.Resolve(context.Background(), ip(t, "10.10.1.42"))
	if !ok {
		t.Fatal("want match")
	}
	if res.Profile != "zen4" || res.Source != "cidr:1" {
		t.Errorf("got %q/%q, want zen4/cidr:1 (/24 beats /8)", res.Profile, res.Source)
	}
}

func TestResolveUnknownIP(t *testing.T) {
	e := testEngine(t, Config{Entries: []Entry{
		{CIDR: "192.168.0.0/16", Profile: "x"},
	}}, newPTR().lookup)

	if _, ok := e.Resolve(context.Background(), ip(t, "10.10.1.42")); ok {
		t.Fatal("want no-match")
	}
}

func TestResolveNoPTRContinuesToCIDR(t *testing.T) {
	// ptr has no entry for 10.10.1.42 ⇒ NXDOMAIN (negative)
	e := testEngine(t, Config{Entries: []Entry{
		{CIDR: "10.10.1.0/24", Profile: "zen4"},
	}}, newPTR().lookup)

	res, ok := e.Resolve(context.Background(), ip(t, "10.10.1.42"))
	if !ok || res.Profile != "zen4" {
		t.Fatalf("got ok=%v profile=%q — CIDR tier must still run", ok, res.Profile)
	}
}

func TestResolvePTRFailureDoesNotFailTheRequest(t *testing.T) {
	// Transient DNS failure (not NXDOMAIN): hostname tier skipped, not cached,
	// no error surfaced.
	ptr := newPTR()
	ptr.fail["10.10.1.42"] = errors.New("connection refused")
	e := testEngine(t, Config{}, ptr.lookup)

	if _, ok := e.Resolve(context.Background(), ip(t, "10.10.1.42")); ok {
		t.Fatal("want no-match")
	}
	if ptr.calls["10.10.1.42"] != 1 {
		t.Errorf("ptr calls = %d — transient failures must not be cached", ptr.calls["10.10.1.42"])
	}
	// Not cached: a second resolve retries the lookup.
	_, _ = e.Resolve(context.Background(), ip(t, "10.10.1.42"))
	if ptr.calls["10.10.1.42"] != 2 {
		t.Errorf("ptr calls = %d, want 2 (failure must not be cached)", ptr.calls["10.10.1.42"])
	}
}

// --- cache behaviour --------------------------------------------------------

func TestResolveNegativeOutcomeIsCached(t *testing.T) {
	ptr := newPTR()
	e := testEngine(t, Config{
		Entries:  []Entry{{CIDR: "10.10.1.0/24", Profile: "zen4"}},
		Hostname: HostnameConfig{LookupTTL: time.Minute, Rules: []Rule{
			{Pattern: `\.x$`, Profile: "x"},
		}},
	}, ptr.lookup)

	for i := 0; i < 3; i++ {
		res, ok := e.Resolve(context.Background(), ip(t, "10.10.1.42"))
		if !ok || res.Profile != "zen4" {
			t.Fatalf("run %d: got ok=%v profile=%q", i, ok, res.Profile)
		}
	}
	if got := ptr.calls["10.10.1.42"]; got != 1 {
		t.Errorf("ptr calls = %d, want 1 (negative outcome cached for TTL)", got)
	}
}

func TestResolvePositiveOutcomeIsCached(t *testing.T) {
	ptr := newPTR()
	ptr.answers["10.10.1.42"] = []string{"gpu-h100-042.a100.cluster.example.com."}
	e := testEngine(t, Config{
		Hostname: HostnameConfig{LookupTTL: time.Minute, Rules: []Rule{
			{Pattern: `\.a100\.`, Profile: "gpu/a100"},
		}},
	}, ptr.lookup)

	for i := 0; i < 3; i++ {
		res, ok := e.Resolve(context.Background(), ip(t, "10.10.1.42"))
		if !ok || res.Profile != "gpu/a100" {
			t.Fatalf("run %d: ok=%v profile=%q", i, ok, res.Profile)
		}
	}
	if got := ptr.calls["10.10.1.42"]; got != 1 {
		t.Errorf("ptr calls = %d, want 1 (positive outcome cached for TTL)", got)
	}
}

func TestResolveCacheExpiresAfterTTL(t *testing.T) {
	ptr := newPTR()
	ptr.answers["10.10.1.42"] = []string{"a.a100."}
	e := testEngine(t, Config{
		Hostname: HostnameConfig{LookupTTL: time.Minute, Rules: []Rule{
			{Pattern: `\.a100\.$`, Profile: "x"},
		}},
	}, ptr.lookup)

	// Freeze the clock: no time passes, nothing expires.
	start := time.Now()
	e.now = func() time.Time { return start }

	_, _ = e.Resolve(context.Background(), ip(t, "10.10.1.42"))
	_, _ = e.Resolve(context.Background(), ip(t, "10.10.1.42"))
	if got := ptr.calls["10.10.1.42"]; got != 1 {
		t.Fatalf("ptr calls = %d, want 1 within TTL", got)
	}

	// Advance past the TTL → fresh lookup.
	e.now = func() time.Time { return start.Add(2 * time.Minute) }
	_, _ = e.Resolve(context.Background(), ip(t, "10.10.1.42"))
	if got := ptr.calls["10.10.1.42"]; got != 2 {
		t.Errorf("ptr calls = %d, want 2 after TTL expiry", got)
	}
}

// --- rule semantics ---------------------------------------------------------

func TestResolveRulesCaseInsensitiveAndOrdered(t *testing.T) {
	ptr := newPTR()
	ptr.answers["10.0.0.1"] = []string{"GPU-H100-042.A100.Cluster.Example.COM."}
	e := testEngine(t, Config{
		Hostname: HostnameConfig{Rules: []Rule{
			{Pattern: `\.gpu\.cluster\.example\.com$`, Profile: "should-not-match"},
			{Pattern: `\.a100\.cluster\.example\.com$`, Profile: "gpu/a100"},
		}},
	}, ptr.lookup)

	res, ok := e.Resolve(context.Background(), ip(t, "10.0.0.1"))
	if !ok || res.Profile != "gpu/a100" || res.Source != "hostname:1" {
		t.Fatalf("got ok=%v %+v — rules must match case-insensitively, in order", ok, res)
	}
	if res.Hostname != "GPU-H100-042.A100.Cluster.Example.COM" {
		t.Errorf("hostname = %q (display name keeps original case, no trailing dot)", res.Hostname)
	}
}

func TestResolveFirstMatchingRuleWins(t *testing.T) {
	ptr := newPTR()
	ptr.answers["10.0.0.1"] = []string{"node.a100.cluster.example.com."}
	e := testEngine(t, Config{
		Hostname: HostnameConfig{Rules: []Rule{
			{Pattern: `\.a100\.`, Profile: "first"},
			{Pattern: `\.cluster\.`, Profile: "second"},
		}},
	}, ptr.lookup)

	res, _ := e.Resolve(context.Background(), ip(t, "10.0.0.1"))
	if res.Profile != "first" {
		t.Errorf("profile = %q, want first (rule order decides)", res.Profile)
	}
}

func TestBuildDefaultsLookupTTL(t *testing.T) {
	e := testEngine(t, Config{}, newPTR().lookup)
	if e.ttl != DefaultLookupTTL {
		t.Errorf("ttl = %v, want default %v", e.ttl, DefaultLookupTTL)
	}
}

func TestBuildRejectsInvalidConfig(t *testing.T) {
	if _, err := Build(Config{Hostname: HostnameConfig{Rules: []Rule{{Pattern: "["}}}}, newPTR().lookup); err == nil {
		t.Fatal("Build must fail on invalid regex")
	}
}
