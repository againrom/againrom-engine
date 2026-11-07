package game

import (
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// secondMapUnitFacings is the facing of each living map unit of w (every
// actor outside the party), by entity ID; each must equal its heading.
func secondMapUnitFacings(t *testing.T, stage string, w *sim.World, party []sim.EntityID) map[sim.EntityID]uint8 {
	t.Helper()
	out := map[sim.EntityID]uint8{}
	for _, e := range w.Entities() {
		if slices.Contains(party, e.ID) || !e.Alive() {
			continue
		}
		if e.DesiredFacing != e.Facing {
			t.Errorf("%s: actor %d (map unit %d) faces %#x heading %#x", stage, e.ID, e.MapUnitID, e.Facing, e.DesiredFacing)
		}
		out[e.ID] = e.Facing
	}
	return out
}

// TestReleaseSecondGameMission42StartsWithDrawnFacings opens the second
// game's mission 42 from the install. Every map unit starts with the facing
// Unit initialization draws, 0x40 + 0..128, equal to its heading, and the
// facings are not one value (R2-ENGINE-959, R2-ENGINE-960). The SAV holds
// each unit's facing and heading in bytes 0 and 1 of its fifth raw block
// (R2-SESSION-362), and a LOAD of that SAV in a new front end restores every
// facing.
func TestReleaseSecondGameMission42StartsWithDrawnFacings(t *testing.T) {
	f := secondGameFront(t)
	f.Options = OptionsStore{}
	app := f.App("second-game start facings")
	app.Layout(1024, 768)
	c := newSecondCampaign()
	c.bank[769] = 1
	c.available = append(c.available, partyPlace{kind: placeMission, id: 42})
	if f.Town == nil {
		f.Town = NewTown(Campaign{})
	}
	f.Town.second = c
	f.Carried = MissionParty(f.StartWeapon.Value(), f.Bodies, f.Table)
	screen := f.TownScreen().(*townScreen)
	app.SetTown(screen)
	destinationTownCampaign{screen}.leave()
	if err := app.OpenMission(f.MissionOpener(42)); err != nil || f.live == nil {
		t.Fatalf("mission 42: %v %s", err, app.HeadlessMessage())
	}
	party := slices.Clone(f.live.mission.state.Start.IDs)
	live := secondMapUnitFacings(t, "fresh", f.live.world, party)
	distinct := map[uint8]bool{}
	for id, facing := range live {
		if facing < 0x40 || facing > 0xC0 {
			t.Errorf("fresh: actor %d starts facing %#x, want 0x40..0xC0", id, facing)
		}
		distinct[facing] = true
	}
	if len(live) == 0 || len(distinct) < 2 {
		t.Fatalf("fresh: %d map units hold %d distinct facings", len(live), len(distinct))
	}
	t.Logf("mission 42: %d map units, %d distinct facings", len(live), len(distinct))

	s, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	byMapUnit := map[uint16][]uint8{}
	for _, e := range f.live.world.Entities() {
		if facing, ok := live[e.ID]; ok && e.MapUnitID != 0 {
			byMapUnit[e.MapUnitID] = append(byMapUnit[e.MapUnitID], facing)
		}
	}
	checked := 0
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Unit" && r.Class != "Human" {
			continue
		}
		v, err := savedStructureValue(r, "T08")
		want := byMapUnit[uint16(v)]
		if err != nil || len(want) != 1 {
			continue
		}
		block := secondRecordRaw(r, "U154")
		if len(block) != 180 || block[0] != want[0] || block[1] != want[0] {
			t.Errorf("SAV: map unit %d writes Mover bytes %#x, want facing and heading %#x", v, block[:min(2, len(block))], want[0])
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("SAV: no record of a map unit with a unique map unit ID")
	}

	cold := secondGameFront(t)
	cold.Options = OptionsStore{}
	open, town, err := cold.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatalf("LOAD town=%t: %v", town, err)
	}
	if err := cold.App("second-game start facings LOAD").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	loaded := secondMapUnitFacings(t, "LOAD", cold.live.world, party)
	for id, facing := range live {
		if got, ok := loaded[id]; !ok || got != facing {
			t.Errorf("LOAD: actor %d faces %#x (held %v), want %#x", id, got, ok, facing)
		}
	}
}
