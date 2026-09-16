package inventory

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// PTRLookup reverse-resolves an IP to hostnames. Abstracted so tests can
// stub DNS; the production adapter is NetPTRLookup.
type PTRLookup func(ctx context.Context, ip string) ([]string, error)

// NetPTRLookup is the production PTR resolver (system resolver).
func NetPTRLookup(ctx context.Context, ip string) ([]string, error) {
	return net.DefaultResolver.LookupAddr(ctx, ip)
}

// Result is a successful resolution outcome.
type Result struct {
	Profile  string
	Hostname string // display hostname when known (exact entry or PTR)
	Source   string // "ip:<i>", "hostname:<i>", "cidr:<i>" — winning entry
}

// Engine evaluates the ordered resolution chain: exact-IP → hostname pattern
// (PTR, TTL-cached incl. negative) → CIDR longest-prefix. First match wins.
type Engine struct {
	cfg   Config // original config (snapshot for the overview UI)
	exact map[string]indexedEntry
	cidrs []cidrEntry
	rules []compiledRule
	ptr   PTRLookup
	ttl   time.Duration

	cacheMu sync.Mutex
	cache   map[string]cacheEntry

	now func() time.Time // injectable clock (tests)
}

type indexedEntry struct {
	entry Entry
	idx   int
}

type cidrEntry struct {
	net     *net.IPNet
	idx     int
	text    string
	profile string
}

type compiledRule struct {
	re      *regexp.Regexp
	pattern string
	profile string
	idx     int
}

type cacheEntry struct {
	names   []string // trailing dots stripped; empty ⇒ negative outcome
	found   bool
	expires time.Time
}

// Build compiles the validated config into an Engine. Call Validate() first —
// Build re-checks and surfaces the same named errors defensively.
func Build(cfg Config, ptr PTRLookup) (*Engine, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	e := &Engine{
		cfg:   cfg,
		exact: make(map[string]indexedEntry),
		ptr:   ptr,
		cache: make(map[string]cacheEntry),
		now:   time.Now,
	}
	e.ttl = cfg.Hostname.LookupTTL
	if e.ttl <= 0 {
		e.ttl = DefaultLookupTTL
	}
	for i, entry := range cfg.Entries {
		if entry.IP != "" {
			ip := net.ParseIP(entry.IP)
			// Validate() guarantees a parseable IP.
			e.exact[ip.String()] = indexedEntry{entry: entry, idx: i}
			continue
		}
		_, ipNet, err := net.ParseCIDR(entry.CIDR)
		if err != nil {
			return nil, fmt.Errorf("entries[%d]: invalid cidr %q: %v", i, entry.CIDR, err)
		}
		e.cidrs = append(e.cidrs, cidrEntry{net: ipNet, idx: i, text: entry.CIDR, profile: entry.Profile})
	}
	for i, rule := range cfg.Hostname.Rules {
		re, err := regexp.Compile(ruleRegex(rule.Pattern))
		if err != nil {
			return nil, fmt.Errorf("hostname.rules[%d]: invalid pattern '%s': %v", i, rule.Pattern, err)
		}
		e.rules = append(e.rules, compiledRule{re: re, pattern: rule.Pattern, profile: rule.Profile, idx: i})
	}
	return e, nil
}

// TTL returns the configured PTR cache TTL (covers negative outcomes too).
func (e *Engine) TTL() time.Duration {
	return e.ttl
}

// Snapshot is the read-only view of the configured sources for the
// overview UI (GET /api/v1/inventory).
type Snapshot struct {
	Entries  []EntryView  `json:"entries"`
	Hostname HostnameView `json:"hostname"`
}

// EntryView is one configured exact-IP or CIDR entry.
type EntryView struct {
	IP       string `json:"ip,omitempty"`
	CIDR     string `json:"cidr,omitempty"`
	Profile  string `json:"profile"`
	Hostname string `json:"hostname,omitempty"` // display data
}

// HostnameView groups the PTR tier configuration.
type HostnameView struct {
	LookupTTL string     `json:"lookup_ttl"`
	Rules     []RuleView `json:"rules"`
}

// RuleView is one ordered hostname-pattern rule.
type RuleView struct {
	Pattern string `json:"pattern"`
	Profile string `json:"profile"`
}

// EmptySnapshot is the snapshot of a service without configured sources.
func EmptySnapshot() Snapshot {
	return Snapshot{
		Entries:  []EntryView{},
		Hostname: HostnameView{Rules: []RuleView{}},
	}
}

// Snapshot renders the configured sources in configured order.
func (e *Engine) Snapshot() Snapshot {
	s := EmptySnapshot()
	s.Hostname.LookupTTL = e.ttl.String()
	for _, entry := range e.cfg.Entries {
		s.Entries = append(s.Entries, EntryView{
			IP:       entry.IP,
			CIDR:     entry.CIDR,
			Profile:  entry.Profile,
			Hostname: entry.Hostname,
		})
	}
	for _, rule := range e.cfg.Hostname.Rules {
		s.Hostname.Rules = append(s.Hostname.Rules, RuleView{Pattern: rule.Pattern, Profile: rule.Profile})
	}
	return s
}

// Resolve evaluates the chain for ip. ok=false ⇒ no-match (the handler
// answers the documented 404 {}).
func (e *Engine) Resolve(ctx context.Context, ip net.IP) (Result, bool) {
	// 1. Exact-IP — the most specific operator statement.
	if entry, ok := e.exact[ip.String()]; ok {
		log.WithFields(log.Fields{
			"ip": ip.String(), "source": fmt.Sprintf("ip:%d", entry.idx),
			"profile": entry.entry.Profile,
		}).Debug("resolve: exact-ip match")
		return Result{
			Profile:  entry.entry.Profile,
			Hostname: entry.entry.Hostname,
			Source:   fmt.Sprintf("ip:%d", entry.idx),
		}, true
	}

	// 2. Hostname patterns — PTR lookup (TTL-cached, incl. negative), each
	// name checked against the ordered rules; first match wins.
	names, cacheHit := e.ptrNames(ctx, ip)
	for _, name := range names {
		for _, rule := range e.rules {
			if rule.re.MatchString(name) {
				log.WithFields(log.Fields{
					"ip": ip.String(), "source": fmt.Sprintf("hostname:%d", rule.idx),
					"pattern": rule.pattern, "profile": rule.profile,
					"hostname": name, "ptr_cache": cacheHit,
				}).Debug("resolve: hostname match")
				return Result{
					Profile:  rule.profile,
					Hostname: name,
					Source:   fmt.Sprintf("hostname:%d", rule.idx),
				}, true
			}
		}
	}

	// 3. CIDR — longest matching prefix; subnet-level fallback.
	if best, ok := e.longestPrefix(ip); ok {
		log.WithFields(log.Fields{
			"ip": ip.String(), "source": fmt.Sprintf("cidr:%d", best.idx),
			"cidr": best.text, "profile": best.profile, "ptr_cache": cacheHit,
		}).Debug("resolve: cidr match")
		return Result{
			Profile: best.profile,
			Source:  fmt.Sprintf("cidr:%d", best.idx),
		}, true
	}

	log.WithFields(log.Fields{"ip": ip.String(), "ptr_cache": cacheHit}).Debug("resolve: no match")
	return Result{}, false
}

// ptrNames returns the cached (or freshly looked-up) PTR names with trailing
// dots stripped, plus whether this came from cache. Negative outcomes are
// cached too; transient DNS errors are NOT cached (retried next request).
func (e *Engine) ptrNames(ctx context.Context, ip net.IP) ([]string, bool) {
	key := ip.String()

	e.cacheMu.Lock()
	cached, hit := e.cache[key]
	e.cacheMu.Unlock()
	if hit && e.now().Before(cached.expires) {
		return cached.names, true
	}

	names, err := e.ptr(ctx, key)
	if err != nil {
		var dnsErr *net.DNSError
		if asDNSError(err, &dnsErr) && dnsErr.IsNotFound {
			// Negative outcome — cache it (spec: TTL cache incl. negative).
			e.store(key, cacheEntry{found: false, expires: e.now().Add(e.ttl)})
			return nil, false
		}
		// Transient failure — do not cache; this request just skips the tier.
		log.WithError(err).WithField("ip", key).Debug("resolve: PTR lookup failed (not cached)")
		return nil, false
	}
	cleaned := make([]string, 0, len(names))
	for _, n := range names {
		cleaned = append(cleaned, strings.TrimSuffix(n, "."))
	}
	e.store(key, cacheEntry{names: cleaned, found: true, expires: e.now().Add(e.ttl)})
	return cleaned, false
}

func (e *Engine) store(key string, c cacheEntry) {
	e.cacheMu.Lock()
	defer e.cacheMu.Unlock()
	e.cache[key] = c
}

func asDNSError(err error, target **net.DNSError) bool {
	if e, ok := err.(*net.DNSError); ok {
		*target = e
		return true
	}
	return false
}

// longestPrefix returns the most specific matching CIDR (ties: first
// configured).
func (e *Engine) longestPrefix(ip net.IP) (cidrEntry, bool) {
	var best cidrEntry
	bestOnes, found := -1, false
	for _, c := range e.cidrs {
		if !c.net.Contains(ip) {
			continue
		}
		ones, _ := c.net.Mask.Size()
		if ones > bestOnes {
			best, bestOnes, found = c, ones, true
		}
	}
	return best, found
}
