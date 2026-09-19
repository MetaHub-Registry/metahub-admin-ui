package inventory

import (
	"context"
	"testing"
)

// learnFixture: static exact + static CIDR + hostname rule — the tiers a
// learned entry must slot between.
func learnedEngine(t *testing.T, seed func(e *Engine)) *Engine {
	t.Helper()
	e := testEngine(t, Config{
		Entries: []Entry{
			{IP: "10.10.1.42", Profile: "static-exact"},
			{CIDR: "10.0.0.0/8", Profile: "subnet-default"},
		},
		Hostname: HostnameConfig{Rules: []Rule{
			{Pattern: `\.learned\.host\.example$`, Profile: "hostname-tier"},
		}},
	}, newPTR().lookup)
	if seed != nil {
		seed(e)
	}
	return e
}

func TestLearnedFiresForUnseededIP(t *testing.T) {
	e := learnedEngine(t, func(e *Engine) {
		if err := e.LearnEntry(Entry{IP: "10.9.9.9", Profile: "node/aws"}); err != nil {
			t.Fatalf("LearnEntry: %v", err)
		}
	})
	res, ok := e.Resolve(context.Background(), ip(t, "10.9.9.9"))
	if !ok || res.Profile != "node/aws" {
		t.Fatalf("got ok=%v %+v", ok, res)
	}
	if res.Source != "learned:10.9.9.9" {
		t.Errorf("source = %q, want learned:10.9.9.9", res.Source)
	}
}

func TestLearnedBeatsCIDRDefault(t *testing.T) {
	// 10.9.9.9 is inside 10.0.0.0/8 → subnet-default; the learned entry
	// (per-node evidence) must win.
	e := learnedEngine(t, func(e *Engine) {
		_ = e.LearnEntry(Entry{IP: "10.9.9.9", Profile: "node/aws"})
	})
	res, _ := e.Resolve(context.Background(), ip(t, "10.9.9.9"))
	if res.Profile != "node/aws" || res.Source != "learned:10.9.9.9" {
		t.Fatalf("got %+v — learned must beat the CIDR default", res)
	}
}

func TestStaticExactBeatsLearned(t *testing.T) {
	e := learnedEngine(t, func(e *Engine) {
		_ = e.LearnEntry(Entry{IP: "10.10.1.42", Profile: "learned-shadow"})
	})
	res, _ := e.Resolve(context.Background(), ip(t, "10.10.1.42"))
	if res.Profile != "static-exact" || res.Source != "ip:0" {
		t.Fatalf("got %+v — static exact must outrank learned", res)
	}
}

func TestHostnameBeatsLearned(t *testing.T) {
	ptr := newPTR()
	ptr.answers["10.9.9.9"] = []string{"node.learned.example.com."}
	e := testEngine(t, Config{
		Hostname: HostnameConfig{Rules: []Rule{
			{Pattern: `\.learned\.example\.com$`, Profile: "hostname-tier"},
		}},
	}, ptr.lookup)
	_ = e.LearnEntry(Entry{IP: "10.9.9.9", Profile: "learned"})

	res, _ := e.Resolve(context.Background(), ip(t, "10.9.9.9"))
	if res.Profile != "hostname-tier" {
		t.Fatalf("got %+v — hostname tier must outrank learned", res)
	}
}

func TestLearnedLastWriteWins(t *testing.T) {
	e := learnedEngine(t, nil)
	if err := e.LearnEntry(Entry{IP: "10.9.9.9", Profile: "first"}); err != nil {
		t.Fatalf("learn 1: %v", err)
	}
	if err := e.LearnEntry(Entry{IP: "10.9.9.9", Profile: "second"}); err != nil {
		t.Fatalf("learn 2 (update): %v", err)
	}
	res, _ := e.Resolve(context.Background(), ip(t, "10.9.9.9"))
	if res.Profile != "second" {
		t.Fatalf("got %q, want second (last-write-wins)", res.Profile)
	}
}

func TestLearnedRejectsInvalid(t *testing.T) {
	e := learnedEngine(t, nil)
	if err := e.LearnEntry(Entry{CIDR: "10.0.0.0/8", Profile: "x"}); err == nil {
		t.Fatal("cidr+learned must be rejected")
	}
	if err := e.LearnEntry(Entry{IP: "10.9.9.9"}); err == nil {
		t.Fatal("empty profile must be rejected")
	}
	if err := e.LearnEntry(Entry{Profile: "x"}); err == nil {
		t.Fatal("missing ip must be rejected")
	}
}

func TestLearnedRemoveRestoresCIDR(t *testing.T) {
	e := learnedEngine(t, func(e *Engine) {
		_ = e.LearnEntry(Entry{IP: "10.9.9.9", Profile: "learned"})
	})
	found, staticHit := e.RemoveEntry("10.9.9.9")
	if !found || staticHit {
		t.Fatalf("RemoveEntry = %v/%v, want true/false", found, staticHit)
	}
	res, ok := e.Resolve(context.Background(), ip(t, "10.9.9.9"))
	if !ok || res.Profile != "subnet-default" || res.Source != "cidr:1" {
		t.Fatalf("after removal got %+v — CIDR tier must be visible again", res)
	}
}

func TestLearnedSnapshotFlag(t *testing.T) {
	e := learnedEngine(t, func(e *Engine) {
		_ = e.LearnEntry(Entry{IP: "10.9.9.9", Profile: "node/aws"})
	})
	snap := e.Snapshot()
	var seen int
	for _, v := range snap.Entries {
		if v.Learned {
			seen++
			if v.IP != "10.9.9.9" || v.Profile != "node/aws" || !v.Runtime {
				t.Errorf("learned snapshot entry = %+v", v)
			}
		}
	}
	if seen != 1 {
		t.Errorf("learned entries in snapshot = %d, want 1", seen)
	}
}

func TestLearnedWipedOnRebuild(t *testing.T) {
	e := learnedEngine(t, func(e *Engine) {
		_ = e.LearnEntry(Entry{IP: "10.9.9.9", Profile: "node/aws"})
	})
	e2 := learnedEngine(t, nil)
	// The learned state is gone — the IP falls back to the CIDR tier.
	res, ok := e2.Resolve(context.Background(), ip(t, "10.9.9.9"))
	if !ok || res.Profile != "subnet-default" {
		t.Fatalf("fresh engine got ok=%v %+v — learned state must be wiped", ok, res)
	}
	_ = e
}
