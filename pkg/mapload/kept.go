package mapload

import (
	"slices"

	"againrom/pkg/sim"
)

// KeptActor is the actor state a party member holds in the town between two
// missions: the end-of-mission cull's stores and what the mission left in
// the fields the cull does not touch (SAV-1353), or the same fields a town
// LOAD read. A nil KeptActor is an actor built in the town, which holds Unit
// constructor values. A value is replaced whole, never written through.
type KeptActor struct {
	// State is the actor state dword +0x50.
	State uint32
	// Done is the byte +0x136 the tick-list walk tests (SAV-1354).
	Done uint8
	// Deadline is the action-run end +0x138; the regeneration rate compares
	// the session sub-tick with it (SAV-1355).
	Deadline uint32
	// Timer is the byte +0x6c.
	Timer uint8
	// Mover is the 180-byte Mover record when the source holds one; the
	// next mission's placement reads its word +0x80 (SAV-1356). Its turn
	// rate byte 10 is 0: the actor's current basis supplies it.
	Mover        [180]byte
	MoverPresent bool
	// StaticRoute and DynamicRoute are the Mover's word lists +0x15c and
	// +0x178 (SAV-1357).
	StaticRoute, DynamicRoute []uint16
	// Order is the 148-byte Order record and its word ring when a LOAD read
	// them. The end-of-mission path holds no Order of its own.
	Order        [148]byte
	OrderRing    []uint16
	OrderPresent bool
}

// keptDeadline starts a kept actor with the action-run end it carried, which
// no STORE, cull or LOAD rewrites; against the restarted sub-tick it keeps
// the regeneration rate at 1 until the clock passes it by 80 (SAV-1355). An
// actor built in the town starts with an unknown run end instead.
func keptDeadline(e *sim.Entity, p PartyMember) {
	if p.Kept != nil {
		e.ActionClock = sim.ActionClock{Known: true, End: p.Kept.Deadline}
	}
}

// Clone copies the route and ring lists.
func (k *KeptActor) Clone() *KeptActor {
	if k == nil {
		return nil
	}
	out := *k
	out.StaticRoute = slices.Clone(k.StaticRoute)
	out.DynamicRoute = slices.Clone(k.DynamicRoute)
	out.OrderRing = slices.Clone(k.OrderRing)
	return &out
}
