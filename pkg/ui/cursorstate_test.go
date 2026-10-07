package ui

import (
	"image"
	"testing"
)

func testRegistry() *CursorRegistry {
	return NewCursorRegistry([]CursorSlot{
		{Name: "default", Frames: []*image.RGBA{{}}, Hotspot: image.Pt(5, 5), FrameCount: 1, PeriodMillis: 2000000000},
		{Name: "wait", Frames: []*image.RGBA{{}, {}, {}}, Hotspot: image.Pt(16, 16), FrameCount: 3, PeriodMillis: 100},
		// "select" carries an animating period over a single frame, on
		// SPR16A-CURSOR-046's own reading of the shipped table — the trap
		// B3's contract names: a small period alone does not mean the slot
		// has more than one frame to advance through.
		{Name: "select", Frames: []*image.RGBA{{}}, Hotspot: image.Pt(3, 4), FrameCount: 1, PeriodMillis: 100},
	})
}

func TestCursorRegistrySlot(t *testing.T) {
	r := testRegistry()
	if _, ok := r.Slot("nonexistent"); ok {
		t.Error("Slot resolved a name the registry does not carry")
	}
	s, ok := r.Slot("wait")
	if !ok {
		t.Fatal("Slot did not resolve \"wait\"")
	}
	if s.FrameCount != 3 || s.PeriodMillis != 100 {
		t.Errorf("wait slot = %+v, want FrameCount 3, PeriodMillis 100", *s)
	}
	var nilRegistry *CursorRegistry
	if _, ok := nilRegistry.Slot("wait"); ok {
		t.Error("a nil registry resolved a slot")
	}
}

// B2: setting an unregistered name, or setting one with no registry
// installed at all, is a no-op that leaves the current cursor unchanged.
func TestCursorManagerSetCursorUnknownNameIsNoOp(t *testing.T) {
	m := NewCursorManager()
	m.SetCursor("wait") // no registry yet
	if m.CurrentName() != "" {
		t.Fatalf("CurrentName = %q before any registry was installed", m.CurrentName())
	}
	m.SetRegistry(testRegistry())
	m.SetCursor("wait")
	if m.CurrentName() != "wait" {
		t.Fatalf("CurrentName = %q, want wait", m.CurrentName())
	}
	m.SetCursor("nope")
	if m.CurrentName() != "wait" {
		t.Fatalf("an unknown name changed the current cursor: CurrentName = %q, want wait unchanged", m.CurrentName())
	}
}

// B2 (AI-CURSOR-193): the idempotence guard. Setting the cursor already
// displayed does nothing — this test proves it by observing that a set to
// the SAME name after Advance has moved the frame index does not reset it,
// which a real (non-idempotent) set would.
func TestCursorManagerSetCursorSameNameIsIdempotent(t *testing.T) {
	m := NewCursorManager()
	m.SetRegistry(testRegistry())
	m.SetCursor("wait")
	m.Advance(0)   // baseline tick
	m.Advance(150) // > 100ms period: advances to frame 1
	pic, _, ok := m.Current()
	if !ok {
		t.Fatal("Current() = not ok after an advance")
	}
	before := pic

	m.SetCursor("wait") // already current: must be a no-op
	pic, _, ok = m.Current()
	if !ok || pic != before {
		t.Fatalf("setting the already-current cursor changed the frame: got %p, want %p unchanged", pic, before)
	}
}

// B2/B3: setting a cursor copies the registration's own frame count and
// period, and zeroes the frame index and the last-tick field.
func TestCursorManagerSetCursorResetsFrameAndClock(t *testing.T) {
	m := NewCursorManager()
	m.SetRegistry(testRegistry())
	m.SetCursor("wait")
	m.Advance(0)
	m.Advance(150) // now on frame 1
	m.Advance(300) // now on frame 2
	if _, _, ok := m.Current(); !ok {
		t.Fatal("Current() = not ok after two advances")
	}
	pic2, _, _ := m.Current()

	m.SetCursor("default") // a different slot: the set fires for real
	m.SetCursor("wait")    // back to wait: frame index and clock reset per B2
	pic0, _, _ := m.Current()
	if pic0 == pic2 {
		t.Fatal("the set did not reset the frame index: Current() is still the frame the counter had reached")
	}

	// THE FIRST ADVANCE AFTER A SET ADVANCES (D-3, AI-CURSOR-172). The decoded
	// routine zeroes the last-tick field on the set and then compares
	// now - 0 against the period, so the compare passes on the first
	// evaluation: "a single 'set cursor' call can only be observed to leave
	// the frame index at 0 or 1". Recording a baseline instead held the
	// frame at 0 for one extra period.
	m.Advance(150)
	pic1, _, _ := m.Current()
	if pic1 == pic0 {
		t.Fatal("the frame did not advance on the first Advance call after a set; AI-CURSOR-172 puts the index at 1 by then")
	}

	// And it stops there until another period elapses.
	m.Advance(200)
	if pic, _, _ := m.Current(); pic != pic1 {
		t.Fatal("the frame advanced again 50ms into a 100ms period")
	}
}

// B3 (AI-CURSOR-172, SPR16A-CURSOR-061): the frame index advances once
// elapsed time exceeds the registered period, and wraps to 0 before it can
// be read at or above the registered frame count.
func TestCursorManagerAdvanceWraps(t *testing.T) {
	m := NewCursorManager()
	m.SetRegistry(testRegistry())
	m.SetCursor("wait") // 3 frames, 100ms period
	m.Advance(0)        // baseline

	frames := map[*image.RGBA]bool{}
	t0 := int64(0)
	for i := 0; i < 3; i++ {
		t0 += 150
		m.Advance(t0)
		pic, _, ok := m.Current()
		if !ok {
			t.Fatalf("Current() = not ok at step %d", i)
		}
		frames[pic] = true
	}
	if len(frames) != 3 {
		t.Fatalf("observed %d distinct frames over 3 advances past the period, want 3 (one per frame, no repeat before the wrap)", len(frames))
	}
	// A fourth advance must wrap back to frame 0, whose picture was already
	// seen above, rather than reading past the registered count.
	t0 += 150
	m.Advance(t0)
	pic, hotspot, ok := m.Current()
	if !ok || !frames[pic] {
		t.Fatalf("the fourth advance produced a frame outside the registered set: %p", pic)
	}
	if hotspot != (image.Point{X: 16, Y: 16}) {
		t.Errorf("hotspot = %v, want the wait slot's own (16,16)", hotspot)
	}
}

// B3's own trap: a small period on a ONE-frame slot never advances, because
// there is no second frame for the wrap to reach.
func TestCursorManagerAdvanceNeverMovesASingleFrameSlot(t *testing.T) {
	m := NewCursorManager()
	m.SetRegistry(testRegistry())
	m.SetCursor("select") // period 100, 1 frame
	m.Advance(0)
	pic0, _, _ := m.Current()
	for _, now := range []int64{200, 500, 100000} {
		m.Advance(now)
		pic, _, ok := m.Current()
		if !ok || pic != pic0 {
			t.Fatalf("a single-frame slot's picture changed after Advance(%d)", now)
		}
	}
}

// PointerHidden/SetPointerHidden: the change-cache App.drawCursor shares
// with pointerModeChange's own shape (cursor.go).
func TestCursorManagerSetPointerHidden(t *testing.T) {
	m := NewCursorManager()
	if m.PointerHidden() {
		t.Fatal("PointerHidden's zero value is true, want false")
	}
	if !m.SetPointerHidden(true) {
		t.Fatal("SetPointerHidden(true) from false reported no change")
	}
	if m.SetPointerHidden(true) {
		t.Fatal("SetPointerHidden(true) reported a change on a repeat")
	}
	if !m.SetPointerHidden(false) {
		t.Fatal("SetPointerHidden(false) from true reported no change")
	}
}
