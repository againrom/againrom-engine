package mapload

import (
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

// Every world-building path standing on all three planes.
//
// The planes are unexported on a world and there is no reader for them, so what
// this file measures is the DIGEST: two maps that differ only in a plane must
// build worlds that differ, on every path. A path that dropped a plane would
// build one world from both.

const cwSide = 24 // wider than twice the eight-cell border, so there is an interior

// cwTiles is a tile plane of one repeated word.
func cwTiles(word uint16) []uint16 {
	out := make([]uint16, cwSide*cwSide)
	for i := range out {
		out[i] = word
	}
	return out
}

// cwLand and cwRoad are two words with the SAME block byte and DIFFERENT costs:
// the word zero blends to Land at 8, and strip group 12 at blend level 5 is Road
// at 6. Neither is water, neither is Mountain and neither sets the impassable
// bit, so a map built from either has the same block plane cell for cell — which
// is what makes a digest difference between them a statement about the COST
// plane and about nothing else.
var (
	cwLand uint16 = 0
	cwRoad uint16 = 12<<6 | 3<<4
)

func cwMap(word uint16, alt uint8) *alm.Map {
	alts := make([]uint8, cwSide*cwSide)
	for i := range alts {
		alts[i] = alt
	}
	return &alm.Map{
		Width: cwSide, Height: cwSide,
		Tiles:     cwTiles(word),
		Altitudes: alts,
		Units:     []alm.Unit{{X: 12 << 8, Y: 12 << 8, ClassID: 3}},
	}
}

// TestTheTwoWordsDifferInCostAndNotInBlocking is the fixture's own premise, and
// it is asserted rather than assumed: if the two words ever came to differ in
// their block byte, every case below would still pass and would be measuring the
// block plane.
func TestTheTwoWordsDifferInCostAndNotInBlocking(t *testing.T) {
	t.Parallel()

	land, road := cwMap(cwLand, 0), cwMap(cwRoad, 0)
	a, b := Passability(land), Passability(road)
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("the two words block differently at cell %d: %#02x against %#02x", i, a[i], b[i])
		}
	}
	ca, cb := Cost(land), Cost(road)
	if ca[0] != 8 || cb[0] != 6 {
		t.Fatalf("the two words cost %d and %d, want 8 and 6", ca[0], cb[0])
	}
}

// cwPaths is every way this package turns a decoded map into a world, each
// reduced to one function of the map so the cases below are one table.
//
// The six are the three the contract names crossed with present and absent:
// a definition table, a party, and a mission script. Two of them REBUILD a world
// from an earlier one, which is the shape most likely to drop a plane — it is
// the one place a second derivation could disagree with the first.
var cwPaths = []struct {
	what  string
	build func(*testing.T, *alm.Map) *sim.World
}{
	{"FromALM, no table", func(t *testing.T, m *alm.Map) *sim.World {
		return FromALM(m)
	}},
	{"FromALMWith, a table", func(t *testing.T, m *alm.Map) *sim.World {
		w, err := FromALMWith(m, &Table{}, DifficultyNormal)
		if err != nil {
			t.Fatalf("FromALMWith: %v", err)
		}
		return w
	}},
	{"StartMission, no party", func(t *testing.T, m *alm.Map) *sim.World {
		w, _, err := StartMission(m, nil, DifficultyNormal, nil)
		if err != nil {
			t.Fatalf("StartMission: %v", err)
		}
		return w
	}},
	{"StartMission, a party — the REBUILD", func(t *testing.T, m *alm.Map) *sim.World {
		w, _, err := StartMission(m, nil, DifficultyNormal, []PartyMember{{Class: 1}})
		if err != nil {
			t.Fatalf("StartMission with a party: %v", err)
		}
		return w
	}},
	{"StartMissionScripted, no script — the OTHER rebuild", func(t *testing.T, m *alm.Map) *sim.World {
		w, _, err := StartMissionScripted(m, nil, DifficultyNormal, nil, nil)
		if err != nil {
			t.Fatalf("StartMissionScripted: %v", err)
		}
		return w
	}},
	{"StartMissionScripted, a party and a script", func(t *testing.T, m *alm.Map) *sim.World {
		sc, _, cerr := CompileScript(m, ScriptRefs{})
		if cerr != nil {
			t.Fatalf("CompileScript: %v", cerr)
		}
		w, _, err := StartMissionScripted(m, nil, DifficultyNormal,
			[]PartyMember{{Class: 1}}, sc)
		if err != nil {
			t.Fatalf("StartMissionScripted with a party: %v", err)
		}
		return w
	}},
}

// TestEveryPathCarriesTheCostPlane — AC-14, the cost half.
func TestEveryPathCarriesTheCostPlane(t *testing.T) {
	t.Parallel()

	for _, p := range cwPaths {
		land := p.build(t, cwMap(cwLand, 0)).Hash()
		road := p.build(t, cwMap(cwRoad, 0)).Hash()
		if land == road {
			t.Errorf("%s: a map of cost-8 ground and one of cost-6 ground build worlds that both "+
				"hash %#016x — this path hands over no cost plane", p.what, land)
		}
		// And the path is deterministic, so the inequality above is a fact about
		// the two maps rather than about two runs.
		if again := p.build(t, cwMap(cwLand, 0)).Hash(); again != land {
			t.Errorf("%s: the same map built twice hashes %#016x then %#016x", p.what, land, again)
		}
	}
}

// TestEveryPathCarriesTheHeightPlane — AC-14, the height half.
//
// The altitude plane reaches the block plane through nothing at all — that
// derivation reads the tile and overlay planes only — so a digest that moves
// with the altitudes moved because the HEIGHT plane reached the world.
func TestEveryPathCarriesTheHeightPlane(t *testing.T) {
	t.Parallel()

	for _, p := range cwPaths {
		flat := p.build(t, cwMap(cwLand, 0)).Hash()
		high := p.build(t, cwMap(cwLand, 200)).Hash()
		if flat == high {
			t.Errorf("%s: a map at altitude 0 and one at altitude 200 build worlds that both "+
				"hash %#016x — this path hands over no height plane", p.what, flat)
		}
	}
}

// TestThePlanesAWorldIsBuiltOverAreTheMapsOwn is the positive half: not merely
// that the digest moves, but that it lands on the world the map's own three
// planes make.
//
// It is asserted on the one path whose entity set is reproducible outside the
// loader — the single-argument one over a map with no placements — because the
// claim needs a world built independently to compare against, and the others
// place units this test would have to re-derive.
func TestThePlanesAWorldIsBuiltOverAreTheMapsOwn(t *testing.T) {
	t.Parallel()

	m := cwMap(cwRoad, 77)
	m.Units = nil

	got := FromALM(m).Hash()
	want, err := sim.NewTerrainWorld(Seed, sim.Bounds{Width: cwSide, Height: cwSide},
		sim.ModeCanonical, Planes(m, nil), nil, nil)
	if err != nil {
		t.Fatalf("NewTerrainWorld: %v", err)
	}
	if err := want.RestoreCurrentPlayers(nil, nil, true); err != nil {
		t.Fatal(err)
	}
	if got != want.Hash() {
		t.Errorf("the loader builds %#016x and the map's own three planes build %#016x",
			got, want.Hash())
	}
	// And NOT the world the defaults make, which is what the comparison above
	// would still pass against if Planes returned nothing.
	bare, err := sim.NewWorld(Seed, sim.Bounds{Width: cwSide, Height: cwSide},
		sim.ModeCanonical, Passability(m), nil)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	if got == bare.Hash() {
		t.Errorf("the loader builds the world the DEFAULT planes make, %#016x", got)
	}
}
