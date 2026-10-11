package sim

import "testing"

// A first-game World holds the session clock an original-save import gives it
// from construction (0/0, SESS-085); a second-game World holds none.
func TestAFirstGameWorldIsConstructedWithTheSessionClock(t *testing.T) {
	b, ents := sample()
	w := mustWorld(t, 1, b, ents)
	if c, ok := w.SessionClock(); !ok || c != (SessionClock{}) || w.SecondGame() {
		t.Fatalf("constructed clock %+v present %v", c, ok)
	}
	asSecondGame(t, w)
	if _, ok := w.SessionClock(); ok || !w.SecondGame() {
		t.Fatal("a second-game World holds a session clock")
	}
}

// A byte form written before the clock was constructed decodes with the clock
// DerivedFullTick derives, and an older engine SAV that marked the clock
// unknown keeps the one its import gave.
func TestAWorldWithoutTheClockTakesTheDerivedOne(t *testing.T) {
	var w World
	if err := w.UnmarshalBinary(rtfBytesHistorical); err != nil {
		t.Fatal(err)
	}
	if c, ok := w.SessionClock(); !ok || c != (SessionClock{SubTick: 7, FullTick: 0}) {
		t.Fatalf("historical form decoded with clock %+v present %v", c, ok)
	}
	w.tick, w.fullTick = 47, 2
	if err := w.RestoreCurrentClock(0, false); err != nil {
		t.Fatal(err)
	}
	if c, ok := w.SessionClock(); !ok || c != (SessionClock{SubTick: 47, FullTick: 2}) {
		t.Fatalf("unknown-clock policy left %+v present %v", c, ok)
	}
	for _, tc := range []struct{ sub, full uint32 }{{0, 0}, {14, 0}, {15, 1}, {16, 1}, {31, 2}, {0xffffffff, 0x10000000}} {
		if got := DerivedFullTick(tc.sub); got != tc.full {
			t.Errorf("DerivedFullTick(%d) = %d, want %d", tc.sub, got, tc.full)
		}
	}
}
