package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// A mover of footprint side n stores its footprint's top-left cell, and the
// centre every distance and bearing reader takes is that corner plus (n-1)*128
// per axis. On mission 20's own terrain a Wimpy mover that has fallen below its
// threshold decides a retreat from a hostile in sight. The expected cell is
// worked here in 8.8 coordinates, from the arithmetic the retreat claims state,
// and not by calling the engine's own routine.

// fleeExpected is the destination three cells away from one hostile: the
// zero axis becomes +1, the larger axis takes the full displacement and the
// other axis is truncated toward zero, then the point is shifted to whole
// cells. sx/sy and hx/hy are 8.8 centres.
func fleeExpected(sx, sy, hx, hy int) (int, int) {
	dx, dy := sx-hx, sy-hy
	if dx == 0 {
		dx = 1
	}
	if dy == 0 {
		dy = 1
	}
	abs := func(v int) int {
		if v < 0 {
			return -v
		}
		return v
	}
	const stride = 3 * 256
	if abs(dx) >= abs(dy) {
		dxs := stride
		if dx < 0 {
			dxs = -stride
		}
		sx, sy = sx+dxs, sy+stride*dy/abs(dx)
	} else {
		dys := stride
		if dy < 0 {
			dys = -stride
		}
		sx, sy = sx+stride*dx/abs(dy), sy+dys
	}
	return sx >> 8, sy >> 8
}

func retreatWorld(t *testing.T, r *areaCostRig, side uint8, dying int32) (*sim.World, [2]int) {
	t.Helper()
	var rel sim.Relations
	rel.Set(2, 3, 1)
	rel.Set(3, 2, 1)
	at := [2]int{r.px + 6, r.py + 6}
	self := sim.Entity{ID: 1, Owner: 2, X: int32(at[0]), Y: int32(at[1]), PostX: int32(at[0]), PostY: int32(at[1]),
		TokenSize: side, HP: 8, MaxHP: 100, Wimpy: 8, ScanRange: 5, Reach: 1, DamageBase: 1, AlwaysHits: true, DyingTime: dying, Speed: 16}
	foe := sim.Entity{ID: 2, Owner: 3, X: int32(at[0] + 4), Y: int32(at[1]), PostX: int32(at[0] + 4), PostY: int32(at[1]),
		HP: 100, MaxHP: 100, ScanRange: 5, Reach: 1, DamageBase: 1, AlwaysHits: true, DyingTime: dying, Speed: 16}
	w, err := sim.NewStructuredWorld(2020, sim.Bounds{Width: int32(r.width), Height: int32(r.height)}, sim.ModeCanonical,
		sim.Terrain{Block: r.block, Cost: r.cost, Height: nil}, []sim.Entity{self, foe}, nil, rel, nil, nil,
		mapload.SpellRules(r.f.Table), sim.GhostTemplate{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return w, at
}

// retreatTarget runs ordinary ticks until the Wimpy mover has decided its
// retreat and returns the destination it wrote.
func retreatTarget(t *testing.T, w *sim.World) (int32, int32) {
	t.Helper()
	for i := 0; i < 64; i++ {
		sim.Step(w, nil)
		for _, e := range w.Entities() {
			if e.ID == 1 && e.HasTarget {
				return e.TargetX, e.TargetY
			}
		}
	}
	t.Fatal("the Wimpy mover never decided a retreat")
	return 0, 0
}

func TestReleaseWideMoverRetreatsFromItsFootprintCentre(t *testing.T) {
	r := openAreaCostRig(t)
	cases := map[uint8]int{1: 0, 2: 0, 3: 0}
	for side := range cases {
		w, at := retreatWorld(t, r, side, 200)
		wantX, wantY := fleeExpected((at[0])*256+int(side)*128, at[1]*256+int(side)*128, (at[0]+4)*256+128, at[1]*256+128)
		gotX, gotY := retreatTarget(t, w)
		if int(gotX) != wantX || int(gotY) != wantY {
			t.Errorf("side %d: retreat destination (%d,%d), want (%d,%d)", side, gotX, gotY, wantX, wantY)
		}
		if side > 1 {
			// Loss control: reading the mover at the centre of its anchor cell
			// gives another cell, so this witness separates the two readings.
			cornerX, cornerY := fleeExpected(at[0]*256+128, at[1]*256+128, (at[0]+4)*256+128, at[1]*256+128)
			if cornerX == wantX && cornerY == wantY {
				t.Errorf("side %d: the anchor-centre reading gives the same cell (%d,%d), so the witness does not see the footprint centre", side, wantX, wantY)
			}
		}
		// SAVE and a cold LOAD carry the decision, and both worlds continue alike.
		form, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var cold sim.World
		if err := cold.UnmarshalBinary(form); err != nil {
			t.Fatal(err)
		}
		if cold.Hash() != w.Hash() {
			t.Fatalf("side %d: the cold world differs from the saved one", side)
		}
		for i := 0; i < 8; i++ {
			sim.Step(w, nil)
			sim.Step(&cold, nil)
			if cold.Hash() != w.Hash() {
				t.Fatalf("side %d: the cold world differs %d ticks after LOAD", side, i+1)
			}
		}
	}
}

// A body that falls on a layered cell is torn down when its dying time ends,
// and the teardown recomputes every cell of its footprint. A first mover is
// felled at the layered cell on the tick after its read decayed the cell's
// byte, before its crossing; a second mover that enters the cell after the
// teardown finds the recomputed byte and takes the baseline transit, where a
// world that never recomputes finds the decayed one.
func TestReleaseTeardownRecomputesTheLayeredCellUnderABody(t *testing.T) {
	r := openAreaCostRig(t)
	scratch := r.world(t, [2]int{}, [2]int{}, nil)
	triples := r.triples(r.cast(t, scratch))
	build := func(a, b, c [2]int, cost []byte, layer bool) *sim.World {
		w := shortDyingAreaWorld(t, r, b, c, cost)
		if layer {
			r.cast(t, w)
		}
		return w
	}
	enter := func(w *sim.World, id sim.EntityID, a [2]int) uint16 {
		sim.Step(w, []sim.Command{sim.MoveTo(id, sim.CellPoint{X: int32(a[0]), Y: int32(a[1])})})
		for _, e := range w.Entities() {
			if e.ID == id {
				if e.X != int32(a[0]) || e.Y != int32(a[1]) {
					t.Fatalf("mover %d stands at (%d,%d), want the layered cell %v", id, e.X, e.Y, a)
				}
				return e.TransitTotal
			}
		}
		return 0
	}
	for _, tr := range triples {
		a, b, c := tr[0], tr[1], tr[2]
		base := r.cost[a[1]*r.width+a[0]]
		decayed := append([]byte(nil), r.cost...)
		decayed[a[1]*r.width+a[0]] = base >> 2
		plain := enter(build(a, b, c, nil, false), areaCostSecond, a)
		quarter := enter(build(a, b, c, decayed, false), areaCostSecond, a)
		if plain == quarter {
			continue
		}
		w := build(a, b, c, nil, true)
		// The first mover's start tick reads the layered cell once.
		sim.Step(w, []sim.Command{sim.MoveTo(areaCostFirst, sim.CellPoint{X: int32(a[0]), Y: int32(a[1])})})
		sim.Step(w, []sim.Command{sim.TerminalKill(areaCostFirst)})
		for i := 0; i < 8; i++ {
			sim.Step(w, nil)
		}
		for _, e := range w.Entities() {
			if e.ID == areaCostFirst && e.Alive() {
				t.Fatal("the first mover is alive after its kill")
			}
		}
		// SAVE and a cold LOAD between the teardown and the entry.
		form, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var cold sim.World
		if err := cold.UnmarshalBinary(form); err != nil {
			t.Fatal(err)
		}
		if got := enter(w, areaCostSecond, a); got != plain {
			t.Errorf("layered cell %v: the entry after a teardown takes transit %d, want the baseline's %d (a decayed byte gives %d)", a, got, plain, quarter)
		}
		if got := enter(&cold, areaCostSecond, a); got != plain {
			t.Errorf("layered cell %v: the cold-loaded world's entry takes transit %d, want %d", a, got, plain)
		}
		if cold.Hash() != w.Hash() {
			t.Error("the cold-loaded world and the continuing world differ after the same entry")
		}
		return
	}
	t.Fatalf("none of %d layered-cell candidates has a baseline byte whose quarter changes the transit", len(triples))
}

// shortDyingAreaWorld is the area rig's arena with the two movers at a and b
// and a dying time of three ticks, so a body's teardown falls well inside the
// life of a cast cloud.
func shortDyingAreaWorld(t *testing.T, r *areaCostRig, a, b [2]int, cost []byte) *sim.World {
	t.Helper()
	if cost == nil {
		cost = r.cost
	}
	cx, cy := r.px+7, r.py+7
	mover := func(id sim.EntityID, at [2]int) sim.Entity {
		return sim.Entity{ID: id, X: int32(at[0]), Y: int32(at[1]), PostX: int32(at[0]), PostY: int32(at[1]),
			HP: 5000, MaxHP: 5000, DyingTime: 3, Speed: 16, Owner: sim.SelfSlot}
	}
	mage := sim.Entity{ID: areaCostCaster, X: int32(cx - 3), Y: int32(cy), PostX: int32(cx - 3), PostY: int32(cy), HP: 5000, MaxHP: 5000, DyingTime: 200,
		Owner: sim.SelfSlot, Mind: 60, MaxMana: 900, Mana: 900, KnownSpells: 1 << r.spell, ScanRange: 6, Reach: 1}
	w, err := sim.NewStructuredWorld(2020, sim.Bounds{Width: int32(r.width), Height: int32(r.height)}, sim.ModeCanonical,
		sim.Terrain{Block: r.block, Cost: cost, Height: nil}, []sim.Entity{mage, mover(areaCostFirst, a), mover(areaCostSecond, b)}, nil, sim.Relations{}, nil, nil,
		mapload.SpellRules(r.f.Table), sim.GhostTemplate{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return w
}
