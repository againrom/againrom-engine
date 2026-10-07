package game

import (
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// Announcer turns a world's latch array into the announcements its script
// raised, WITHOUT the simulation carrying an announcement at all.
//
// WHY THIS IS NOT A pkg/sim ARM. The action that raises mission text changes no
// simulation state — in the original it posts a client window message and the
// server's own state does not branch on it — and pkg/sim can hold nothing extra
// anyway: World's field set is pinned for exact equality by a test whose purpose
// is catching state the byte form cannot reach, and the byte form's next version
// is spoken for. So the fact is derived here, above the wall, and no file under
// pkg/sim changes.
//
// WHAT MAKES IT DERIVABLE is the shape of the pass that writes the latches. An
// inert trigger is skipped before its latch is touched; a one-shot that has
// already fired is skipped whole, keeping its latch; every other trigger has its
// latch CLEARED at the top of the pass and set to 1 only if its conditions hold.
// So a 0-to-1 transition is a firing, exactly.
//
// SAMPLE IT ONCE PER STEP. The transition must be observed at the granularity
// the world advances at, not at the frame seam: one frame advances up to a
// catch-up bound of whole ticks — some sixteen script passes at the fast end of
// the cadence ladder — and a repeating trigger that fires and then does not
// inside one such frame has its latch cleared again before a frame-level sampler
// could look. That raise would not be merged with a neighbour; it would be lost.
//
// A repeating trigger held true never returns its latch to 0, so on a step that
// ran the pass its set latch is one more firing (DLG-LIFE-005 decides display).
type Announcer struct {
	// events is the announcements each latch raises, in slot order. It is keyed
	// by latch rather than ranged, because a latch is a trigger's position in
	// the MAP's array and the map's positions are sparse: a trigger the builder
	// dropped leaves a hole rather than shifting its neighbours.
	events map[int32][]int32

	// order is the latches that raise anything, in trigger order, so the
	// announcements of one step come out in a fixed order rather than a Go
	// map's.
	order []int32

	// prev is what each of those latches read at the previous sample. It is
	// seeded from the world the announcer is built for, so a world that arrives
	// with a one-shot already latched — one resumed from its bytes — does not
	// re-announce what fired before it got here.
	prev map[int32]bool

	// repeating is the latches of triggers neither fire-once nor inert.
	repeating map[int32]bool
}

// NewAnnouncer builds an announcer for w from the raise list its map's compile
// produced.
//
// THE RAISE LIST IS THE COMPILE'S because the compile owns the authored action
// id to runtime instant join. A build-time action becomes no instant and leaves
// no runtime trigger slot; a message hit becomes one row here.
//
// A nil world is a valid input: nothing is seeded and nothing can ever be
// raised, which is what an announcer for a map with no script should do.
func NewAnnouncer(w *sim.World, raises []mapload.ScriptRaise) *Announcer {
	a := &Announcer{
		events: make(map[int32][]int32, len(raises)),
		prev:   make(map[int32]bool, len(raises)),
	}
	if w != nil && w.Script() != nil {
		a.repeating = make(map[int32]bool)
		for _, t := range w.Script().Triggers() {
			if !t.Once && !t.Inert {
				a.repeating[t.Latch] = true
			}
		}
	}
	for _, r := range raises {
		if _, seen := a.events[r.Latch]; !seen {
			a.order = append(a.order, r.Latch)
		}
		a.events[r.Latch] = append(a.events[r.Latch], r.Event)
	}
	for _, latch := range a.order {
		a.prev[latch] = w.ScriptLatched(latch)
	}
	return a
}

// Sample reports the announcements w's script raised since the previous sample,
// and advances the announcer's memory.
//
// IT IS CALLED ONCE PER STEP and it reads the world and nothing else: no clock,
// no file, no counter. Nothing it does reaches the world, so a run that samples
// and a run that does not produce the same ticks and the same digests.
//
// The result is nil when nothing fired, which is every step but one in sixteen
// even on a busy map — the pass runs on one phase of the tick cycle, so most
// samples see no change at all and allocate nothing.
func (a *Announcer) Sample(w *sim.World) []int32 {
	if a == nil || len(a.order) == 0 {
		return nil
	}
	var out []int32
	pass := w.ScriptPassJustRan()
	for _, latch := range a.order {
		now := w.ScriptLatched(latch)
		// A firing is a rising edge or, for a repeating trigger, a set latch
		// on a pass step. A one-shot's spent latch is not a firing.
		if now && (!a.prev[latch] || pass && a.repeating[latch] && !w.ScriptTriggerBlocked(latch)) {
			out = append(out, a.events[latch]...)
		}
		a.prev[latch] = now
	}
	return out
}
