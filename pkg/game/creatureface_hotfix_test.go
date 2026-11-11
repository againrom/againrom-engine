package game

import (
	"image/color"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

// The second game's Ancient Forest places Druid_Male2 (face 2) with secondary
// key 1, and a hired druid enters on the Druid_Mercenary face 4. Each shows
// its own picture on a fresh mission and after a LOAD, never the first one.
func TestSecondGameDruidShowsItsOwnPortrait(t *testing.T) {
	const druidType, druidServer = 100, 5032
	row := make([]int32, 56)
	for i := range row {
		row[i] = -1
	}
	row[0x1d], row[0x1e], row[55] = druidType, 2, druidServer
	table := &mapload.Table{Units: dbCollection{{}, {name: "Druid_Male2", params: row}}, UnitKeys: mapload.ServerUnitKeys}
	placement := alm.Unit{UnitID: 32, ClassID: druidType, ClassSubID: 1, ServerID: druidServer}

	placed := sim.Entity{ID: 0, X: 2, Y: 3, HP: 10, MaxHP: 10, MapUnitID: 32, TypeID: druidType, Class: druidType}
	built, _, err := mapload.ConstructActorBasis(placed, mapload.PartyMember{}, &placement, table, 1, 1, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if built.SourceBinding.Face != 2 {
		t.Errorf("the SAVE constructor gives the placed druid face %d, want its row's 2", built.SourceBinding.Face)
	}

	m := worldFixtureMap()
	m.Units = []alm.Unit{placement}
	party := []mapload.PartyMember{manFighter()[0], {Name: "Druid", Class: druidType, FigureFace: 4}}
	world := func() *sim.World {
		t.Helper()
		w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
			[]sim.Entity{placed,
				{ID: 1, X: 4, Y: 3, HP: 10, MaxHP: 10, Owner: 1, Humanoid: true},
				{ID: 2, X: 5, Y: 3, HP: 10, MaxHP: 10, Owner: 1, TypeID: druidType, Class: druidType}},
			nil, sim.Relations{}, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		return w
	}
	art := worldFixtureArt(16, 16, 8, 16, 3, 3, 1)
	art.Portrait = "druid"
	units := &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{druidType: art}}
	src := missionSource{
		"graphics/infowindow/druid.bmp":  synthBMP(2, 2, color.RGBA{R: 1, A: 0xff}),
		"graphics/infowindow/druid2.bmp": synthBMP(2, 2, color.RGBA{R: 2, A: 0xff}),
		"graphics/infowindow/druid4.bmp": synthBMP(2, 2, color.RGBA{R: 4, A: 0xff}),
	}
	open := func(state *SnapshotSAVDocument) *mapWorld {
		ms := &Mission{Number: 50, Map: m, World: world(), Party: party,
			Start: mapload.Start{IDs: []sim.EntityID{1, 2}}, savedDocument: state}
		return openMission(ms, table, units, worldFixtureViewer(t, m), src, nil, nil)
	}
	// The record an earlier build saved: the secondary key as the face byte.
	state := &SnapshotSAVDocument{Document: &sav.DocumentData{Objects: []sav.DocumentRecordData{
		savedTierRecord(32, druidType, 1),
	}}, Actors: []SnapshotSAVActor{{EntityID: 0, ObjectIndex: 1}}}
	for _, run := range []struct {
		when  string
		state *SnapshotSAVDocument
	}{{"fresh", nil}, {"after LOAD", state}} {
		mw := open(run.state)
		for _, want := range []struct {
			id       sim.EntityID
			who      string
			tier     int
			portrait uint8
		}{{0, "placed Druid_Male2", 2, 2}, {2, "hired druid", 4, 4}} {
			if got := mw.tiers[want.id]; got != want.tier {
				t.Errorf("%s: %s tier %d, want %d", run.when, want.who, got, want.tier)
			}
			pic := mw.inspectionUnitPicture(uint32(want.id))
			if pic == nil || pic.RGBAAt(0, 0).R != want.portrait {
				t.Errorf("%s: %s shows %v, want infowindow/druid%d.bmp", run.when, want.who, pic, want.portrait)
			}
		}
		if _, ok := mw.tiers[1]; ok && data.ComposesFigure(party[0].Class) {
			t.Errorf("%s: the composed hero took a creature tier", run.when)
		}
	}
}

// A first-game creature keeps its secondary key as its face: the key selected
// its row by that face, so the record's face and the row's agree.
func TestFirstGameCreatureFaceIsItsSecondaryKey(t *testing.T) {
	table := tierTable()
	placement := alm.Unit{UnitID: 7, ClassID: unitsArmKey, ClassSubID: 3}
	e := sim.Entity{ID: 0, MapUnitID: 7, TypeID: unitsArmKey, Class: unitsArmKey}
	built, _, err := mapload.ConstructActorBasis(e, mapload.PartyMember{}, &placement, table, 1, 1, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if built.SourceBinding.Face != 3 {
		t.Errorf("first-game face %d, want the secondary key 3", built.SourceBinding.Face)
	}
	m := &alm.Map{Units: []alm.Unit{placement}}
	state := &SnapshotSAVDocument{Document: &sav.DocumentData{Objects: []sav.DocumentRecordData{
		savedTierRecord(7, unitsArmKey, 3),
	}}, Actors: []SnapshotSAVActor{{EntityID: 0, ObjectIndex: 1}}}
	if got := savedActorTiers(m, table, entityTiers(m, table), []sim.Entity{e}, state)[0]; got != 3 {
		t.Errorf("first-game loaded tier %d, want 3", got)
	}
}
