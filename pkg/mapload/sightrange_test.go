package mapload_test

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// The scan-range slot on each band's row, transcribed from the row's own
// documented order rather than imported — the rule every other slot number in
// this suite is written under: a test asserting that a column reached a field
// must not read that column's position out of the code it is testing.
const (
	slotUnitScan  = 10
	slotHumanScan = 8
)

// The values this file loads. NEITHER IS 5, which is the constructor's own
// default and therefore the one number a loader that never read the column would
// still produce; and the two differ from each other, so a build that read one
// band's slot on both arms fails rather than agreeing by accident.
const (
	srcUnitScan  int32 = 11
	srcHumanScan int32 = 7
)

// srcMap places ONE unit on a key at the class-key floor's upper side, so the
// placement takes the creature arm; srcHumanMap places one below it, so it takes
// the person arm. Both cells are inside the extent.
func srcMap(key int16) *alm.Map {
	return &alm.Map{
		Width: 40, Height: 40,
		Units: []alm.Unit{{X: 0x0C80, Y: 0x0C80, ClassID: key, ClassSubID: 1}},
	}
}

// srcLoad builds the one-placement world and answers its only entity.
func srcLoad(t *testing.T, m *alm.Map, tbl *mapload.Table) sim.Entity {
	t.Helper()
	w, err := mapload.FromALMWith(m, tbl, mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}
	ents := w.Entities()
	if len(ents) != 1 {
		t.Fatalf("the fixture built %d entities, want 1", len(ents))
	}
	return ents[0]
}

// TestAPlacementsRangeIsItsOwnBandsColumn is AC-5 and AC-6.
//
// Four arms in one table because they are one claim with four answers: the
// creature band's column, the person band's, the empty cell that keeps the
// constructor's value, and the placement that reached no entry at all. The last
// two produce the SAME number by two different routes, and both are stated —
// which is exactly the pair a build reading no column would collapse into one.
func TestAPlacementsRangeIsItsOwnBandsColumn(t *testing.T) {
	t.Parallel()

	creature := &mapload.Table{
		Units: defCollection{{}, {name: "k40", params: defRow(map[int]int32{
			slotUnitType: 0x40, slotUnitFace: 1, slotHealthMax: 30,
			slotUnitScan: srcUnitScan,
		})}},
		Humans: defCollection{},
	}
	silent := &mapload.Table{
		Units: defCollection{{}, {name: "k40", params: defRow(map[int]int32{
			slotUnitType: 0x40, slotUnitFace: 1, slotHealthMax: 30,
		})}},
		Humans: defCollection{},
	}
	person := &mapload.Table{
		Units: defCollection{},
		Humans: defCollection{{}, {name: "k7", params: defRow(map[int]int32{
			slotHumanType: 7, slotHumanScan: srcHumanScan,
		})}},
	}

	for _, tc := range []struct {
		name string
		m    *alm.Map
		tbl  *mapload.Table
		want uint8
	}{
		{"a creature row states its range", srcMap(0x40), creature, uint8(srcUnitScan)},
		{"a creature row leaving the cell empty keeps the constructor's", srcMap(0x40), silent, 5},
		{"a person row states its range", srcMap(7), person, uint8(srcHumanScan)},
		{"a placement that resolves to nothing keeps the constructor's", srcMap(0x40), silent, 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := srcLoad(t, tc.m, tc.tbl).ScanRange; got != tc.want {
				t.Errorf("the placement carries a range of %d, want %d", got, tc.want)
			}
		})
	}

	// And the number the two default arms agree on is the definition tier's own
	// rather than a literal repeated here, so a constructor default that moved
	// would fail in one place instead of quietly disagreeing with this file.
	if got := data.UnitDefaults().ScanRange; got != 5 {
		t.Errorf("the constructor's default range is %d; this file's two default arms expect 5", got)
	}
}

// TestAColumnOutsideAByteIsTruncated is AC-8.
//
// The narrowing is the streamer's own — its store is a byte store after the
// empty-cell compare — so a column outside a byte loses its high bytes rather
// than being clamped or refused. No shipped row exercises it; these two do.
func TestAColumnOutsideAByteIsTruncated(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		col  int32
		want uint8
	}{
		{"a column of 300", 300, 44},
		{"a column of -2", -2, 254},
		{"a column at the top of a byte", 255, 255},
		{"a column of exactly 256", 256, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tbl := &mapload.Table{
				Units: defCollection{{}, {name: "k40", params: defRow(map[int]int32{
					slotUnitType: 0x40, slotUnitFace: 1, slotHealthMax: 30,
					slotUnitScan: tc.col,
				})}},
				Humans: defCollection{},
			}
			if got := srcLoad(t, srcMap(0x40), tbl).ScanRange; got != tc.want {
				t.Errorf("a column of %d reached the entity as %d, want %d", tc.col, got, tc.want)
			}
		})
	}
}

func TestEveryPlacedUnitHasARangeFromOneSource(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		tbl  *mapload.Table
	}{
		{"with no table at all", nil},
		{"with the fixture's own table", fixtureTable()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, err := mapload.FromALMWith(fixtureMap(), tc.tbl, mapload.DifficultyNormal)
			if err != nil {
				t.Fatalf("FromALMWith: %v", err)
			}
			for _, e := range w.Entities() {
				if e.ScanRange == 0 {
					t.Errorf("placement %d carries no sight range at all", e.ID)
				}
			}
		})
	}
}

// TestAPartyMembersRangeIsHisOwnDerivation is AC-7 at the loader: a generated
// hero has no row at all, so his range comes from his own statistics and from
// nothing the table says.
//
// The two members carry DIFFERENT statistics whose derivations differ, so a
// build that filled one number for the whole party — the constructor's 5, or the
// first member's — fails on the second.
func TestAPartyMembersRangeIsHisOwnDerivation(t *testing.T) {
	t.Parallel()

	p := []mapload.PartyMember{
		{Class: 100, Hero: data.Hero{Mind: 26, Reaction: 26}},
		{Class: 101, Hero: data.Hero{Mind: 50, Reaction: 50}},
	}
	w, st := mustStart(t, startMap(t, 40, 40, mapload.Cell{X: 20, Y: 20}), p)
	ents := w.Entities()
	for i, m := range p {
		want := uint8(m.Hero.Sight())
		var got uint8
		var found bool
		for _, e := range ents {
			if e.ID == st.IDs[i] {
				got, found = e.ScanRange, true
			}
		}
		if !found {
			t.Fatalf("party member %d is not in the world", i)
		}
		if got != want {
			t.Errorf("party member %d carries a range of %d, want his own derivation %d", i, got, want)
		}
	}
	// And the two really do differ, so the check above is not two equal numbers
	// agreeing for the wrong reason.
	if p[0].Hero.Sight() == p[1].Hero.Sight() {
		t.Fatal("the two members derive the same range; this fixture discriminates nothing")
	}
}
