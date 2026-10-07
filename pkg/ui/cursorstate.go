package ui

import "image"

// CursorSlot is one of the engine's 28 named cursor registrations, resolved
// to pictures this package can draw (SPR16A-CURSOR-067, SPR16A-CURSOR-046,
// SPR256-CURSOR-046).
//
// ONLY THE RESOLVED SHAPE CROSSES HERE. Frames is every frame of the sheet, in
// sheet order, already the premultiplied pictures this package's upload path
// wants — no archive, no container, no decoder — on SetAttackPointer's own
// rule (cursor.go): this package gains no format knowledge and no import for
// one. The tier that owns the install (pkg/game) is what builds a
// CursorRegistry; this type only holds what that tier resolved.
type CursorSlot struct {
	// Name is the slot's name, e.g. "wait", "default", "attack" (SPR16A-CURSOR-067).
	Name string
	// Frames holds len(Frames) == FrameCount pictures in sheet order. A slot
	// with one frame (nineteen of the 28) still carries a one-element slice.
	Frames []*image.RGBA
	// Hotspot is the pixel within a frame that the cursor point names, read out
	// of the executable rather than chosen (SPR16A-CURSOR-046,
	// SPR256-CURSOR-046).
	Hotspot image.Point
	// FrameCount is the registered frame count, an executable constant
	// independent of the sheet's own frame count (G2, SPR16A-CURSOR-061). It
	// equals len(Frames) for every one of the 28 shipped registrations; a
	// caller comparing the two is comparing a resource fact to an executable
	// one that happen to agree today.
	FrameCount int
	// PeriodMillis is the registered period argument in milliseconds, the
	// second executable constant a set copies into the manager
	// (AI-CURSOR-172, SPR16A-CURSOR-061).
	PeriodMillis int64
}

// CursorRegistry is the build's 28-slot table, in slot order (SPR16A-CURSOR-067).
type CursorRegistry struct {
	// Slots is the 28 registrations in slot order (0 = default .. 27 = backpack).
	Slots  []CursorSlot
	byName map[string]*CursorSlot
}

// NewCursorRegistry builds a registry from slots, indexing it by name. A slot
// naming an already-present name replaces the earlier one; the build's own 28
// slot names are distinct so this never fires against shipped data (a test may
// still exercise it against a synthetic table).
func NewCursorRegistry(slots []CursorSlot) *CursorRegistry {
	r := &CursorRegistry{Slots: slots, byName: make(map[string]*CursorSlot, len(slots))}
	for i := range r.Slots {
		r.byName[r.Slots[i].Name] = &r.Slots[i]
	}
	return r
}

// Slot returns the named registration and whether the registry carries one.
func (r *CursorRegistry) Slot(name string) (*CursorSlot, bool) {
	if r == nil {
		return nil, false
	}
	s, ok := r.byName[name]
	return s, ok
}

// CursorManager is the build's one current-cursor state (B3, AI-CURSOR-172):
// which slot is current, its frame index, and the wall-clock instant the
// index last advanced. One instance is shared by every screen that draws a
// cursor through it — App and every Viewer it hands the same pointer to
// — mirroring the decoded engine's own single manager object.
type CursorManager struct {
	registry *CursorRegistry

	current    string
	hasCurrent bool
	frameIndex int

	lastTick      int64
	pointerHidden bool
}

// NewCursorManager returns a manager with no registry and no current cursor.
func NewCursorManager() *CursorManager { return &CursorManager{} }

// SetRegistry installs the resolved 28-slot table this manager draws from. A
// nil registry is the ordinary state before an install has resolved one; every
// method below degrades to "no cursor" rather than panicking.
func (m *CursorManager) SetRegistry(r *CursorRegistry) { m.registry = r }

func (m *CursorManager) currentSlot() *CursorSlot {
	if m == nil || m.registry == nil || !m.hasCurrent {
		return nil
	}
	s, _ := m.registry.Slot(m.current)
	return s
}

// SetCursor makes name the current cursor: it copies the registration's own
// frame count and period, and zeroes the frame index and the last-tick field
// (B2, AI-CURSOR-172's "copies the five incoming fields ... unconditionally
// zeroes").
//
// A NAME ALREADY CURRENT IS A NO-OP, the idempotence guard AI-CURSOR-193
// reads off a whole-image scan of the engine's own current-cursor global: the
// set-cursor adapter is never called for a cursor already displayed. An
// unknown name (absent from the registry, or the registry not yet installed)
// is also a no-op and leaves the current cursor unchanged — B2's "an exit
// that sets no cursor leaves the displayed cursor unchanged" applied to a name
// this build cannot resolve as well as to a caller that names none.
func (m *CursorManager) SetCursor(name string) {
	if m == nil || name == "" {
		return
	}
	if m.hasCurrent && name == m.current {
		return
	}
	if _, ok := m.registry.Slot(name); !ok {
		return
	}
	m.current, m.hasCurrent = name, true
	m.frameIndex = 0
	m.lastTick = 0
}

// Advance moves the frame index forward from nowMillis, the manager's own
// wall-clock reading in the units AI-CURSOR-172's `timeGetTime` source uses.
//
// A slot with one frame, or no current slot, never advances: the wrap this
// mirrors only exists to keep a multi-frame reader from observing an index at
// or above the registered count (SPR16A-CURSOR-061), and a single-frame slot
// has no second index to reach.
//
// THE FIRST CALL AFTER A SET ADVANCES, and that is the decoded shape rather
// than a choice. AI-CURSOR-172 states the consequence: "a single 'set
// cursor' call can only be observed to leave the frame index at 0 or 1". So
// SetCursor zeroes lastTick here for the same reason, and this call
// subtracts from zero rather than recording a baseline. Every one of the
// nine multi-frame registrations carries a period of 66 or 100 ms, far below
// any clock reading this build passes in, so the dependence on the clock's
// magnitude is not reachable on shipped data.
func (m *CursorManager) Advance(nowMillis int64) {
	if m == nil {
		return
	}
	slot := m.currentSlot()
	if slot == nil || slot.FrameCount <= 1 {
		return
	}
	if nowMillis-m.lastTick > slot.PeriodMillis {
		m.lastTick = nowMillis
		m.frameIndex++
		// The wrap happens HERE, before any reader below can observe an index
		// at or above the count (AI-CURSOR-172, SPR16A-CURSOR-061).
		if m.frameIndex >= slot.FrameCount {
			m.frameIndex = 0
		}
	}
}

// Current returns the picture and hotspot the manager's own frame index names,
// and whether a cursor is set with at least one resolved frame at all.
func (m *CursorManager) Current() (*image.RGBA, image.Point, bool) {
	slot := m.currentSlot()
	if slot == nil || len(slot.Frames) == 0 {
		return nil, image.Point{}, false
	}
	idx := m.frameIndex
	if idx < 0 || idx >= len(slot.Frames) {
		idx = 0
	}
	return slot.Frames[idx], slot.Hotspot, true
}

// CurrentName reports the current cursor's name, or "" for none set.
func (m *CursorManager) CurrentName() string {
	if m == nil || !m.hasCurrent {
		return ""
	}
	return m.current
}

// PointerHidden reports what this manager last told a caller it asked the
// engine for, and SetPointerHidden updates that cache and reports whether the
// engine must actually be told — pointerModeChange's own shape (cursor.go),
// generalized to whichever screen draws through this manager rather than
// through the map's own attackPointer field.
func (m *CursorManager) PointerHidden() bool { return m != nil && m.pointerHidden }

func (m *CursorManager) SetPointerHidden(hidden bool) bool {
	if m == nil || hidden == m.pointerHidden {
		return false
	}
	m.pointerHidden = hidden
	return true
}
