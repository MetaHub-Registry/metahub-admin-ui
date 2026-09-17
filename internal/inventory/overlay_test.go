package inventory

import (
	"context"
	"testing"
)

// engineWithOverlay builds an engine with static sources to test overlay
// precedence against (mirrors the dev seed).
func engineWithOverlay(t *testing.T, overlayCfg func(e *Engine)) *Engine {
	t.Helper()
	e := testEngine(t, Config{
		Entries: []Entry{
			{IP: "10.10.1.42", Profile: "gpu/a100"},
			{CIDR: "10.10.1.0/24", Profile: "zen4"},
			{CIDR: "10.0.0.0/8", Profile: "generic"},
		},
	}, newPTR().lookup)
	if overlayCfg != nil {
		overlayCfg(e)
	}
	return e
}

func TestOverlayAddAndResolve(t *testing.T) {
	e := engineWithOverlay(t, nil)
	if err := e.AddEntry(Entry{IP: "192.168.5.115", Profile: "e2e/42", Hostname: "frame.hvk"}); err != nil {
		t.Fatalf("AddEntry: %v", err)
	}

	res, ok := e.Resolve(context.Background(), ip(t, "192.168.5.115"))
	if !ok || res.Profile != "e2e/42" {
		t.Fatalf("got ok=%v %+v", ok, res)
	}
	if res.Source != "runtime-ip:192.168.5.115" {
		t.Errorf("source = %q, want runtime-ip:192.168.5.115", res.Source)
	}
	if res.Hostname != "frame.hvk" {
		t.Errorf("hostname = %q", res.Hostname)
	}
}

func TestOverlayBeatsStaticExact(t *testing.T) {
	e := engineWithOverlay(t, func(e *Engine) {
		if err := e.AddEntry(Entry{IP: "10.10.1.42", Profile: "e2e/override"}); err != nil {
			t.Fatalf("AddEntry: %v", err)
		}
	})
	res, ok := e.Resolve(context.Background(), ip(t, "10.10.1.42"))
	if !ok || res.Profile != "e2e/override" || res.Source != "runtime-ip:10.10.1.42" {
		t.Fatalf("got ok=%v %+v — runtime must win over static exact", ok, res)
	}
}

func TestOverlayCIDRTieBreakAndPrefix(t *testing.T) {
	e := engineWithOverlay(t, func(e *Engine) {
		// Same prefix as the static /24 → runtime wins the tie.
		if err := e.AddEntry(Entry{CIDR: "10.10.1.0/24", Profile: "e2e/subnet"}); err != nil {
			t.Fatalf("AddEntry: %v", err)
		}
	})
	res, ok := e.Resolve(context.Background(), ip(t, "10.10.1.99"))
	if !ok || res.Profile != "e2e/subnet" {
		t.Fatalf("got ok=%v %+v — runtime must win the prefix tie", ok, res)
	}
	if res.Source != "runtime-cidr:10.10.1.0/24" {
		t.Errorf("source = %q", res.Source)
	}

	// Static exact entry still beats the overlay CIDR (exact tier first).
	if err := e.AddEntry(Entry{IP: "10.10.1.42", Profile: "e2e/exact"}); err != nil {
		t.Fatalf("AddEntry: %v", err)
	}
	res, _ = e.Resolve(context.Background(), ip(t, "10.10.1.42"))
	if res.Source != "runtime-ip:10.10.1.42" {
		t.Errorf("source = %q — exact tier must outrank any CIDR", res.Source)
	}

	// A longer static prefix outranks a shorter overlay prefix.
	e2 := engineWithOverlay(t, func(e *Engine) {
		if err := e.AddEntry(Entry{CIDR: "10.0.0.0/8", Profile: "e2e/wide"}); err != nil {
			t.Fatalf("AddEntry: %v", err)
		}
	})
	res, _ = e2.Resolve(context.Background(), ip(t, "10.10.1.99"))
	if res.Profile != "zen4" || res.Source != "cidr:1" {
		t.Fatalf("got %+v — static /24 must outrank overlay /8", res)
	}
}

func TestOverlayRemoveRestoresStatic(t *testing.T) {
	e := engineWithOverlay(t, func(e *Engine) {
		if err := e.AddEntry(Entry{IP: "10.10.1.42", Profile: "e2e/tmp"}); err != nil {
			t.Fatalf("AddEntry: %v", err)
		}
	})
	res, _ := e.Resolve(context.Background(), ip(t, "10.10.1.42"))
	if res.Profile != "e2e/tmp" {
		t.Fatalf("overlay entry not effective: %+v", res)
	}

	found, staticHit := e.RemoveEntry("10.10.1.42")
	if !found || staticHit {
		t.Fatalf("RemoveEntry = %v/%v, want true/false", found, staticHit)
	}
	res, _ = e.Resolve(context.Background(), ip(t, "10.10.1.42"))
	if res.Profile != "gpu/a100" || res.Source != "ip:0" {
		t.Fatalf("after removal static must be visible again, got %+v", res)
	}
}

func TestOverlayRemoveSemantics(t *testing.T) {
	e := engineWithOverlay(t, func(e *Engine) {
		_ = e.AddEntry(Entry{CIDR: "192.168.7.0/24", Profile: "e2e/c"})
	})

	// Unknown IP that also exists as a static entry ⇒ staticHit hint.
	found, staticHit := e.RemoveEntry("10.10.1.42")
	if found || !staticHit {
		t.Fatalf("(found, staticHit) = %v/%v, want false/true", found, staticHit)
	}
	// Unknown IP entirely ⇒ both false.
	found, staticHit = e.RemoveEntry("172.16.1.1")
	if found || staticHit {
		t.Fatalf("(found, staticHit) = %v/%v, want false/false", found, staticHit)
	}
	// Remove the overlay CIDR by string.
	found, staticHit = e.RemoveEntry("192.168.7.0/24")
	if !found || staticHit {
		t.Fatalf("(found, staticHit) = %v/%v, want true/false", found, staticHit)
	}
	// Removing a static CIDR ⇒ staticHit.
	found, staticHit = e.RemoveEntry("10.10.1.0/24")
	if found || !staticHit {
		t.Fatalf("(found, staticHit) = %v/%v, want false/true", found, staticHit)
	}
}

func TestOverlayValidationAndDuplicates(t *testing.T) {
	e := engineWithOverlay(t, nil)

	cases := []struct {
		name    string
		entry   Entry
		wantSub string
	}{
		{"empty profile", Entry{IP: "10.9.9.9"}, "empty profile"},
		{"both ip and cidr", Entry{IP: "10.9.9.9", CIDR: "10.0.0.0/8", Profile: "x"}, "mutually exclusive"},
		{"neither", Entry{Profile: "x"}, "neither ip nor cidr"},
		{"bad ip", Entry{IP: "nope", Profile: "x"}, "invalid ip"},
		{"bad cidr", Entry{CIDR: "10.0.0.0/99", Profile: "x"}, "invalid cidr"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := e.AddEntry(tc.entry)
			if err == nil {
				t.Fatalf("expected error")
			}
			if !contains(err.Error(), tc.wantSub) {
				t.Errorf("error %q missing %q", err.Error(), tc.wantSub)
			}
		})
	}

	// Duplicate (exact and cidr) → ErrDuplicate. Note: an overlay entry may
	// shadow a static key (that is the override mechanism); duplicates are
	// only rejected within the overlay itself.
	if err := e.AddEntry(Entry{IP: "10.10.1.42", Profile: "a"}); err != nil {
		t.Fatalf("first add: %v", err)
	}
	if err := e.AddEntry(Entry{CIDR: "192.168.7.0/24", Profile: "a"}); err != nil {
		t.Fatalf("first cidr add: %v", err)
	}
	if err := e.AddEntry(Entry{IP: "10.10.1.42", Profile: "b"}); err == nil {
		t.Fatal("duplicate exact must error")
	}
	if err := e.AddEntry(Entry{CIDR: "192.168.7.0/24", Profile: "b"}); err == nil {
		t.Fatal("duplicate cidr must error")
	}
	if e.overlay.Len() != 2 {
		t.Errorf("overlay len = %d, want 2 (failed adds must not be stored)", e.overlay.Len())
	}
}

func TestOverlaySnapshotFlagsRuntime(t *testing.T) {
	e := engineWithOverlay(t, func(e *Engine) {
		_ = e.AddEntry(Entry{IP: "192.168.5.115", Profile: "e2e/42"})
	})
	snap := e.Snapshot()
	var runtimeSeen int
	for _, v := range snap.Entries {
		if v.Runtime {
			runtimeSeen++
			if v.IP != "192.168.5.115" || v.Profile != "e2e/42" {
				t.Errorf("runtime entry = %+v", v)
			}
		}
	}
	if runtimeSeen != 1 {
		t.Errorf("runtime entries in snapshot = %d, want 1", runtimeSeen)
	}
}

func TestOverlayWipedOnRebuild(t *testing.T) {
	e := engineWithOverlay(t, func(e *Engine) {
		_ = e.AddEntry(Entry{IP: "192.168.5.115", Profile: "e2e/42"})
	})
	// A fresh Build (what a restart does) starts with an empty overlay —
	// there is no persistence.
	e2 := engineWithOverlay(t, nil)
	if e2.overlay.Len() != 0 {
		t.Fatalf("fresh engine overlay len = %d, want 0", e2.overlay.Len())
	}
	if _, ok := e2.Resolve(context.Background(), ip(t, "192.168.5.115")); ok {
		t.Fatal("fresh engine must not know the previous overlay entry")
	}
	_ = e
}
