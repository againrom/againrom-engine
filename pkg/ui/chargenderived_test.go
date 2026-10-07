package ui

import (
	"strconv"
	"testing"
)

// derivedSetup is chargenLegalSetup with a Derive that reports one line
// per statistic, carrying that statistic's own current value. It is
// deliberately a FUNCTION OF THE RESULT and of nothing else, so a line that
// fails to move when a statistic moves is a defect in the model's own
// plumbing rather than in the fixture.
func derivedSetup(calls *int) ChargenSetup {
	s := chargenLegalSetup()
	s.Derive = func(res ChargenResult) []ChargenDerived {
		if calls != nil {
			*calls++
		}
		out := make([]ChargenDerived, 0, len(res.Stats))
		for i, v := range res.Stats {
			out = append(out, ChargenDerived{
				Name:  "consequence of " + s.Stats[i].Name,
				Value: strconv.Itoa(v * 10),
			})
		}
		return out
	}
	return s
}

// TestDerivedTextIsSilentWithoutACallback is the backwards-compatibility
// half: every setup built before Derive existed leaves it nil, and such a
// screen paints exactly the frame it painted before.
func TestDerivedTextIsSilentWithoutACallback(t *testing.T) {
	c := NewChargen(chargenLegalSetup())
	if got := c.DerivedText(); got != nil {
		t.Errorf("DerivedText with no callback = %q, want nothing at all", got)
	}
}

// TestDerivedTextShowsWhatTheSpreadBuysAndMovesWithIt is the owner's own
// request: while generating, the screen says what the current spread buys,
// and moving a statistic moves it. The title leads, each consequence is
// indented into RowText's own marker column, and the value is the wiring
// tier's own string.
func TestDerivedTextShowsWhatTheSpreadBuysAndMovesWithIt(t *testing.T) {
	c := NewChargen(derivedSetup(nil))

	// Start is 3 on both statistic rows, so the fixture reports 30 and 30.
	want := []string{chargenDerivedTitle, "  consequence of A: 30", "  consequence of B: 30"}
	got := c.DerivedText()
	if len(got) != len(want) {
		t.Fatalf("DerivedText = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("DerivedText = %q, want %q", got, want)
		}
	}

	// The focus opens on the CHOICE row, so one Move reaches statistic A.
	c.Move(1)
	c.Adjust(1)
	if got := c.DerivedText(); got[1] != "  consequence of A: 40" {
		t.Errorf("after raising A by one the block reads %q, want the line to have moved to 40 — "+
			"a block that does not move is the defect this exists to fix", got)
	}
	if got := c.DerivedText(); got[2] != "  consequence of B: 30" {
		t.Errorf("raising A moved B's line to %q; only the statistic that moved may move", got[2])
	}
}

// TestDerivedTextRefusesAnIllegalSpreadAndNeverAsks is the trap this block
// had to avoid: while the spread is one generation could not have produced,
// there is nothing to derive, and the last legal numbers would be WORSE than
// none — a player would read a stale health as the one he is about to get.
//
// It also asserts the callback is not called at all, which is the stronger
// claim: the refusal is Result's, inherited, not a second opinion formed
// after asking.
func TestDerivedTextRefusesAnIllegalSpreadAndNeverAsks(t *testing.T) {
	calls := 0
	s := chargenIllegalSetup()
	s.Derive = derivedSetup(&calls).Derive
	c := NewChargen(s)

	if c.Legal() {
		t.Fatal("the illegal fixture starts legal; the rest of this test proves nothing")
	}
	got := c.DerivedText()
	want := []string{chargenDerivedTitle, chargenDerivedNone}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("DerivedText on an illegal spread = %q, want %q", got, want)
	}
	if calls != 0 {
		t.Errorf("Derive called %d times on an illegal spread, want 0 — "+
			"there is no result to hand it and Result already refused to make one", calls)
	}
}

// TestDerivedTextIsSilentWhenNothingIsReported is the lone-heading case: a
// callback answering no consequences leaves no title standing over nothing,
// which would be a screen element saying only that something is missing.
func TestDerivedTextIsSilentWhenNothingIsReported(t *testing.T) {
	s := chargenLegalSetup()
	s.Derive = func(ChargenResult) []ChargenDerived { return nil }
	if got := NewChargen(s).DerivedText(); got != nil {
		t.Errorf("DerivedText over an empty report = %q, want nothing at all", got)
	}
}
