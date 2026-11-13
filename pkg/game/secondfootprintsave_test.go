package game

import (
	"encoding/binary"
	"maps"
	"testing"

	"againrom/pkg/formats/sav"
)

// secondStandingKeys is the Token key of each living Unit and Human record
// of the second-game file raw that stands at rest, by its cell.
func secondStandingKeys(t *testing.T, raw []byte) map[uint16]uint32 {
	t.Helper()
	sf, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	doc, origins, err := sav.DecodeDocumentDataWithOrigins(raw)
	if err != nil {
		t.Fatal(err)
	}
	object := map[uint16]uint16{}
	for _, o := range origins {
		object[o.ArchiveIndex] = o.ObjectIndex
	}
	locations, err := sf.DocumentActorLocations()
	if err != nil {
		t.Fatal(err)
	}
	key := map[int]uint32{}
	for _, l := range locations {
		at := object[l.ArchiveIndex]
		if at == 0 {
			t.Fatalf("the %s record at %#x has no object", l.Class, l.Off)
		}
		for _, v := range doc.Objects[at-1].Values {
			if v.Name == "Identity" {
				key[l.Off] = v.Value
			}
		}
	}
	out := map[uint16]uint32{}
	for _, a := range sf.Actors {
		if a.Dead() || !a.AtRest() {
			continue
		}
		if key[a.Off] == 0 {
			t.Fatalf("the actor at %#x has no key", a.Off)
		}
		out[a.Cell] = key[a.Off]
	}
	return out
}

// A second-game mission SAVE names the unit that stands on a cell twice in
// the cell's terrain entry, as every original mission file does: as the
// occupant, +0x04 on the ground and +0x08 in the air, and as the footprint
// occupant, +0x2c and +0x30. The original's refresh of an entry cell takes
// the cell's plane-two occupancy bits from the footprint fields
// (R2-ENGINE-758, R2-ENGINE-673). The file LOADs back to the World of the
// engine document it carries. Fails while the footprint fields are zero.
func TestSecondMissionSaveNamesEachStandingUnitAsItsCellsFootprintOccupant(t *testing.T) {
	f, app, screen, _ := secondBagFrontOf(t, nil, secondMageMission())
	secondMageItems(f)
	secondBagEnter(t, f, app, screen)
	before := secondPlacedKeys(f.live.world)
	raw := secondPageSave(t, f, app)
	checkSecondMissionFile(t, raw, 10)

	standing := secondStandingKeys(t, raw)
	if len(standing) < 7 {
		t.Fatalf("the file holds %d standing units, want the hero, two knights and four creatures", len(standing))
	}
	sf, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	named := map[uint16]bool{}
	for i := range sf.World.CellRecCount {
		e, err := sf.CellRecord(i)
		if err != nil {
			t.Fatal(err)
		}
		cell, p := binary.LittleEndian.Uint16(e), e[2:]
		ground, air := binary.LittleEndian.Uint32(p[0x04:]), binary.LittleEndian.Uint32(p[0x08:])
		groundPrint, airPrint := binary.LittleEndian.Uint32(p[0x2c:]), binary.LittleEndian.Uint32(p[0x30:])
		if groundPrint != ground || airPrint != air {
			t.Errorf("entry %#04x: occupants %#x and %#x, footprint occupants %#x and %#x", cell, ground, air, groundPrint, airPrint)
		}
		if key, ok := standing[cell]; ok {
			if (ground != key && air != key) || (groundPrint != key && airPrint != key) {
				t.Errorf("entry %#04x: the unit %#x stands there; occupants %#x and %#x, footprint occupants %#x and %#x",
					cell, key, ground, air, groundPrint, airPrint)
			}
			named[cell] = true
		}
	}
	for cell, key := range standing {
		if !named[cell] {
			t.Errorf("the unit %#x stands on %#04x, which has no terrain entry", key, cell)
		}
	}

	loaded, _ := secondMageLoad(t, raw)
	own, err := sav.EngineDocument(raw)
	if err != nil {
		t.Fatal(err)
	}
	engine, _ := secondMageLoad(t, own)
	if got, want := loaded.live.world.Hash(), engine.live.world.Hash(); got != want {
		t.Fatalf("the LOAD's World hash is %x, the engine document's LOAD %x", got, want)
	}
	if got := secondPlacedKeys(loaded.live.world); !maps.Equal(got, before) {
		t.Fatalf("the LOAD holds keys %v, want %v", got, before)
	}
	gotAt, wantAt := map[uint16][2]int32{}, map[uint16][2]int32{}
	for _, e := range loaded.live.world.Entities() {
		gotAt[e.MapUnitID] = [2]int32{int32(e.X), int32(e.Y)}
	}
	for _, e := range f.live.world.Entities() {
		wantAt[e.MapUnitID] = [2]int32{int32(e.X), int32(e.Y)}
	}
	if !maps.Equal(gotAt, wantAt) {
		t.Fatalf("the LOAD stands its units at %v, want %v", gotAt, wantAt)
	}
}
