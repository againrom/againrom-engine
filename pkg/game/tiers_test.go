package game

import (
	"image/color"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

// tierMark is the palette entry the fixture frames disagree on, and the one an
// assertion reads back: it says which SLICE the seam took, without counting
// pointers and without a second copy of the selection rule.
const tierMark = 5

// tierArtClass is one drawable class with three tiers: the base frames, and two
// slices over the SAME pixels whose mark entry differs. The frames-per-slice
// count is one, so every selection lands on frame 0 and the only thing an
// assertion can be reading is the colour.
func tierArtClass(tiers int) *terrain.UnitClass {
	frame := func(mark uint8) *terrain.StaticFrame {
		f := &terrain.StaticFrame{
			Width: 1, Height: 1,
			Pixels: []terrain.StaticPixel{{Index: tierMark, Opaque: true}},
		}
		f.Palette[tierMark] = color.RGBA{R: mark, A: 0xff}
		return f
	}
	base := []*terrain.StaticFrame{frame(0)}
	c := &terrain.UnitClass{
		Width: 2, Height: 2, CenterX: 1, CenterY: 1,
		Frames: base,
		// One standing frame per block, and a dying block that EXISTS: the
		// death selection refuses a class with no dying slot, so a descriptor
		// without one would send the corpse case down the live arm and the
		// substitution would never be exercised.
		Anim: terrain.UnitAnim{S: 1, D: 1, Total: 1, MoveOK: true, DyingBase: 0, DyingSlot: 1},
	}
	for tier := 1; tier <= tiers; tier++ {
		if tier == 1 {
			// Tier 1 is the sheet's own, by identity — what the loader builds
			// wherever a tier's table equals the sheet's.
			c.Tiers = append(c.Tiers, base)
			continue
		}
		c.Tiers = append(c.Tiers, []*terrain.StaticFrame{frame(uint8(tier))})
	}
	c.Corpse = c
	return c
}

// tierMap places the units-arm class twice — two tiers of it — and one placement
// diverted to the npc path, which reaches no entry and so states no tier, after
// the fixture map's own four, whose ids the rest of the suite asserts over.
//
// The diverted one carries a key BELOW the class-key floor, because the flag
// word only diverts inside the humans band: at or above the floor the class
// key decides on its own and the flag is never read.
func tierMap() *alm.Map {
	m := worldFixtureMap()
	m.Units = append(m.Units,
		alm.Unit{X: 0x1580, Y: 0x1700, ClassID: unitsArmKey, ClassSubID: 1},
		alm.Unit{X: 0x1600, Y: 0x1700, ClassID: unitsArmKey, ClassSubID: 3},
		alm.Unit{X: 0x1680, Y: 0x1700, ClassID: npcArmKey, ClassSubID: 3, Flags: 1},
	)
	return m
}

// tierRow is a units row on the units-arm key at a stated tier column. The key
// pair a search matches on is (typeID, face), so the face column is both the
// search key and the tier — which is why two rows differing in it are two
// entries rather than one.
func tierRow(name string, face int32) dbEntry {
	p := make([]int32, 38)
	for i := range p {
		p[i] = -1
	}
	p[0x1d], p[0x1e] = unitsArmKey, face
	return dbEntry{name: name, params: p}
}

func tierTable() *mapload.Table {
	return &mapload.Table{Units: dbCollection{{}, tierRow("first", 1), tierRow("third", 3)}}
}

// markOf reads which slice a drawn frame came from: the mark entry of its
// palette, and -1 for an entity that crossed with no frame at all.
func markOf(d frameCarrier) int {
	if d == nil {
		return -1
	}
	return int(d.Palette[tierMark].R)
}

type frameCarrier = *terrain.StaticFrame

// AC-7. Two placements of one class at two tier columns reach the seam with
// different colour tables over the same pixel indices, and the placement that
// resolves to no entry states no tier and takes the sheet's own.
func TestTheTierColumnReachesTheDrawnFrame(t *testing.T) {
	m := tierMap()
	set := &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{
		unitsArmKey: tierArtClass(3), npcArmKey: tierArtClass(3)}}
	mw := mustOpenMapWorld(t, m, tierTable(), set, worldFixtureViewer(t, m))

	draws := mw.entityDraws()
	if len(draws) != len(m.Units) {
		t.Fatalf("%d draws for %d placements", len(draws), len(m.Units))
	}
	first, third, npc := draws[len(draws)-3], draws[len(draws)-2], draws[len(draws)-1]

	if got := markOf(first.Frame); got != 0 {
		t.Errorf("the tier-1 placement drew mark %d, want 0 — the sheet's own", got)
	}
	if got := markOf(third.Frame); got != 3 {
		t.Errorf("the tier-3 placement drew mark %d, want 3", got)
	}
	if got := markOf(npc.Frame); got != 0 {
		t.Errorf("the npc-arm placement drew mark %d, want 0 — no tier stated", got)
	}
	if first.Frame == third.Frame {
		t.Fatal("the two tiers drew the very same frame")
	}
	// The pixels are the same picture: a tier is a colour table and nothing
	// else, so the indices and the geometry are untouched.
	if first.Frame.Width != third.Frame.Width || first.Frame.Height != third.Frame.Height {
		t.Errorf("the two tiers differ in size: %dx%d against %dx%d",
			first.Frame.Width, first.Frame.Height, third.Frame.Width, third.Frame.Height)
	}
	for i := range first.Frame.Pixels {
		if first.Frame.Pixels[i] != third.Frame.Pixels[i] {
			t.Fatalf("pixel %d differs between the two tiers", i)
		}
	}
	// The four placements the fixture already had reach no units entry, so they
	// state no tier and draw nothing at all under this bundle.
	for i := 0; i < len(draws)-3; i++ {
		if draws[i].Frame != nil {
			t.Errorf("placement %d drew a frame; its class key resolves to no art here", i)
		}
	}
}

func TestTheTierSurvivesTheCorpseSubstitution(t *testing.T) {
	m := tierMap()
	set := &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{
		unitsArmKey: tierArtClass(3), npcArmKey: tierArtClass(3)}}
	mw := mustOpenMapWorld(t, m, tierTable(), set, worldFixtureViewer(t, m))

	// Kill the tier-3 placement outright, then push.
	dead := sim.EntityID(len(m.Units) - 2)
	for i := 0; i < 64 && mw.world.Entities()[dead].Alive(); i++ {
		mw.affect(uint32(dead), true)
		mw.tick()
	}
	if mw.world.Entities()[dead].Alive() {
		t.Fatal("the fixture entity would not die")
	}

	d := mw.entityDraws()[dead]
	if d.Frame == nil {
		t.Fatal("the corpse crossed with no frame")
	}
	// The DEATH arm must be the one that drew it: a descriptor the death
	// selection refuses would fall through to the live arm and this test would
	// pass on the wrong path.
	if d.Art != mw.units.Classes[unitsArmKey].Corpse {
		t.Fatal("the corpse did not draw through the substituted class")
	}
	if _, _, ok := terrain.SelectDeathFrame(d.Art.Anim, len(d.Art.Frames), 0, 0); !ok {
		t.Fatal("the fixture's dying block is refused; the death arm cannot be reached")
	}
	if got := markOf(d.Frame); got != 3 {
		t.Errorf("the corpse drew mark %d, want 3 — the tier is the placement's", got)
	}
}

func TestTierResolutionReachesNoWorldState(t *testing.T) {
	m := tierMap()
	tbl := tierTable()
	mw := mustOpenMapWorld(t, m, tbl, nil, worldFixtureViewer(t, m))

	// The comparison world is built by the loading tier DIRECTLY, with no map
	// screen and no tier resolution anywhere near it.
	bare, err := mapload.FromALMWith(m, tbl, openDifficulty)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}

	got, want := mw.world.Entities(), bare.Entities()
	if len(got) != len(want) {
		t.Fatalf("%d entities against %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("entity %d is %+v, want %+v", i, got[i], want[i])
		}
	}
	if mw.world.Hash() != bare.Hash() {
		t.Error("the digest moved")
	}
	a, err := mw.world.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	b, err := bare.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if string(a) != string(b) {
		t.Error("the byte form moved")
	}
}

func TestTheTierIsFixedAtMapOpen(t *testing.T) {
	m := tierMap()
	set := &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{
		unitsArmKey: tierArtClass(3), npcArmKey: tierArtClass(3)}}
	mw := mustOpenMapWorld(t, m, tierTable(), set, worldFixtureViewer(t, m))

	first := mw.entityDraws()
	before := len(mw.tiers)
	second := mw.entityDraws()
	if len(mw.tiers) != before {
		t.Errorf("the lookup grew from %d to %d entries across a build", before, len(mw.tiers))
	}
	for i := range first {
		if first[i].Frame != second[i].Frame {
			t.Errorf("entity %d drew a different frame on the second build", i)
		}
	}
}

func TestOnlyTheStatArmStatesATier(t *testing.T) {
	m := tierMap()
	tiers := entityTiers(m, tierTable())

	last := len(m.Units) - 1
	for _, tc := range []struct {
		id    sim.EntityID
		tier  int
		found bool
	}{
		{sim.EntityID(last - 2), 1, true},
		{sim.EntityID(last - 1), 3, true},
		{sim.EntityID(last), 0, false}, // the npc arm
		{0, 0, false},                  // a humans-arm placement of the base fixture
	} {
		got, ok := tiers[tc.id]
		if ok != tc.found {
			t.Errorf("entity %d: present = %v, want %v", tc.id, ok, tc.found)
		}
		if got != tc.tier {
			t.Errorf("entity %d: tier %d, want %d", tc.id, got, tc.tier)
		}
	}

	// A map opened with no table states no tier at all, which is what leaves
	// every hand-assembled front-end drawing exactly what it drew.
	for _, tbl := range []*mapload.Table{nil, {}} {
		if got := entityTiers(m, tbl); len(got) != 0 {
			t.Errorf("a table-free open stated %d tier(s)", len(got))
		}
	}
	if got := entityTiers(nil, tierTable()); len(got) != 0 {
		t.Errorf("a nil map stated %d tier(s)", len(got))
	}
}
