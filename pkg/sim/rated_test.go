package sim

// The three fields the rate arrives as: what makes a mover rated, what a
// crossing may look like, and that each of the three reaches the digest.
//
// The byte form's own offsets, its pin and its refusals are binary_test.go's;
// what is here is the pair of rules the CONSTRUCTOR and the decoder share, and
// the widening rule that decides whether a mover has a rate at all.

import (
	"strings"
	"testing"
)

// TestOnlyAPositiveSpeedIsARate pins the widening rule in the one place it is
// decided, so a change to it is a failure here rather than a surprise in a tick.
func TestOnlyAPositiveSpeedIsARate(t *testing.T) {
	for _, c := range []struct {
		speed int32
		want  bool
	}{{-2147483648, false}, {-1, false}, {0, false}, {1, true}, {35, true}, {2147483647, true}} {
		if got := rated(Entity{Speed: c.speed}); got != c.want {
			t.Errorf("rated at speed %d = %v, want %v", c.speed, got, c.want)
		}
	}
}

// TestTheConstructorRefusesACrossingNoTickCanLeave is the constructor's half of
// AC-6, and it is the same rule the decoder applies — one function, two callers,
// so the two cannot come to disagree about which pairs exist.
//
// Every refused case is named by its own error, so a caller is told which of the
// three it asked for rather than that something was wrong.
func TestTheConstructorRefusesACrossingNoTickCanLeave(t *testing.T) {
	b := Bounds{Width: 4, Height: 4}
	cases := []struct {
		name    string
		e       Entity
		wantSub string
	}{
		{"a transit one longer than the law can make",
			Entity{ID: 1, Transit: 0, TransitTotal: maxTransit + 1}, "the longest the law can produce"},
		{"a transit at the top of its range",
			Entity{ID: 1, Transit: 0, TransitTotal: 0xffff}, "the longest the law can produce"},
		{"an owed count at its own transit's length",
			Entity{ID: 1, Transit: 9, TransitTotal: 9}, "which no tick can leave"},
		{"an owed count past its own transit's length",
			Entity{ID: 1, Transit: 10, TransitTotal: 9}, "which no tick can leave"},
		{"an owed count with no transit to owe it to",
			Entity{ID: 1, Transit: 1, TransitTotal: 0}, "no transit to owe them to"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, err := NewWorld(1, b, ModeCanonical, nil, []Entity{tc.e})
			if err == nil {
				t.Fatalf("built a world holding %+v", tc.e)
			}
			if w != nil {
				t.Errorf("a refused construction returned a world")
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Errorf("the error is %q, which does not name %q", err, tc.wantSub)
			}
		})
	}

	// The legal boundary, so the cases above are refusing the pair and not the
	// fields: the longest crossing the law can make, with every owed count in it
	// legal at its own end.
	for _, e := range []Entity{
		{ID: 1, Transit: 0, TransitTotal: 0},
		{ID: 1, Transit: 0, TransitTotal: maxTransit},
		{ID: 1, Transit: maxTransit - 1, TransitTotal: maxTransit},
	} {
		if _, err := NewWorld(1, b, ModeCanonical, nil, []Entity{e}); err != nil {
			t.Errorf("refused the legal pair %d/%d: %v", e.Transit, e.TransitTotal, err)
		}
	}
}

// TestACorpseCarriesNoCrossing is the constructor's NORMALISATION, beside the
// refusals above: a unit that is not alive is crossing nothing, so its pair is
// residue and is dropped exactly as its order is. Its SPEED is not — that is a
// property of the class it was, carried whole like its health.
func TestACorpseCarriesNoCrossing(t *testing.T) {
	w := mustWorld(t, 1, Bounds{Width: 4, Height: 4}, []Entity{
		{ID: 1, X: 1, Y: 1, HP: -1, MaxHP: 10, Speed: 19, Transit: 4, TransitTotal: 9},
		{ID: 2, X: 2, Y: 2, HP: 0, MaxHP: 10, Speed: 19, Transit: 4, TransitTotal: 9},
		{ID: 3, X: 3, Y: 3, HP: 10, MaxHP: 10, Speed: 19, Transit: 4, TransitTotal: 9},
	})
	got := w.Entities()
	for _, e := range got[:2] {
		if e.Transit != 0 || e.TransitTotal != 0 {
			t.Errorf("entity %d is at %d/%d and carries a transit of %d/%d",
				e.ID, e.HP, e.MaxHP, e.Transit, e.TransitTotal)
		}
		if e.Speed != 19 {
			t.Errorf("entity %d lost its speed: %d", e.ID, e.Speed)
		}
	}
	if e := got[2]; e.Transit != 4 || e.TransitTotal != 9 {
		t.Errorf("the living unit's transit is %d/%d, want 4/9", e.Transit, e.TransitTotal)
	}
}

func TestEachNewFieldReachesTheDigest(t *testing.T) {
	b := Bounds{Width: 4, Height: 4}
	base := Entity{ID: 1, X: 1, Y: 1, Speed: 19, Transit: 4, TransitTotal: 9}
	ref := mustWorld(t, 1, b, []Entity{base}).Hash()

	moved := base
	moved.Speed = 20
	if got := mustWorld(t, 1, b, []Entity{moved}).Hash(); got == ref {
		t.Errorf("a changed speed left the digest at %#016x", got)
	}

	moved = base
	moved.Transit = 5
	if got := mustWorld(t, 1, b, []Entity{moved}).Hash(); got == ref {
		t.Errorf("a changed owed count left the digest at %#016x", got)
	}

	moved = base
	moved.TransitTotal = 10
	if got := mustWorld(t, 1, b, []Entity{moved}).Hash(); got == ref {
		t.Errorf("a changed transit length left the digest at %#016x", got)
	}
}
