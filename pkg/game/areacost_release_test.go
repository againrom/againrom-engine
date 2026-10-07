package game

import (
	"bytes"
	"math/bits"
	"reflect"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// The mutable movement-cost byte under a cast area spell, on mission 20's own
// terrain, cost plane and spell rules. The caster, the two movers and every
// order are fixtures; the cloud, its cells and the cost bytes the movers read
// are the install's and the engine's.
//
// The first mover leaves a layered cell and the second enters it on the same
// tick. Each read of a layered cell divides its stored byte by four and stores
// it back, and the recompute at a crossing comes later, so the second read
// answers a quarter of the first. The first read answers the baseline.

type areaCostRig struct {
	f      *FrontEnd
	block  []byte
	cost   []byte
	width  int
	height int
	px, py int
	spell  uint16
}

const (
	areaCostCaster = sim.EntityID(1)
	areaCostFirst  = sim.EntityID(2)
	areaCostSecond = sim.EntityID(3)
)

func openAreaCostRig(t *testing.T) *areaCostRig {
	t.Helper()
	f := releaseFront(t)
	m := releaseMissionMap(t, f, 20)
	r := &areaCostRig{f: f, block: mapload.PassabilityWith(m, f.Table), cost: mapload.Cost(m), width: m.Width, height: m.Height}
	px, py, found := openGroundSquare(r.block, r.width, r.height, 15)
	if !found {
		t.Fatal("mission 20 has no open ground square of 15 cells")
	}
	r.px, r.py = px, py
	for _, rule := range mapload.SpellRules(f.Table) {
		_, layer := areaLayerIndex(rule.ID)
		if layer && rule.Area && rule.AreaMode() == sim.AreaModeCloud && rule.Damaging && (r.spell == 0 || rule.ID == 12) {
			r.spell = rule.ID
		}
	}
	if r.spell == 0 {
		t.Fatal("the install has no damaging area spell that paints a layer")
	}
	return r
}

func (r *areaCostRig) key(x, y int) uint16 { return uint16(y)<<8 | uint16(x) }

// world builds the arena over the install's planes with the movers at a and b.
// A nil cost plane takes the map's own.
func (r *areaCostRig) world(t *testing.T, a, b [2]int, cost []byte) *sim.World {
	t.Helper()
	if cost == nil {
		cost = r.cost
	}
	cx, cy := r.px+7, r.py+7
	mover := func(id sim.EntityID, at [2]int) sim.Entity {
		return sim.Entity{ID: id, X: int32(at[0]), Y: int32(at[1]), PostX: int32(at[0]), PostY: int32(at[1]),
			HP: 5000, MaxHP: 5000, DyingTime: 200, Speed: 16, Owner: sim.SelfSlot}
	}
	mage := sim.Entity{ID: areaCostCaster, X: int32(cx - 3), Y: int32(cy), PostX: int32(cx - 3), PostY: int32(cy), HP: 5000, MaxHP: 5000, DyingTime: 200,
		Owner: sim.SelfSlot, Mind: 60, MaxMana: 900, Mana: 900, KnownSpells: 1 << r.spell, ScanRange: 6, Reach: 1}
	ents := []sim.Entity{mage}
	if a != [2]int{} {
		ents = append(ents, mover(areaCostFirst, a), mover(areaCostSecond, b))
	}
	w, err := sim.NewStructuredWorld(2020, sim.Bounds{Width: int32(r.width), Height: int32(r.height)}, sim.ModeCanonical,
		sim.Terrain{Block: r.block, Cost: cost, Height: nil}, ents, nil, sim.Relations{}, nil, nil,
		mapload.SpellRules(r.f.Table), sim.GhostTemplate{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

// cast runs the caster's cloud and returns its cells once it stands.
func (r *areaCostRig) cast(t *testing.T, w *sim.World) []uint16 {
	t.Helper()
	cx, cy := int32(r.px+7), int32(r.py+7)
	sim.Step(w, []sim.Command{sim.CastAt(areaCostCaster, sim.SpellID(r.spell), sim.CellPoint{X: cx, Y: cy})})
	for i := 0; ; i++ {
		states, err := w.NativeAreaSaveStates()
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range states {
			if s.Mode == sim.AreaModeCloud && s.Spell == r.spell {
				sim.Step(w, nil)
				return s.Cells
			}
		}
		if i > 300 {
			t.Fatalf("the caster never raised spell %d", r.spell)
		}
		sim.Step(w, nil)
	}
}

// landing casts the layer and returns the number of steps, the first carrying
// the cast order, after which the cloud stands.
func (r *areaCostRig) landing(t *testing.T, w *sim.World) int {
	t.Helper()
	cx, cy := int32(r.px+7), int32(r.py+7)
	sim.Step(w, []sim.Command{sim.CastAt(areaCostCaster, sim.SpellID(r.spell), sim.CellPoint{X: cx, Y: cy})})
	for n := 1; n < 300; n++ {
		states, err := w.NativeAreaSaveStates()
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range states {
			if s.Mode == sim.AreaModeCloud && s.Spell == r.spell {
				return n
			}
		}
		sim.Step(w, nil)
	}
	t.Fatalf("the caster never raised spell %d", r.spell)
	return 0
}

func (r *areaCostRig) open(x, y int) bool {
	return x >= r.px && y >= r.py && x < r.px+15 && y < r.py+15 && r.block[y*r.width+x] == 0
}

// triple finds a layered cell a with two distinct uncovered open orthogonal
// neighbours: b, the second mover's start, and c, the first mover's goal.
func (r *areaCostRig) triples(cells []uint16) (out [][3][2]int) {
	covered := map[uint16]bool{}
	for _, k := range cells {
		covered[k] = true
	}
	free := func(x, y int) bool {
		return r.open(x, y) && !covered[r.key(x, y)] && !(x == r.px+1 && y == r.py+7)
	}
	steps := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	for _, k := range cells {
		ax, ay := int(k&255), int(k>>8)
		if !r.open(ax, ay) {
			continue
		}
		for _, sb := range steps {
			for _, sc := range steps {
				bx, by, cx, cy := ax+sb[0], ay+sb[1], ax+sc[0], ay+sc[1]
				if sb != sc && free(bx, by) && free(cx, cy) {
					out = append(out, [3][2]int{{ax, ay}, {bx, by}, {cx, cy}})
				}
			}
		}
	}
	return out
}

// transits runs the one tick of orders and reports the two movers' transits.
func areaCostTransits(t *testing.T, w *sim.World, a, b, c [2]int) (first, second uint16) {
	t.Helper()
	sim.Step(w, []sim.Command{
		sim.MoveTo(areaCostFirst, sim.CellPoint{X: int32(c[0]), Y: int32(c[1])}),
		sim.MoveTo(areaCostSecond, sim.CellPoint{X: int32(a[0]), Y: int32(a[1])}),
	})
	for _, e := range w.Entities() {
		switch e.ID {
		case areaCostFirst:
			if e.X != int32(c[0]) || e.Y != int32(c[1]) {
				t.Fatalf("the first mover stands at (%d,%d), want its goal %v", e.X, e.Y, c)
			}
			first = e.TransitTotal
		case areaCostSecond:
			if e.X != int32(a[0]) || e.Y != int32(a[1]) {
				t.Fatalf("the second mover stands at (%d,%d), want the layered cell %v", e.X, e.Y, a)
			}
			second = e.TransitTotal
		}
	}
	return first, second
}

// areaCostWitness is the claim under test: over the layered cell the first
// mover's transit is the baseline's and the second mover's is the one a
// quarter-cost cell gives. It returns what differs.
func areaCostWitness(want [2]uint16, first, second uint16) []string {
	var bad []string
	if first != want[0] {
		bad = append(bad, "the first read is not the baseline's")
	}
	if second != want[1] {
		bad = append(bad, "the second read is not the divided byte's")
	}
	return bad
}

// Mission 20, a cast area spell, and two movers over its cell on one tick:
// the second reader's transit is the one a cost byte divided by four gives,
// and the first reader's is the unlayered baseline's. With no layer standing
// the same witness fails.
func TestReleaseAreaLayerMovementCost(t *testing.T) {
	r := openAreaCostRig(t)
	scratch := r.world(t, [2]int{}, [2]int{}, nil)
	cells := r.cast(t, scratch)
	triples := r.triples(cells)
	if len(triples) == 0 {
		t.Fatalf("spell %d painted %04x with no layered cell that has two open uncovered neighbours", r.spell, cells)
	}
	for _, tr := range triples {
		a, b, c := tr[0], tr[1], tr[2]
		base := r.cost[a[1]*r.width+a[0]]
		decayed := append([]byte(nil), r.cost...)
		decayed[a[1]*r.width+a[0]] = base >> 2

		bare := r.world(t, a, b, nil)
		plain1, plain2 := areaCostTransits(t, bare, a, b, c)
		quarter := r.world(t, a, b, decayed)
		_, want2 := areaCostTransits(t, quarter, a, b, c)
		if want2 == plain2 {
			continue
		}

		layered := r.world(t, a, b, nil)
		if got := r.cast(t, layered); len(got) != len(cells) {
			t.Fatalf("the cast covered %d cells here and %d in the scratch world", len(got), len(cells))
		}
		first, second := areaCostTransits(t, layered, a, b, c)
		if bad := areaCostWitness([2]uint16{plain1, want2}, first, second); len(bad) != 0 {
			t.Fatalf("layered cell %v (baseline %d), second mover from %v, first mover to %v: transits %d and %d, want %d and %d: %v",
				a, base, b, c, first, second, plain1, want2, bad)
		}
		// Loss control: the same witness over a world with no layer standing
		// reports the second reader.
		if bad := areaCostWitness([2]uint16{plain1, want2}, plain1, plain2); len(bad) == 0 {
			t.Fatal("the witness passes over bare ground, so it does not see the layer")
		}
		t.Logf("layered cell %v baseline %d: first transit %d (baseline %d), second transit %d (bare %d, divided %d)", a, base, first, plain1, second, plain2, want2)
		return
	}
	t.Fatalf("none of %d layered-cell candidates has a baseline byte whose quarter changes the transit", len(triples))
}

// areaCostCrossingTick is the transit tick on which a stride's cell changes,
// counted from the tick that started the transit: the position is one
// sixteen-bit value per axis, starts at fraction 0x80 and gains the signed step
// on every tick, the starting tick included.
func areaCostCrossingTick(t *testing.T, s sim.NativeStride) int {
	t.Helper()
	x, y := int(s.FromX)<<8|0x80, int(s.FromY)<<8|0x80
	for tick := 0; tick < 300; tick++ {
		nx, ny := x+int(s.StepX), y+int(s.StepY)
		if nx>>8 != x>>8 || ny>>8 != y>>8 {
			return tick
		}
		x, y = nx, ny
	}
	t.Fatalf("stride %+v never changes cell", s)
	return 0
}

// areaCostPair runs the first mover from a to c on tick 0 and the second mover
// from b onto a on tick delay, and returns the second mover's transit and the
// first mover's crossing tick.
func areaCostPair(t *testing.T, w *sim.World, a, b, c [2]int, delay int) (second uint16, crossing int) {
	t.Helper()
	sim.Step(w, []sim.Command{sim.MoveTo(areaCostFirst, sim.CellPoint{X: int32(c[0]), Y: int32(c[1])})})
	for _, e := range w.Entities() {
		if e.ID == areaCostFirst {
			if !e.Stride.Present {
				t.Fatal("the first mover has no stride")
			}
			crossing = areaCostCrossingTick(t, e.Stride)
		}
	}
	for i := 1; i < delay; i++ {
		sim.Step(w, nil)
	}
	sim.Step(w, []sim.Command{sim.MoveTo(areaCostSecond, sim.CellPoint{X: int32(a[0]), Y: int32(a[1])})})
	for _, e := range w.Entities() {
		if e.ID == areaCostSecond {
			if e.X != int32(a[0]) || e.Y != int32(a[1]) {
				t.Fatalf("the second mover stands at (%d,%d), want the layered cell %v", e.X, e.Y, a)
			}
			second = e.TransitTotal
		}
	}
	return second, crossing
}

// The first mover leaves a layered cell, and its crossing recomputes the cell
// on one tick. A second mover that reads the cell one tick earlier finds the
// byte the first mover's read left, a quarter of the baseline; on the crossing
// tick, after the first mover in the entity list, it finds the recomputed byte
// and gets the baseline transit. Bare ground gives the baseline on both ticks.
func TestReleaseAreaLayerCostRecomputesOnTheCrossingTick(t *testing.T) {
	r := openAreaCostRig(t)
	scratch := r.world(t, [2]int{}, [2]int{}, nil)
	triples := r.triples(r.cast(t, scratch))
	for _, tr := range triples {
		a, b, c := tr[0], tr[1], tr[2]
		base := r.cost[a[1]*r.width+a[0]]
		decayed := append([]byte(nil), r.cost...)
		decayed[a[1]*r.width+a[0]] = base >> 2
		_, plain2 := areaCostTransits(t, r.world(t, a, b, nil), a, b, c)
		_, want2 := areaCostTransits(t, r.world(t, a, b, decayed), a, b, c)
		if want2 == plain2 {
			continue
		}
		layered := func() *sim.World {
			w := r.world(t, a, b, nil)
			r.cast(t, w)
			return w
		}
		probe := layered()
		_, crossing := areaCostPair(t, probe, a, b, c, 2)
		if crossing < 3 {
			t.Fatalf("crossing tick %d, a transit crosses no earlier than its third tick", crossing)
		}
		before, _ := areaCostPair(t, layered(), a, b, c, crossing-1)
		on, _ := areaCostPair(t, layered(), a, b, c, crossing)
		if before != want2 {
			t.Errorf("layered cell %v: a read on tick %d gives transit %d, want the quarter byte's %d", a, crossing-1, before, want2)
		}
		if on != plain2 {
			t.Errorf("layered cell %v: a read on crossing tick %d gives transit %d, want the baseline's %d", a, crossing, on, plain2)
		}
		// Loss control: over bare ground neither tick shows a layer.
		bareBefore, _ := areaCostPair(t, r.world(t, a, b, nil), a, b, c, crossing-1)
		bareOn, _ := areaCostPair(t, r.world(t, a, b, nil), a, b, c, crossing)
		if bareBefore != plain2 || bareOn != plain2 {
			t.Errorf("bare ground gives %d and %d, want %d on both ticks", bareBefore, bareOn, plain2)
		}
		t.Logf("layered cell %v baseline %d crossing tick %d: transits %d before, %d on (quarter %d, baseline %d)", a, base, crossing, before, on, want2, plain2)
		return
	}
	t.Fatalf("none of %d layered-cell candidates has a baseline byte whose quarter changes the transit", len(triples))
}

// The area-effect tick runs after every actor tick. A cast whose cloud lands
// on a tick is laid after that tick's movers: the first mover leaves the cell
// on the landing tick and reads no layer, so a second mover entering on the
// next tick is the first reader and gets the baseline. With the cloud already
// standing the same two movers give the second one the quarter byte.
func TestReleaseAreaLayerLandsAfterTheActorsOfItsTick(t *testing.T) {
	r := openAreaCostRig(t)
	scratch := r.world(t, [2]int{}, [2]int{}, nil)
	triples := r.triples(r.cast(t, scratch))
	for _, tr := range triples {
		a, b, c := tr[0], tr[1], tr[2]
		base := r.cost[a[1]*r.width+a[0]]
		decayed := append([]byte(nil), r.cost...)
		decayed[a[1]*r.width+a[0]] = base >> 2
		_, plain2 := areaCostTransits(t, r.world(t, a, b, nil), a, b, c)
		_, want2 := areaCostTransits(t, r.world(t, a, b, decayed), a, b, c)
		if want2 == plain2 {
			continue
		}
		land := r.landing(t, r.world(t, a, b, nil))
		w := r.world(t, a, b, nil)
		cx, cy := int32(r.px+7), int32(r.py+7)
		for step := 1; step <= land; step++ {
			var cmds []sim.Command
			if step == 1 {
				cmds = append(cmds, sim.CastAt(areaCostCaster, sim.SpellID(r.spell), sim.CellPoint{X: cx, Y: cy}))
			}
			if step == land {
				cmds = append(cmds, sim.MoveTo(areaCostFirst, sim.CellPoint{X: int32(c[0]), Y: int32(c[1])}))
			}
			sim.Step(w, cmds)
		}
		sim.Step(w, []sim.Command{sim.MoveTo(areaCostSecond, sim.CellPoint{X: int32(a[0]), Y: int32(a[1])})})
		var second uint16
		for _, e := range w.Entities() {
			if e.ID == areaCostSecond {
				second = e.TransitTotal
			}
		}
		if second != plain2 {
			t.Errorf("cloud landing on tick %d over %v: the second mover's transit is %d, want the baseline's %d", land, a, second, plain2)
		}
		// Loss control: the cloud standing before the first mover leaves.
		standing := r.world(t, a, b, nil)
		r.cast(t, standing)
		sim.Step(standing, []sim.Command{sim.MoveTo(areaCostFirst, sim.CellPoint{X: int32(c[0]), Y: int32(c[1])})})
		sim.Step(standing, []sim.Command{sim.MoveTo(areaCostSecond, sim.CellPoint{X: int32(a[0]), Y: int32(a[1])})})
		for _, e := range standing.Entities() {
			if e.ID == areaCostSecond && e.TransitTotal != want2 {
				t.Errorf("with the cloud standing the second mover's transit is %d, want the quarter byte's %d", e.TransitTotal, want2)
			}
		}
		return
	}
	t.Fatalf("none of %d layered-cell candidates has a baseline byte whose quarter changes the transit", len(triples))
}

// A mage raises an area layer on mission 20 and walks into it. The mission is
// saved and loaded cold; the cost plane is not part of the file, so the loaded
// world takes the layered cell's byte from the layers the file carries. After
// the same orders the continuing and loaded worlds marshal alike. A world
// loaded from before the cast is the loss control.
func TestReleaseAreaLayerCostContinuesAcrossSaveAndLoad(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	const mission = 20
	app := f.App("area cost save")
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpenerWith(mission, MissionPartyAs(true, f.StartWeapon.Value(), f.Bodies, f.Table))); err != nil {
		t.Fatal(err)
	}
	for k := 0; k < 16 && f.live.mission.open; k++ {
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	var mage sim.Entity
	for _, e := range f.live.world.Entities() {
		if e.Owner == sim.SelfSlot && e.Alive() && e.KnownSpells != 0 {
			mage = e
		}
	}
	spell := uint16(0)
	for known := mage.KnownSpells; known != 0 && spell == 0; known &= known - 1 {
		id := uint16(bits.TrailingZeros32(known))
		rule, ok := f.live.world.Spell(uint32(id))
		if _, cloud := areaLayerIndex(id); ok && cloud && rule.Area && rule.AreaMode() == sim.AreaModeCloud {
			spell = id
		}
	}
	if spell == 0 {
		t.Fatalf("mission %d mage %d knows no cloud spell: %032b", mission, mage.ID, mage.KnownSpells)
	}
	// Loss control: a save taken before any layer stands.
	bareStore := SaveStore{Dir: t.TempDir()}
	bareName, _ := deadPatrolSave(t, f, bareStore)
	f.live.attackOrCast(uint32(mage.ID), 0, uint32(spell), int(mage.X)+3, int(mage.Y), true)
	for i := 0; !cloudCellWitness(t, f, mage.ID).Present; i++ {
		if i > 256 {
			t.Fatalf("mage %d never cast cloud spell %d", mage.ID, spell)
		}
		f.live.tick()
	}
	f.live.tick()
	mage = releaseEntity(t, f.live, mage.ID)
	at := cloudCellWitness(t, f, mage.ID)
	target, found := [2]int{}, false
	for _, key := range at.Cells {
		x, y := int(key&255), int(key>>8)
		if x == int(mage.X)+1 && y == int(mage.Y) {
			target, found = [2]int{x, y}, true
		}
	}
	if !found {
		t.Fatalf("the cloud %04x does not cover the cell beside the mage at (%d,%d)", at.Cells, mage.X, mage.Y)
	}

	store := SaveStore{Dir: t.TempDir()}
	name, _ := deadPatrolSave(t, f, store)
	g := loadLocalLegacySave(t, store, name)
	if got := cloudCellWitness(t, g, mage.ID); !reflect.DeepEqual(got, at) {
		t.Fatalf("LOAD cloud %+v, want %+v", got, at)
	}

	var h *FrontEnd
	order := func(x *FrontEnd) sim.Entity {
		x.live.pending = append(x.live.pending, sim.MoveTo(mage.ID, sim.CellPoint{X: int32(target[0]), Y: int32(target[1])}))
		x.live.tick()
		e := releaseEntity(t, x.live, mage.ID)
		for i := 0; i < 5 && x == h && (e.X != int32(target[0]) || e.Y != int32(target[1])); i++ {
			x.live.tick()
			e = releaseEntity(t, x.live, mage.ID)
		}
		if e.X != int32(target[0]) || e.Y != int32(target[1]) {
			t.Fatalf("the mage stands at (%d,%d) tick %d x==h %v, want the layered cell %v", e.X, e.Y, x.live.world.Tick(), x == h, target)
		}
		return e
	}
	predicted := func(x *FrontEnd) (int32, int32) {
		rate, transit, adjacent, ok := x.live.world.StepRate(mage.ID, int32(target[0]), int32(target[1]))
		if !ok || !adjacent {
			t.Fatalf("no rate onto the layered cell: ok %v adjacent %v", ok, adjacent)
		}
		return rate, transit
	}

	fr, ft := predicted(f)
	gr, gt := predicted(g)
	if fr != gr || ft != gt {
		t.Errorf("before the step the continuing world rates %d/%d and the loaded world %d/%d", fr, ft, gr, gt)
	}
	h = loadLocalLegacySave(t, bareStore, bareName)

	fe, ge := order(f), order(g)
	if fe.TransitTotal != ge.TransitTotal || fe.X != ge.X || fe.Y != ge.Y {
		t.Errorf("after the step the continuing mage has transit %d at (%d,%d) and the loaded mage %d at (%d,%d)",
			fe.TransitTotal, fe.X, fe.Y, ge.TransitTotal, ge.X, ge.Y)
	}
	if int32(fe.TransitTotal) != ft {
		t.Errorf("the continuing mage's transit is %d, its rate query said %d", fe.TransitTotal, ft)
	}

	// The mage's rate sits at its ceiling, so transit cannot show the layer.
	// The step back reads the cell twice over and leaves a decayed byte that
	// the native form carries: continuing and loaded worlds must marshal alike.
	back := func(x *FrontEnd) uint16 {
		e := releaseEntity(t, x.live, mage.ID)
		for i := 0; i < 400 && e.Transit != 0; i++ {
			x.live.tick()
			e = releaseEntity(t, x.live, mage.ID)
		}
		if e.Transit != 0 {
			t.Fatal("the transit onto the layered cell never ended")
		}
		x.live.pending = append(x.live.pending, sim.MoveTo(mage.ID, sim.CellPoint{X: int32(mage.X), Y: int32(mage.Y)}))
		for i := 0; i < 10 && (e.X != mage.X || e.Y != mage.Y); i++ {
			x.live.tick()
			e = releaseEntity(t, x.live, mage.ID)
		}
		if e.X != mage.X || e.Y != mage.Y {
			t.Fatalf("the mage stands at (%d,%d) after the step back, want (%d,%d)", e.X, e.Y, mage.X, mage.Y)
		}
		return e.TransitTotal
	}
	order(h)
	fb, gb, _ := back(f), back(g), back(h)
	if fb != gb {
		t.Errorf("the step back takes %d in the continuing world and %d in the loaded one", fb, gb)
	}
	form := func(x *FrontEnd) []byte {
		b, err := x.live.world.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	if !bytes.Equal(form(f), form(g)) {
		t.Error("after the same orders the continuing and loaded worlds marshal to different bytes")
	}
	if bytes.Equal(form(f), form(h)) {
		t.Error("a world loaded from before the cast marshals like the layered one, so the comparison does not see the layer")
	}
}
