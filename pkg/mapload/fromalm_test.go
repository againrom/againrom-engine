// Package mapload_test exercises the loader from outside, over its exported
// surface and nothing else.
//
// Everything here is synthetic: the fixture is an alm.Map literal built in
// this file, so the suite reads no game install, opens no file, consults no
// clock and needs no window. The import block is the whole evidence for that
// claim.
package mapload_test

import (
	"bytes"
	"encoding/binary"
	"hash/fnv"
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// fixtureMap returns the fixture, built fresh on every call.
//
// It is a literal rather than bytes run through alm.Open: routing a known
// position through the decoder to recover the position it was written from adds
// a dependency whose failure would read as a loader defect. It carries what the
// transform reads — the extent and the units — and nothing else.
//
// THE EXTENT IS 72x68 AND EVERY PLACED UNIT STANDS IN THE INTERIOR, which is
// what leaves this a world in which anything can happen: a derived plane blocks
// the outer eight rings of every map, so a unit placed inside them holds its
// cell whatever it is ordered to do, and a test that then asserted motion would
// fail for a reason unrelated to what it measures. The two extents differ, so a
// world that swapped them is a different world; neither is a round number, so an
// off-by-one in either direction shows.
//
// The five positions are chosen so that the arithmetic cannot pass by accident:
//
//   - unit 0 sits at the centre of its cell, the usual 0x80 low byte;
//   - unit 1 is raw zero in both LOW bytes — the whole-cell case;
//   - unit 2 has low bytes 0x04 and 0xFF — 27+255/256 of a cell is still row 27,
//     so the fraction is dropped and not rounded;
//   - unit 3 is the far interior corner: one cell short of the ring on both
//     axes, so a border derived one cell deeper would seal it in;
//   - unit 4 is in the top half of the uint32 range, where the unsigned shift
//     and a signed division by 256 stop agreeing. Nothing constrains a unit to
//     the map's extent — bounds are recorded state, not a movement constraint —
//     so this loads like any other, and it is left OUTSIDE the extent
//     deliberately.
//
// The slice order matches neither ascending X nor row-major cell order, so ids
// taken from a sort rather than from the slice would not be the ids below.
//
// The class keys are assorted on the same principle. Unit 1's is negative — the
// sign-extension witness, since its 16-bit pattern zero-extended would be a
// value near 65000, not -24. Unit 2's names no unit-registry class — the domain
// is sparse over 1..80, and the loader consults no registry, so 200 must arrive
// as raw as any other key. No two keys are equal, so a loader that smeared one
// key across the slice would not produce the wants below.
func fixtureMap() *alm.Map {
	return &alm.Map{
		Width:  72,
		Height: 68,
		Units: []alm.Unit{
			{X: 0x1780, Y: 0x1580, ClassID: 7},
			{X: 0x1400, Y: 0x1400, ClassID: -24},
			{X: 0x2104, Y: 0x1BFF, ClassID: 200},
			{X: 0x3F80, Y: 0x3B80, ClassID: 80},
			{X: 0xFF000080, Y: 0x00000280, ClassID: 1},
		},
	}
}

// wantEntities is the world fixtureMap must load as, written out by hand.
//
// These cells and class ids are literals on purpose. Computing them as u.X>>8
// or int32(u.ClassID) would assert the loader's arithmetic against itself and
// would hold for any conversion the loader happened to use. Class -24 in
// particular: a loader that widened the record's 16-bit field unsigned would
// load 65512 here, and this literal is what catches it.
//
// The health pair is written out as 100 and 100 and not as mapload.SpawnHP, for
// the same reason: read off the constant, this table would agree with whatever
// that constant said, and a spawn health silently changed to 1 would pass.
func wantEntities() []sim.Entity {
	// Every placement here resolves to no units entry, so each takes the
	// provisional health pair, the constructor's own default speed AND the
	// constructor's own eight combat numbers — a charge of 8, a relax of 4 and
	// zero on the six a blow reads. All of them are written out here rather than
	// reached for, so a change to any default is a failure in this table and not
	// a silent re-rating or a silent re-arming.
	//
	// The six zeros are spelled by being ABSENT, which is the one thing this
	// table cannot say more loudly than by omission: an unresolved placement
	// hits for nothing, is hit by everything, and absorbs none of it.
	//
	// THE SIGHT RANGE IS THE CONSTRUCTOR'S 5 for the same reason as the speed: an
	// unresolved placement is handed the whole constructor definition and every
	// number it carries comes off that one value. It is written out here rather
	// than reached for, so the arm that would leave a placement blind is a
	// failure in this table.
	//
	// The DYING TIME is the constructor's too, and it comes off the substituted
	// definition with the speed and the cadence rather than being decided here.
	//
	// THE ACTOR STATE IS 11 ON ALL FIVE — sim.NewWorld's own guard value,
	// written as the literal rather than as a named constant because that
	// constant is unexported and this package reads sim from outside. It is
	// unconditional: the loader hands the constructor no placement that could
	// make it anything else, so every entity this table names comes back guard
	// regardless of what a later loader arm supplies.
	//
	// AND THE REACH IS 1 ON ALL FIVE, on the sight range's own ground rather
	// than the actor state's: FromALM passes no table at all, so no placement
	// here can reach the item collections 0104's join reads (T3) whatever its
	// class row would have named, and the constructor's own floor of 1 is what
	// comes back regardless. THE POST IS EVERY ENTITY'S OWN CELL: the world
	// constructor writes it unconditionally from X and Y, so an unresolved
	// placement's post is not a fifth default reached for here — it is the
	// same X and Y this table already names, repeated.
	//
	// AND THE TWO PERIODS ARE THE CONSTRUCTOR'S 100 AND 50 ON ALL FIVE: the
	// unresolved arm now substitutes the whole constructor definition, periods
	// included, in the same composite literal as the speed and the sight range
	// above — so a placement this table cannot resolve regenerates at the
	// base rate rather than never. THE MANA PAIR STAYS ABSENT: the
	// constructor's own maximum is zero, so it is a fifth pair of zeros spelled
	// the way the six combat zeros already are.
	//
	// AND MIND IS THE CONSTRUCTOR'S 20 ON ALL FIVE: the same substituted
	// definition names it, off the one field this table has not yet had reason
	// to spell. THE EXPERIENCE VALUE, THE CREDITED SLOT AND THE SIX SLOT
	// EXPERIENCES STAY ABSENT: the constructor's own value is zero, the
	// constructor's own combat carries no weapon-driven skill, and nothing this
	// loader builds has earned anything yet, so all eight are a sixth run of
	// zeros spelled the way the others already are. THE FLAG IS ABSENT TOO, and
	// for a different reason than a zero: an unresolved placement does not
	// gain, so `false` is what this table already says about it by never
	// writing `GainsXP: true`.
	return []sim.Entity{
		{ID: 0, X: 23, Y: 21, Class: 7, HP: 100, MaxHP: 100, Speed: 10, ScanRange: 5, AttackCharge: 8, AttackRelax: 4, DyingTime: 8, ActorState: 11, Reach: 1, PostX: 23, PostY: 21, HealthRegenPeriod: 100, ManaRegenPeriod: 50, RotationSpeed: 16, Reaction: 30, Mind: 20, Spirit: 20, Capacity: 300},
		{ID: 1, X: 20, Y: 20, Class: -24, HP: 100, MaxHP: 100, Speed: 10, ScanRange: 5, AttackCharge: 8, AttackRelax: 4, DyingTime: 8, ActorState: 11, Reach: 1, PostX: 20, PostY: 20, HealthRegenPeriod: 100, ManaRegenPeriod: 50, RotationSpeed: 16, Reaction: 30, Mind: 20, Spirit: 20, Capacity: 300},
		{ID: 2, X: 33, Y: 27, Class: 200, HP: 100, MaxHP: 100, Speed: 10, ScanRange: 5, AttackCharge: 8, AttackRelax: 4, DyingTime: 8, ActorState: 11, Reach: 1, PostX: 33, PostY: 27, HealthRegenPeriod: 100, ManaRegenPeriod: 50, RotationSpeed: 16, Reaction: 30, Mind: 20, Spirit: 20, Capacity: 300},
		{ID: 3, X: 63, Y: 59, Class: 80, HP: 100, MaxHP: 100, Speed: 10, ScanRange: 5, AttackCharge: 8, AttackRelax: 4, DyingTime: 8, ActorState: 11, Reach: 1, PostX: 63, PostY: 59, HealthRegenPeriod: 100, ManaRegenPeriod: 50, RotationSpeed: 16, Reaction: 30, Mind: 20, Spirit: 20, Capacity: 300},
		{ID: 4, X: 16711680, Y: 2, Class: 1, HP: 100, MaxHP: 100, Speed: 10, ScanRange: 5, AttackCharge: 8, AttackRelax: 4, DyingTime: 8, ActorState: 11, Reach: 1, PostX: 16711680, PostY: 2, HealthRegenPeriod: 100, ManaRegenPeriod: 50, RotationSpeed: 16, Reaction: 30, Mind: 20, Spirit: 20, Capacity: 300},
	}
}

// wantBounds is the fixture's extent, likewise by hand. Width and Height differ,
// so a world that swapped them is a different world.
var wantBounds = sim.Bounds{Width: 72, Height: 68}

// wantGrid is the plane the fixture map must load with, WRITTEN FROM THE
// CONTRACT and never taken from the derivation — read from a call to it, the pin
// below would be an agreement rather than a pin.
func wantGrid() []byte {
	w, h := int(wantBounds.Width), int(wantBounds.Height)
	g := make([]byte, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x < 8 || y < 8 || x >= w-8 || y >= h-8 {
				g[y*w+x] = 0x03
			}
		}
	}
	return g
}

// gridSection is the grid a world's canonical byte form carries, read straight
// out of the bytes at the offsets the form documents: a cell count at +30 and
// that many cells from +34.
func gridSection(t *testing.T, w *sim.World) []byte {
	t.Helper()
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	const header = 34
	if len(b) < header {
		t.Fatalf("the byte form is %d byte(s), shorter than its own %d-byte header", len(b), header)
	}
	n := int(binary.LittleEndian.Uint32(b[30:34]))
	if len(b) < header+n {
		t.Fatalf("the byte form declares %d grid cell(s) and carries %d byte(s) after its header",
			n, len(b)-header)
	}
	return b[header : header+n]
}

// TestFixtureMapIsBuiltFresh checks the instrument, not the code under test. An
// alm.Map holds slices, so if two calls shared one units array, the
// "the map is unchanged" comparison below would be comparing a map against
// itself and would pass whatever the loader did to it.
func TestFixtureMapIsBuiltFresh(t *testing.T) {
	a, b := fixtureMap(), fixtureMap()
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("two builds of the fixture differ:\n a = %+v\n b = %+v", a, b)
	}
	a.Units[0].X = 0xDEAD
	a.Width = 1
	if b.Units[0].X == 0xDEAD || b.Width == 1 {
		t.Fatal("the two fixture maps share storage; every comparison against a second build is vacuous")
	}
}

// TestFromALMBuildsOneEntityPerUnit is AC-8's first half: N entities at the
// shifted cells, ids 0..N-1 in the map's slice order, bounds from the map.
func TestFromALMBuildsOneEntityPerUnit(t *testing.T) {
	w := mapload.FromALM(fixtureMap())

	if got, want := w.Entities(), wantEntities(); !reflect.DeepEqual(got, want) {
		t.Errorf("entities:\n got %+v\nwant %+v", got, want)
	}
	if got := w.Bounds(); got != wantBounds {
		t.Errorf("bounds: got %+v, want %+v", got, wantBounds)
	}
	if got := w.Tick(); got != 0 {
		t.Errorf("tick: got %d, want a world that has not advanced", got)
	}
}

// TestFromALMIsIdenticalAcrossLoads is AC-8's "identical across both loads",
// over two independently built maps rather than one map loaded twice: the
// loader must depend on the map's contents and on nothing it carried over from
// the last call.
func TestFromALMIsIdenticalAcrossLoads(t *testing.T) {
	w1 := mapload.FromALM(fixtureMap())
	w2 := mapload.FromALM(fixtureMap())

	if !reflect.DeepEqual(w1.Entities(), w2.Entities()) {
		t.Errorf("two loads differ:\n %+v\n %+v", w1.Entities(), w2.Entities())
	}
	if w1.Hash() != w2.Hash() {
		t.Errorf("two loads hash differently: %#x vs %#x", w1.Hash(), w2.Hash())
	}
}

// The expected seed, grid, entities and current Player are independent inputs.
// The digest distinguishes another seed and an absent grid.
func TestFromALMSeedsFromTheConstant(t *testing.T) {
	got := mapload.FromALM(fixtureMap())

	want, err := sim.NewWorld(mapload.Seed, wantBounds, sim.ModeCanonical, wantGrid(), wantEntities())
	if err != nil {
		t.Fatalf("NewWorld over the expected entities: %v", err)
	}
	if err := want.RestoreCurrentPlayers([]sim.SavedGroupPlayer{{ID: 1, Slot: 0}}, []sim.PlayerParticipant{{PlayerID: 1, Value: 1}}, true); err != nil {
		t.Fatal(err)
	}
	if got.Hash() != want.Hash() {
		t.Errorf("loaded world hashes %#x, want %#x", got.Hash(), want.Hash())
	}

	other, err := sim.NewWorld(mapload.Seed+1, wantBounds, sim.ModeCanonical, wantGrid(), wantEntities())
	if err != nil {
		t.Fatalf("NewWorld over the expected entities: %v", err)
	}
	if got.Hash() == other.Hash() {
		t.Fatal("a world seeded differently hashes the same; this test cannot see the seed at all")
	}

	// ...and a world over NO grid is a different world, so the equality above is
	// a statement about the derived plane and not one the grid falls out of.
	flat, err := sim.NewWorld(mapload.Seed, wantBounds, sim.ModeCanonical, nil, wantEntities())
	if err != nil {
		t.Fatalf("NewWorld over the expected entities: %v", err)
	}
	if got.Hash() == flat.Hash() {
		t.Fatal("the loaded world hashes as one built over no grid at all; the loader passed none")
	}
}

// TestEveryMapBuiltUnitIsBornAtTheSpawnHealth is AC-9's first half. The pair is
// checked on EVERY unit of the loaded world rather than on the table as a whole,
// so a loader that filled the first record and left the rest at zero names the
// unit it forgot; and it is checked against the exported constant here, where
// the claim is that both fields come from that one number, having been checked
// against literals in the table above, where the claim is what the number is.
//
// A spawned unit is ALIVE and has a health system: the two together are what
// make it damageable at all, so both are asserted rather than the pair's values
// alone.
func TestEveryMapBuiltUnitIsBornAtTheSpawnHealth(t *testing.T) {
	w := mapload.FromALM(fixtureMap())
	ents := w.Entities()
	if len(ents) != len(fixtureMap().Units) {
		t.Fatalf("the loaded world holds %d entities for %d units", len(ents), len(fixtureMap().Units))
	}
	for _, e := range ents {
		if e.HP != mapload.SpawnHP || e.MaxHP != mapload.SpawnHP {
			t.Errorf("unit %d is born at %d/%d, want %d/%d — both fields come from the one constant",
				e.ID, e.HP, e.MaxHP, mapload.SpawnHP, mapload.SpawnHP)
		}
		if !e.Alive() || e.Dead() || e.Downed() {
			t.Errorf("unit %d is born at %d/%d and is not alive", e.ID, e.HP, e.MaxHP)
		}
	}
	if mapload.SpawnHP <= 0 {
		t.Errorf("SpawnHP is %d; a unit born with no positive maximum has no health system at all",
			mapload.SpawnHP)
	}
}

func TestNoOtherWayOfBuildingAWorldSetsEitherHealthField(t *testing.T) {
	given := []sim.Entity{
		{ID: 0},
		{ID: 1, HP: 3, MaxHP: 7},
		{ID: 2, HP: -400, MaxHP: 100},
	}
	w, err := sim.NewWorld(1, wantBounds, sim.ModeCanonical, nil, given)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	for i, e := range w.Entities() {
		if e.HP != given[i].HP || e.MaxHP != given[i].MaxHP {
			t.Errorf("a hand-built unit %d came back at %d/%d, want the %d/%d it was given",
				e.ID, e.HP, e.MaxHP, given[i].HP, given[i].MaxHP)
		}
	}
}

func TestSteppingALoadedWorldLeavesTheMapUnchanged(t *testing.T) {
	m := fixtureMap()
	untouched := fixtureMap()

	w := mapload.FromALM(m)
	for i := 0; i < 4; i++ {
		sim.Step(w, []sim.Command{
			{Entity: 0, X: 9, Y: 9},
			{Entity: 3, X: 8, Y: 8},
			{Entity: 4, X: 16711680, Y: 20},
		})
	}

	// Guard against a vacuous pass: if the world had not moved, an aliasing
	// loader would have had nothing to write through. It is stated over a moved
	// CELL and not over a moved record — a target field written on a unit that
	// then held its cell would satisfy a whole-entity comparison while nothing
	// walked anywhere, which is exactly the shape a blocked fixture fails in.
	moved := 0
	for i, e := range w.Entities() {
		if e.X != wantEntities()[i].X || e.Y != wantEntities()[i].Y {
			moved++
		}
	}
	if moved == 0 {
		t.Fatalf("no entity left its start cell in four ticks (%+v), so the map could not have been "+
			"written through either", w.Entities())
	}
	if !reflect.DeepEqual(m, untouched) {
		t.Errorf("the map changed under a stepped world:\n got %+v\nwant %+v", m, untouched)
	}
}

// TestTheWorldCarriesTheDerivedBytesAndOnlyThisPathDerivesOne is AC-8 (SC-6),
// read through the byte form on all four counts.
//
// The three build paths are the three there are, and each must answer for
// itself: a world from a decoded map carries the derived plane, and a world
// assembled by hand carries EXACTLY what it was handed — over a grid, and over
// none. That last one is the negative that matters. Were a grid derived or
// defaulted anywhere below the loader, every hand-built world in the tree would
// change state and digest and every byte-form pin would have moved for a reason
// unrelated to this contract.
func TestTheWorldCarriesTheDerivedBytesAndOnlyThisPathDerivesOne(t *testing.T) {
	loaded := mapload.FromALM(fixtureMap())
	if got := gridSection(t, loaded); !bytes.Equal(got, wantGrid()) {
		t.Errorf("the loaded world's grid section is %d byte(s) and differs from the plane the contract "+
			"describes; first difference at cell %d", len(got), firstDifference(got, wantGrid()))
	}

	// A hand-built world over a grid of its own: no cell of it agrees with the
	// derived plane by accident, since the derived plane is 0x03 on the ring and
	// this one is 0x01 there and 0x02 in the interior.
	given := make([]byte, wantBounds.Width*wantBounds.Height)
	for i := range given {
		given[i] = byte(1 + i%2)
	}
	overGrid, err := sim.NewWorld(mapload.Seed, wantBounds, sim.ModeCanonical, given, wantEntities())
	if err != nil {
		t.Fatalf("NewWorld over a hand-built grid: %v", err)
	}
	if got := gridSection(t, overGrid); !bytes.Equal(got, given) {
		t.Errorf("a world built over a hand-written grid came back with a different one; first difference "+
			"at cell %d", firstDifference(got, given))
	}

	// ...and one over none: gridCells zero bytes, and not a plane anybody derived.
	overNone, err := sim.NewWorld(mapload.Seed, wantBounds, sim.ModeCanonical, nil, wantEntities())
	if err != nil {
		t.Fatalf("NewWorld over no grid: %v", err)
	}
	zeroes := make([]byte, wantBounds.Width*wantBounds.Height)
	if got := gridSection(t, overNone); !bytes.Equal(got, zeroes) {
		t.Errorf("a world built over NO grid carries a nonzero plane; first difference at cell %d — "+
			"something below the loader derives or defaults a grid", firstDifference(got, zeroes))
	}

	// The fourth map: everything but the extent, the tile plane and the overlay
	// plane is different, and the plane is the same.
	other := fixtureMap()
	other.Name = "another map entirely"
	other.Description = "different in every section a grid does not read"
	other.Altitudes = make([]uint8, wantBounds.Width*wantBounds.Height)
	for i := range other.Altitudes {
		other.Altitudes[i] = uint8(i)
	}
	other.Objects = []alm.Object{{X: 0x1400, Y: 0x1400, Kind: 0x21}}
	other.Triggers = alm.Triggers{EntryCount: 5, Body: []byte{1, 2, 3}}
	other.Groups = []alm.Group{{Scalar: 5000, Name: "g"}}
	other.Units = append(other.Units, alm.Unit{X: 0x1500, Y: 0x1500, ClassID: 12})

	fourth := mapload.FromALM(other)
	if got, want := gridSection(t, fourth), gridSection(t, loaded); !bytes.Equal(got, want) {
		t.Errorf("a map differing in altitudes, units, objects, triggers and name derives a different "+
			"plane; first difference at cell %d", firstDifference(got, want))
	}
	// ...and the two worlds are not simply equal: the added unit is there, so
	// the equality above is about the grid section and not about two identical
	// forms.
	if fourth.Hash() == loaded.Hash() {
		t.Error("the fourth world hashes as the first; the sections that were meant to differ did not")
	}
}

// firstDifference is where two planes part, or -1 when they do not. It names a
// cell rather than dumping thousands of bytes into a failure message.
func firstDifference(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	if len(a) != len(b) {
		return n
	}
	return -1
}

// TestEveryMapShapeBuildsAUsableWorld is AC-10's world half (SC-6): each
// degenerate shape its own case, each built into a world, and each world shown
// to be one a caller can actually use — it marshals, it reads back to the same
// bytes, and it takes a tick without raising.
//
// "No error and no panic" is not a claim a test can make by observing a return
// value, since this signature has neither; what it can do is exercise the world
// that comes back, which is what a caller would have done with it.
func TestEveryMapShapeBuildsAUsableWorld(t *testing.T) {
	full := make([]uint16, 24*24)
	full[10*24+10] = 0x2000 // bit 13, on an interior cell of a 24x24 map

	cases := []struct {
		what  string
		m     *alm.Map
		want  sim.Bounds
		cells int
	}{
		{"no map at all", nil, sim.Bounds{}, 0},
		{"a map with no tile plane", &alm.Map{Width: 24, Height: 24}, sim.Bounds{Width: 24, Height: 24}, 576},
		{"a tile plane shorter than the extent",
			&alm.Map{Width: 24, Height: 24, Tiles: full[:300]}, sim.Bounds{Width: 24, Height: 24}, 576},
		{"a tile plane longer than the extent",
			&alm.Map{Width: 24, Height: 24, Tiles: append(append([]uint16(nil), full...), full...)},
			sim.Bounds{Width: 24, Height: 24}, 576},
		{"no overlay plane",
			&alm.Map{Width: 24, Height: 24, Tiles: full}, sim.Bounds{Width: 24, Height: 24}, 576},
		{"a width of 0", &alm.Map{Width: 0, Height: 24}, sim.Bounds{Width: 0, Height: 24}, 0},
		{"a negative height", &alm.Map{Width: 24, Height: -1}, sim.Bounds{Width: 24, Height: -1}, 0},
	}

	for _, c := range cases {
		t.Run(c.what, func(t *testing.T) {
			w := mapload.FromALM(c.m)
			if got := w.Bounds(); got != c.want {
				t.Fatalf("bounds %+v, want %+v", got, c.want)
			}
			if got := gridSection(t, w); len(got) != c.cells {
				t.Fatalf("the grid section is %d byte(s), want %d", len(got), c.cells)
			}

			form, err := w.MarshalBinary()
			if err != nil {
				t.Fatalf("MarshalBinary: %v", err)
			}
			var back sim.World
			if err := back.UnmarshalBinary(form); err != nil {
				t.Fatalf("the world this map built does not read back: %v", err)
			}
			if again, err := back.MarshalBinary(); err != nil || !bytes.Equal(again, form) {
				t.Fatalf("the round trip changed the form (err %v)", err)
			}

			// A tick over it, with an order for an entity no such world holds:
			// the shapes above place none, so this is the caller's ordinary use
			// of a world and it must simply advance.
			sim.Step(w, []sim.Command{{Entity: 0, X: 12, Y: 12}})
			if got := w.Tick(); got != 1 {
				t.Fatalf("the world stands at tick %d after one step, want 1", got)
			}
		})
	}
}

// preStoryDigest and preStoryFormLength are FromALM(fixtureMap())'s digest and
// byte-form length, recorded and pasted here as literals.
//
// They were RECORDED BEFORE the definition table reached this package, at
// 0x6d7c57873172163c over 5125 bytes, and that recording is what said the
// single-argument entry point had not moved when it was re-expressed through
// the new one. **0056 moved both deliberately**: the movement rate widened
// every entity record by eight bytes, which is the whole of the 40 here, and
// then gave every placement a speed, which is the whole of the digest change.
// **0059 moved both again, and only the form's width deliberately**: the group
// rate term widened every record by ONE byte, which is the whole of the 5 here,
// and raised the version byte, which is the whole of the digest change — this
// loader writes no group term, so every one of the five is zero.
//
// They are RE-RECORDED at that story rather than derived, exactly as they were
// first taken, and what they still do is the same: a later change that moves
// either fails here instead of moving a save silently. What they no longer do
// is reach back past 0059 — the numbers before it are in this comment, which is
// where a reader can find them. 0056's pair was 0x30095fbf4808477d over 5165
// bytes.
//
// **0063 moved both a third time, and neither deliberately**: the mission
// script's section is appended to every form and the version byte rises to 9.
// This loader compiles no script, so the section is 1421 zeros — which is the
// whole of the 1421 here — and the version byte is the whole of the digest
// change. Both were checked rather than merely re-recorded: stripping the 1421
// bytes, confirming them zero and putting byte 0 back to 8 reproduces the
// version-8 digest below exactly. 0059's pair was 0x8f2d690d4655f1ae over 5170
// bytes.
//
// **0064 moved both a fourth time, and neither deliberately either**: the attack
// cycle widens every entity record by thirty-nine bytes and the version byte
// rises to 10. This loader gives a placement no combat number and issues no
// order, so all thirty-nine are zero on each of this fixture's FIVE records —
// 5 x 39 = 195, which is the whole of the 195 here — and the version byte is the
// whole of the digest change. Checked by 0063's own recipe rather than
// re-recorded: lifting the thirty-nine out of every record, confirming each run
// is zero and putting byte 0 back to 9 reproduces 0xf852d97fb442c9a9 exactly,
// which was the version-9 pair's digest over 6591 bytes.
//
// **0067 moves the DIGEST a fifth time and the LENGTH not at all**, and this
// one IS deliberate: a placement now carries the eight numbers a fight
// reads, and every placement on this fixture resolves to nothing, so each of
// the five records takes the constructor's charge of 8 and relax of 4 and
// keeps zero on the other six.
//
// The new value was DERIVED, not recorded. Taking the pre-story form, writing 8
// and 4 into each record's own two cadence slots and hashing the result yields
// the literal below; a number pasted out of a run of the new loader would only
// have said the new loader agrees with itself. zeroedEightDigest re-derives it in
// the other direction on every run, from the code as it now stands.
// 0064's pair was 0x2f029f5557deeb6a over 6786 bytes.
const (
	// 1047 appends the inactive turn pair (desired facing repeating the
	// current facing, remaining progress zero) to every record and moves
	// the version byte to 63. 1045's pair was 0x4890714c41b23c96.
	preStoryDigest     uint64 = 0x1534e82402777f6a
	preStoryFormLength        = 6786

	// zeroedPreStoryDigest is what preStoryDigest's own form hashes to once the
	// eight fields' bytes are lifted out of every record: the digest this
	// fixture carried before this story, exactly. It is the second, independent
	// derivation R-1 asks for, and it is a literal that PREDATES this story's
	// code — so the two pins can only agree if this story's bytes are the eight
	// fields' bytes and nothing else moved.
	//
	// 1047 adds the inactive turn pair to every record outside these eight
	// fields, so this value moves too. 1045's value was 0xf67e4ed3c7a472c2.
	zeroedPreStoryDigest uint64 = 0x655d10358e5dc106
)

// TestTheSingleArgumentEntryPointIsUnmoved is AC-7's first half and SC-7's.
func TestTheSingleArgumentEntryPointIsUnmoved(t *testing.T) {
	w := mapload.FromALM(fixtureMap())

	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	// The group story DID add a field and so did the owner story, so the
	// pre-story digest and length are no longer this world's — they are what this
	// world becomes when both words are lifted back out of every record and the
	// version byte put back. Asserted that way round, the two literals stay older
	// than all four stories.
	if want := preStoryFormLength + len(w.Entities())*(10+decayBlockLen+actorStateTailLen+1+postLen+regenLen+commandGroupLen+experienceLen+knownSpellsLen+skillBlockLen+weaponSpellLen+spellStateLen+offMapLen+lastArmsLen+spellEffectsLen+corpseLootLen+itemAttributionLen+carriedWeightLen+mapUnitIDLen+weaponResistanceLen+withdrawalThresholdLen+1+2+1+32+15+113+2+1+35) + 2*preStoryCells +
		formRelationLen + groupSectionCountLen + sackSectionCountLen +
		len(w.Entities())*carryCountLen + len(w.Entities())*equipRecordLen + len(w.Entities())*16 + purseLen + spellCountLen + itemWeightCountLen + 2*castingCountLen +
		relationSlotsInForm + cellTailCountLen + structureCountLen + 8 + 189*len(w.Entities()) + 4 + 4 + 4 + 4 + 5 + 4 + 4 + absentSavedPlayerFooterLen + absentNativeStrideFooterLen + absentSavedMotionFooterLen + absentSavedCellFooterLen + absentSavedObjectsFooterLen + absentCarriedResumeFooterLen + 28 + entityIDFloorFooterLen + 4 + 26; len(form) != want {
		t.Errorf("the byte form is %d byte(s), want the pre-story %d plus every record's own growth, "+
			"two more planes, the relation, the empty group section, the empty sack section, the "+
			"empty carry section, the empty equipment section, the purse section this fixture's "+
			"ownerless units write, the empty spell table, this story's own twenty-four-byte skill "+
			"block (0135) and the six-byte weapon-spell tail this fixture's placements resolve to "+
			"no spell (0139), the five-byte autocast and mark tail (0154), 0166's own seven "+
			"bytes a record and its script-state section, the structure section's own "+
			"four-byte zero count (1033 B3), absent span footers and one current Player",
			len(form), want)
	}
	if got := strippedOfTheOwnerAndGroupWords(t, form, len(w.Structures())); got != preStoryDigest {
		t.Errorf("FromALM with its owner and group words lifted out hashes %#016x, want the "+
			"pre-story %#016x — a byte moved outside those two words", got, preStoryDigest)
	}
	if got, want := w.Entities(), wantEntities(); !reflect.DeepEqual(got, want) {
		t.Errorf("entities:\n got %+v\nwant %+v", got, want)
	}

	// And the two entry points are ONE implementation, not two that agree today.
	with, err := mapload.FromALMWith(fixtureMap(), nil, mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("FromALMWith over no table: %v", err)
	}
	if with.Hash() != w.Hash() {
		t.Errorf("the two entry points disagree over no table: %#016x vs %#016x", with.Hash(), w.Hash())
	}
}

// fixtureTable resolves two of the fixture map's five placements.
//
// The keys are the fixture's own: 200 and 80 are at or above the class-key floor
// and are themselves. The other three carry keys -24, 7 and 1, all below the
// floor and so all in the humans band — 7 reaches a humans entry that EXISTS, the
// other two reach nothing, so both shapes of "no stat block" are on the map at
// once. The k232 units row stays: nothing may reach it, which is the assertion.
func fixtureTable() *mapload.Table {
	return &mapload.Table{
		Units: defCollection{
			{},
			{name: "k232", params: unitDefRow(232, 0, 40)},
			{name: "k200", params: unitDefRow(200, 0, 99)},
			{name: "k80", params: unitDefRow(80, 0, 1)},
		},
		Humans: defCollection{
			{},
			{name: "k7", params: defRow(map[int]int32{slotHumanType: 7})},
		},
	}
}

// TestAWorldBuiltWithATableCarriesTheResolvedHealth is AC-7's second half: the
// resolved placements take their definition's ADJUSTED maximum on both fields
// and every other placement keeps the provisional pair — and the worlds differ
// in nothing else.
func TestAWorldBuiltWithATableCarriesTheResolvedHealth(t *testing.T) {
	// Written out by hand, per difficulty, for the placements that resolve. Id 4
	// is absent because it must not move.
	//
	// ID 0 IS A PERSON and 0087 put it here. If this fixture's own figure
	// moved, a change in the definition tier's own defaults moved it, not this
	// test.
	for _, tc := range []struct {
		name string
		diff mapload.Difficulty
		want map[sim.EntityID]int32
	}{
		{"easy", mapload.DifficultyEasy, map[sim.EntityID]int32{0: 70, 2: 65, 3: 0}},
		{"normal", mapload.DifficultyNormal, map[sim.EntityID]int32{0: 70, 2: 99, 3: 1}},
		{"hard", mapload.DifficultyHard, map[sim.EntityID]int32{0: 70, 2: 148, 3: 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := mapload.FromALMWith(fixtureMap(), fixtureTable(), tc.diff)
			if err != nil {
				t.Fatalf("FromALMWith: %v", err)
			}
			plain := mapload.FromALM(fixtureMap())

			ents, base := got.Entities(), plain.Entities()
			if len(ents) != len(base) {
				t.Fatalf("the table changed the entity count: %d vs %d", len(ents), len(base))
			}
			for i := range ents {
				want := mapload.SpawnHP
				_, resolved := tc.want[ents[i].ID]
				if resolved {
					want = tc.want[ents[i].ID]
				}
				if ents[i].HP != want || ents[i].MaxHP != want {
					t.Errorf("entity %d is at %d/%d, want %d/%d",
						ents[i].ID, ents[i].HP, ents[i].MaxHP, want, want)
				}
				// The to-hit and the defence, per placement, ASSERTED BEFORE
				// being neutralised below so the pair is checked rather than
				// excused. Three populations and three answers:
				//
				//   - a CREATURE carries its row's columns, which this fixture
				//     leaves at the constructor's zero, plus the hard setting's
				//     own constant;
				//   - a PERSON carries the derivation's, off statistics this row
				//     leaves at the constructor's 30 each: to-hit is
				//     ftol((1.1^30 + 1.1^30)/5) = 6 and defence is 30/3 = 10,
				//     BOTH WORKED OUT HERE rather than read back off the tier
				//     that computes them, and both unmoved by the setting;
				//   - anything unresolved carries nothing.
				wantToHit, wantDefence := int32(0), int32(0)
				// wantGainsXP is 0125's own third population, ASSERTED BEFORE being
				// neutralised below on the pair's own principle: a PERSON gains and
				// neither a CREATURE nor anything unresolved does, whatever the setting
				// — the flag is not one of the eight difficulty can reach, so it does
				// not vary with tc.diff the way to-hit and defence's hard case does.
				wantGainsXP := false
				switch {
				case ents[i].ID == 0:
					wantToHit, wantDefence = 6, 10
				case resolved && tc.diff == mapload.DifficultyHard:
					wantToHit, wantDefence = 50, 50
				}
				if ents[i].ToHit != wantToHit || ents[i].Defence != wantDefence {
					t.Errorf("entity %d carries to-hit %d and defence %d, want %d and %d",
						ents[i].ID, ents[i].ToHit, ents[i].Defence, wantToHit, wantDefence)
				}
				if ents[i].GainsXP != wantGainsXP {
					t.Errorf("entity %d gains experience: %v, want %v", ents[i].ID, ents[i].GainsXP, wantGainsXP)
				}
				wantProtection := [5]int32{}
				wantTokenSize := uint8(0)
				if resolved {
					wantTokenSize = 1
				}
				if ents[i].ID == 0 {
					wantProtection = [5]int32{10, 10, 10, 10, 10}
				}
				if ents[i].Protection != wantProtection || ents[i].TokenSize != wantTokenSize {
					t.Errorf("entity %d carries protection %v and token size %d, want %v and %d",
						ents[i].ID, ents[i].Protection, ents[i].TokenSize, wantProtection, wantTokenSize)
				}
				// Carrying capacity: a PERSON carries body x 10 + 1 off the
				// constructor default of 30, 301; a CREATURE, resolved or not, the
				// constructor's body x 10, 300.
				wantCapacity := int32(300)
				if ents[i].ID == 0 {
					wantCapacity = 301
				}
				if ents[i].Capacity != wantCapacity {
					t.Errorf("entity %d carries capacity %d, want %d", ents[i].ID, ents[i].Capacity, wantCapacity)
				}
				wantHumanoid := ents[i].ID == 0
				if ents[i].Humanoid != wantHumanoid {
					t.Errorf("entity %d humanoid = %v, want %v from its resolved definition kind",
						ents[i].ID, ents[i].Humanoid, wantHumanoid)
				}
				wantTurnRate := base[i].RotationSpeed
				if wantHumanoid {
					wantTurnRate = 18
				}
				if ents[i].RotationSpeed != wantTurnRate {
					t.Errorf("entity %d turn rate %d, want %d", ents[i].ID, ents[i].RotationSpeed, wantTurnRate)
				}
				if ents[i].NativeTraining.Present != wantHumanoid || ents[i].NativeTraining.Levels != ([data.SkillSlots]int32{}) {
					t.Fatalf("entity %d native training = %+v", ents[i].ID, ents[i].NativeTraining)
				}
				class := sim.NativeClass{}
				if wantHumanoid {
					class = sim.NativeClass{Present: true, Fighter: true}
				}
				if ents[i].NativeClass != class {
					t.Fatalf("entity %d native class = %+v, want %+v", ents[i].ID, ents[i].NativeClass, class)
				}
				// Everything but the health pair, the adjusted pair and the flag must be
				// untouched — the cadence, the damage pair, the absorption and the
				// mark included, so a setting that reached one of those would fail here.
				a, b := ents[i], base[i]
				if resolved {
					if !a.NativeBasis.BodyPresent || !a.NativeBasis.BodyKnown || a.NativeBasis.Body != 30 {
						t.Fatal("resolved constructor Body was not captured", a.ID, a.NativeBasis)
					}
				} else if a.NativeBasis.HasValues() {
					t.Fatal("unresolved constructor fabricated native basis", a.ID, a.NativeBasis)
				}
				a.NativeBasis, b.NativeBasis = sim.NativeActorBasis{}, sim.NativeActorBasis{}
				a.NativeTraining, b.NativeTraining = sim.NativeTraining{}, sim.NativeTraining{}
				a.NativeClass, b.NativeClass = sim.NativeClass{}, sim.NativeClass{}
				a.HP, a.MaxHP, b.HP, b.MaxHP = 0, 0, 0, 0
				a.ToHit, a.Defence, b.ToHit, b.Defence = 0, 0, 0, 0
				a.GainsXP, b.GainsXP = false, false
				a.TypeID, b.TypeID = 0, 0
				a.GoldChance, b.GoldChance = 0, 0
				a.TreasureMin, b.TreasureMin = 0, 0
				a.TreasureMax, b.TreasureMax = 0, 0
				a.Protection, b.Protection = [5]int32{}, [5]int32{}
				a.TokenSize, b.TokenSize = 0, 0
				a.Capacity, b.Capacity = 0, 0
				a.Humanoid, b.Humanoid = false, false
				a.RotationSpeed, b.RotationSpeed = 0, 0
				if a != b {
					t.Errorf("entity %d differs outside the health pair, the adjusted pair and the flag:\n got %+v\nwant %+v",
						ents[i].ID, a, b)
				}
			}
			if got.Bounds() != plain.Bounds() {
				t.Errorf("the table moved the bounds: %+v vs %+v", got.Bounds(), plain.Bounds())
			}
			if got.Hash() == plain.Hash() {
				t.Error("the table changed no digest at all; nothing resolved")
			}
		})
	}
}

func TestAuthoredCurrentHealthIsSignedUnscaledAndIndependentOfResolution(t *testing.T) {
	for _, tc := range []struct {
		name        string
		diff        mapload.Difficulty
		creatureMax int32
		creatureDef int32
	}{
		{"easy", mapload.DifficultyEasy, 65, 0},
		{"normal", mapload.DifficultyNormal, 99, 0},
		{"hard", mapload.DifficultyHard, 148, 25},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := fixtureMap()
			// A resolved human, a resolved creature and an unresolved low-key
			// placement exercise all three resolution populations. The positive
			// value deliberately exceeds the unresolved maximum.
			m.Units[0].CurrentHP, m.Units[0].HasCurrentHP = 0, true
			m.Units[2].CurrentHP, m.Units[2].HasCurrentHP = -10, true
			m.Units[1].CurrentHP, m.Units[1].HasCurrentHP = 321, true

			w, err := mapload.FromALMWith(m, fixtureTable(), tc.diff)
			if err != nil {
				t.Fatalf("FromALMWith: %v", err)
			}
			ents := w.Entities()
			human, unresolved, creature := ents[0], ents[1], ents[2]
			if human.HP != 0 || human.MaxHP != 70 || human.Decay != sim.DecayFallen || human.Defence != 5 {
				t.Errorf("human = HP %d/%d decay %d defence %d, want 0/70 fallen/5",
					human.HP, human.MaxHP, human.Decay, human.Defence)
			}
			if creature.HP != -10 || creature.MaxHP != tc.creatureMax ||
				creature.Decay != sim.DecayFallen || creature.Defence != tc.creatureDef {
				t.Errorf("creature = HP %d/%d decay %d defence %d, want -10/%d fallen/%d",
					creature.HP, creature.MaxHP, creature.Decay, creature.Defence,
					tc.creatureMax, tc.creatureDef)
			}
			if unresolved.HP != 321 || unresolved.MaxHP != mapload.SpawnHP || !unresolved.Alive() {
				t.Errorf("unresolved = HP %d/%d alive %v, want authored 321/%d alive",
					unresolved.HP, unresolved.MaxHP, unresolved.Alive(), mapload.SpawnHP)
			}
		})
	}
}

// The slot numbers 0109 T3 reads, spelled out here for the reason
// slotHealthMax and its neighbours (spawn_test.go) already are: a test
// asserting that a column moved a placement must not read that column's
// position out of the code it is testing.
const (
	slotUnitHealthRegenPeriod = 5
	slotUnitManaMax           = 6
	slotUnitManaRegenPeriod   = 7
	slotHumanManaMax          = 5
)

func TestAResolvedUnitsPlacementCarriesItsRowsPeriodsAndPool(t *testing.T) {
	m := &alm.Map{Width: 10, Height: 10,
		Units: []alm.Unit{{X: 5 << 8, Y: 5 << 8, ClassID: 90}}}
	table := &mapload.Table{
		Units: defCollection{
			{},
			{name: "k90", params: defRow(map[int]int32{
				slotUnitType: 90, slotUnitFace: 0, slotHealthMax: 40,
				slotUnitHealthRegenPeriod: 7, slotUnitManaMax: 60, slotUnitManaRegenPeriod: 13,
			})},
		},
	}
	w, err := mapload.FromALMWith(m, table, mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}
	e := w.Entities()[0]
	if e.HealthRegenPeriod != 7 || e.ManaRegenPeriod != 13 {
		t.Errorf("entity carries periods %d/%d, want the row's own 7/13", e.HealthRegenPeriod, e.ManaRegenPeriod)
	}
	if e.Mana != 60 || e.MaxMana != 60 {
		t.Errorf("entity carries mana %d/%d, want the row's own 60/60", e.Mana, e.MaxMana)
	}
}

func TestTheHumansArmCarriesTheConstructorsPeriodsAndTheGraphsOwnManaMaximum(t *testing.T) {
	m := &alm.Map{Width: 10, Height: 10,
		Units: []alm.Unit{{X: 5 << 8, Y: 5 << 8, ClassID: 9}}}
	table := &mapload.Table{
		Humans: defCollection{
			{},
			{name: "h9", params: defRow(map[int]int32{slotHumanType: 9, slotHumanManaMax: 44})},
		},
	}
	w, err := mapload.FromALMWith(m, table, mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}
	e := w.Entities()[0]
	if e.HealthRegenPeriod != 100 || e.ManaRegenPeriod != 50 {
		t.Errorf("person carries periods %d/%d, want the constructor's 100/50 — the Humans table "+
			"has no such column", e.HealthRegenPeriod, e.ManaRegenPeriod)
	}
	if e.Mana != 42 || e.MaxMana != 42 {
		t.Errorf("person carries mana %d/%d, want the graph's own derived 42/42 for this row's "+
			"zero statistics — not its raw ManaMax cell of 44", e.Mana, e.MaxMana)
	}
}

// slotHumanSpirit and slotHumanSkillBase are two more Humans row slots,
// transcribed the same way every other slot constant in this suite is:
// Spirit at its own slot 3, and the six skill positions starting at slot 10
// (data.NewHumanDef's own switch, humandef.go), General through Shooting.
const (
	slotHumanSpirit    = 3
	slotHumanSkillBase = 10
)

// witnessArmorDefenceColumn and witnessArmorAbsorptionColumn are the Armors
// row's own two value columns (data.wear.go's armorDefenceColumn and
// armorAbsorptionColumn), transcribed here because wear_test.go's own
// armorRow leaves both at the format's empty cell — that suite asks only
// which SLOT a piece lands in, never what it is worth, and this test asks
// the opposite question.
const (
	witnessArmorSlotColumn       = slotArmorSlot
	witnessArmorDefenceColumn    = 9
	witnessArmorAbsorptionColumn = 10
)

// valuedArmorRow is an Armors row wide enough to carry Slot, Defence and
// Absorption, stating all three — wear_test.go's own armorRow states Slot
// alone, on the ground that no test there reads a value; this one does.
func valuedArmorRow(slot, defence, absorption int32) []int32 {
	p := make([]int32, witnessArmorAbsorptionColumn+1)
	p[witnessArmorSlotColumn] = slot
	p[witnessArmorDefenceColumn] = defence
	p[witnessArmorAbsorptionColumn] = absorption
	return p
}

// witnessHumanRow is a person with real statistics, a trained skill in every
// slot, and a mana column — the same shape of row Scrakan's own carries on
// mission 151 (docs/hotfix/LEDGER.md), synthetic so this suite reads no
// install.
func witnessHumanRow() []int32 {
	return humanRow(map[int]int32{
		slotHumanType: 7, slotHumanBody: 17, slotHumanReaction: 24,
		slotHumanMind: 42, slotHumanSpirit: 50, slotHumanManaMax: 70,
		slotHumanSkillBase + 0: 0, slotHumanSkillBase + 1: 93,
		slotHumanSkillBase + 2: 91, slotHumanSkillBase + 3: 95,
		slotHumanSkillBase + 4: 97, slotHumanSkillBase + 5: 99,
	})
}

// TestAPlacedPersonsSheetAtLoadEqualsAFullRecompute is the hotfix's own
// witness (docs/hotfix/LEDGER.md): a map-placed person's combat block and
// mana maximum, AT LOAD, must equal what mapload.ResolveEquipmentLoadout
// plus data.HumanDef.DerivedWithLoadout give the SAME row over the SAME
// resolved worn set — the graph a live skill-raise (pkg/game rearm.go
// recomputeRaisedSkills) already recomputes him through. It reads the
// resolved worn set BACK OFF THE WORLD (sim.World.Equipped) rather than
// reimplementing wearRow's own cell-to-slot resolution, so this test cannot
// disagree with the loader about which cell armed which slot — only about
// whether the two derivations agree past that point.
func TestAPlacedPersonsSheetAtLoadEqualsAFullRecompute(t *testing.T) {
	shapes, materials := identityScale(), identityScale()
	weapons := wearWeapons()
	armors := defCollection{{}, {name: "Arm4", params: valuedArmorRow(4, 12, 6)}}
	tbl := &mapload.Table{
		Units:  defCollection{{}},
		Humans: defCollection{{}, {name: "k7", params: witnessHumanRow(), strings: []string{"Sword", "", "Arm4"}}},
		Shapes: shapes, Materials: materials, Weapons: weapons, Armors: armors,
	}

	w, err := mapload.FromALMWith(humanMap(), tbl, mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}
	ents := w.Entities()
	if len(ents) != 1 {
		t.Fatalf("the fixture built %d entities, want 1", len(ents))
	}
	e := ents[0]

	d, err := data.NewHumanDef("k7", witnessHumanRow())
	if err != nil {
		t.Fatalf("NewHumanDef: %v", err)
	}
	eq, ok := w.Equipped(e.ID)
	if !ok {
		t.Fatalf("Equipped: no entry for the one entity this fixture built")
	}
	var equipment data.Equipment
	for slot, code := range eq {
		equipment.SetCode(slot+1, data.ItemCode(code))
	}
	sword, err := data.ResolveWeapon("Sword", shapes, materials, weapons)
	if err != nil {
		t.Fatalf("ResolveWeapon: %v", err)
	}
	// everEquipped is FALSE, on the same ground blockFor's own hotfix comment
	// gives (fromalm.go): a map placement has never had anything taken off.
	loadout, ok := mapload.ResolveEquipmentLoadout(equipment, &sword, false, tbl)
	if !ok {
		t.Fatalf("ResolveEquipmentLoadout refused a table this fixture built whole")
	}
	// PLAIN Recompute, NOT HumanDef.DerivedWithLoadout: the two agree on
	// every field this test reads (DerivedWithLoadout's own doc — the
	// cadence-override tail it adds past Recompute touches
	// Combat.AttackChargeTime and Combat.AttackRelaxTime alone), and reading
	// this side off the older, narrower method keeps this test's own
	// expectation independent of the new method the fix under test adds.
	want := d.Hero().Recompute(d.Profile(), loadout)

	if want.Combat.Defence == 0 {
		t.Fatalf("fixture error: the recompute's own Defence is 0 — the armour fold is not exercised")
	}
	if e.ToHit != want.Combat.ToHit || e.Defence != want.Combat.Defence || e.Absorption != want.Combat.Absorption ||
		e.DamageBase != want.Combat.DamageBase || e.DamageSpread != want.Combat.DamageSpread {
		t.Errorf("entity at load {ToHit:%d Defence:%d Absorption:%d DamageBase:%d DamageSpread:%d} != "+
			"the recompute {ToHit:%d Defence:%d Absorption:%d DamageBase:%d DamageSpread:%d}",
			e.ToHit, e.Defence, e.Absorption, e.DamageBase, e.DamageSpread,
			want.Combat.ToHit, want.Combat.Defence, want.Combat.Absorption, want.Combat.DamageBase, want.Combat.DamageSpread)
	}
	if e.MaxMana != want.ManaMax {
		t.Errorf("entity at load MaxMana = %d, want the recompute's ManaMax %d", e.MaxMana, want.ManaMax)
	}
	if e.Mana != e.MaxMana {
		t.Errorf("entity at load Mana = %d, MaxMana = %d, want a fresh placement spawned at full", e.Mana, e.MaxMana)
	}
}

// slotUnitMind, slotHumanMind and slotUnitXPValue are 0125's own slot
// numbers, spelled out here for the reason slotUnitHealthRegenPeriod and its
// neighbours above already are: a test asserting that a column moved a
// placement must not read that column's position out of the code it is
// testing. Mind sits at the SAME slot on both collections; XPValue is a
// UNITS row's own last slot, and the Humans collection has no such column at
// all — TestTheHumansArmCarriesTheConstructorsExperienceValueWithTheRowsMind
// below is about exactly that gap.
const (
	slotUnitMind    = 2
	slotHumanMind   = 2
	slotUnitXPValue = 37
)

func TestAResolvedUnitsPlacementCarriesItsRowsExperienceValueAndMind(t *testing.T) {
	m := &alm.Map{Width: 10, Height: 10,
		Units: []alm.Unit{{X: 5 << 8, Y: 5 << 8, ClassID: 90}}}
	table := &mapload.Table{
		Units: defCollection{
			{},
			{name: "k90", params: defRow(map[int]int32{
				slotUnitType: 90, slotUnitFace: 0, slotHealthMax: 40,
				slotUnitMind: 33, slotUnitXPValue: 77,
			})},
		},
	}
	w, err := mapload.FromALMWith(m, table, mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}
	e := w.Entities()[0]
	if e.Mind != 33 || e.XPValue != 77 {
		t.Errorf("entity carries Mind %d and experience value %d, want the row's own 33 and 77", e.Mind, e.XPValue)
	}
	if e.GainsXP {
		t.Error("a creature gains experience; FR-3 says its class must not")
	}
	if e.XPSlot != 0 {
		t.Errorf("entity credits slot %d, want SkillGeneral (0) — a units row derives no weapon skill", e.XPSlot)
	}
}

func TestTheHumansArmCarriesTheConstructorsExperienceValueWithTheRowsMind(t *testing.T) {
	m := &alm.Map{Width: 10, Height: 10,
		Units: []alm.Unit{{X: 5 << 8, Y: 5 << 8, ClassID: 9}}}
	table := &mapload.Table{
		Humans: defCollection{
			{},
			{name: "h9", params: defRow(map[int]int32{slotHumanType: 9, slotHumanMind: 41})},
		},
	}
	w, err := mapload.FromALMWith(m, table, mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}
	e := w.Entities()[0]
	if e.Mind != 41 {
		t.Errorf("person carries Mind %d, want the row's own 41", e.Mind)
	}
	if e.XPValue != 0 {
		t.Errorf("person carries experience value %d, want the constructor's 0 — the Humans table "+
			"has no such column", e.XPValue)
	}
	if e.GainsXP {
		t.Error("a zero-mode low-TypeID Human gains experience; only the player-character band may train")
	}
}

func TestAnUnresolvedPlacementCarriesTheConstructorsExperienceValueAndMind(t *testing.T) {
	m := &alm.Map{Width: 10, Height: 10,
		Units: []alm.Unit{{X: 5 << 8, Y: 5 << 8, ClassID: 12345}}}
	w := mapload.FromALM(m)
	e := w.Entities()[0]
	def := data.UnitDefaults()
	if e.Mind != def.Mind || e.XPValue != def.XPValue {
		t.Errorf("entity carries Mind %d and experience value %d, want the constructor's %d and %d",
			e.Mind, e.XPValue, def.Mind, def.XPValue)
	}
	if e.GainsXP {
		t.Error("an unresolved placement gains experience; FR-3 says it must not")
	}
}

// TestAdjustLeavesTheFourNewRegenFieldsAlone is the two easy mistakes'
// first: difficulty scales HealthMax alone, and the regeneration pass's gain
// is proportional to a maximum, so the scaling already falls out — Adjust
// itself must not touch a mana field or a period at any of the three
// settings.
func TestAdjustLeavesTheFourNewRegenFieldsAlone(t *testing.T) {
	in := data.UnitDef{HealthMax: 50, HealthRegenPeriod: 11, Mana: 22, ManaMax: 33, ManaRegenPeriod: 44}
	for _, diff := range []mapload.Difficulty{mapload.DifficultyEasy, mapload.DifficultyNormal, mapload.DifficultyHard} {
		got, err := mapload.Adjust(in, diff)
		if err != nil {
			t.Fatalf("Adjust(difficulty %d): %v", int32(diff), err)
		}
		if got.HealthRegenPeriod != 11 || got.Mana != 22 || got.ManaMax != 33 || got.ManaRegenPeriod != 44 {
			t.Errorf("Adjust(difficulty %d) moved a regeneration field: got %+v, want periods 11/44 "+
				"and mana 22/33 unchanged", int32(diff), got)
		}
	}
}

// preStoryFormVersion is the byte the canonical form opened with BEFORE this
// story, written out as a literal rather than read from the package that emits
// it — read from there it would agree with whatever that package said, which is
// the one thing this assertion exists to refuse.
const preStoryFormVersion byte = 12

// TestATableBuiltWorldRoundTripsAtTheVersionTheTreeCarries is AC-7 and SC-6: the
// eight this story fills enter the byte form and come back out of it identical.
//
// It was ...AtTheUnmovedVersion, and the version-is-unmoved half of it belonged
// to a story that added no field. Two later stories did add one, so what is
// asserted now is the version the tree carries — the literal moves with the form
// and the round trip is what the test still measures.
//
// The world is built FROM A TABLE, so its records carry the eight at values a
// table produced rather than at the constructor's — a round trip over an
// all-defaults world would pass just as well with a decoder that read the fields
// as zero.
func TestATableBuiltWorldRoundTripsAtTheVersionTheTreeCarries(t *testing.T) {
	t.Parallel()

	w, err := mapload.FromALMWith(fixtureMap(), fixtureTable(), mapload.DifficultyHard)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back sim.World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if got, want := back.Entities(), w.Entities(); !reflect.DeepEqual(got, want) {
		t.Errorf("the round trip changed the entities:\n got %+v\nwant %+v", got, want)
	}
	if back.Hash() != w.Hash() {
		t.Errorf("the round trip hashes %#016x, want %#016x", back.Hash(), w.Hash())
	}

	again, err := mapload.FromALMWith(fixtureMap(), fixtureTable(), mapload.DifficultyHard)
	if err != nil {
		t.Fatalf("FromALMWith, second load: %v", err)
	}
	if again.Hash() != w.Hash() {
		t.Errorf("two loads of one input hash differently: %#016x vs %#016x", again.Hash(), w.Hash())
	}
}

// pinMap is the fixture the three table shapes below are pinned against: an
// extent, both terrain planes, two placed units and three placed structures —
// one of which resolves in the table that carries Buildings.
//
// It carries units so the pinned digest covers an entity list and not only a
// plane, and its unit keys reach no entry in any table used here, so every world
// built from it spawns at the provisional health and the ONLY thing a table can
// move is the plane.
func pinMap() *alm.Map {
	m := passMap(24, 24)
	m.Tiles[12*24+12] = 600 // water, so a structure has something to subtract
	m.Units = []alm.Unit{
		{X: 10<<8 | 0x80, Y: 10 << 8, ClassID: 0x50, ClassSubID: 1},
		{X: 11<<8 | 0x80, Y: 12 << 8, ClassID: 0x51, ClassSubID: 2},
	}
	m.Objects = []alm.Object{
		place(1, 12, 12, nil),
		place(200, 13, 13, nil),
		place(0x21, 14, 14, ext(2, 2)),
	}
	return m
}

// pinDigest is pinMap's world digest, recorded by running this fixture and
// copied here by hand. It stood at 0x7f65b8eca4e34a24 before a structure could
// reach a plane at all, was re-recorded at 0056, where the movement rate widened
// every record and gave every placement a speed, and again at 0059, where the
// group rate term widened every record by one zero byte and raised the version.
// 0056's value was 0x45d2e9bf5ebe5541.
//
// 0067 moves it a fifth time and DELIBERATELY, which is what the four moves
// before it were not: both of this map's placements reach no units entry, so
// each now carries the constructor's charge of 8 and relax of 4 where it carried
// two zeros. The claim above is narrowed by exactly that and by nothing else —
// a world built with no table, a nil table or a table carrying no Buildings must
// still be byte for byte the world this map produced before a structure could
// reach a plane, once those two numbers are allowed for. Derived the same way
// its neighbour was, from the pre-story bytes rather than from a run, and
// checked in the other direction by zeroedEightDigest below. 0064's value was
// 0xf0db1255eab165cc.
//
// 1047 appends the inactive turn pair to every record outside the owner and
// group words, so this value moves too. 1045's value was 0x8baf7105add3ff12.
const pinDigest uint64 = 0x9efebe07086c13e6

// zeroedPinDigest is pinDigest's form with the eight fields' bytes lifted out of
// both records: the digest this fixture carried before this story.
//
// 1047's value replaces 1045's 0xb0c6ae14c317e84a.
const zeroedPinDigest uint64 = 0x93c363dc601e216e

// groupWordAt and ownerWordAt are those two words' offsets inside one entity
// record, and preGroupFormVersion the version byte the form carried before
// either existed. Together they are what lets the two literals below — both
// older than the group story — stay the fixed point: zeroedEightDigest lifts the
// eight fields AND both words out of every record and puts the version back, so
// what it hashes is the form as it stood before any of the three stories, and a
// byte written anywhere else fails against a number nothing in this story could
// have chosen.
const (
	groupWordAt              = 83
	ownerWordAt              = 87
	preGroupFormVersion byte = 10
)

// combatFieldsAt is the byte span of the eight fields this story fills, inside
// one entity record: the two cadence numbers at +54 and +58, the five a blow
// reads at +62…+81, and the always-hits byte at +82. They are CONTIGUOUS and
// close the record, which is why one span names all eight.
//
// The offsets are written out here from the form's own documented layout rather
// than reached for, so a field that moved would fail this test instead of being
// followed by it.
const (
	combatFieldsFrom = 54
	combatFieldsTo   = 83
	entityRecordLen  = 91
	entityCountAt    = 25
	gridCountAt      = 30
	formHeaderLen    = 34
)

// groupSectionCountLen is the width of the group section's own record count —
// pkg/sim's 0095. Every fixture in this package places units with no Owner,
// so the section every one of them writes is this many bytes and no record.
const groupSectionCountLen = 4

// spellStateLen is the width 0154 added to one entity record: AutoSpell (a
// uint16), CastWait, SpellFX and SpellFXSpell (a byte each) — pkg/sim's
// own unexported entityLen delta, written out here on weaponSpellLen's own
// terms: these tests cannot import it, so a change to it fails here loudly
// rather than quietly rewriting what these digests mean.
const spellStateLen = 5

// preSpellStateFormVersion is the version byte the form carries once 0154's
// five-byte tail is peeled back off — version 44, 0153's own landed state.
const preSpellStateFormVersion byte = 44

// strippedOfTheScriptItem undoes version 46's compiled-instant ITEM tail — a
// uint16 code and a presence byte per instant record — and restores version 45.
//
// EVERY FIXTURE IN THIS PACKAGE COMPILES NO INSTANTS: none of them authors a
// script action, so every one writes a script section whose instant count is
// zero and the inverse removes no bytes at all. What is left of the undo is the
// version byte, and that is the whole claim this peel makes — 0156 wrote nothing
// into any fixture here except the byte that names the version.
//
// It is a function beside its siblings rather than an inlined assignment because
// that is what keeps the claim testable. Should a fixture in this package ever
// author a script action, this is where the three bytes per instant have to come
// out, and the digest chain is what will say so.
//
// It runs after strippedOfTheOffMap, which is the newest peel.
func strippedOfTheScriptItem(form []byte) []byte {
	out := append([]byte(nil), form...)
	out[0] = 45
	return out
}

// offMapLen is the width 0164 added to one entity record: the off-map bit, a
// single 0/1 byte at the record's very tail.
const offMapLen = 1

// strippedOfTheOffMap removes 0164's off-map bit from every record and
// restores version 46. It is the NEWEST peel, so every older derivation runs
// on its output.
//
// It is STRUCTURAL rather than a zero check: it removes the byte by offset and
// width alone. Nothing in this package takes a placement off the map — only a
// mission-script arm sets the bit, and no fixture here fires one — so the byte
// it removes is always the zero the constructor leaves it at.
// lastArmsLen is how many bytes 0166 added to each entity record: six of escort
// triple at the tail, and one from widening the spell mark's remaining ticks to
// a word. cellTailCountLen is the width of that story's own cell-tail count.
const (
	lastArmsLen        = 7
	spellEffectsLen    = 30
	itemAttributionLen = 6
	cellTailCountLen   = 4
)

// mapUnitIDLen is the width 1029 added to each entity record: the authored map
// id of the placement, a uint16 at the record's very tail. Check opcode 9
// answers with it, so it has to reach the simulation and be carried by the
// form.
const mapUnitIDLen = 2

// preMapUnitIDFormVersion is the version byte the form carries once 1029's two
// bytes a record are peeled back off: version 56, 1025's own landed state.
const preMapUnitIDFormVersion byte = 56

// strippedOfTheMapUnitID removes the two bytes 1029 added to every entity
// record and restores version 56. It runs after strippedOfTheStructureSection,
// which is the newest peel.
//
// The peel is STRUCTURAL, on strippedOfTheOffMap's own rule: it removes by
// offset and width alone, not by checking the value. Every fixture whose form
// this chain runs on leaves alm.Unit.UnitID at 0 on every placement, so the
// bytes it removes are in fact zeros -- which is also why none of the digest
// pins in this package witnesses the loader carrying the field.
// mapunitid_test.go is where that is witnessed, on fixtures of its own.
//
// 1029 also widened the compiled CHECK record by three bytes, an item code and
// its presence flag. No fixture in this package authors a script condition, so
// every one of them writes a script section whose check count is zero and no
// byte has to come out there. That is asserted by
// TestNoFixtureInThisPackageCompilesAScriptCheck rather than left to this
// comment; should a fixture here ever author a condition, this is where the
// three bytes per check have to come out.
func strippedOfTheMapUnitID(form []byte) []byte {
	out := append([]byte(nil), form...)
	n := int(binary.LittleEndian.Uint32(out[entityCountAt : entityCountAt+4]))
	cells := int(binary.LittleEndian.Uint32(out[gridCountAt : gridCountAt+4]))
	base := formHeaderLen + 3*cells
	const currentWidth = 277
	for i := n - 1; i >= 0; i-- {
		r := base + currentWidth*i
		out = append(out[:r+currentWidth-mapUnitIDLen], out[r+currentWidth:]...)
	}
	out[0] = preMapUnitIDFormVersion
	return out
}

// TestNoFixtureInThisPackageCompilesAScriptCheck is what strippedOfTheMapUnitID
// asserts instead of claiming it in a comment. 1029 widened the compiled check
// record from 73 bytes to 76, and that peel removes no byte from the script
// section. That is correct only while every fixture whose form the peel chain
// runs on compiles zero checks, which is what is measured here. A fixture that
// gains a condition fails this test, and the peel is where the three bytes per
// check then have to come out.
func TestNoFixtureInThisPackageCompilesAScriptCheck(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		world *sim.World
	}{
		{"fixtureMap", mapload.FromALM(fixtureMap())},
		{"pinMap", mapload.FromALM(pinMap())},
		{"gfMap", mapload.FromALM(gfMap())},
	} {
		if n := len(tc.world.Script().Checks()); n != 0 {
			t.Errorf("%s compiles %d script check(s); strippedOfTheMapUnitID removes no byte "+
				"from the script section and has to grow one that narrows each check record "+
				"from 76 bytes back to 73", tc.name, n)
		}
	}
}

// scriptSectionEmptyLen is the script section's own fixed head when it
// compiles no trigger, check or instant record: pkg/sim's own scriptStateLen
// (400 bytes of register, 1000 of latch, the two win/lose counters and the
// outcome byte, 1409) plus scriptCountsLen (12), restated here on this file's
// own rule for every other width it walks past.
// TestNoFixtureInThisPackageCompilesAScriptCheck is what keeps this true for
// checks; no fixture in this package fires a mission-script instant or
// trigger either, so every one of them writes exactly this many bytes and no
// record.
const scriptSectionEmptyLen = 1421

// structureCountLen is the width 1033 B3 added: a whole new STRUCTURE
// SECTION, between 0166's own script-state section and the script section,
// its own count. structureRecordLen is one record's own width, an id and
// Field42 (structurebinary.go): pinMap() is the one fixture in this package
// that declares any (three, off its own three Objects, "Structures" —
// fromalm.go — minting one per placed type-4 record), so the peel below
// takes the count as its own argument rather than assuming it zero.
const (
	structureCountLen  = 4
	structureRecordLen = 22
)

// preStructureSectionFormVersion is the version byte the form carries once
// 1033's own section is peeled back off: version 57, this tree's own state
// before it.
const preStructureSectionFormVersion byte = 57

// withdrawalThresholdLen is the two-int32 record tail 1037 adds.
const withdrawalThresholdLen = 8

// strippedOfTurnDuration removes 1047's pass-3 request-time duration from
// every current entity record and restores the preceding form. It is the
// newest peel, so all older independent digest recipes continue to receive
// the exact bytes and record widths they measured before this correction.
func strippedOfTurnDuration(form []byte) []byte {
	out := strippedOfConsumables1090(form)
	n := int(binary.LittleEndian.Uint32(out[entityCountAt : entityCountAt+4]))
	cells := int(binary.LittleEndian.Uint32(out[gridCountAt : gridCountAt+4]))
	base := formHeaderLen + 3*cells
	const currentWidth = 294
	for i := n - 1; i >= 0; i-- {
		r := base + currentWidth*i
		out = append(out[:r+currentWidth-1], out[r+currentWidth:]...)
	}
	out[0] = 63
	return out
}

// Form80 appends a four-byte span after the complete form79 payload. ALM
// fixtures have no saved Player provenance or retired Group highwater, so
// their span is exactly zero. Assert that absence before removing the footer;
// a newly populated section must not silently disappear behind an old pin.
const absentSavedPlayerFooterLen = 4
const absentNativeStrideFooterLen = 4
const absentSavedMotionFooterLen = 4
const absentSavedCellFooterLen = 4
const absentSavedObjectsFooterLen = 4

const absentCarriedResumeFooterLen = 4

const entityIDFloorFooterLen = 8

func strippedOfPlayerCarriers(form []byte) []byte {
	out := append([]byte(nil), form...)
	if len(out) >= 13+formHeaderLen && out[0] == 112 && string(out[len(out)-4:]) == "NLB1" {
		n := uint64(binary.LittleEndian.Uint32(out[len(out)-9:]))
		base := out[len(out)-5]
		if n > uint64(len(out)-9-formHeaderLen) || n < 63 || (n-4)%59 != 0 || base >= 112 {
			panic("invalid native live block fixture span")
		}
		start := len(out) - 9 - int(n)
		if uint64(binary.LittleEndian.Uint32(out[start:])) != (n-4)/59 {
			panic("native live block fixture count differs from span")
		}
		out = out[:start]
		out[0] = base
	}
	if len(out) >= 14 && out[0] == 111 && string(out[len(out)-4:]) == "CPP1" {
		n := uint64(binary.LittleEndian.Uint32(out[len(out)-9:]))
		base := out[len(out)-5]
		if n > uint64(len(out)-9-formHeaderLen) || n < 5 || base >= 111 {
			panic("current Player span exceeds payload")
		}
		start := len(out) - 9 - int(n)
		count := uint64(binary.LittleEndian.Uint32(out[start+1:]))
		if out[start] > 1 || count > 65535 || n != 5+12*count {
			panic("invalid current Player population")
		}
		var previous uint32
		for i := uint64(0); i < count; i++ {
			at := start + 5 + 12*int(i)
			id := binary.LittleEndian.Uint32(out[at:])
			if id == 0 || id <= previous || out[start] == 0 && binary.LittleEndian.Uint32(out[at+8:]) != 0 {
				panic("invalid current Player identity or absence")
			}
			previous = id
		}
		out = out[:start]
		out[0] = base
	}
	if len(out) >= 9 && out[0] == 109 && string(out[len(out)-4:]) == "NAB1" {
		n := uint64(binary.LittleEndian.Uint32(out[len(out)-9:]))
		base := out[len(out)-5]
		if n > uint64(len(out)-9) || n < 111 || base >= 109 {
			panic("native basis span exceeds payload")
		}
		start := len(out) - 9 - int(n)
		count := uint64(binary.LittleEndian.Uint32(out[start:]))
		if count == 0 || count > 65535 || n != 4+107*count {
			panic("invalid native basis population")
		}
		out = out[:start]
		out[0] = base
	}
	if len(out) >= 9 && out[0] == 107 && string(out[len(out)-4:]) == "CLS1" {
		n := int(binary.LittleEndian.Uint32(out[len(out)-9:]))
		base := out[len(out)-5]
		if n > len(out)-9 || n < 9 || base >= 107 {
			panic("native class span exceeds payload")
		}
		out = out[:len(out)-9-n]
		out[0] = base
	}
	if len(out) >= 9 && out[0] == 105 && string(out[len(out)-4:]) == "TRN1" {
		n := int(binary.LittleEndian.Uint32(out[len(out)-9:]))
		base := out[len(out)-5]
		if n > len(out)-9 {
			panic("native training span exceeds payload")
		}
		out = out[:len(out)-9-n]
		out[0] = base
	}
	if len(out) > 0 && out[0] == 95 {
		n := int(binary.LittleEndian.Uint32(out[len(out)-4:]))
		out = out[:len(out)-4-n]
		out[0] = 94
	}
	if len(out) > 0 && out[0] == 94 {
		out = out[:len(out)-8]
		out[0] = 93
	}
	if len(out) > 0 && out[0] == 93 {
		if binary.LittleEndian.Uint32(out[len(out)-4:]) != 0 {
			panic("ALM fixture acquired an attack notice")
		}
		out = out[:len(out)-4]
		out[0] = 92
	}
	if len(out) > 0 && out[0] == 92 {
		// Form92 changes area ordering semantics, not the byte layout.
		out[0] = 91
	}
	if len(out) > 0 && out[0] == 91 {
		if len(out) < formHeaderLen+4 || binary.LittleEndian.Uint32(out[len(out)-4:]) != 0 {
			panic("ALM fixture acquired original world-effect continuation")
		}
		out = out[:len(out)-4]
		out[0] = 90
	}
	if len(out) > 0 && out[0] == 90 {
		if len(out) < formHeaderLen+4 || binary.LittleEndian.Uint32(out[len(out)-4:]) != 0 {
			panic("ALM fixture acquired exact saved formations")
		}
		out = out[:len(out)-4]
		out[0] = 89
	}
	if len(out) > 0 && out[0] == 89 {
		span := int(binary.LittleEndian.Uint32(out[len(out)-4:]))
		if span > len(out)-formHeaderLen-4 {
			panic("invalid structure-use fixture span")
		}
		out = out[:len(out)-4-span]
		out[0] = 88
	}
	if len(out) > 0 && out[0] == 88 {
		if len(out) < formHeaderLen+4 || binary.LittleEndian.Uint32(out[len(out)-4:]) != 0 {
			panic("ALM fixture acquired scorched cells")
		}
		out = out[:len(out)-4]
		out[0] = 87
	}
	if len(out) > 0 && out[0] == 87 {
		if len(out) < formHeaderLen+4 || binary.LittleEndian.Uint32(out[len(out)-4:]) != 0 {
			panic("ALM fixture acquired native Roam counter")
		}
		out = out[:len(out)-4]
		out[0] = 86
	}
	if len(out) > 0 && out[0] == 86 {
		span := int(binary.LittleEndian.Uint32(out[len(out)-4:]))
		if span%8 != 0 || span > len(out)-formHeaderLen-4 {
			panic("invalid action clock fixture span")
		}
		out = out[:len(out)-4-span]
		out[0] = 85
	}
	if len(out) > 0 && out[0] == 85 {
		if len(out) < formHeaderLen+absentCarriedResumeFooterLen || binary.LittleEndian.Uint32(out[len(out)-absentCarriedResumeFooterLen:]) != 0 {
			panic("strippedOfPlayerCarriers: ALM fixture must carry absent carried resume state")
		}
		out = out[:len(out)-absentCarriedResumeFooterLen]
		out[0] = 84
	}
	// ALM fixtures have no source-backed current-object registry. Only its
	// explicit absent span may be removed; all predecessor pins stay literal.
	if len(out) > 0 && out[0] == 84 {
		if len(out) < formHeaderLen+absentSavedObjectsFooterLen || binary.LittleEndian.Uint32(out[len(out)-absentSavedObjectsFooterLen:]) != 0 {
			panic("strippedOfPlayerCarriers: ALM fixture must carry absent current objects")
		}
		out = out[:len(out)-absentSavedObjectsFooterLen]
		out[0] = 83
	}
	if len(out) > 0 && out[0] == 83 {
		if len(out) < formHeaderLen+4 || binary.LittleEndian.Uint32(out[len(out)-4:]) != 0 {
			panic("strippedOfPlayerCarriers: ALM fixture must carry absent original cell planes")
		}
		out = out[:len(out)-4]
		out[0] = 82
	}
	if len(out) > 0 && out[0] == 82 {
		if len(out) < formHeaderLen+4 || binary.LittleEndian.Uint32(out[len(out)-4:]) != 0 {
			panic("strippedOfPlayerCarriers: ALM fixture must carry absent saved motion")
		}
		out = out[:len(out)-4]
		out[0] = 81
	}
	// A later ordinary stride may populate form81's independent tail. Remove
	// that captured history to compare the older gameplay-state goldens.
	if len(out) > 0 && out[0] == 81 {
		if len(out) < formHeaderLen+4 {
			panic("strippedOfPlayerCarriers: truncated stride footer")
		}
		span := uint64(binary.LittleEndian.Uint32(out[len(out)-4:]))
		if span > uint64(len(out)-formHeaderLen-4) {
			panic("strippedOfPlayerCarriers: stride span exceeds payload")
		}
		out = out[:len(out)-4-int(span)]
		out[0] = 80
	}
	if out[0] >= 80 {
		if len(out) < formHeaderLen+absentSavedPlayerFooterLen || binary.LittleEndian.Uint32(out[len(out)-absentSavedPlayerFooterLen:]) != 0 {
			panic("strippedOfPlayerCarriers: ALM fixture must carry an absent Player footer")
		}
		out = out[:len(out)-absentSavedPlayerFooterLen]
		out[0] = 79
	}
	return out
}

// Literal form73 layout, independent of sim's private record-width constant.
// Removing the new provenance byte must leave every older digest unchanged.
func strippedOfCurrentProfile1107(form []byte) []byte {
	out := strippedOfPlayerCarriers(form)
	if out[0] >= 79 {
		span := int(binary.LittleEndian.Uint32(out[len(out)-4:]))
		out = out[:len(out)-4-span]
		out[0] = 78
	}
	if out[0] >= 78 {
		span := int(binary.LittleEndian.Uint32(out[len(out)-4:]))
		out = out[:len(out)-4-span]
		out[0] = 77
	}
	if out[0] >= 77 {
		out = out[:len(out)-5]
		out[0] = 76
	}
	if out[0] >= 76 {
		base := 34 + 3*int(binary.LittleEndian.Uint32(out[30:34]))
		for i := int(binary.LittleEndian.Uint32(out[25:29])) - 1; i >= 0; i-- {
			at := base + 492*i + 457
			out = append(out[:at], out[at+35:]...)
		}
		out[0] = 75
	}
	if out[0] >= 75 {
		end := len(out) - 2500 - 4
		span := int(binary.LittleEndian.Uint32(out[end:]) & 0x7fffffff)
		out = append(out[:end-span], out[end+4:]...)
		out[0] = 74
	}
	if out[0] >= 74 {
		end := len(out) - 2500 - 4
		span := int(binary.LittleEndian.Uint32(out[end:]))
		out = append(out[:end-span], out[end+4:]...)
		out[0] = 73
	}
	if out[0] >= 73 {
		base := 34 + 3*int(binary.LittleEndian.Uint32(out[30:34]))
		for i := int(binary.LittleEndian.Uint32(out[25:29])) - 1; i >= 0; i-- {
			at := base + 457*i + 456
			out = append(out[:at], out[at+1:]...)
		}
		out[0] = 72
	}
	return out
}

func strippedOfSecondPhysical1104(form []byte) []byte {
	out := strippedOfCurrentProfile1107(form)
	if out[0] >= 72 {
		base := 34 + 3*int(binary.LittleEndian.Uint32(out[30:34]))
		for i := int(binary.LittleEndian.Uint32(out[25:29])) - 1; i >= 0; i-- {
			at := base + 456*i + 454
			out = append(out[:at], out[at+2:]...)
		}
		out[0] = 71
	}
	return out
}

func strippedOfSpellbook1101(form []byte) []byte {
	out := strippedOfSecondPhysical1104(form)
	if out[0] >= 71 {
		base := 34 + 3*int(binary.LittleEndian.Uint32(out[30:34]))
		for i := int(binary.LittleEndian.Uint32(out[25:29])) - 1; i >= 0; i-- {
			at := base + 454*i + 341
			out = append(out[:at], out[at+113:]...)
		}
		out[0] = 70
	}
	return out
}

func strippedOfOriginalDead1100(form []byte) []byte {
	out := strippedOfSpellbook1101(form)
	if out[0] >= 70 {
		end := len(out) - 2500
		span := int(binary.LittleEndian.Uint32(out[end-4:])) + 4
		out = append(out[:end-span], out[end:]...)
		out[0] = 69
	}
	return out
}

func strippedOfConsumables1090(form []byte) []byte {
	out := strippedOfOriginalDead1100(form)
	if out[0] >= 69 {
		n := int(binary.LittleEndian.Uint32(out[25:29]))
		base := 34 + 3*int(binary.LittleEndian.Uint32(out[30:34]))
		for i := n - 1; i >= 0; i-- {
			at := base + 341*i + 326
			out = append(out[:at], out[at+15:]...)
		}
		out[0] = 68
	}
	if out[0] < 68 {
		return out
	}
	n := int(binary.LittleEndian.Uint32(out[25:29]))
	base := 34 + 3*int(binary.LittleEndian.Uint32(out[30:34]))
	at := len(out) - formRelationLen - scriptSectionEmptyLen - 4
	out = append(out[:at], out[at+4:]...)
	for i := n - 1; i >= 0; i-- {
		at := base + 326*i + 294
		out = append(out[:at], out[at+32:]...)
	}
	out[0] = 67
	return out
}

// strippedOfItemState removes format 61's empty canonical item-instance
// section from the historical no-table fixtures in this file. Those worlds
// own no sack, carried or equipped instance; the section is therefore its two
// four-byte counts plus 189 bytes per entity.
func strippedOfItemState(form []byte) []byte {
	out := append([]byte(nil), form...)
	n := int(binary.LittleEndian.Uint32(out[entityCountAt : entityCountAt+4]))
	sectionLen := 8 + 189*n
	end := len(out) - formRelationLen - scriptSectionEmptyLen
	at := end - sectionLen
	out = append(out[:at], out[end:]...)
	out[0] = 60
	return out
}

// weaponResistanceLen is the five-byte record tail 1039 adds. Resistance is
// stored in active-skill order: blade, axe, bludgeon, pike and shooting.
const weaponResistanceLen = 5

// strippedOfWeaponResistance removes 1039's newest record tail and restores
// version 58. It is deliberately the first peel: every older peel continues
// to walk the record width of the version it originally followed.
func strippedOfWeaponResistance(form []byte) []byte {
	out := strippedOfWithdrawalThresholds(strippedOfTurnDuration(form))
	n := int(binary.LittleEndian.Uint32(out[entityCountAt : entityCountAt+4]))
	cells := int(binary.LittleEndian.Uint32(out[gridCountAt : gridCountAt+4]))
	base := formHeaderLen + 3*cells
	const currentWidth = 282
	for i := n - 1; i >= 0; i-- {
		r := base + currentWidth*i
		out = append(out[:r+currentWidth-weaponResistanceLen], out[r+currentWidth:]...)
	}
	out[0] = 58
	return out
}

// strippedOfWithdrawalThresholds removes 1037's current record tail and
// restores version 59, so every older peel keeps walking its historical width.
func strippedOfWithdrawalThresholds(form []byte) []byte {
	out := strippedOfItemState(form)
	n := int(binary.LittleEndian.Uint32(out[entityCountAt : entityCountAt+4]))
	cells := int(binary.LittleEndian.Uint32(out[gridCountAt : gridCountAt+4]))
	base := formHeaderLen + 3*cells
	const currentWidth = 290
	for i := n - 1; i >= 0; i-- {
		r := base + currentWidth*i
		out = append(out[:r+currentWidth-withdrawalThresholdLen], out[r+currentWidth:]...)
	}
	out[0] = 59
	return out
}

// strippedOfTheStructureSection removes 1033 B3's STRUCTURE SECTION and
// restores version 57. It runs immediately after the resistance peel, so every
// older derivation still receives the version it was written against.
//
// n is the caller's own w.Structures() count. The section's end sits
// scriptSectionEmptyLen and formRelationLen bytes from the end of the form on
// every fixture in this package: the script section is always its own bare
// 1421-byte head on scriptSectionEmptyLen's own reason, and the relation is
// always formRelationLen bytes at the very end. Its start is n records and
// the count field back from there, which is what n is for: the offset is
// found from the end rather than by walking every section in front of it,
// but the section's own width is not fixed the way every OTHER end-relative
// peel's is, pinMap's placements being the one fixture here that puts
// anything in it.
func strippedOfTheStructureSection(form []byte, n int) []byte {
	end := len(form) - formRelationLen - scriptSectionEmptyLen
	at := end - structureCountLen - structureRecordLen*n
	out := append([]byte(nil), form[:at]...)
	out = append(out, form[end:]...)
	out[0] = preStructureSectionFormVersion
	return out
}

// carriedWeightLen is the width 1025 added to each entity record: four bytes
// of Load and four of Capacity at the record's tail. itemWeightCountLen is
// that story's own section count, between the spell table and the casting
// section.
const (
	carriedWeightLen   = 8
	itemWeightCountLen = 2
)

// strippedOfCarriedWeight removes everything 1025 added to the form and
// restores version 55: the eight bytes each record grew by, and the
// ITEM-WEIGHT SECTION between the spell table and the casting section. It ran
// on the live form until 1029, and now runs on strippedOfTheMapUnitID's
// output, so the record width it walks is version 56's 275 rather than the
// live one.
//
// It is STRUCTURAL rather than a zero check, on strippedOfTheOffMap's own
// rule: it removes by offset and width alone. On every fixture in this package
// the bytes it removes are in fact zeros -- no fixture here declares an item
// weight, and a placement carries neither holdings nor a derived capacity --
// but the peel does not depend on that.
//
// The section's offset is found by walking every counted section in front of
// it, exactly as strippedOfTheLastArms walks the same chain, with the record
// width being the live one.
func strippedOfCarriedWeight(form []byte) []byte {
	n := int(binary.LittleEndian.Uint32(form[entityCountAt : entityCountAt+4]))
	cells := int(binary.LittleEndian.Uint32(form[gridCountAt : gridCountAt+4]))
	const currentWidth = 275
	o := formHeaderLen + 3*cells + n*currentWidth
	for range n {
		count := int(binary.LittleEndian.Uint32(form[o : o+4]))
		o += 4 + 8*count
	}
	groups := int(binary.LittleEndian.Uint32(form[o : o+4]))
	o += 4 + 18*groups
	sacks := int(binary.LittleEndian.Uint32(form[o : o+4]))
	o += 4
	for range sacks {
		items := int(binary.LittleEndian.Uint32(form[o+12 : o+16]))
		o += 16 + 2*items
	}
	for range n {
		items := int(binary.LittleEndian.Uint32(form[o : o+4]))
		o += 4 + 2*items
	}
	o += n * sim.EquipSlots * 2
	o += n * 16
	o += relationSlotsInForm * 4
	spells := int(binary.LittleEndian.Uint16(form[o : o+2]))
	if spells != 0 {
		panic("strippedOfCarriedWeight: this peel is only applied to forms whose spell table is empty")
	}
	o += 2
	out := append([]byte(nil), form[:o]...)
	out = append(out, form[o+itemWeightCountLen:]...)
	base := formHeaderLen + 3*cells
	for i := n - 1; i >= 0; i-- {
		r := base + currentWidth*i
		out = append(out[:r+currentWidth-carriedWeightLen], out[r+currentWidth:]...)
	}
	out[0] = 55
	return out
}

func strippedOfItemAttribution(form []byte) []byte {
	out := append([]byte(nil), form...)
	n := int(binary.LittleEndian.Uint32(out[entityCountAt : entityCountAt+4]))
	cells := int(binary.LittleEndian.Uint32(out[gridCountAt : gridCountAt+4]))
	base := formHeaderLen + 3*cells
	const currentWidth = 267
	for i := n - 1; i >= 0; i-- {
		o := base + currentWidth*i
		out = append(out[:o+currentWidth-itemAttributionLen], out[o+currentWidth:]...)
	}
	out[0] = 54
	return out
}

// strippedOfSpellEffects removes 1001's five protection dwords and token-size
// byte from every entity record. Empty spell/casting sections keep their old
// widths, so these fixtures need no second removal outside the records.
func strippedOfCorpseLoot(form []byte) []byte {
	out := append([]byte(nil), form...)
	n := int(binary.LittleEndian.Uint32(out[entityCountAt : entityCountAt+4]))
	cells := int(binary.LittleEndian.Uint32(out[gridCountAt : gridCountAt+4]))
	base := formHeaderLen + 3*cells
	const currentWidth = 261
	for i := n - 1; i >= 0; i-- {
		o := base + currentWidth*i
		out = append(out[:o+currentWidth-corpseLootLen], out[o+currentWidth:]...)
	}
	out[0] = 53
	return out
}

const corpseLootLen = 1

func strippedOfSpellEffects(form []byte) []byte {
	n := int(binary.LittleEndian.Uint32(form[entityCountAt : entityCountAt+4]))
	cells := int(binary.LittleEndian.Uint32(form[gridCountAt : gridCountAt+4]))
	const oldWidth = 230
	base := formHeaderLen + 3*cells
	out := append([]byte(nil), form...)
	for i := n - 1; i >= 0; i-- {
		o := base + (oldWidth+spellEffectsLen)*i
		out = append(out[:o+oldWidth], out[o+oldWidth+spellEffectsLen:]...)
	}
	out[0] = 50
	return out
}

// strippedOfTheLastArms removes everything 0166 added to the form and restores
// version 49: the SCRIPT-STATE SECTION between the casting section and the
// script section, and the seven bytes each record grew by. It is the NEWEST
// peel, so every older derivation runs on its output.
//
// It is STRUCTURAL rather than a zero check, on strippedOfTheOffMap's own rule.
// Nothing in this package writes an escort order or a spell mark — both are
// mission-script arms and no fixture here fires one — and nothing writes a cell
// tail either, so the tail half of the section is a bare zero count. The
// FORMATION BLOCK it removes is NOT zero: it is the default in every slot,
// which is what a world nothing has written a mode into carries.
//
// The section's offset is found by walking every counted section in front of
// it, exactly as strippedOfTheCasting walks the same chain, with the record
// width being the live one this peel runs on.
func strippedOfTheLastArms(form []byte) []byte {
	n := int(binary.LittleEndian.Uint32(form[entityCountAt : entityCountAt+4]))
	cells := int(binary.LittleEndian.Uint32(form[gridCountAt : gridCountAt+4]))
	width := entityRecordLen + 1 + decayBlockLen + 1 + actorStateTailLen + 1 + postLen + regenLen + commandGroupLen + experienceLen + knownSpellsLen + skillBlockLen + weaponSpellLen + spellStateLen + offMapLen + lastArmsLen
	o := formHeaderLen + 3*cells + n*width
	for range n {
		count := int(binary.LittleEndian.Uint32(form[o : o+4]))
		o += 4 + 8*count
	}
	groups := int(binary.LittleEndian.Uint32(form[o : o+4]))
	o += 4 + 18*groups
	sacks := int(binary.LittleEndian.Uint32(form[o : o+4]))
	o += 4
	for range sacks {
		items := int(binary.LittleEndian.Uint32(form[o+12 : o+16]))
		o += 16 + 2*items
	}
	for range n {
		items := int(binary.LittleEndian.Uint32(form[o : o+4]))
		o += 4 + 2*items
	}
	o += n * sim.EquipSlots * 2
	o += n * 16
	o += relationSlotsInForm * 4
	spells := int(binary.LittleEndian.Uint16(form[o : o+2]))
	if spells != 0 {
		panic("strippedOfTheLastArms: this peel is only applied to forms whose spell table is empty")
	}
	o += 2 + 2*castingCountLen
	out := append([]byte(nil), form[:o]...)
	out = append(out, form[o+relationSlotsInForm+cellTailCountLen:]...)
	// The seven per-record bytes, records walked BACKWARDS so that each
	// removal leaves the earlier records at the offsets the contract gives
	// them. Within a record the escort triple comes off first, then the spell
	// mark's high byte, on the same rule.
	base := formHeaderLen + 3*cells
	for i := n - 1; i >= 0; i-- {
		r := base + width*i
		out = append(out[:r+width-6], out[r+width:]...)
		out = append(out[:r+width-9], out[r+width-8:]...)
	}
	out[0] = preLastArmsFormVersion
	return out
}

// preLastArmsFormVersion is the version byte the form carries once 0166's own
// bytes are peeled back off — version 49, 0165's own landed state.
const preLastArmsFormVersion byte = 49

// strippedOfTheCasting removes 0165's CASTING SECTION and restores version 48.
// It runs on strippedOfTheLastArms' output.
//
// The section is the two counts, eight zero bytes on every fixture in this
// package: nothing here creates a pending cast or an area effect, both being
// mission-script arms and no fixture firing one. It sits between the spell
// table and the script section, so its offset is found by walking every
// counted section in front of it, exactly as strippedOfDeathGold walks the
// same chain — the record width being the live one this peel runs on, which
// is the width strippedOfTheOffMap names.
//
// The spell record also grew by four bytes at this version, and this peel does
// not undo that: it is applied to forms whose spell table is empty, and the
// length check below asserts that.
func strippedOfTheCasting(form []byte) []byte {
	n := int(binary.LittleEndian.Uint32(form[entityCountAt : entityCountAt+4]))
	cells := int(binary.LittleEndian.Uint32(form[gridCountAt : gridCountAt+4]))
	width := entityRecordLen + 1 + decayBlockLen + 1 + actorStateTailLen + 1 + postLen + regenLen + commandGroupLen + experienceLen + knownSpellsLen + skillBlockLen + weaponSpellLen + spellStateLen + offMapLen
	o := formHeaderLen + 3*cells + n*width
	for range n {
		count := int(binary.LittleEndian.Uint32(form[o : o+4]))
		o += 4 + 8*count
	}
	groups := int(binary.LittleEndian.Uint32(form[o : o+4]))
	o += 4 + 18*groups
	sacks := int(binary.LittleEndian.Uint32(form[o : o+4]))
	o += 4
	for range sacks {
		items := int(binary.LittleEndian.Uint32(form[o+12 : o+16]))
		o += 16 + 2*items
	}
	for range n {
		items := int(binary.LittleEndian.Uint32(form[o : o+4]))
		o += 4 + 2*items
	}
	o += n * sim.EquipSlots * 2
	o += n * 16
	o += relationSlotsInForm * 4
	spells := int(binary.LittleEndian.Uint16(form[o : o+2]))
	if spells != 0 {
		panic("strippedOfTheCasting: this peel is only applied to forms whose spell table is empty")
	}
	o += 2
	out := append([]byte(nil), form[:o]...)
	out = append(out, form[o+2*castingCountLen:]...)
	out[0] = preCastingFormVersion
	return out
}

// preCastingFormVersion is the version byte the form carries once 0165's own
// section is peeled back off — version 48, 0164's own landed state.
const preCastingFormVersion byte = 48

// castingCountLen is the width of each of the casting section's two counts, and
// relationSlotsInForm the roster-slot count the purse section is sized by. Both
// are pkg/sim's own constants restated here, on this file's own rule for every
// other width it walks past.
const (
	castingCountLen     = 4
	relationSlotsInForm = 50
)

func strippedOfTheOffMap(form []byte) []byte {
	out := append([]byte(nil), form...)
	n := int(binary.LittleEndian.Uint32(out[entityCountAt : entityCountAt+4]))
	cells := int(binary.LittleEndian.Uint32(out[gridCountAt : gridCountAt+4]))
	base := formHeaderLen + 3*cells
	width := entityRecordLen + 1 + decayBlockLen + 1 + actorStateTailLen + 1 + postLen + regenLen + commandGroupLen + experienceLen + knownSpellsLen + skillBlockLen + weaponSpellLen + spellStateLen + offMapLen
	for i := n - 1; i >= 0; i-- {
		o := base + width*i
		out = append(out[:o+width-offMapLen], out[o+width:]...)
	}
	out[0] = preOffMapFormVersion
	return out
}

// preOffMapFormVersion is the version byte the form carries once 0164's own
// byte is peeled back off — version 46, 0156's own landed state.
const preOffMapFormVersion byte = 46

// strippedOfTheSpellState removes 0154's autocast pair and spell effect mark
// from every record and restores version 44. It is the NEWEST peel, so every
// older derivation runs on its output. Nothing in this package gives a
// placement an autocast and no cast has run, so the five bytes it removes are
// always the zeros the constructor leaves them at.
func strippedOfTheSpellState(form []byte) []byte {
	out := append([]byte(nil), form...)
	n := int(binary.LittleEndian.Uint32(out[entityCountAt : entityCountAt+4]))
	cells := int(binary.LittleEndian.Uint32(out[gridCountAt : gridCountAt+4]))
	base := formHeaderLen + 3*cells
	width := entityRecordLen + 1 + decayBlockLen + 1 + actorStateTailLen + 1 + postLen + regenLen + commandGroupLen + experienceLen + knownSpellsLen + skillBlockLen + weaponSpellLen + spellStateLen
	for i := n - 1; i >= 0; i-- {
		o := base + width*i
		out = append(out[:o+width-spellStateLen], out[o+width:]...)
	}
	out[0] = preSpellStateFormVersion
	return out
}

// strippedOfDeathGold removes version 44's four-int32 record per entity and
// restores version 41. The section follows equipment, so the walk crosses the
// variable route, sack and carry sections rather than assuming an offset.
func strippedOfDeathGold(form []byte) []byte {
	n := int(binary.LittleEndian.Uint32(form[entityCountAt : entityCountAt+4]))
	cells := int(binary.LittleEndian.Uint32(form[gridCountAt : gridCountAt+4]))
	o := formHeaderLen + 3*cells + n*217
	for range n {
		count := int(binary.LittleEndian.Uint32(form[o : o+4]))
		o += 4 + 8*count
	}
	groups := int(binary.LittleEndian.Uint32(form[o : o+4]))
	o += 4 + 18*groups
	sacks := int(binary.LittleEndian.Uint32(form[o : o+4]))
	o += 4
	for range sacks {
		items := int(binary.LittleEndian.Uint32(form[o+12 : o+16]))
		o += 16 + 2*items
	}
	for range n {
		items := int(binary.LittleEndian.Uint32(form[o : o+4]))
		o += 4 + 2*items
	}
	o += n * sim.EquipSlots * 2
	out := append([]byte(nil), form[:o]...)
	out = append(out, form[o+n*16:]...)
	out[0] = 41
	return out
}

// zeroedEightDigest is AC-8's recipe, run over a world this story's code built:
// marshal it, write zero over the eight fields' bytes in EVERY record, and hash
// what is left the way a world hashes itself.
//
// It is the whole of R-1's mitigation. A pinned digest pasted out of a run is a
// pin that cannot fail — it agrees with whatever produced it — so the two pins
// this story moves are each checked against a SECOND derivation whose target
// literal predates the story. If this story wrote a byte anywhere but in these
// eight fields, the two would disagree and only this check would say so.
func zeroedEightDigest(t *testing.T, w *sim.World) uint64 {
	t.Helper()
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if len(form) < formHeaderLen {
		t.Fatalf("the byte form is %d byte(s), shorter than its own header", len(form))
	}
	form = strippedOfThePlanes(strippedOfTheFacing(strippedOfTheRelation(strippedOfTheDecayBlock(strippedOfTheSight(strippedOfTheGroupSection(strippedOfTheActorState(strippedOfTheSacks(strippedOfTheReach(strippedOfThePost(strippedOfTheRegen(strippedOfTheCarryAndPurse(strippedOfTheCommandGroup(strippedOfTheExperience(strippedOfTheKnownSpells(strippedOfTheSkill(strippedOfTheWeaponSpell(strippedOfDeathGold(strippedOfTheSpellState(strippedOfTheScriptItem(strippedOfTheOffMap(strippedOfTheCasting(strippedOfTheLastArms(strippedOfSpellEffects(strippedOfCorpseLoot(strippedOfItemAttribution(strippedOfCarriedWeight(strippedOfTheMapUnitID(strippedOfTheStructureSection(strippedOfWeaponResistance(form), len(w.Structures()))))))))))))))))))))))))))))))
	n := int(binary.LittleEndian.Uint32(form[entityCountAt : entityCountAt+4]))
	cells := int(binary.LittleEndian.Uint32(form[gridCountAt : gridCountAt+4]))
	base := formHeaderLen + cells
	if want := base + n*entityRecordLen; len(form) < want {
		t.Fatalf("the byte form is %d byte(s) and its %d record(s) need %d", len(form), n, want)
	}
	// Backwards, so removing one record's tail leaves the earlier records where
	// they are.
	for i := n - 1; i >= 0; i-- {
		o := base + entityRecordLen*i
		for b := o + combatFieldsFrom; b < o+combatFieldsTo; b++ {
			form[b] = 0
		}
		form = append(form[:o+ownerWordAt], form[o+entityRecordLen:]...)
		form = append(form[:o+groupWordAt], form[o+ownerWordAt:]...)
	}
	form[0] = preGroupFormVersion
	h := fnv.New64a()
	h.Write(form)
	return h.Sum64()
}

// preCostFormVersion is the version byte the form carried before the cost and
// height planes existed, and strippedOfThePlanes is the inverse of the widening
// that added them: one form with the two plane sections cut out from behind the
// block plane and the version byte put back.
//
// It runs FIRST in both derivations below, because the strippings come off
// newest story first — and it is what keeps every pre-story digest in this file
// a fixed point rather than a number to be replaced whenever a plane arrives.
const preCostFormVersion byte = 12

// currentFormVersion is the version this tree writes today, and preStoryCells the
// fixture map's cell count — the plane length the two new sections each add.
const (
	// currentFormVersion tracks pkg/sim's own formatVersion by hand: 41 since
	// WeaponSpell and WeaponSpellLevel arrived on the entity record's own tail
	// (0139 T2; 40 is out to a lane running in parallel off the same master
	// this one branched from, and this package carries none of its change),
	// before that 39 since an entity's six skill levels landed as a tail on the
	// entity record (0135 T1), before that 38 since a compiled instant's SECOND
	// unit reference landed as a tail on the instant record (0129 T2; 37 is out
	// to a lane running in parallel off the same master this one branched from,
	// and this package carries none of its change), before that 36 since a
	// KnownSpells mask arrived on the entity record's own tail and the spell
	// table arrived between the purse section and the script section (0127 T4),
	// before that 35 since an entity's experience from use arrived on the
	// entity record's own tail (0125 T2), before that 34 since the equipment
	// section landed between the carry section and the purse (0124 T2; 33 is
	// out to a lane running in parallel off the same master this one branched
	// from, and this package carries none of its section), before that 32 since
	// the command group arrived on the entity record's own tail (0117 T1),
	// before that 31 since a compiled check's second player reference landed as
	// a tail on the check record (0122 T2), before that 26 since the carry and
	// purse sections landed between the sack section and the script section
	// (0112 T2). It has to move every time that constant does, because this
	// package asserts an encoded form's version byte against it rather than
	// importing the unexported constant. It is 56 since the carried load and
	// the carrying capacity landed on the entity record's own tail together
	// with the world's own item-weight section (1025), was 55 while delayed
	// kill attribution closed the record, and 54 while the corpse-loot
	// suppression byte did. It is 57 since the authored map id landed on the
	// entity record's own tail and the compiled check record grew an item
	// reference (1029). It is 58 since the structure section landed between the
	// script-state section and the script section, and the compiled check and
	// instant records each grew a structure reference (1033 B3). It is 59 since
	// the five weapon-kind resistance bytes landed on each entity record
	// (1039). It is 63 since DesiredFacing and TurnRemaining landed on the
	// entity record's own tail (1047), and 64 since the same story's pass-3
	// correction appended request-time TurnTotal. It is 65 since destructible
	// structures gained their map shape (1052), and 66 since cloud Remaining
	// became the raw counter (1063). Form80 appends exact saved
	// Player-container provenance; form81 appends captured native strides
	// (1115). The independent peels preserve the prior gameplay-state goldens.
	// It is 94 since the unconditional entity-ID allocation floor landed
	// outermost, after attack notices (1177).
	currentFormVersion byte = 95
	preStoryCells           = 72 * 68
)

// formRelationLen is how long pkg/sim's relation block is: 50 slots square, one
// byte each. The constant is unexported there, so it is written out here on
// gfScriptSection's own terms — a length this package's tests have to know and
// cannot import, pinned so that moving it in sim fails here loudly rather than
// quietly rewriting what these digests mean.
const formRelationLen = 50 * 50

// preSightFormVersion is the version byte the form carried before THE SIGHT
// RANGE existed — which is version 17, the tree as 0089 left it, not version 16.
// The two stories were written in parallel off one master and 0089 landed first,
// so "pre-sight" is its post-landing state.
//
// strippedOfTheSight is the inverse of the widening that added the range: one
// form with the one byte each record grew by cut off its tail and the version
// byte put back. It ran first, ahead of the decay block's, until 0095's group
// section arrived outside every record — the strippings come off newest story
// first, which is what keeps every pre-story digest in this file a fixed
// point rather than a number to be replaced whenever a field arrives, and
// strippedOfTheGroupSection now runs ahead of this one for that reason.
const preSightFormVersion byte = 17

// preSkillFormVersion is the version byte the form carries once 0135's
// twenty-four-byte tail is peeled back off — version 38, the value 0129's
// second unit reference landed at and currentFormVersion sat on before
// this story bumped it to 39.
//
// strippedOfTheSkill is the inverse of the widening that added it: one
// form with the twenty-four bytes each record grew by cut off its tail
// and the version byte put back. It is the NEWEST peel, so it runs BEFORE
// strippedOfTheKnownSpells and everything below it — the strippings still
// come off newest story first.
//
// THE PEEL IS STRUCTURAL, NOT A ZERO CHECK, on strippedOfTheKnownSpells's
// own reason: this loader fills no skill level yet (0135's own mapload
// task is a later story), so it always finds the same twenty-four zero
// bytes at the tail of every record it strips, but the peel removes them
// by offset and width alone regardless.
const preSkillFormVersion byte = 38

// skillBlockLen is the width 0135 added to one entity record: six int32
// skill levels — pkg/sim's own unexported entityLen delta, written out
// here on knownSpellsLen's own terms: these tests cannot import it, so a
// change to it fails here loudly rather than quietly rewriting what these
// digests mean.
const skillBlockLen = 24

func strippedOfTheSkill(form []byte) []byte {
	out := append([]byte(nil), form...)
	n := int(binary.LittleEndian.Uint32(out[entityCountAt : entityCountAt+4]))
	cells := int(binary.LittleEndian.Uint32(out[gridCountAt : gridCountAt+4]))
	base := formHeaderLen + 3*cells
	width := entityRecordLen + 1 + decayBlockLen + 1 + actorStateTailLen + 1 + postLen + regenLen + commandGroupLen + experienceLen + knownSpellsLen + skillBlockLen
	for i := n - 1; i >= 0; i-- {
		o := base + width*i
		out = append(out[:o+width-skillBlockLen], out[o+width:]...)
	}
	out[0] = preSkillFormVersion
	return out
}

// preSpellFormVersion is the version byte the form carries once 0127's
// four-byte tail is peeled back off — version 35, the value 0125's
// experience-from-use block landed at and currentFormVersion sat on before
// this story bumped it to 36. The entity record's own tail is the only
// thing this peel moves; the spell table section 0127 also adds is
// strippedOfTheCarryAndPurse's own widening, on 0124 T2's own precedent
// for the equipment section below.
//
// strippedOfTheKnownSpells is the inverse of the widening that added it:
// one form with the four bytes each record grew by cut off its tail and
// the version byte put back. It was the newest peel until 0135's own
// strippedOfTheSkill arrived above it; it runs BEFORE
// strippedOfTheExperience and everything below it — the strippings still
// come off newest story first.
//
// THE PEEL IS STRUCTURAL, NOT A ZERO CHECK, on strippedOfTheExperience's
// own reason: this loader fills no KnownSpells mask yet (0127's own
// mapload task is a later story), so it always finds the same four zero
// bytes at the tail of every record it strips, but the peel removes them
// by offset and width alone regardless.
const preSpellFormVersion byte = 35

// knownSpellsLen is the width 0127 added to one entity record: one
// KnownSpells uint32 — pkg/sim's own unexported entityLen delta, written
// out here on experienceLen's own terms: these tests cannot import it, so
// a change to it fails here loudly rather than quietly rewriting what
// these digests mean.
const knownSpellsLen = 4

func strippedOfTheKnownSpells(form []byte) []byte {
	out := append([]byte(nil), form...)
	n := int(binary.LittleEndian.Uint32(out[entityCountAt : entityCountAt+4]))
	cells := int(binary.LittleEndian.Uint32(out[gridCountAt : gridCountAt+4]))
	base := formHeaderLen + 3*cells
	width := entityRecordLen + 1 + decayBlockLen + 1 + actorStateTailLen + 1 + postLen + regenLen + commandGroupLen + experienceLen + knownSpellsLen
	for i := n - 1; i >= 0; i-- {
		o := base + width*i
		out = append(out[:o+width-knownSpellsLen], out[o+width:]...)
	}
	out[0] = preSpellFormVersion
	return out
}

// preWeaponSpellFormVersion is the version byte the form carries once the
// tail this peel removes is gone — version 39, 0135's own landed state
// (its own skill block sits between the known-spells tail and this one,
// untouched by this peel, so the bytes it leaves behind still carry it;
// tagging them 38 would claim a form that predates it). It is the NEWEST
// peel now, ahead of the skill block's own, and it is STRUCTURAL rather
// than a zero check: it removes the tail's six bytes by offset and width
// alone, on the spellbook peel's own reason — no fixture in this package
// places a weapon carrying a castSpell attachment, so the field this peel
// removes has always been the zero WeaponSpell's own doc names as "none"
// (world.go).
const preWeaponSpellFormVersion byte = 39

// weaponSpellLen is the width 0139 added to one entity record: WeaponSpell
// (a uint16) and WeaponSpellLevel (an int32) — pkg/sim's own unexported
// entityLen delta, written out here on knownSpellsLen's own terms: these
// tests cannot import it, so a change to it fails here loudly rather than
// quietly rewriting what these digests mean.
const weaponSpellLen = 6

func strippedOfTheWeaponSpell(form []byte) []byte {
	out := append([]byte(nil), form...)
	n := int(binary.LittleEndian.Uint32(out[entityCountAt : entityCountAt+4]))
	cells := int(binary.LittleEndian.Uint32(out[gridCountAt : gridCountAt+4]))
	base := formHeaderLen + 3*cells
	width := entityRecordLen + 1 + decayBlockLen + 1 + actorStateTailLen + 1 + postLen + regenLen + commandGroupLen + experienceLen + knownSpellsLen + skillBlockLen + weaponSpellLen
	for i := n - 1; i >= 0; i-- {
		o := base + width*i
		out = append(out[:o+width-weaponSpellLen], out[o+width:]...)
	}
	out[0] = preWeaponSpellFormVersion
	return out
}

// preExperienceFormVersion is the version byte the form carries once 0125's
// thirty-four-byte tail is peeled back off — version 34, the value 0124's
// EQUIPMENT section landed at and currentFormVersion sat on before that
// story bumped it to 35. NOT 32: 33 went to 0119 and never lands in this
// file's own byte form, but 34 did land, between the carry section and the
// purse, and it is still THERE in every byte this peel leaves untouched —
// only the entity record's own tail moves. Tagging the peeled buffer 32
// would claim a form that predates the equipment section for bytes that
// still carry it.
//
// strippedOfTheExperience is the inverse of the widening that added it: one
// form with the thirty-four bytes each record grew by cut off its tail and
// the version byte put back. It was the newest peel until 0127's own
// strippedOfTheKnownSpells arrived above it; it runs BEFORE
// strippedOfTheCommandGroup and everything below it — the strippings still
// come off newest story first.
//
// THE PEEL IS STRUCTURAL, NOT A ZERO CHECK, on strippedOfTheCommandGroup's
// own reason: this loader fills none of the five new fields yet (0125's own
// mapload task is later than T2), so it always finds the same thirty-four
// zero bytes at the tail of every record it strips, but the peel removes
// them by offset and width alone regardless.
const preExperienceFormVersion byte = 34

// experienceLen is the width 0125 added to one entity record: six SkillXP
// int32, Mind, XPValue, XPSlot and GainsXP — pkg/sim's own unexported
// entityLen delta, written out here on commandGroupLen's own terms: these
// tests cannot import it, so a change to it fails here loudly rather than
// quietly rewriting what these digests mean.
const experienceLen = 34

func strippedOfTheExperience(form []byte) []byte {
	out := append([]byte(nil), form...)
	n := int(binary.LittleEndian.Uint32(out[entityCountAt : entityCountAt+4]))
	cells := int(binary.LittleEndian.Uint32(out[gridCountAt : gridCountAt+4]))
	base := formHeaderLen + 3*cells
	width := entityRecordLen + 1 + decayBlockLen + 1 + actorStateTailLen + 1 + postLen + regenLen + commandGroupLen + experienceLen
	for i := n - 1; i >= 0; i-- {
		o := base + width*i
		out = append(out[:o+width-experienceLen], out[o+width:]...)
	}
	out[0] = preExperienceFormVersion
	return out
}

// preCommandGroupFormVersion is the version byte the form carried before
// pkg/sim's 0117 COMMAND GROUP existed — version 31, the value 0122's
// check-record widening left currentFormVersion at before this story
// bumped it to 32.
//
// strippedOfTheCommandGroup is the inverse of the widening that added it:
// one form with the four bytes each record grew by cut off its tail and
// the version byte put back. It is no longer the newest peel —
// strippedOfTheExperience above is — but it keeps its place ahead of
// strippedOfTheCarryAndPurse and everything below it: the strippings still
// come off newest story first.
//
// THE PEEL IS STRUCTURAL, NOT A ZERO CHECK, on strippedOfTheCarryAndPurse's
// own reason: this loader gives no placement a command group either, so it
// always finds the same four zero bytes at the tail of every record it
// strips, but the peel removes them by offset and width alone regardless.
const preCommandGroupFormVersion byte = 31

// commandGroupLen is the width the command group added to one entity
// record: one uint32.
const commandGroupLen = 4

func strippedOfTheCommandGroup(form []byte) []byte {
	out := append([]byte(nil), form...)
	n := int(binary.LittleEndian.Uint32(out[entityCountAt : entityCountAt+4]))
	cells := int(binary.LittleEndian.Uint32(out[gridCountAt : gridCountAt+4]))
	base := formHeaderLen + 3*cells
	width := entityRecordLen + 1 + decayBlockLen + 1 + actorStateTailLen + 1 + postLen + regenLen + commandGroupLen
	for i := n - 1; i >= 0; i-- {
		o := base + width*i
		out = append(out[:o+width-commandGroupLen], out[o+width:]...)
	}
	out[0] = preCommandGroupFormVersion
	return out
}

// preCarryFormVersion is the version byte the form carried before pkg/sim's
// 0112 CARRY AND PURSE sections existed — version 25, currentFormVersion's
// own value before that story bumped it to 26.
//
// strippedOfTheCarryAndPurse is the inverse of the pair 0112 added between
// the sack section and the script section: one form with the two blocks
// this story wrote cut out and the version byte put back. It is no longer
// the newest peel — strippedOfTheCommandGroup above is — but it keeps its
// place ahead of strippedOfTheRegen and everything below it: the strippings
// still come off newest story first.
//
// Unlike a per-record tail, the pair sits in the MIDDLE of the form —
// strippedOfTheGroupSection's and strippedOfTheSacks' own shape rather than
// strippedOfThePost's — so the inverse locates it by the records', the
// (empty) routes', the group section's and the sack section's own lengths,
// exactly as strippedOfTheSacks locates its own section.
//
// No placement FromALM builds is ever given anything to carry or any gold —
// mapload writes neither field — so the carry section every fixture in this
// package produces is always a zero count per entity, carryCountLen bytes
// each and no codes, and the purse section is always its own fixed purseLen
// zeroed bytes. That is what lets this peel cut a flat n*carryCountLen+
// purseLen span once it has found where the sack section ends, rather than
// walking each entity's own carried-code count.
//
// WIDENED BY 0124 T2 (pkg/sim): no placement FromALM builds is ever
// equipped either, so the equipment section this story added — between
// the carry section and the purse — is always a flat n*equipRecordLen
// zeroed span too, and the one span this function already cuts widens to
// cover it rather than gaining a peel of its own. The name and the version
// byte stay the pair's own: what this function reconstructs is still "the
// form before pkg/sim's carry-section-shaped sections arrived", and every
// byte this file's chain removes here is disposable in transit anyway — no
// test asserts preCarryFormVersion except by way of the FINAL peel further
// down overwriting it again.
//
// WIDENED AGAIN BY 0127 (pkg/sim): no fixture in this package ever names a
// spell table either, so the section this story added — right after the
// purse — is always its own bare spellCountLen zero count and no records,
// on the equipment section's own precedent immediately above: the span
// this function cuts widens once more rather than gaining a peel of its
// own, and what it reconstructs is still "the form before pkg/sim's
// carry-section-shaped sections arrived".
const preCarryFormVersion byte = 25

// carryCountLen, equipRecordLen, purseLen and spellCountLen are the four
// sections' own fixed widths — pkg/sim's unexported carryCountLen,
// equipRecordLen (EquipSlots*2), purseLen (relationSlots*4) and
// spellCountLen, written out here on sackSectionCountLen's own terms:
// these tests cannot import them, so a change to any one fails here
// loudly rather than quietly rewriting what these digests mean.
const (
	carryCountLen  = 4
	equipRecordLen = 24
	purseLen       = 200
	spellCountLen  = 2
)

func strippedOfTheCarryAndPurse(form []byte) []byte {
	out := append([]byte(nil), form...)
	n := int(binary.LittleEndian.Uint32(out[entityCountAt : entityCountAt+4]))
	cells := int(binary.LittleEndian.Uint32(out[gridCountAt : gridCountAt+4]))
	width := entityRecordLen + 1 + decayBlockLen + 1 + actorStateTailLen + 1 + postLen + regenLen // 145: the record's current full width
	base := formHeaderLen + 3*cells + width*n + 4*n                                               // past the records and the empty routes
	groupCount := int(binary.LittleEndian.Uint32(out[base : base+4]))
	sackBase := base + 4 + 18*groupCount // past the group section
	sackCount := int(binary.LittleEndian.Uint32(out[sackBase : sackBase+4]))
	off := sackBase + 4
	for i := 0; i < sackCount; i++ {
		items := int(binary.LittleEndian.Uint32(out[off+12 : off+16]))
		off += 16 + 2*items
	}
	// off is now past the sack section: the carry section starts here, one
	// zero count per entity, then the equipment section (0124 T2), then the
	// purse section, then the spell table's own bare count (0127).
	carryLen := carryCountLen * n
	equipLen := equipRecordLen * n
	out = append(out[:off], out[off+carryLen+equipLen+purseLen+spellCountLen:]...)
	out[0] = preCarryFormVersion
	return out
}

// preRegenFormVersion is the version byte the form carried before pkg/sim's
// 0109 REGENERATION BLOCK existed, and strippedOfTheRegen is the inverse of
// the widening that added it: one form with the eighteen bytes each record
// grew by cut off its tail and the version byte put back. It is no longer
// the newest peel — strippedOfTheCarryAndPurse above is — but it keeps its
// place ahead of strippedOfThePost, which runs ahead of strippedOfTheReach
// in turn — the strippings still come off newest story first.
//
// THE PEEL IS STRUCTURAL, NOT A ZERO CHECK. Before 0109's own T3 this
// package wired no placement to a period or a mana pool, so it always found
// six zero bytes at the tail of every record it strips; T3 now fills the two
// periods on every path and the mana pair where a row names one, and the
// peel keeps working unchanged because it removes the tail's eighteen bytes
// by offset and width alone, whatever value a loader arm put there.
const preRegenFormVersion byte = 24

// regenLen is the width the regeneration block added to one entity record:
// four int32 and two bytes.
const regenLen = 18

func strippedOfTheRegen(form []byte) []byte {
	out := append([]byte(nil), form...)
	n := int(binary.LittleEndian.Uint32(out[entityCountAt : entityCountAt+4]))
	cells := int(binary.LittleEndian.Uint32(out[gridCountAt : gridCountAt+4]))
	base := formHeaderLen + 3*cells
	width := entityRecordLen + 1 + decayBlockLen + 1 + actorStateTailLen + 1 + postLen + regenLen
	for i := n - 1; i >= 0; i-- {
		o := base + width*i
		out = append(out[:o+width-regenLen], out[o+width:]...)
	}
	out[0] = preRegenFormVersion
	return out
}

// prePostFormVersion is the version byte the form carried before pkg/sim's
// 0106 POST TAIL existed, and strippedOfThePost is the inverse of the
// widening that added it: one form with the eight bytes each record grew by
// cut off its tail and the version byte put back. It is no longer the
// newest peel — strippedOfTheRegen above is — but it keeps its place ahead
// of strippedOfTheReach, which runs ahead of strippedOfTheSacks in turn —
// the strippings still come off newest story first.
//
// NewWorld writes the post from a placement's own cell unconditionally, so
// this peel always finds that placement's own X, Y — never a value a later
// loader arm supplied — at the tail of every record it strips.
const prePostFormVersion byte = 23

// postLen is the width the post tail added to one entity record: two int32.
const postLen = 8

func strippedOfThePost(form []byte) []byte {
	out := append([]byte(nil), form...)
	n := int(binary.LittleEndian.Uint32(out[entityCountAt : entityCountAt+4]))
	cells := int(binary.LittleEndian.Uint32(out[gridCountAt : gridCountAt+4]))
	base := formHeaderLen + 3*cells
	width := entityRecordLen + 1 + decayBlockLen + 1 + actorStateTailLen + 1 + postLen
	for i := n - 1; i >= 0; i-- {
		o := base + width*i
		out = append(out[:o+width-postLen], out[o+width:]...)
	}
	out[0] = prePostFormVersion
	return out
}

// preReachFormVersion is the version byte the form carried before pkg/sim's
// 0104 REACH TAIL existed, and strippedOfTheReach is the inverse of the
// widening that added it: one form with the one byte each record grew by
// cut off its tail and the version byte put back. It is no longer the
// newest peel — strippedOfThePost above is — but it keeps its place ahead
// of strippedOfTheSacks, which runs ahead of everything below it in turn —
// the strippings still come off newest story first.
//
// FromALM builds with no table at all, so no placement here can reach the
// item collections 0104's join reads (T3) whatever its class row would have
// named, and this peel always finds the same byte, 1 — the constructor's own
// floor — at the tail of every record it strips.
const preReachFormVersion byte = 22

func strippedOfTheReach(form []byte) []byte {
	out := append([]byte(nil), form...)
	n := int(binary.LittleEndian.Uint32(out[entityCountAt : entityCountAt+4]))
	cells := int(binary.LittleEndian.Uint32(out[gridCountAt : gridCountAt+4]))
	base := formHeaderLen + 3*cells
	width := entityRecordLen + 1 + decayBlockLen + 1 + actorStateTailLen + 1
	for i := n - 1; i >= 0; i-- {
		o := base + width*i
		out = append(out[:o+width-1], out[o+width:]...)
	}
	out[0] = preReachFormVersion
	return out
}

// preActorStateFormVersion is the version byte the form carried before
// pkg/sim's 0099 ACTOR STATE TAIL existed, and strippedOfTheActorState is the
// inverse of the widening that added it: one form with the eighteen bytes
// each record grew by cut off its tail and the version byte put back.
//
// It WAS the newest peel; pkg/sim's 0103 sack section is, now — see
// strippedOfTheSacks below, which runs ahead of this one — but it keeps its
// place ahead of every peel below it, including the group section's: the
// strippings still come off newest story first, which is what keeps every
// pre-story digest in this file a fixed point rather than a number to be
// replaced whenever a field arrives.
const preActorStateFormVersion byte = 20

// actorStateTailLen is how many bytes 0099 added to each record: the actor
// state byte, the two-cell ring and the leg. No placement FromALM builds is
// ever a patroller, so this tail is guard with an empty ring on every record
// this package's fixtures produce — the same six-field shape
// TestThePinIsThePreStoryPinPlusTheActorState pins in pkg/sim's own
// binary_test.go.
const actorStateTailLen = 18

func strippedOfTheActorState(form []byte) []byte {
	out := append([]byte(nil), form...)
	n := int(binary.LittleEndian.Uint32(out[entityCountAt : entityCountAt+4]))
	cells := int(binary.LittleEndian.Uint32(out[gridCountAt : gridCountAt+4]))
	base := formHeaderLen + 3*cells
	width := entityRecordLen + 1 + decayBlockLen + 1 + actorStateTailLen
	for i := n - 1; i >= 0; i-- {
		o := base + width*i
		out = append(out[:o+width-actorStateTailLen], out[o+width:]...)
	}
	out[0] = preActorStateFormVersion
	return out
}

// preGroupSectionFormVersion is the version byte the form carried before
// pkg/sim's 0095 GROUP SECTION existed — a name distinct from
// preGroupFormVersion above on purpose: that one is the group WORD's own
// fixed point (version 10), and this is the frozen notice-radius record's,
// nine stories newer.
const preGroupSectionFormVersion byte = 18

// strippedOfTheGroupSection is the inverse of that widening, and it is unlike
// every peel below it: the section is not a tail on the entity record and not
// a block closing the form, it is a new COUNTED block of its own sitting
// between the routes and the script section. So the inverse locates it by the
// records' and the (empty) routes' own lengths rather than by a fixed offset
// or a per-record width. It is no longer the newest peel — strippedOfTheActorState
// is, above — but it keeps running before strippedOfTheSight, on a form
// strippedOfTheActorState has already brought back to its own current record
// width, because every peel below it assumes the group section is already gone.
//
// It reads the group section's own count rather than assuming one, so a
// fixture that ever carried an owned entity would still peel correctly; every
// fixture this package builds today carries none, and every entity built by
// FromALM (never stepped before these tests marshal it) holds no route, so
// the routes section is a plain 4-byte zero count per entity — which is what
// lets the group section's start be computed without parsing a route.
func strippedOfTheGroupSection(form []byte) []byte {
	out := append([]byte(nil), form...)
	n := int(binary.LittleEndian.Uint32(out[entityCountAt : entityCountAt+4]))
	cells := int(binary.LittleEndian.Uint32(out[gridCountAt : gridCountAt+4]))
	width := entityRecordLen + 1 + decayBlockLen + 1 // 100: the record's current full width
	base := formHeaderLen + 3*cells + width*n + 4*n  // past the records and the empty routes
	count := int(binary.LittleEndian.Uint32(out[base : base+4]))
	// The record is 18 bytes since pkg/sim's 0096 (owner, group, base, order,
	// commandedX, commandedY) — grown in place from the 9 it opened at.
	out = append(out[:base], out[base+4+18*count:]...)
	out[0] = preGroupSectionFormVersion
	return out
}

// preSackSectionFormVersion is the version byte the form carried before
// pkg/sim's 0103 SACK SECTION existed — version 21, 0099's own landed
// state.
//
// sackSectionCountLen is the width of that section's own record count.
// Every fixture in this package places no ground record, so the section
// every one of them writes is this many bytes and no record.
const (
	preSackSectionFormVersion byte = 21
	sackSectionCountLen            = 4
)

// strippedOfTheSacks is the inverse of that widening. It WAS the newest
// peel; pkg/sim's 0104 reach tail is, now — see strippedOfTheReach above,
// which runs ahead of this one — but it keeps its place ahead of
// strippedOfTheActorState and everything below it, which is what pushed
// that one's own place in the chain down by one when THIS section arrived.
// Like
// strippedOfTheGroupSection, the section is a new COUNTED block rather than
// a tail or a form-closing block, sitting between the group section and the
// script section, so its own start is found past the group section's own
// (possibly owned) records rather than by a fixed offset.
//
// Unlike strippedOfTheGroupSection, a sack record is itself variable-width —
// an item count and that many codes — so this cannot multiply a declared
// count by one fixed record width; it walks each sack's own head to find the
// next one, on decodeSacks' own rule. Every fixture in this package places
// no ground record, so that walk runs zero times either way.
func strippedOfTheSacks(form []byte) []byte {
	out := append([]byte(nil), form...)
	n := int(binary.LittleEndian.Uint32(out[entityCountAt : entityCountAt+4]))
	cells := int(binary.LittleEndian.Uint32(out[gridCountAt : gridCountAt+4]))
	width := entityRecordLen + 1 + decayBlockLen + 1 + actorStateTailLen // 118: the record's current full width
	base := formHeaderLen + 3*cells + width*n + 4*n                      // past the records and the empty routes
	groupCount := int(binary.LittleEndian.Uint32(out[base : base+4]))
	sackBase := base + 4 + 18*groupCount // past the group section
	sackCount := int(binary.LittleEndian.Uint32(out[sackBase : sackBase+4]))
	off := sackBase + 4
	for i := 0; i < sackCount; i++ {
		items := int(binary.LittleEndian.Uint32(out[off+12 : off+16]))
		off += 16 + 2*items
	}
	out = append(out[:sackBase], out[off:]...)
	out[0] = preSackSectionFormVersion
	return out
}

func strippedOfTheSight(form []byte) []byte {
	out := append([]byte(nil), form...)
	n := int(binary.LittleEndian.Uint32(out[entityCountAt : entityCountAt+4]))
	cells := int(binary.LittleEndian.Uint32(out[gridCountAt : gridCountAt+4]))
	base := formHeaderLen + 3*cells
	width := entityRecordLen + 1 + decayBlockLen + 1
	for i := n - 1; i >= 0; i-- {
		o := base + width*i
		out = append(out[:o+width-1], out[o+width:]...)
	}
	out[0] = preSightFormVersion
	return out
}

// preDecayFormVersion is the version byte the form carried before THE DECAY
// LADDER existed, and strippedOfTheDecayBlock is the inverse of the widening that
// added it: one form with the seven bytes each record grew by cut off its tail
// and the version byte put back. It runs on the peel above's output.
const preDecayFormVersion byte = 16

// decayBlockLen is how many bytes the decay ladder added to each record: the
// stage, the dwell and the dying time.
const decayBlockLen = 7

func strippedOfTheDecayBlock(form []byte) []byte {
	out := append([]byte(nil), form...)
	n := int(binary.LittleEndian.Uint32(out[entityCountAt : entityCountAt+4]))
	cells := int(binary.LittleEndian.Uint32(out[gridCountAt : gridCountAt+4]))
	base := formHeaderLen + 3*cells
	width := entityRecordLen + 1 + decayBlockLen
	for i := n - 1; i >= 0; i-- {
		o := base + width*i
		out = append(out[:o+entityRecordLen+1], out[o+width:]...)
	}
	out[0] = preDecayFormVersion
	return out
}

// preRelationFormVersion is the version byte the form carried before THE
// RELATION existed, and strippedOfTheRelation is the inverse of the widening
// that added it: the block closes the form, so the inverse is a truncation and
// the version byte put back.
const preRelationFormVersion byte = 14

func strippedOfTheRelation(form []byte) []byte {
	out := append([]byte(nil), form[:len(form)-formRelationLen]...)
	out[0] = preRelationFormVersion
	return out
}

// preFacingFormVersion is the version byte the form carried before the FACING
// existed, and strippedOfTheFacing is the inverse of the widening that added it:
// one form with the one byte each record grew by cut off its tail and the version
// byte put back.
//
// It is the NEWEST peel, so it runs before the planes come off and before either
// word does — the strippings come off newest story first, which is what keeps
// every pre-story digest in this file a fixed point rather than a number to be
// replaced whenever a field arrives.
const preFacingFormVersion byte = 13

func strippedOfTheFacing(form []byte) []byte {
	out := append([]byte(nil), form...)
	n := int(binary.LittleEndian.Uint32(out[entityCountAt : entityCountAt+4]))
	cells := int(binary.LittleEndian.Uint32(out[gridCountAt : gridCountAt+4]))
	base := formHeaderLen + 3*cells
	for i := n - 1; i >= 0; i-- {
		o := base + (entityRecordLen+1)*i
		out = append(out[:o+entityRecordLen], out[o+entityRecordLen+1:]...)
	}
	out[0] = preFacingFormVersion
	return out
}

func strippedOfThePlanes(form []byte) []byte {
	cells := int(binary.LittleEndian.Uint32(form[gridCountAt : gridCountAt+4]))
	out := append([]byte(nil), form[:formHeaderLen+cells]...)
	out = append(out, form[formHeaderLen+3*cells:]...)
	out[0] = preCostFormVersion
	return out
}

// strippedOfTheOwnerAndGroupWords is the inverse of the two widenings that have
// been made to this record since the digests in this file were taken: one form,
// minus the four bytes the owner story added and the four the group story added
// before it, with the version byte put back, hashed the way a world hashes
// itself. It is what keeps every pre-story digest in this file usable as a fixed
// point rather than as a number to be replaced.
//
// The two words come off in REVERSE ORDER of arrival, so each removal leaves the
// earlier one at the offset the contract gives it.
//
// structs is the caller's own w.Structures() count, on
// strippedOfTheStructureSection's own reason (1033 B3): pinMap's placements
// are the one fixture this function ever runs on that puts anything in the
// section, so the count is not assumed zero.
func strippedOfTheOwnerAndGroupWords(t *testing.T, form []byte, structs int) uint64 {
	t.Helper()
	out := strippedOfThePlanes(strippedOfTheFacing(strippedOfTheRelation(strippedOfTheDecayBlock(strippedOfTheSight(strippedOfTheGroupSection(strippedOfTheActorState(strippedOfTheSacks(strippedOfTheReach(strippedOfThePost(strippedOfTheRegen(strippedOfTheCarryAndPurse(strippedOfTheCommandGroup(strippedOfTheExperience(strippedOfTheKnownSpells(strippedOfTheSkill(strippedOfTheWeaponSpell(strippedOfDeathGold(strippedOfTheSpellState(strippedOfTheScriptItem(strippedOfTheOffMap(strippedOfTheCasting(strippedOfTheLastArms(strippedOfSpellEffects(strippedOfCorpseLoot(strippedOfItemAttribution(strippedOfCarriedWeight(strippedOfTheMapUnitID(strippedOfTheStructureSection(strippedOfWeaponResistance(form), structs)))))))))))))))))))))))))))))
	n := int(binary.LittleEndian.Uint32(out[entityCountAt : entityCountAt+4]))
	cells := int(binary.LittleEndian.Uint32(out[gridCountAt : gridCountAt+4]))
	base := formHeaderLen + cells
	for i := n - 1; i >= 0; i-- {
		o := base + entityRecordLen*i
		out = append(out[:o+ownerWordAt], out[o+entityRecordLen:]...)
		out = append(out[:o+groupWordAt], out[o+ownerWordAt:]...)
	}
	out[0] = preGroupFormVersion
	h := fnv.New64a()
	h.Write(out)
	return h.Sum64()
}

// TestTheNewPinsAreTheOldPinsPlusTheEight is AC-8's second half and SC-7's, over
// both pinned fixtures at once.
//
// Every placement on both of them resolves to nothing, so the only bytes this
// story can have written are a charge of 8 and a relax of 4 in each record. That
// is asserted here as bytes, beside the digest, because the digest alone would
// hold just as well if the story had written those numbers somewhere else and
// zeroed something it should not have.
func TestTheNewPinsAreTheOldPinsPlusTheEight(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		world  *sim.World
		zeroed uint64
	}{
		{"fixtureMap", mapload.FromALM(fixtureMap()), zeroedPreStoryDigest},
		{"pinMap", mapload.FromALM(pinMap()), zeroedPinDigest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := zeroedEightDigest(t, tc.world); got != tc.zeroed {
				t.Errorf("zeroing the eight fields hashes %#016x, want the pre-story %#016x — "+
					"this story moved a byte outside them", got, tc.zeroed)
			}
			for _, e := range tc.world.Entities() {
				if e.AttackCharge != 8 || e.AttackRelax != 4 {
					t.Errorf("entity %d carries cadence %d/%d, want the constructor's 8/4",
						e.ID, e.AttackCharge, e.AttackRelax)
				}
				if e.ToHit != 0 || e.Defence != 0 || e.Absorption != 0 ||
					e.DamageBase != 0 || e.DamageSpread != 0 || e.AlwaysHits {
					t.Errorf("entity %d resolves to nothing and carries %+v, want six zeros", e.ID, e)
				}
			}
		})
	}
}

// AC-8 — the three table shapes that must move nothing, and the census running
// with no install.
func TestStructurePassLeavesATablelessWorldWhereItWas(t *testing.T) {
	t.Parallel()

	m := pinMap()
	empty := &mapload.Table{Units: defCollection{}, Humans: defCollection{}}

	worlds := map[string]*sim.World{}
	worlds["no table"] = mapload.FromALM(m)
	nilTable, err := mapload.FromALMWith(m, nil, mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("nil table: %v", err)
	}
	worlds["nil table"] = nilTable
	noBuildings, err := mapload.FromALMWith(m, empty, mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("table without Buildings: %v", err)
	}
	worlds["Buildings absent"] = noBuildings

	base := mapload.Passability(m)
	for name, w := range worlds {
		if w == nil {
			t.Fatalf("%s: no world", name)
		}
		// Through the two record widenings' own inverse, so the literal stays the
		// one that predates them: three table shapes that move nothing must still
		// strip back to exactly the same pre-story form.
		form, err := w.MarshalBinary()
		if err != nil {
			t.Fatalf("%s: MarshalBinary: %v", name, err)
		}
		if got := strippedOfTheOwnerAndGroupWords(t, form, len(w.Structures())); got != pinDigest {
			t.Errorf("%s: world strips to %#016x, want the pre-story %#016x", name, got, pinDigest)
		}
	}
	for name, plane := range map[string][]byte{
		"nil table":        mapload.PassabilityWith(m, nil),
		"empty table":      mapload.PassabilityWith(m, &mapload.Table{}),
		"Buildings absent": mapload.PassabilityWith(m, empty),
	} {
		if len(plane) != len(base) {
			t.Fatalf("%s: plane is %d bytes, want %d", name, len(plane), len(base))
		}
		for i := range base {
			if plane[i] != base[i] {
				t.Fatalf("%s: byte %d is %#02x, want the pre-story %#02x", name, i, plane[i], base[i])
			}
		}
	}

	// And the wiring reaches the world: with a Buildings collection the same
	// map builds a DIFFERENT world, or nothing above this tier would ever see
	// the pass at all.
	full := &mapload.Table{
		Units:     defCollection{},
		Humans:    defCollection{},
		Buildings: bldTable(map[int][]int32{1: bldRow(1, 1, 0, 1)}).Buildings,
	}
	built, err := mapload.FromALMWith(m, full, mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("table with Buildings: %v", err)
	}
	if built.Hash() == pinDigest {
		t.Errorf("a table carrying Buildings built the table-less world — the pass does not reach FromALMWith")
	}

	// The census, headless, with no install present: both levels reported.
	c := mapload.StructureCensus(m, full)
	if c.Resolved == 0 || c.Unresolved == 0 {
		t.Errorf("census %+v: want both a resolved and an unresolved placement", c)
	}
	if c.Attached == 0 || c.Opened == 0 {
		t.Errorf("census %+v: want both cell-level directions reported", c)
	}
}

// domainMap places one unit per domain case of AC-1, on five distinct keys.
//
// The keys are the low bytes the search truncates to, and every one of them is
// at or above the humans bound, so all five take the units arm and the arm is
// not what this fixture varies. The cells are inside the extent and clear of the
// border ring, so no placement is refused for a reason unrelated to its column.
func domainMap() *alm.Map {
	return &alm.Map{
		Width: 40, Height: 40,
		Units: []alm.Unit{
			{X: 0x0C80, Y: 0x0C80, ClassID: 0x40},
			{X: 0x0D80, Y: 0x0C80, ClassID: 0x41},
			{X: 0x0E80, Y: 0x0C80, ClassID: 0x42},
			{X: 0x0F80, Y: 0x0C80, ClassID: 0x43},
			{X: 0x1080, Y: 0x0C80, ClassID: 0x44},
		},
	}
}

// TestTheDomainColumnDecidesTheMoverSDomain is AC-1 and SC-1.
//
// The five rows are the contract's five cases in its own order: the three codes,
// an empty cell, and a value outside them. The wanted domains are written as the
// simulation's own constants rather than as the column's numbers, which is the
// whole of what the mapping has to get right — the two numberings differ, and a
// loader that cast one to the other would make every ground unit a ghost.
func TestTheDomainColumnDecidesTheMoverSDomain(t *testing.T) {
	tbl := &mapload.Table{Units: defCollection{
		{},
		{name: "ground", params: domainRow(0x40, 1)},
		{name: "ghost", params: domainRow(0x41, 2)},
		{name: "air", params: domainRow(0x42, 3)},
		{name: "empty-cell", params: unitDefRow(0x43, 0, 30)},
		{name: "outside", params: domainRow(0x44, 4)},
	}}

	w, err := mapload.FromALMWith(domainMap(), tbl, mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}

	want := []sim.Domain{
		sim.DomainGround, sim.DomainGhost, sim.DomainAir, sim.DomainGround, sim.DomainGround,
	}
	ents := w.Entities()
	if len(ents) != len(want) {
		t.Fatalf("the world holds %d entities for %d placements", len(ents), len(want))
	}
	for i, e := range ents {
		if e.Domain != want[i] {
			t.Errorf("placement %d landed in domain %d, want %d", i, e.Domain, want[i])
		}
	}
}

// domainRow is a units row keyed on typeID with a domain column written into it.
//
// Face is written as 0 and not left empty: an empty cell takes the constructor's
// 1, and the placements above carry no subkey, so a row that let the default
// stand would fail the second half of the units search and resolve to nothing.
func domainRow(typeID, movement int32) []int32 {
	return defRow(map[int]int32{
		slotUnitType: typeID, slotUnitFace: 0, slotMovement: movement, slotHealthMax: 30})
}

func TestOnlyAMatchedUnitsEntryCanYieldANonGroundMover(t *testing.T) {
	m := &alm.Map{
		Width: 40, Height: 40,
		Units: []alm.Unit{
			{X: 0x0C80, Y: 0x0C80, ClassID: 7, Flags: 1},   // npc
			{X: 0x0D80, Y: 0x0C80, ClassID: 7, DefID: 900}, // server id
			{X: 0x0E80, Y: 0x0C80, ClassID: 7},             // humans, by type
			{X: 0x0F80, Y: 0x0C80, ClassID: 0x55},          // units, no match
			{X: 0x1080, Y: 0x0C80, ClassID: 0x40},          // units, matched
		},
	}
	tbl := &mapload.Table{
		Units:  defCollection{{}, {name: "flyer", params: domainRow(0x40, 3)}},
		Humans: defCollection{{}, {name: "k7", params: defRow(map[int]int32{slotHumanType: 7, slotServerID: 900})}},
	}

	w, err := mapload.FromALMWith(m, tbl, mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}

	ents := w.Entities()
	// The DOMAIN half is the whole of this test's question and it covers all
	// four: no arm but a matched units entry may yield a non-ground mover, and
	// the humans arm has its own movement column which this row leaves empty.
	for i := 0; i < 4; i++ {
		if ents[i].Domain != sim.DomainGround {
			t.Errorf("placement %d took an arm that cannot resolve a units entry, yet landed in domain %d",
				i, ents[i].Domain)
		}
	}
	// The HEALTH half now splits, and 0087 split it. Placement 0 carries the
	// npc flag and this table holds no npc registry, so its rung reaches
	// nothing at all; placement 3's key names no units row. Both keep the
	// provisional pair, and 0 does so for a reason that has nothing to do with
	// the collection it would have searched.
	//
	// 70 is the same all-default derivation TestAWorldBuiltWithATableCarriesTheResolvedHealth
	// pins for the same shape of row (type id and, here, a server id — neither
	// reaches the graph): the row is empty everywhere else, so it was the
	// constructor's health column, 30, before this story and is
	// HumanDef.DerivedMaximum's own answer now.
	const humansHealth = int32(70)
	for _, i := range []int{1, 2} {
		if ents[i].HP != humansHealth || ents[i].MaxHP != humansHealth {
			t.Errorf("placement %d is at %d/%d, want the humans row's own %d",
				i, ents[i].HP, ents[i].MaxHP, humansHealth)
		}
	}
	for _, i := range []int{0, 3} {
		if ents[i].HP != mapload.SpawnHP || ents[i].MaxHP != mapload.SpawnHP {
			t.Errorf("placement %d is at %d/%d, want the provisional pair", i, ents[i].HP, ents[i].MaxHP)
		}
	}
	if ents[4].Domain != sim.DomainAir {
		t.Errorf("the matched units placement landed in domain %d, want air", ents[4].Domain)
	}
}

// TestTwoClassesTakeTheirOwnUnscaledMaxima is AC-10 and SC-8: the second thing
// the table now delivers to the running game, at the difficulty it uses.
//
// The two maxima are distinct and neither is the provisional constant, so a
// build that resolved nothing, resolved both to one row, or scaled either would
// fail here rather than pass on a number that looked plausible.
func TestTwoClassesTakeTheirOwnUnscaledMaxima(t *testing.T) {
	m := &alm.Map{
		Width: 40, Height: 40,
		Units: []alm.Unit{
			{X: 0x0C80, Y: 0x0C80, ClassID: 0x40},
			{X: 0x0D80, Y: 0x0C80, ClassID: 0x41},
		},
	}
	tbl := &mapload.Table{Units: defCollection{
		{},
		{name: "weak", params: unitDefRow(0x40, 0, 37)},
		{name: "strong", params: unitDefRow(0x41, 0, 211)},
	}}

	w, err := mapload.FromALMWith(m, tbl, mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}

	for i, want := range []int32{37, 211} {
		e := w.Entities()[i]
		if e.HP != want || e.MaxHP != want {
			t.Errorf("placement %d is at %d/%d, want %d/%d", i, e.HP, e.MaxHP, want, want)
		}
		if want == mapload.SpawnHP {
			t.Fatalf("the fixture chose the provisional constant for placement %d; it proves nothing", i)
		}
	}
}

func TestTwoWorldsFromOneMapAndOneTableStayIdentical(t *testing.T) {
	tbl := &mapload.Table{Units: defCollection{
		{},
		{name: "ground", params: domainRow(0x40, 1)},
		{name: "ghost", params: domainRow(0x41, 2)},
		{name: "air", params: domainRow(0x42, 3)},
	}}

	build := func() *sim.World {
		w, err := mapload.FromALMWith(domainMap(), tbl, mapload.DifficultyNormal)
		if err != nil {
			t.Fatalf("FromALMWith: %v", err)
		}
		return w
	}
	a, b := build(), build()

	cmds := []sim.Command{
		{Entity: 0, X: 30, Y: 30}, {Entity: 1, X: 30, Y: 31}, {Entity: 2, X: 30, Y: 32},
	}
	for tick := 0; tick < 40; tick++ {
		sim.Step(a, cmds)
		sim.Step(b, cmds)
		cmds = nil
		if a.Hash() != b.Hash() {
			t.Fatalf("the two worlds diverged at tick %d: %#016x vs %#016x", tick, a.Hash(), b.Hash())
		}
	}

	fa, err := a.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	fb, err := b.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if !bytes.Equal(fa, fb) {
		t.Error("the two worlds marshalled to different bytes")
	}
}

// ---------------------------------------------------------------------------
// AC-1 — an entity carries the group its map placed it in.
// ---------------------------------------------------------------------------

// TestEveryEntityCarriesItsRecordsGroup is AC-1's first half. The group ids are
// written out as literals for the reason the class ids in wantEntities are: read
// off the fixture's own field, this table would agree with whatever the loader
// happened to copy, the neighbouring record's word included.
//
// The zero one is deliberate and is not padding: it is a real group, and a
// loader that treated it as "no group" and substituted something would be caught
// here rather than at the point a check counted its members.
func TestEveryEntityCarriesItsRecordsGroup(t *testing.T) {
	t.Parallel()

	m := &alm.Map{
		Width:  8,
		Height: 8,
		Units: []alm.Unit{
			{X: 0x0100, Y: 0x0100, ClassID: 7, UnitID: 1, GroupID: 3},
			{X: 0x0200, Y: 0x0100, ClassID: 7, UnitID: 2, GroupID: 3},
			{X: 0x0300, Y: 0x0100, ClassID: 7, UnitID: 3, GroupID: 0},
			{X: 0x0400, Y: 0x0100, ClassID: 7, UnitID: 4, GroupID: 4294967295},
		},
	}
	want := []uint32{3, 3, 0, 4294967295}
	for i, e := range mapload.FromALM(m).Entities() {
		if e.Group != want[i] {
			t.Errorf("entity %d carries group %d, want %d", i, e.Group, want[i])
		}
	}
}

// TestAWorldBuiltFromNoMapCarriesTheZeroGroup is AC-1's second half. There is no
// record to take a group from, so every entity is in group zero — which is a
// group, and the same one a placed unit whose record says 0 is in. Nothing about
// that is a defect: a check naming a group no entity carries answers zero, and a
// check naming zero over such a world answers the count of the world.
func TestAWorldBuiltFromNoMapCarriesTheZeroGroup(t *testing.T) {
	t.Parallel()

	for _, e := range mapload.FromALM(fixtureMap()).Entities() {
		if e.Group != 0 {
			t.Errorf("entity %d of a map that names no group carries group %d, want 0", e.ID, e.Group)
		}
	}
	w := mapload.FromALM(nil)
	if n := len(w.Entities()); n != 0 {
		t.Errorf("a world built from no map holds %d entities, want none", n)
	}
}

// ---------------------------------------------------------------------------
// AC-2 — an entity carries the OWNER its map placed it under.
// ---------------------------------------------------------------------------

// TestEveryEntityCarriesItsRecordsOwner is AC-2's first half. The owner slots
// are written out as literals for the reason the group ids above are: read off
// the fixture's own field, this table would agree with whatever the loader
// happened to copy, the neighbouring record's word included.
//
// The zero one is the opposite of the group table's zero and that is the whole
// point of it being here. A group's zero is a real group; a roster slot's zero
// is NO SLOT, because the space is 1-based. So a loader that substituted the
// first roster entry for it — the plausible mistake, since slot 1 is what a map
// gives the player — is caught here, and the two neighbouring fields are made to
// disagree about what their zero means, deliberately.
//
// The group words differ from the owner words on every record, so an owner read
// out of GroupID would land on a value this table names elsewhere.
func TestEveryEntityCarriesItsRecordsOwner(t *testing.T) {
	t.Parallel()

	m := &alm.Map{
		Width:  8,
		Height: 8,
		Units: []alm.Unit{
			{X: 0x0100, Y: 0x0100, ClassID: 7, UnitID: 1, GroupID: 3, Owner: 1},
			{X: 0x0200, Y: 0x0100, ClassID: 7, UnitID: 2, GroupID: 3, Owner: 2},
			{X: 0x0300, Y: 0x0100, ClassID: 7, UnitID: 3, GroupID: 9, Owner: 0},
			{X: 0x0400, Y: 0x0100, ClassID: 7, UnitID: 4, GroupID: 5, Owner: 4294967295},
		},
	}
	want := []uint32{1, 2, 0, 4294967295}
	ents := mapload.FromALM(m).Entities()
	if len(ents) != len(want) {
		t.Fatalf("the world holds %d entities, want %d", len(ents), len(want))
	}
	for i, e := range ents {
		if e.Owner != want[i] {
			t.Errorf("entity %d carries owner %d, want %d", i, e.Owner, want[i])
		}
	}
}

// TestAWorldBuiltFromNoMapCarriesNoOwner is AC-2's second half, and it is the
// half the contract states as a NEGATIVE: no path silently gives an entity the
// first roster entry.
//
// Three ways of reaching a world without a placement's owner are asked at once —
// a map whose records name none, a world built from no map at all, and the party
// placement one tier up, which is where the hero comes from. Each must be zero,
// which is nobody, and none may be 1.
func TestAWorldBuiltFromNoMapCarriesNoOwner(t *testing.T) {
	t.Parallel()

	for _, e := range mapload.FromALM(fixtureMap()).Entities() {
		if e.Owner != 0 {
			t.Errorf("entity %d of a map that names no owner carries owner %d, want 0", e.ID, e.Owner)
		}
	}
	if n := len(mapload.FromALM(nil).Entities()); n != 0 {
		t.Errorf("a world built from no map holds %d entities, want none", n)
	}
}

// TestTheOwnerSurvivesTheFormAndReachesTheDigest is AC-7's loader-side half: the
// word a map placed reaches the byte form, comes back out of it, and two worlds
// that differ only in it are different worlds.
//
// The digest half matters more than it looks. Nothing in this build reads an
// owner, so every behavioural test in the tree would pass with the field dropped
// on the floor between the loader and the form. This is the assertion that does
// not care what the digest IS — only that it moves.
func TestTheOwnerSurvivesTheFormAndReachesTheDigest(t *testing.T) {
	t.Parallel()

	build := func(o uint32) *sim.World {
		return mapload.FromALM(&alm.Map{
			Width: 8, Height: 8,
			Units: []alm.Unit{
				{X: 0x0100, Y: 0x0100, ClassID: 7, UnitID: 1, Owner: o},
				{X: 0x0200, Y: 0x0100, ClassID: 7, UnitID: 2, Owner: 2},
			},
		})
	}
	a, b := build(1), build(3)
	if a.Hash() == b.Hash() {
		t.Errorf("two worlds differing only in one placement's owner hash alike (%#016x) — "+
			"the field does not reach the byte form", a.Hash())
	}
	if build(1).Hash() != a.Hash() {
		t.Errorf("one world built twice hashes differently")
	}

	form, err := a.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back sim.World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if got, want := back.Entities(), a.Entities(); !reflect.DeepEqual(got, want) {
		t.Errorf("the round trip changed the entities:\n got %+v\nwant %+v", got, want)
	}
}

// ---------------------------------------------------------------------------
// 0086 T4 / AC-15 — the map authors the world's relation.
//
// The store is the loader's and not the format's, so every assertion here is
// about arithmetic: which slot a record's row lands on, which column each of its
// sixteen words takes, what becomes of the byte above the one that is kept, and
// what the diagonal ends up as whatever the file said.
//
// The relation is read out of the BYTE FORM rather than through the world's own
// accessor wherever a row's untouched half is the point. sim's accessor answers
// zero for a cell the matrix does not hold, so it cannot tell "column 0 was never
// written" from "column 0 cannot be addressed"; the form's block is the store
// itself, 2500 bytes at stride 50, and a write anywhere in it is visible.
// ---------------------------------------------------------------------------

// relSlots and relBlockLen mirror pkg/sim's relation geometry. They are written
// out here because it is unexported there, and because a test that took the
// number from the thing it measures would measure nothing.
const (
	relSlots    = 50
	relBlockLen = relSlots * relSlots
)

// relBlock is the world's relation before the clock and optional saved-Group,
// Structure and Player suffixes, row-major at stride relSlots.
//
// It ends the form, which is what makes a tail slice the whole of the
// extraction; that it ends there is pinned separately in gridform_test.go.
func relBlock(t *testing.T, w *sim.World) []byte {
	t.Helper()
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if len(form) < relBlockLen {
		t.Fatalf("the byte form is %d bytes, shorter than the relation block alone", len(form))
	}
	form = strippedOfPlayerCarriers(form)
	end := len(form)
	for range 2 { // Structure79 then Group78, each variable-sized
		span := int(binary.LittleEndian.Uint32(form[end-4:]))
		end -= 4 + span
	}
	end -= 5 // session clock77
	return form[end-relBlockLen : end]
}

// rosterRow builds one type-5 roster entry carrying row.
func rosterRow(name string, row [16]uint16) alm.Group {
	return alm.Group{Name: name, Relation: row}
}

func TestTheRosterAuthorsTheWorldsRelation(t *testing.T) {
	t.Parallel()

	var first, second [16]uint16
	first[0] = 0       // slot 1 -> slot 1: the diagonal, and the file spells it 0
	first[1] = 1       // slot 1 -> slot 2: hostile
	first[2] = 3       // slot 1 -> slot 3: hostile, with the locked bit also set
	first[15] = 2      // slot 1 -> slot 16: the last column a row can reach
	second[0] = 1      // slot 2 -> slot 1: hostile
	second[1] = 5      // slot 2 -> slot 2: the diagonal, spelled hostile, and forced away
	second[2] = 0x0201 // low byte 1 under a high byte of 2 — a narrowing keeps the 1
	second[3] = 0xff00 // low byte 0 under a nonzero word — a wider store keeps this

	w := mapload.FromALM(&alm.Map{
		Width: 8, Height: 8,
		Groups: []alm.Group{rosterRow("Self", first), rosterRow("Monsters", second)},
	})

	want := make([]byte, relBlockLen)
	for k := 0; k < 16; k++ {
		want[1*relSlots+k+1] = byte(first[k])
		want[2*relSlots+k+1] = byte(second[k])
	}
	want[1*relSlots+1] = 2 // the forced diagonals, after the row and over it
	want[2*relSlots+2] = 2

	if got := relBlock(t, w); !bytes.Equal(got, want) {
		if i := firstDifference(got, want); i >= 0 {
			t.Fatalf("the relation block differs at byte %d ([%d][%d]): got %d, want %d",
				i, i/relSlots, i%relSlots, got[i], want[i])
		}
		t.Fatal("the relation block differs from the store the roster authors")
	}

	// The same statement through the world's own reader, which is what a rule
	// actually consults: the direction the file authored is the direction that
	// holds, and the two diagonals that were forced hold neither way.
	rel := w.Relations()
	for _, c := range []struct {
		from, to uint32
		want     bool
	}{
		{1, 2, true}, {2, 1, true}, {1, 3, true}, {3, 1, false},
		{1, 1, false}, {2, 2, false}, {1, 16, false}, {2, 4, false},
	} {
		if got := rel.Hostile(c.from, c.to); got != c.want {
			t.Errorf("Hostile(%d, %d) = %v, want %v", c.from, c.to, got, c.want)
		}
	}
}

// TestNoRosterAuthorsNothing is AC-15's other half. A map with no type-5 record
// is not a roster of empty rows: it writes no row at all, so not even a diagonal
// is forced, and the world is the all-zero one every world built before this
// story existed was.
func TestNoRosterAuthorsNothing(t *testing.T) {
	t.Parallel()

	zero := make([]byte, relBlockLen)
	for _, c := range []struct {
		what string
		w    *sim.World
	}{
		{"a map with no type-5 roster", mapload.FromALM(&alm.Map{Width: 8, Height: 8})},
		{"no map at all", mapload.FromALM(nil)},
		{"the fixture", mapload.FromALM(fixtureMap())},
	} {
		if got := relBlock(t, c.w); !bytes.Equal(got, zero) {
			t.Errorf("%s authored %d relation byte(s), want none",
				c.what, len(got)-bytes.Count(got, []byte{0}))
		}
	}
}

// TestARosterSlotOutsideTheMatrixContributesNothing is the fence: the engine's
// store is a byte write into a fixed block, so a roster longer than the matrix
// writes its excess rows nowhere and the map still loads.
//
// It is asked as an EQUALITY against the shorter roster rather than as "row 50
// is absent", because row 50 is not addressable and its absence is therefore not
// observable. What is observable is that two loud extra records changed nothing
// anywhere in the block.
func TestARosterSlotOutsideTheMatrixContributesNothing(t *testing.T) {
	t.Parallel()

	var loud [16]uint16
	for k := range loud {
		loud[k] = 0xffff
	}
	inside := make([]alm.Group, relSlots-1) // slots 1..49, every one addressable
	for i := range inside {
		inside[i] = rosterRow("g", loud)
	}
	over := append(append([]alm.Group(nil), inside...),
		rosterRow("slot 50", loud), rosterRow("slot 51", loud))

	build := func(gs []alm.Group) *sim.World {
		return mapload.FromALM(&alm.Map{Width: 8, Height: 8, Groups: gs})
	}
	short, long := relBlock(t, build(inside)), relBlock(t, build(over))
	if !bytes.Equal(short, long) {
		i := firstDifference(short, long)
		t.Fatalf("two roster records past the matrix wrote at byte %d ([%d][%d])",
			i, i/relSlots, i%relSlots)
	}
	// And the last row that IS addressable was written, so the equality above is
	// not two empty blocks agreeing.
	if short[49*relSlots+49] != 2 {
		t.Errorf("slot 49's forced diagonal is %d, want 2 — the roster reached no row at all",
			short[49*relSlots+49])
	}
}

// TestARosterWordReachesTheDigest is the assertion that does not care what the
// digest is, only that it moves. Two maps differing in one roster word are two
// worlds; one map built twice is one world.
func TestARosterWordReachesTheDigest(t *testing.T) {
	t.Parallel()

	build := func(v uint16) *sim.World {
		var row [16]uint16
		row[3] = v
		return mapload.FromALM(&alm.Map{
			Width: 8, Height: 8,
			Groups: []alm.Group{rosterRow("Self", row)},
		})
	}
	a, b := build(0), build(1)
	if a.Hash() == b.Hash() {
		t.Errorf("two worlds differing only in one roster word hash alike (%#016x) — "+
			"the row does not reach the byte form", a.Hash())
	}
	if build(0).Hash() != a.Hash() {
		t.Error("one map built twice hashes differently")
	}
}

// TestALoadedMapFightsWithoutBeingTold is the story's point measured from the
// loader rather than from a hand-built world: nothing is commanded, nothing is
// ordered, and the map's own roster is the whole of what starts the fight.
//
// The relation is ONE WAY on purpose. Slot 2 is hostile to slot 1 and slot 1 is
// hostile to nobody, so the placement that acquires is the one the file said
// would, and a symmetric store — which every map in the corpus would forgive —
// fails here.
//
// SINCE 0108 THE ASYMMETRY IS READ OFF THE MATRIX AND NOT OFF WHO ACQUIRED.
// A connecting blow now flips both of a struck pair's cells, so slot 1 does
// turn hostile to slot 2 the moment it is hit, and by tick 64 both placements
// hold a target. That is the story working, not the store leaking: the store
// is checked directly, before a single tick, which is a stricter reading of
// the same property than the acquisition it used to be inferred from — a
// symmetric store fails it on tick zero instead of surviving to tick 64.
// The retaliation is then asserted rather than forbidden, because on a loaded
// map it is what a player watches: strike a neutral and his faction turns.
//
// Slot 1 stands its ground and every other slot guards, so the acquiring
// side has to be the one at slot 2: a stand-ground group vetoes a candidate
// past reach, and these two stand two cells apart.
func TestALoadedMapFightsWithoutBeingTold(t *testing.T) {
	t.Parallel()

	var self, monsters [16]uint16
	monsters[0] = 1 // slot 2 -> slot 1: hostile, and nothing the other way

	w := mapload.FromALM(&alm.Map{
		Width: 40, Height: 40,
		Groups: []alm.Group{rosterRow("Self", self), rosterRow("Monsters", monsters)},
		Units: []alm.Unit{
			{X: 20 << 8, Y: 20 << 8, ClassID: 7, UnitID: 1, GroupID: 1, Owner: 1},
			{X: 22 << 8, Y: 20 << 8, ClassID: 7, UnitID: 2, GroupID: 2, Owner: 2},
		},
	})
	// The store itself, before anything has been stepped: one way and one way
	// only. A symmetric store fails right here.
	if rel := w.Relations(); rel.Byte(2, 1) != 1 || rel.Byte(1, 2) != 0 {
		t.Fatalf("the loaded store is [2][1]=%d [1][2]=%d, want 1 and 0 — "+
			"the file named one direction and the store is symmetric",
			rel.Byte(2, 1), rel.Byte(1, 2))
	}
	for i := 0; i < 96; i++ {
		sim.Step(w, nil)
	}
	got := w.Entities()
	if len(got) != 2 {
		t.Fatalf("the world holds %d entities, want 2", len(got))
	}
	if !got[1].HasAttackTarget || got[1].AttackTarget != got[0].ID {
		t.Errorf("the hostile placement holds target %v/%v, want entity %d — "+
			"no map authored a relation and nothing acquired",
			got[1].AttackTarget, got[1].HasAttackTarget, got[0].ID)
	}
	// And the struck side fought back (0108): its blank cell was flipped by the
	// blow, so the pair the file left one-way ends the run mutual.
	// The direction that was ALREADY hostile is byte-identical: both sides swing
	// by the end of the run, and a blow into an already-hostile cell writes
	// nothing at all.
	if rel := w.Relations(); rel.Byte(1, 2) != 1 || rel.Byte(2, 1) != 1 {
		t.Errorf("the pair ends [1][2]=%d [2][1]=%d, want 1 and 1 — the struck cell gains "+
			"bit 0 and the one already hostile is left alone",
			rel.Byte(1, 2), rel.Byte(2, 1))
	}
	if !got[0].HasAttackTarget || got[0].AttackTarget != got[1].ID {
		t.Errorf("the struck placement holds target %v/%v, want entity %d — "+
			"it was hit, so it is hostile now and acquires its striker",
			got[0].AttackTarget, got[0].HasAttackTarget, got[1].ID)
	}
}
