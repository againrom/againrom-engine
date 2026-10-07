package game

import (
	"image"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/ui"
)

// guardsOnScreen is the guards' frame the composed square shows: the fixture
// art paints frame n as a block whose red channel is n+1, at the guards' own
// origin.
func guardsOnScreen(pix *image.RGBA) int {
	return int(pix.RGBAAt(184, 158).R) - 1
}

// guardSounds counts the requests the recorder holds for the two halberd
// samples. The fixture bank numbers each sample by its place in
// exteriorSoundPaths, so guard1 is 6 and guard2 is 7.
func guardSounds(r *exteriorRecorder) (guard1, guard2 int) {
	for _, sample := range r.samples {
		switch sample.PCM[0] {
		case 6:
			guard1++
		case 7:
			guard2++
		}
	}
	return guard1, guard2
}

// townGuardsLeave enters one door of the square and leaves it by Escape, both
// through App input, and requires the square back.
func townGuardsLeave(t *testing.T, f *FrontEnd, a *ui.App, s *townScreen, door string, room townRoom) {
	t.Helper()
	f.Shop = NewShop(1000)
	if err := a.HeadlessActivate(door); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if s.room != room {
		t.Fatalf("%s opened room %d, want %d", door, s.room, room)
	}
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if s.room != roomSquare {
		t.Fatalf("Escape in %s left room %d, want the square", door, s.room)
	}
}

// The guards stand at the last frame of their sheet when the square is first
// shown, and a pointer that asks for more than the last frame moves nothing
// (TOWN-476, TOWN-477).
func TestTownGuardsRestAtTheSheetsLastFrameOnEntry(t *testing.T) {
	_, a, s, now, _, r := exteriorFixture(t)
	if got := s.TownSquareView().Exterior.Guard; got != 7 {
		t.Fatalf("the square before any update shows the guards at frame %d, want 7", got)
	}
	exteriorPointer(t, a, 0)
	for i := 0; i < 10; i++ {
		if got := guardsOnScreen(exteriorPaint(t, a, now, 68*time.Millisecond)); got != 7 {
			t.Fatalf("hub %d: guards at frame %d, want 7", i+1, got)
		}
	}
	if guard1, guard2 := guardSounds(r); guard1 != 0 || guard2 != 1 || s.exterior.guardStep != 0 {
		t.Fatalf("guard1 %d, guard2 %d, step %d; want one guard2 request and no motion", guard1, guard2, s.exterior.guardStep)
	}
}

// Leaving the shop or the tavern, with the gate holding a quest or not, shows
// the guards at the last frame of their sheet, and the pointer the square then
// receives moves nothing: the halberds do not swing on the way back.
func TestTownGuardsRestAtTheSheetsLastFrameAfterARoomExit(t *testing.T) {
	for _, tc := range []struct {
		name  string
		door  string
		room  townRoom
		quest bool
	}{
		{"shop without a quest", "SHOP", roomShop, false},
		{"shop with a quest", "SHOP", roomShop, true},
		{"tavern without a quest", "TAVERN", roomTavern, false},
		{"tavern with a quest", "TAVERN", roomTavern, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, a, s, now, _, r := exteriorFixture(t)
			if !tc.quest {
				clearTownTestGateLatches(f.Town)
			}
			townGuardsLeave(t, f, a, s, tc.door, tc.room)
			if s.exterior.guardStep != 0 {
				t.Fatalf("leaving %s left guard step %d", tc.door, s.exterior.guardStep)
			}
			// The square is back and no pointer has been delivered to it yet.
			if got := guardsOnScreen(exteriorPaint(t, a, now, 68*time.Millisecond)); got != 7 {
				t.Fatalf("after leaving %s, before any pointer: guards at frame %d, want 7", tc.door, got)
			}
			exteriorPointer(t, a, 0)
			for i := 0; i < 10; i++ {
				if got := guardsOnScreen(exteriorPaint(t, a, now, 68*time.Millisecond)); got != 7 {
					t.Fatalf("after leaving %s, hub %d: guards at frame %d, want 7", tc.door, i+1, got)
				}
			}
			if s.exterior.guardStep != 0 {
				t.Fatalf("guard step %d after the hubs, want 0", s.exterior.guardStep)
			}
			if guard1, guard2 := guardSounds(r); guard1 != 0 || guard2 != 1 {
				t.Fatalf("halberd sounds after the exit: guard1 %d, guard2 %d; want one guard2 request", guard1, guard2)
			}
		})
	}
}

// A room exit drops the guards' frame, step and latch: the sweep down that the
// gate with nothing on offer starts, and the sound latch it set, do not carry
// into the next visit. The next visit stands at frame 7 with the latch clear,
// so a first sweep down is silent and the sweep back up asks for guard2 once.
func TestTownGuardsForgetTheLastVisitsSweep(t *testing.T) {
	f, a, s, now, _, r := exteriorFixture(t)
	clearTownTestGateLatches(f.Town)
	exteriorPointer(t, a, 40)
	want := []int{6, 5, 4, 3, 2, 1, 0, 0}
	for i, frame := range want {
		if got := guardsOnScreen(exteriorPaint(t, a, now, 68*time.Millisecond)); got != frame {
			t.Fatalf("first sweep, hub %d: guards at frame %d, want %d", i+1, got, frame)
		}
	}
	if guard1, guard2 := guardSounds(r); guard1 != 0 || guard2 != 0 {
		t.Fatalf("a first sweep down asked for guard1 %d and guard2 %d, want none: the latch is clear", guard1, guard2)
	}
	townGuardsLeave(t, f, a, s, "TAVERN", roomTavern)
	if s.exterior.frame.Guard != 7 || s.exterior.guardStep != 0 || s.exterior.guardLatch {
		t.Fatalf("after the exit: frame %d, step %d, latch %v; want 7, 0, false", s.exterior.frame.Guard, s.exterior.guardStep, s.exterior.guardLatch)
	}
	exteriorPointer(t, a, 40)
	for i, frame := range want {
		if got := guardsOnScreen(exteriorPaint(t, a, now, 68*time.Millisecond)); got != frame {
			t.Fatalf("second visit, sweep down, hub %d: guards at frame %d, want %d", i+1, got, frame)
		}
	}
	if guard1, guard2 := guardSounds(r); guard1 != 0 || guard2 != 0 {
		t.Fatalf("a sweep down from a fresh visit asked for guard1 %d and guard2 %d, want none", guard1, guard2)
	}
	exteriorPointer(t, a, 0)
	for i, frame := range []int{1, 2, 3, 4, 5, 6, 7, 7} {
		if got := guardsOnScreen(exteriorPaint(t, a, now, 68*time.Millisecond)); got != frame {
			t.Fatalf("sweep back, hub %d: guards at frame %d, want %d", i+1, got, frame)
		}
	}
	if guard1, guard2 := guardSounds(r); guard1 != 0 || guard2 != 1 {
		t.Fatalf("the sweep back asked for guard1 %d and guard2 %d, want one guard2", guard1, guard2)
	}
	exteriorPointer(t, a, 40)
	exteriorPaint(t, a, now, 68*time.Millisecond)
	if guard1, guard2 := guardSounds(r); guard1 != 1 || guard2 != 1 {
		t.Fatalf("the next sweep down asked for guard1 %d and guard2 %d, want one guard1 now that the latch is set", guard1, guard2)
	}
	if r.ones != 0 {
		t.Fatal("the guards used an unrestricted one-shot")
	}
	for _, p := range r.places {
		if p != (audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}) {
			t.Fatal(p)
		}
	}
}
