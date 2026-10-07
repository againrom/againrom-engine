package mapload_test

import (
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// The channel fixture. A 24x24 map, so the interior is the 8x8 block at columns
// and rows 8..15; the water runs the whole of row 12 with ONE land cell in it,
// at column 14.
//
// The unit stands at (13,10) and is sent to (13,14) — straight down its own
// column, which the channel closes at (13,12). Reaching the target at all means
// leaving that column for the gap and coming back, so a unit that walked the
// straight line would end up standing on water.
const (
	rtW, rtH   = 24, 24
	rtChannelY = 12
	rtGapX     = 14
	rtFromX    = 13
	rtFromY    = 10
	rtToX      = 13
	rtToY      = 14
)

// rtMap is that map. gap says whether the land cell is there at all; blocker
// places a SECOND unit standing on it, which closes the crossing to a mover
// while leaving the terrain exactly as it was.
//
// The water runs the full width of the row rather than the interior's: outside
// the interior the ring already blocks, so the two are one plane, and a channel
// drawn to the map's own edges is what a map carries.
func rtMap(gap, blocker bool) *alm.Map {
	m := &alm.Map{
		Width: rtW, Height: rtH,
		Tiles:   make([]uint16, rtW*rtH),
		Overlay: make([]uint8, rtW*rtH),
		Units:   []alm.Unit{{X: rtFromX<<8 | 0x80, Y: rtFromY<<8 | 0x80}},
	}
	for x := 0; x < rtW; x++ {
		if gap && x == rtGapX {
			continue
		}
		m.Tiles[rtChannelY*rtW+x] = 0x0208 // index 520, inside [512,768)
	}
	if blocker {
		m.Units = append(m.Units, alm.Unit{X: rtGapX<<8 | 0x80, Y: rtChannelY<<8 | 0x80})
	}
	return m
}

// rtWantPlane is the plane that map describes, written from the contract: 0x03
// within 8 of an edge, 0x01 on the channel, 0x00 everywhere else — and the gap,
// when there is one, open like any other interior cell.
func rtWantPlane(gap bool) []byte {
	g := make([]byte, rtW*rtH)
	for y := 0; y < rtH; y++ {
		for x := 0; x < rtW; x++ {
			switch {
			case x < 8 || y < 8 || x >= rtW-8 || y >= rtH-8:
				g[y*rtW+x] = 0x03
			case y == rtChannelY && !(gap && x == rtGapX):
				g[y*rtW+x] = 0x01
			}
		}
	}
	return g
}

// rtWorld is that map's world in the given mode.
//
// Canonical is the loader's own. The optimised one is assembled beside it from
// the SAME derived plane — read out of the loaded world's byte form, so no
// second derivation exists here — and from the same entities, because the mode
// is fixed at construction and this loader names canonical.
func rtWorld(t *testing.T, m *alm.Map, mode sim.Mode) (*sim.World, []byte) {
	t.Helper()
	base := mapload.FromALM(m)
	plane := gridSection(t, base)
	if mode == sim.ModeCanonical {
		return base, plane
	}
	w, err := sim.NewWorld(mapload.Seed, base.Bounds(), mode, plane, base.Entities())
	if err != nil {
		t.Fatalf("NewWorld in mode %d over the derived plane: %v", mode, err)
	}
	return w, plane
}

// rtModes are the two, run as subtests so a failure names which one.
var rtModes = []struct {
	name string
	mode sim.Mode
}{{"canonical", sim.ModeCanonical}, {"optimised", sim.ModeOptimised}}

// TestTheFixturePlaneIsTheChannelTheContractDescribes checks the instrument
// before anything is measured with it: the cells the walk below is judged
// against are the cells the contract puts there, not merely the cells the
// derivation produced.
func TestTheFixturePlaneIsTheChannelTheContractDescribes(t *testing.T) {
	for _, gap := range []bool{true, false} {
		_, plane := rtWorld(t, rtMap(gap, false), sim.ModeCanonical)
		want := rtWantPlane(gap)
		if len(plane) != len(want) {
			t.Fatalf("gap=%v: the plane is %d byte(s), want %d", gap, len(plane), len(want))
		}
		for i := range want {
			if plane[i] != want[i] {
				t.Fatalf("gap=%v: cell (%d,%d) is %#02x, want %#02x",
					gap, i%rtW, i/rtW, plane[i], want[i])
			}
		}
	}
}

// TestAUnitCrossesAtTheGapAndNeverStandsOnWater is AC-9's arrival clause (SC-7),
// in both routing modes: the unit reaches its target, and after EVERY tick on
// the way its cell is checked against the plane.
//
// It also has to leave its own column, which is the water doing the turning: the
// straight line down column 13 runs into the channel, so a unit that never stood
// anywhere but column 13 either did not arrive or stood on water.
func TestAUnitCrossesAtTheGapAndNeverStandsOnWater(t *testing.T) {
	for _, m := range rtModes {
		t.Run(m.name, func(t *testing.T) {
			w, plane := rtWorld(t, rtMap(true, false), m.mode)

			// The bound is a cell count scaled by what a cell now costs: a
			// placement this fixture makes resolves to no definition, so it
			// takes the constructor's default speed and one straight cell is
			// 26 ticks, one diagonal 37.
			turned, arrived, ticks := false, false, 0
			for tick := 1; tick <= 32*37 && !arrived; tick++ {
				var cmds []sim.Command
				if tick == 1 {
					cmds = []sim.Command{{Entity: 0, X: rtToX, Y: rtToY}}
				}
				sim.Step(w, cmds)
				ticks = tick

				e := w.Entities()[0]
				if b := plane[int(e.Y)*rtW+int(e.X)]; b&0x01 != 0 {
					t.Fatalf("after tick %d the unit stands on (%d,%d), whose byte is %#02x — it walked "+
						"onto a cell that blocks ground", tick, e.X, e.Y, b)
				}
				if e.X != rtFromX {
					turned = true
				}
				arrived = e.X == rtToX && e.Y == rtToY
			}

			if !arrived {
				e := w.Entities()[0]
				t.Fatalf("the unit stands on (%d,%d) after %d ticks and never reached (%d,%d)",
					e.X, e.Y, ticks, rtToX, rtToY)
			}
			if !turned {
				t.Errorf("the unit reached its target without ever leaving column %d, which the channel "+
					"closes — the walk above was never turned by the water", rtFromX)
			}
			if e := w.Entities()[0]; e.HasTarget {
				t.Errorf("the arrived unit still heads for (%d,%d); an arrival clears the target",
					e.TargetX, e.TargetY)
			}
		})
	}
}

// TestAChannelWithNoGapIsWalkedUpToAndNeverCrossed is AC-9's second half as this
// tree actually behaves, and 0037 moves it again — so the criterion is once more
// witnessed against the tree's landed rule rather than against AC-9's sentence,
// and the disagreement reported rather than absorbed.
//
// AC-9 says the unit holds its target and gives up on the sixteenth. That is
// what a crossing closed by a BODY does, which the test below measures. A
// crossing closed by WATER used to end the order in the tick that found it
// — 0029 AC-2's "cleared on the first tick" over a wall with no gap, the
// very shape this fixture draws in water — and now does neither: the far
// search SETTLES for the labelled cell nearest the target, which on this map
// is the near bank, so the unit walks up to the water and stops there.
//
// The invariant that survives all three readings is the one this fixture exists
// for, and it is what is asserted here rather than a cell: THE UNIT NEVER
// CROSSES. It stays on its own side of the channel, never stands on a blocking
// cell, and ends with no residue — and the optimised mode, whose plane is built
// outward from a target it refuses, still never sets out at all.
func TestAChannelWithNoGapIsWalkedUpToAndNeverCrossed(t *testing.T) {
	for _, m := range rtModes {
		t.Run(m.name, func(t *testing.T) {
			w, plane := rtWorld(t, rtMap(false, false), m.mode)

			for tick := 1; tick <= 16; tick++ {
				var cmds []sim.Command
				if tick == 1 {
					cmds = []sim.Command{{Entity: 0, X: rtToX, Y: rtToY}}
				}
				sim.Step(w, cmds)

				e := w.Entities()[0]
				if e.Y >= rtChannelY {
					t.Fatalf("after tick %d the unit stands on (%d,%d), at or past the channel row %d — "+
						"no route crosses a channel with no gap in it", tick, e.X, e.Y, rtChannelY)
				}
				if b := plane[int(e.Y)*rtW+int(e.X)]; b&0x01 != 0 {
					t.Fatalf("after tick %d the unit's own cell (%d,%d) is %#02x", tick, e.X, e.Y, b)
				}
				if m.mode == sim.ModeOptimised && (e.X != rtFromX || e.Y != rtFromY) {
					t.Fatalf("after tick %d the unit stands on (%d,%d), want the cell it started on (%d,%d)",
						tick, e.X, e.Y, rtFromX, rtFromY)
				}
				if e.Stall != 0 {
					t.Fatalf("after tick %d the unit carries stall %d; it settles by arriving, not by "+
						"waiting", tick, e.Stall)
				}
			}
			// One step at most ONCE FACING THE BANK, and then it is settled: the
			// order ends on arrival, not on a stall. The canonical unit pays its
			// turn first — up to eight ticks at this fixture's own about-face and
			// RotationSpeed 16 — so the sixteen-tick budget above is read whole
			// rather than tick by tick, and the order must be clear by its end in
			// both modes.
			if e := w.Entities()[0]; e.HasTarget {
				t.Fatalf("after all 16 ticks the unit still heads for (%d,%d) — the bank is reachable and "+
					"an arrival clears the order", e.TargetX, e.TargetY)
			}
		})
	}
}

// TestACrossingHeldByABodyIsWaitedOutAndGivenUpOn is AC-9's "holds its side and
// gives up", over the fixture that actually produces it: the gap is there, and a
// second unit is standing in it.
//
// Terrain and occupancy part here, and that is the whole of the case. The far
// search reads the map alone, so it finds the crossing and stores a route through
// it; the near search reads the bodies as they stand, so it can never label the
// far bank, and the count runs out.
//
// What it may do first is WALK UP TO the water, because a near search that cannot
// label its waypoint settles for the cheapest cell it did reach and every such
// cell is on this side. So the tick the order ends on is not fixed and is not
// asserted; what is asserted every tick is that the mover is still on its own
// side, still off every blocking cell, and still carrying a count below the limit
// — the last being the state the byte form refuses, so a count that outlived the
// tick that raised it would be caught here rather than at the decode.
func TestACrossingHeldByABodyIsWaitedOutAndGivenUpOn(t *testing.T) {
	const stallLimit = 16

	for _, m := range rtModes {
		t.Run(m.name, func(t *testing.T) {
			w, plane := rtWorld(t, rtMap(true, true), m.mode)
			if got := len(w.Entities()); got != 2 {
				t.Fatalf("the fixture holds %d entities, want the mover and the body in the gap", got)
			}
			if b := plane[rtChannelY*rtW+rtGapX]; b != 0x00 {
				t.Fatalf("the gap cell (%d,%d) is %#02x, want 0x00 — the crossing must be closed by the "+
					"BODY and not by the terrain", rtGapX, rtChannelY, b)
			}

			gaveUp := 0
			for tick := 1; tick <= 4*stallLimit && gaveUp == 0; tick++ {
				var cmds []sim.Command
				if tick == 1 {
					cmds = []sim.Command{{Entity: 0, X: rtToX, Y: rtToY}}
				}
				sim.Step(w, cmds)

				e := w.Entities()[0]
				if e.Y >= rtChannelY {
					t.Fatalf("after tick %d the mover stands on (%d,%d), at or past the channel row %d — "+
						"the only crossing is held by a body", tick, e.X, e.Y, rtChannelY)
				}
				if b := plane[int(e.Y)*rtW+int(e.X)]; b&0x01 != 0 {
					t.Fatalf("after tick %d the mover's own cell (%d,%d) is %#02x", tick, e.X, e.Y, b)
				}
				if body := w.Entities()[1]; body.X != rtGapX || body.Y != rtChannelY {
					t.Fatalf("after tick %d the body stands on (%d,%d), want the gap (%d,%d) — it takes no "+
						"order and must not have moved", tick, body.X, body.Y, rtGapX, rtChannelY)
				}
				if e.Stall >= stallLimit {
					t.Fatalf("after tick %d the mover carries stall %d, at or above the limit — the give-up "+
						"fires in the tick that raises the count, so no stored count reaches it",
						tick, e.Stall)
				}

				if e.HasTarget {
					if e.TargetX != rtToX || e.TargetY != rtToY {
						t.Fatalf("after tick %d the mover heads for (%d,%d), want (%d,%d) — nothing here "+
							"re-aims an order", tick, e.TargetX, e.TargetY, rtToX, rtToY)
					}
					continue
				}
				gaveUp = tick

				// The count is spent inside the same tick that raised it, so
				// nothing is left behind.
				if e.Stall != 0 || e.TargetX != 0 || e.TargetY != 0 {
					t.Errorf("the given-up mover carries stall %d and target (%d,%d), want no residue at all",
						e.Stall, e.TargetX, e.TargetY)
				}
			}
			if gaveUp == 0 {
				t.Fatalf("the mover still holds its order after %d ticks — the crossing is held and the "+
					"count must run out", 4*stallLimit)
			}

			// ...and the world it ends in is one its own form reads back, which is
			// what says the give-up left no state the decoder refuses.
			form, err := w.MarshalBinary()
			if err != nil {
				t.Fatalf("MarshalBinary: %v", err)
			}
			var back sim.World
			if err := back.UnmarshalBinary(form); err != nil {
				t.Fatalf("the world the give-up left does not read back: %v", err)
			}
		})
	}
}
