package game

import (
	"encoding/binary"
	"maps"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// secondHeroSlot is the hero's entry in secondActorKeys.
const secondHeroSlot = ^uint32(0)

// secondActorKeys is each Unit and Human record of doc with a +8, and the
// hero, by its +8, with its +4 and whether the hero's Player owns it.
func secondActorKeys(t *testing.T, doc sav.DocumentData) (keys map[uint32]uint32, heroOwned map[uint32]bool) {
	t.Helper()
	hero, err := secondTownHeroRecord(doc)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := savedStructureValue(&doc.Objects[hero-1], "Reference")
	if err != nil {
		t.Fatal(err)
	}
	keys, heroOwned = map[uint32]uint32{}, map[uint32]bool{}
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Unit" && r.Class != "Human" {
			continue
		}
		id, _ := savedStructureValue(r, "T08")
		key, err := savedStructureValue(r, "RuntimeID")
		if err != nil {
			t.Fatal(err)
		}
		ref, _ := savedStructureValue(r, "Reference")
		if uint16(i+1) == hero {
			id = secondHeroSlot
		} else if id == 0 {
			continue
		}
		keys[id], heroOwned[id] = key, ref == owner
	}
	return keys, heroOwned
}

// checkSecondMapActorKeys: every actor of raw's records that the map placed,
// whoever owns it, holds +4 = 0x6000 plus its map id, and the hero holds a
// key below 0x6000.
func checkSecondMapActorKeys(t *testing.T, name string, raw []byte, placed []uint16) {
	t.Helper()
	keys, heroOwned := secondActorKeys(t, checkSecondMissionFile(t, raw, 10))
	if key := keys[secondHeroSlot]; key == 0 || key >= 0x6000 || !heroOwned[secondHeroSlot] {
		t.Fatalf("%s: the hero holds +4 = %d", name, key)
	}
	for _, id := range placed {
		if got := keys[uint32(id)]; got != 0x6000+uint32(id) {
			t.Errorf("%s: the actor of map id %d holds +4 = %d, want %d", name, id, got, 0x6000+uint32(id))
		}
	}
}

// secondPlacedKeys is the World's key of each actor the map placed.
func secondPlacedKeys(w *sim.World) map[uint16]uint32 {
	out := map[uint16]uint32{}
	for _, e := range w.Entities() {
		if e.MapUnitID != 0 {
			out[e.MapUnitID] = e.SourceBinding.RuntimeID
		}
	}
	return out
}

// A second-game mission SAVE writes each creature and person the map placed
// with +4 = 0x6000 plus its map id, the key the original's client builds
// the actor's unit object under from the map (R2-ENGINE-156, R2-ENGINE-513,
// R2-ENGINE-615); the hero keeps his key. The engine document keeps the
// allocator's keys, so the engine's LOAD restores them; a LOAD of the
// records alone SAVEs the same keys, and the selection names a rekeyed
// actor by its new key. Fails while the records hold the allocator's keys.
func TestSecondMissionSavesEachMapActorWithItsMapIDPlus0x6000(t *testing.T) {
	c := secondMageMission()
	f, app, screen, _ := secondBagFrontOf(t, nil, c)
	secondMageItems(f)
	secondBagEnter(t, f, app, screen)
	before := secondPlacedKeys(f.live.world)
	placed := slices.Sorted(maps.Keys(before))
	if len(placed) != 6 {
		t.Fatalf("the World holds placed actors %v, want the two knights and four creatures", placed)
	}
	for id, key := range before {
		if key == 0 || key >= 0x6000 {
			t.Fatalf("the World's actor of map id %d holds key %d, want an allocator key", id, key)
		}
	}
	raw := secondPageSave(t, f, app)
	checkSecondMapActorKeys(t, "the SAVE", raw, placed)
	engine, _ := secondActorKeys(t, engineSaveDocument(t, raw))
	for id, key := range before {
		if engine[uint32(id)] != key {
			t.Errorf("the engine document's actor of map id %d holds %d, want the World's %d", id, engine[uint32(id)], key)
		}
	}

	loaded, _ := secondMageLoad(t, raw)
	if got := secondPlacedKeys(loaded.live.world); !maps.Equal(got, before) {
		t.Fatalf("the LOAD holds keys %v, want %v", got, before)
	}
	bare, bareApp := secondMageLoad(t, secondRecordsAlone(t, raw))
	checkSecondMapActorKeys(t, "the records' LOAD's SAVE", secondPageSave(t, bare, bareApp), placed)

	doc := engineSaveDocument(t, raw)
	heroKey := engine[secondHeroSlot]
	selection := make([]byte, 8)
	binary.LittleEndian.PutUint32(selection, heroKey)
	binary.LittleEndian.PutUint32(selection[4:], before[secondMageNecro])
	n, err := patchStateLeaf(doc.State.RootKind, doc.State.DirectoryRecords, doc.State.ValueRecords, "/Objects/Selection", func(leaf *sav.CityStateValueData) error {
		leaf.Bytes = selection
		return nil
	})
	if err != nil || n != 1 {
		t.Fatalf("the engine document holds %d selections: %v", n, err)
	}
	keyed := slices.DeleteFunc(slices.Clone(secondFormHolders), func(h savedFormHolder) bool { return h.field != "RuntimeID" })
	if err := writeSecondTownHolders(&doc, keyed, Snapshot{Mission: 10}, nil); err != nil {
		t.Fatal(err)
	}
	at := stateLeafRecords(doc.State.RootKind, doc.State.DirectoryRecords, doc.State.ValueRecords, "/Objects/Selection")
	got := doc.State.ValueRecords[at[0]].Value.Bytes
	if len(got) != 8 || binary.LittleEndian.Uint32(got) != heroKey || binary.LittleEndian.Uint32(got[4:]) != 0x6000+secondMageNecro {
		t.Fatalf("the selection holds % x, want the hero's %d and the necromancer's %d", got, heroKey, 0x6000+secondMageNecro)
	}
}

// A map-built actor of the hero's Player is keyed as every other map-built
// actor: its record's +4 and the selection that names it hold 0x6000 plus
// its map id (R2-ENGINE-950), while the hero, carried into the mission,
// keeps his key. Fails while the hero's Player's actors keep the
// allocator's key.
func TestSecondMissionSavesTheHeroPlayersMapActorWithItsMapIDPlus0x6000(t *testing.T) {
	c := secondMageMission()
	ours := uint16(secondMagePlain)
	for i := range c.units {
		if c.units[i].UnitID == ours {
			c.units[i].Owner = sim.SelfSlot
		}
	}
	f, app, screen, _ := secondBagFrontOf(t, nil, c)
	secondMageItems(f)
	secondBagEnter(t, f, app, screen)
	before := secondPlacedKeys(f.live.world)
	if key := before[ours]; key == 0 || key >= 0x6000 {
		t.Fatalf("the World's actor of map id %d holds key %d, want an allocator key", ours, key)
	}
	raw := secondPageSave(t, f, app)
	keys, heroOwned := secondActorKeys(t, checkSecondMissionFile(t, raw, 10))
	if !heroOwned[uint32(ours)] {
		t.Fatalf("the hero's Player does not own the actor of map id %d", ours)
	}
	checkSecondMapActorKeys(t, "the SAVE", raw, slices.Sorted(maps.Keys(before)))
	if got := keys[uint32(ours)]; got != 0x6000+uint32(ours) {
		t.Fatalf("the hero's Player's actor of map id %d holds +4 = %d", ours, got)
	}
	engine, _ := secondActorKeys(t, engineSaveDocument(t, raw))
	if engine[uint32(ours)] != before[ours] {
		t.Errorf("the engine document's actor of map id %d holds %d, want the World's %d", ours, engine[uint32(ours)], before[ours])
	}

	doc := engineSaveDocument(t, raw)
	heroKey := engine[secondHeroSlot]
	selection := make([]byte, 8)
	binary.LittleEndian.PutUint32(selection, heroKey)
	binary.LittleEndian.PutUint32(selection[4:], before[ours])
	n, err := patchStateLeaf(doc.State.RootKind, doc.State.DirectoryRecords, doc.State.ValueRecords, "/Objects/Selection", func(leaf *sav.CityStateValueData) error {
		leaf.Bytes = selection
		return nil
	})
	if err != nil || n != 1 {
		t.Fatalf("the engine document holds %d selections: %v", n, err)
	}
	keyed := slices.DeleteFunc(slices.Clone(secondFormHolders), func(h savedFormHolder) bool { return h.field != "RuntimeID" })
	// A member carried into the mission from a file whose record held map
	// id secondMageNecro keeps the key it holds.
	carried := Snapshot{Mission: 10, Party: []mapload.PartyMember{{Saved: &mapload.Saved{MapUnitID: secondMageNecro}}}, CurrentPartyIDs: []sim.EntityID{0}}
	if err := writeSecondTownHolders(&doc, keyed, carried, nil); err != nil {
		t.Fatal(err)
	}
	if kept, _ := secondActorKeys(t, doc); kept[secondMageNecro] != before[secondMageNecro] || kept[uint32(ours)] != 0x6000+uint32(ours) {
		t.Errorf("the carried member holds %d, want its %d, and the actor %d, want %d", kept[secondMageNecro], before[secondMageNecro], kept[uint32(ours)], 0x6000+uint32(ours))
	}
	at := stateLeafRecords(doc.State.RootKind, doc.State.DirectoryRecords, doc.State.ValueRecords, "/Objects/Selection")
	got := doc.State.ValueRecords[at[0]].Value.Bytes
	if len(got) != 8 || binary.LittleEndian.Uint32(got) != heroKey || binary.LittleEndian.Uint32(got[4:]) != 0x6000+uint32(ours) {
		t.Fatalf("the selection holds % x, want the hero's %d and the actor's %d", got, heroKey, 0x6000+uint32(ours))
	}
}
