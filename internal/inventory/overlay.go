package inventory

import (
	"fmt"
	"net"
	"sync"
)

// ErrDuplicate marks an already-registered runtime key (HTTP 409).
var ErrDuplicate = fmt.Errorf("duplicate runtime entry")

// Overlay is the in-memory runtime entry store on top of the YAML sources
// (openspec change add-runtime-entry-api). Entries registered via
// POST /api/v1/entries live here until the process dies — a restart wipes
// the overlay, so automation can never poison operator-managed data.
//
// Resolution consults the overlay FIRST within the exact-IP tier (a runtime
// registration wins over a static entry for the same IP) and merges its
// CIDRs into the longest-prefix scan (runtime wins ties). The overlay never
// mutates static sources.
type Overlay struct {
	mu      sync.RWMutex
	exact   map[string]Entry // canonical net.IP.String() → entry
	cidrs   []overlayCIDR    // registration order
	learned map[string]Entry // canonical net.IP.String() → entry (last-write-wins)
}

type overlayCIDR struct {
	net   *net.IPNet
	text  string // as registered
	entry Entry
}

// newOverlay returns an empty overlay.
func newOverlay() Overlay {
	return Overlay{
		exact:   make(map[string]Entry),
		learned: make(map[string]Entry),
	}
}

// Add validates and registers one entry. Validation matches the YAML
// sources (exactly one of ip/cidr, non-empty profile). Re-registering the
// same key returns an error wrapping ErrDuplicate — deregister first
// (HTTP 409 at the API layer).
func (o *Overlay) Add(e Entry) error {
	if err := validateEntry(e, "entry"); err != nil {
		return err
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if e.IP != "" {
		key := net.ParseIP(e.IP).String() // validateEntry guarantees parseable
		if _, dup := o.exact[key]; dup {
			return fmt.Errorf("runtime entry %s: already registered: %w", key, ErrDuplicate)
		}
		o.exact[key] = e
		return nil
	}
	_, ipNet, err := net.ParseCIDR(e.CIDR)
	if err != nil {
		return fmt.Errorf("entry: invalid cidr %q: %v", e.CIDR, err)
	}
	for _, c := range o.cidrs {
		if c.net.String() == ipNet.String() {
			return fmt.Errorf("runtime entry %s: already registered: %w", e.CIDR, ErrDuplicate)
		}
	}
	o.cidrs = append(o.cidrs, overlayCIDR{net: ipNet, text: e.CIDR, entry: e})
	return nil
}

// Remove deregisters a runtime or learned entry by its ip or cidr string.
// Returns found=true when an overlay entry was removed. Static entries are
// never touched here — the Engine distinguishes the static case for the API
// hint.
func (o *Overlay) Remove(key string) bool {
	o.mu.Lock()
	defer o.mu.Unlock()

	if ip := net.ParseIP(key); ip != nil {
		canonical := ip.String()
		if _, ok := o.exact[canonical]; ok {
			delete(o.exact, canonical)
			return true
		}
		if _, ok := o.learned[canonical]; ok {
			delete(o.learned, canonical)
			return true
		}
		return false
	}
	if _, ipNet, err := net.ParseCIDR(key); err == nil {
		text := ipNet.String()
		for i, c := range o.cidrs {
			if c.net.String() == text {
				o.cidrs = append(o.cidrs[:i], o.cidrs[i+1:]...)
				return true
			}
		}
	}
	return false
}

// Learn stores/updates a learned ip→profile mapping (openspec
// add-learned-entries). Learned entries are last-write-wins per IP — they
// represent "the last explicit login from this IP" — and are consulted in
// the resolution chain AFTER hostname patterns and BEFORE CIDR entries.
// Only exact-IP mappings are accepted (a learned entry is a per-IP
// observation, not subnet knowledge).
func (o *Overlay) Learn(e Entry) error {
	if e.CIDR != "" {
		return fmt.Errorf("learned entry: cidr not supported — learned mappings are per-IP")
	}
	if err := validateEntry(e, "learned entry"); err != nil {
		return err
	}
	key := net.ParseIP(e.IP).String() // validateEntry guarantees parseable
	o.mu.Lock()
	defer o.mu.Unlock()
	o.learned[key] = e
	return nil
}

// Learned returns a snapshot of the learned mappings.
func (o *Overlay) Learned() map[string]Entry {
	o.mu.RLock()
	defer o.mu.RUnlock()
	out := make(map[string]Entry, len(o.learned))
	for k, v := range o.learned {
		out[k] = v
	}
	return out
}

// Entries returns a snapshot of the overlay (CIDRs in registration order,
// then exact IPs).
func (o *Overlay) Entries() []Entry {
	o.mu.RLock()
	defer o.mu.RUnlock()
	out := make([]Entry, 0, len(o.exact)+len(o.cidrs))
	for _, c := range o.cidrs {
		out = append(out, Entry{CIDR: c.text, Profile: c.entry.Profile, Hostname: c.entry.Hostname})
	}
	for _, e := range o.exact {
		out = append(out, e)
	}
	return out
}

// Len returns the number of overlay entries.
func (o *Overlay) Len() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return len(o.exact) + len(o.cidrs)
}
