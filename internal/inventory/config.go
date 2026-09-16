// Package inventory implements the profile-resolution chain and the
// inventory config model behind GET /api/v1/resolve.
//
// The ENTIRE resolution chain lives here (metahub-api is a thin consumer):
// ordered, first match wins — exact-IP entries, then hostname patterns
// (DNS PTR, TTL-cached incl. negative), then CIDR longest-prefix.
package inventory

import (
	"fmt"
	"net"
	"regexp"
	"time"

	"github.com/spf13/viper"
)

// DefaultLookupTTL is used when hostname.lookup_ttl is unset.
const DefaultLookupTTL = 60 * time.Second

// Config is the `inventory` config block.
type Config struct {
	Entries  []Entry        `mapstructure:"entries"`
	Hostname HostnameConfig `mapstructure:"hostname"`
}

// Entry is an exact-IP or CIDR entry — exactly one of IP/CIDR must be set.
type Entry struct {
	IP       string `mapstructure:"ip"`
	CIDR     string `mapstructure:"cidr"`
	Profile  string `mapstructure:"profile"`
	Hostname string `mapstructure:"hostname"` // optional display data
}

// HostnameConfig groups the PTR-backed hostname-pattern tier.
type HostnameConfig struct {
	LookupTTL time.Duration `mapstructure:"lookup_ttl"`
	Rules     []Rule        `mapstructure:"rules"`
}

// Rule is an ordered hostname-pattern → profile mapping.
type Rule struct {
	Pattern string `mapstructure:"pattern"`
	Profile string `mapstructure:"profile"`
}

// LoadConfig reads the config file and decodes the `inventory` block.
// A missing or unreadable FILE is the caller's decision (main runs with an
// empty inventory + warning); a present-but-invalid file errors from
// Validate() instead.
func LoadConfig(path string) (Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := v.UnmarshalKey("inventory", &cfg); err != nil {
		return Config{}, fmt.Errorf("decode inventory block from %s: %w", path, err)
	}
	return cfg, nil
}

// Validate checks every source and fails fast naming the offending entry —
// the spec contract (specs/inventory-lookup: "Config-driven sources,
// fail-fast validation"). Invalid regex/CIDR/IP or empty profile ⇒ error.
func (c Config) Validate() error {
	for i, e := range c.Entries {
		name := fmt.Sprintf("entries[%d]", i)
		hasIP, hasCIDR := e.IP != "", e.CIDR != ""
		if !hasIP && !hasCIDR {
			return fmt.Errorf("%s: neither ip nor cidr set", name)
		}
		if hasIP && hasCIDR {
			return fmt.Errorf("%s: ip and cidr are mutually exclusive", name)
		}
		if e.Profile == "" {
			return fmt.Errorf("%s: empty profile", name)
		}
		if hasIP {
			if net.ParseIP(e.IP) == nil {
				return fmt.Errorf("%s: invalid ip %q", name, e.IP)
			}
		} else if _, _, err := net.ParseCIDR(e.CIDR); err != nil {
			return fmt.Errorf("%s: invalid cidr %q: %v", name, e.CIDR, err)
		}
	}
	for i, r := range c.Hostname.Rules {
		name := fmt.Sprintf("hostname.rules[%d]", i)
		if r.Pattern == "" {
			return fmt.Errorf("%s: empty pattern", name)
		}
		if r.Profile == "" {
			return fmt.Errorf("%s: empty profile", name)
		}
		// Rules are matched case-insensitively (spec: Hostname-pattern
		// resolution) — compile exactly as the engine will.
		if _, err := regexp.Compile(ruleRegex(r.Pattern)); err != nil {
			return fmt.Errorf("%s: invalid pattern '%s': %v", name, r.Pattern, err)
		}
	}
	return nil
}

// ruleRegex wraps an operator pattern for case-insensitive RE2 matching.
func ruleRegex(pattern string) string {
	return "(?i)" + pattern
}
