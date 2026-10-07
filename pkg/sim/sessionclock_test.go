package sim

import "testing"

func TestSessionClockKeepsIndependentSavedWords(t *testing.T) {
	c := SessionClock{SubTick: 9343, FullTick: 584}
	if c.FullTick == c.SubTick>>4 {
		t.Fatal("discriminating pair was replaced by a quotient-compatible one")
	}
	c.advanceSubTick()
	if c != (SessionClock{SubTick: 9344, FullTick: 584}) {
		t.Fatalf("subtick mutation changed the fulltick: %+v", c)
	}
	c.advanceFullTick()
	if c != (SessionClock{SubTick: 9344, FullTick: 585}) {
		t.Fatalf("fulltick mutation changed the subtick: %+v", c)
	}
}

func TestSessionClockReadsEntryAndPostBodyPhasesSeparately(t *testing.T) {
	for _, tc := range []struct {
		entry         uint32
		script, pools bool
		report        bool
	}{
		{5, false, false, false},
		{6, true, false, false},
		{11, false, false, false},
		{12, false, true, false},
		{14, false, false, true},
		{15, false, false, false},
		{22, true, false, false},
		{28, false, true, false},
		{30, false, false, true},
		{31, false, false, false},
	} {
		c := SessionClock{SubTick: tc.entry, FullTick: 99}
		if c.scriptDue() != tc.script || c.poolsDue() != tc.pools {
			t.Fatalf("entry%d script=%t pools=%t", tc.entry, c.scriptDue(), c.poolsDue())
		}
		c.advanceSubTick()
		if c.reportDue() != tc.report {
			t.Fatalf("entry%d post-body%d report=%t", tc.entry, c.SubTick, c.reportDue())
		}
		if c.reportDue() {
			c.advanceFullTick()
		}
		wantFull := uint32(99)
		if tc.report {
			wantFull = 100
		}
		if c.FullTick != wantFull {
			t.Fatalf("entry%d full=%d want%d", tc.entry, c.FullTick, wantFull)
		}
	}
}

func TestSessionClockSignedHalfAndIndependentWrapping(t *testing.T) {
	for _, sub := range []uint32{0x80000000, 0x80000006, 0x8000000c, 0x8000000f, 0xfffffff6, 0xfffffffc, 0xffffffff} {
		c := SessionClock{SubTick: sub}
		if c.scriptDue() || c.poolsDue() || c.reportDue() {
			t.Fatalf("negative signed subtick%08x admitted a positive phase", sub)
		}
	}
	c := SessionClock{SubTick: 0xffffffff, FullTick: 17}
	c.advanceSubTick()
	if c != (SessionClock{SubTick: 0, FullTick: 17}) {
		t.Fatalf("subtick wrap %+v", c)
	}
	c = SessionClock{SubTick: 14, FullTick: 0xffffffff}
	c.advanceSubTick()
	if !c.reportDue() || c.FullTick != 0xffffffff {
		t.Fatal("report must observe the not-yet-incremented fulltick")
	}
	c.advanceFullTick()
	if c != (SessionClock{SubTick: 15, FullTick: 0}) {
		t.Fatalf("fulltick wrap %+v", c)
	}
}

func TestSessionClockFullTickFiltersDoNotReadSubTick(t *testing.T) {
	for _, tc := range []struct {
		full         uint32
		health, dead bool
	}{
		{0, true, true}, {1, false, false}, {2, false, true}, {3, false, false},
		{4, true, true}, {0x80000000, true, true}, {0x80000002, false, true},
		{0xfffffffc, true, true}, {0xffffffff, false, false},
	} {
		c := SessionClock{SubTick: 12, FullTick: tc.full}
		if c.healthDue() != tc.health || c.liveDecayDue() != tc.health || c.deadDecayDue() != tc.dead {
			t.Fatalf("full%08x health=%t liveDecay=%t deadDecay=%t", tc.full, c.healthDue(), c.liveDecayDue(), c.deadDecayDue())
		}
	}
}
