package game

import (
	"bytes"
	"encoding/binary"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// secondTownSave is a second-game town SAVE: an axe hero with 500 gold in
// the first town, the first mission open, an axe and a staff in the bag.
func secondTownSave(t *testing.T) []byte {
	t.Helper()
	f, _, _ := secondCampaignFixtureFiles(t, false)
	f.Table = secondDollTable()
	hero := mapload.PartyMember{ID: "hero", Name: "Aldor", PlayerCharacter: true, StartingHero: true}
	hero.Hero.Skill[2] = 20
	f.Carried = []mapload.PartyMember{hero}
	c := secondShopSample()
	c.current, c.available, c.room = partyPlace{2, 1}, []partyPlace{{2, 1}, {1, 10}}, secondTownShop
	c.bank[768] = 10
	*c.heroBag() = []sim.ItemInstance{secondDollAxe, secondDollStaff}
	f.Town.second, f.Town.gold = c, 500
	s, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// withoutEngineDocument is the file as the second game's original reads
// it: its records alone, without the engine document leaf.
func withoutEngineDocument(t *testing.T, raw []byte) []byte {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	n := len(doc.State.ValueRecords)
	doc.State.ValueRecords = slices.DeleteFunc(slices.Clone(doc.State.ValueRecords), func(r sav.CityStateRecordData) bool {
		return strings.EqualFold(r.Path, sav.NativeFormPath)
	})
	if len(doc.State.ValueRecords) != n-1 {
		t.Fatal("the town file holds no engine document leaf")
	}
	bare, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	return bare
}

func coldSecondTown(t *testing.T, raw []byte) *FrontEnd {
	t.Helper()
	cold, _, _ := secondCampaignFixtureFiles(t, false)
	cold.Table = secondDollTable()
	if _, town, err := cold.RestoreOriginal(raw); err != nil || !town {
		t.Fatal("cold town LOAD", town, err)
	}
	return cold
}

// checkSecondTown: the LOAD stands in the first town with the first mission
// open, the hero, 500 gold and the axe and staff in the bag.
func checkSecondTown(t *testing.T, name string, cold *FrontEnd) {
	t.Helper()
	c := cold.Town.second
	if c == nil || c.current != (partyPlace{2, 1}) || !slices.Equal(c.available, []partyPlace{{2, 1}, {1, 10}}) || c.bank[768] != 10 {
		t.Fatalf("%s: campaign position %+v", name, c)
	}
	if cold.Town.gold != 500 || len(cold.Carried) != 1 || cold.Carried[0].Name != "Aldor" {
		t.Fatalf("%s: gold %d party %d", name, cold.Town.gold, len(cold.Carried))
	}
	var codes []uint16
	for _, m := range c.party {
		for _, it := range m.CarriedItems {
			codes = append(codes, it.Code)
		}
	}
	if !slices.Equal(codes, []uint16{secondDollAxe.Code, secondDollStaff.Code}) {
		t.Fatalf("%s: bag %#04x", name, codes)
	}
}

// TestSecondTownSaveWritesTheSecondGameRecords: a second-game town SAVE is a
// Bsg& file whose records hold the town, the hero with his main skill, the
// bag in his Inventory, the purse and the campaign tail, so that its records
// alone LOAD to the same town; the whole file LOADs the same town through
// the engine's own document, which is the Asg& document the engine wrote
// before and still loads.
func TestSecondTownSaveWritesTheSecondGameRecords(t *testing.T) {
	raw := secondTownSave(t)
	if !bytes.HasPrefix(raw, []byte("Bsg&")) {
		t.Fatalf("the town file opens with %q", raw[:4])
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	tail := doc.SecondTail
	if doc.Game != "rom2" || tail == nil || tail.Bank[768] != 10 || tail.Current != 0 || len(tail.Locations) != 2 ||
		tail.Locations[0].ID != 1 || tail.Locations[0].Kind != 2 || tail.Locations[1].ID != 10 || tail.Locations[1].Kind != 1 {
		t.Fatalf("game %q tail %+v", doc.Game, tail)
	}
	if _, found, _ := doc.State.ReadLeaf(sav.NativeActionsPath); found {
		t.Fatal("the second game's records carry the engine's action leaf")
	}
	hero, err := secondTownHeroRecord(doc)
	if err != nil {
		t.Fatal(err)
	}
	r := &doc.Objects[hero-1]
	xp := documentRawBytes(t, r, "H1CC")
	insert, _ := savedStructureValue(r, "Inventory1C")
	if binary.LittleEndian.Uint32(xp) != 2 || insert != 0x7fff {
		t.Fatalf("hero main skill %d insert index %d", binary.LittleEndian.Uint32(xp), insert)
	}
	// Each bag Item is owned by the hero's Player and the bag's weight is
	// unit weight times quantity summed (R2-ENGINE-402, R2-ENGINE-403).
	owner, _ := savedStructureValue(r, "Reference")
	var weight uint32
	for _, slot := range r.RefSlots {
		if slot.Name != "Inventory" {
			continue
		}
		if len(slot.Objects) != 2 {
			t.Fatalf("hero Inventory %v", slot.Objects)
		}
		for _, ref := range slot.Objects {
			item := &doc.Objects[ref-1]
			unit, _ := savedStructureValue(item, "F4A")
			count, _ := savedStructureValue(item, "F42")
			if got, _ := savedStructureValue(item, "Reference"); got != owner || count == 0 {
				t.Fatalf("bag %s owner %d quantity %d, want owner %d", item.Class, got, count, owner)
			}
			weight += unit * count
		}
	}
	if got, _ := savedStructureValue(r, "Inventory20"); got != weight {
		t.Fatalf("bag weight %d, want %d", got, weight)
	}
	sf, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	chars, err := sf.Party()
	if err != nil {
		t.Fatal(err)
	}
	if len(chars) != 1 || !chars[0].Hero || chars[0].MainSkill != 2 || len(chars[0].Items) != 2 {
		t.Fatalf("the original's reader finds %+v", chars)
	}
	checkSecondTown(t, "the second game's records", coldSecondTown(t, withoutEngineDocument(t, raw)))
	checkSecondTown(t, "the town file", coldSecondTown(t, raw))
	engine, err := sav.EngineDocument(raw)
	if err != nil || !bytes.HasPrefix(engine, []byte("Asg&")) {
		t.Fatalf("engine document: %v", err)
	}
	checkSecondTown(t, "the engine document", coldSecondTown(t, engine))
}

// TestSecondTownSaveKeepsTheLoadedHeroRecord: a town LOAD of a second-game
// original and its town SAVE keep the hero's Humans row (R2-ENGINE-319), his
// party identity 21 (R2-ENGINE-105), the actor state his record holds and
// the Player colour (the fixture's 2), and a LOAD of that SAVE keeps them
// again.
func TestSecondTownSaveKeepsTheLoadedHeroRecord(t *testing.T) {
	hero := secondFixtureHero()
	hero.townState, hero.townDone, hero.timer, hero.identity = 0x0b, 1, 128, 21
	raw := secondFixtureTown("town", hero, 44)
	humans := make(dbCollection, 30)
	humans[28] = actorRegistryTable().Humans.(dbCollection)[1]
	humans[28].name = "Start_Gildarius"
	for round := range 2 {
		cold, _, _ := secondCampaignFixtureFiles(t, false)
		cold.Table = secondDollTable()
		cold.Table.Humans = humans
		if _, town, err := cold.RestoreOriginal(raw); err != nil || !town {
			t.Fatal("cold town LOAD", round, town, err)
		}
		s, label, err := cold.Snapshot(false)
		if err != nil {
			t.Fatal(err)
		}
		if raw, err = cold.ExportCurrentSave(s, label); err != nil {
			t.Fatal(err)
		}
		got, err := sav.DecodeDocumentData(withoutEngineDocument(t, raw))
		if err != nil {
			t.Fatal(err)
		}
		at, err := secondTownHeroRecord(got)
		if err != nil {
			t.Fatal(err)
		}
		h := &got.Objects[at-1]
		for name, want := range map[string]uint32{"T0C": 28, "U136": 1, "U6C": 128, "U148": 21} {
			if v, _ := savedStructureValue(h, name); v != want {
				t.Errorf("SAVE %d: hero %s %d, want %d", round, name, v, want)
			}
		}
		if b := documentRawBytes(t, h, "U50"); binary.LittleEndian.Uint32(b) != 0x0b {
			t.Errorf("SAVE %d: hero U50 % x, want the town state 0x0b", round, b)
		}
		if v, _ := savedStructureValue(&got.Objects[got.Players[0]-1], "F44"); v != 2 {
			t.Errorf("SAVE %d: Player colour %d, want 2", round, v)
		}
		if v, _ := savedStructureValue(&got.Objects[got.Players[0]-1], "F48"); v != 0 {
			t.Errorf("SAVE %d: Player F48 %#x, want 0", round, v)
		}
	}
}

// TestSaveFormKeepsTheEngineRecords: the first game writes the engine's own
// records.
func TestSaveFormKeepsTheEngineRecords(t *testing.T) {
	doc := sav.DocumentData{Label: []byte("x")}
	if _, ok, err := (firstCampaignRules{}).saveForm(nil, Snapshot{}, doc); ok || err != nil {
		t.Fatalf("the first game's SAVE changed form: %v", err)
	}
}

// engineSaveDocument is the engine's own document a SAV carries: the
// AgainromForm leaf of a second-game town file, else the file itself.
func engineSaveDocument(t *testing.T, raw []byte) sav.DocumentData {
	t.Helper()
	engine, err := sav.EngineDocument(raw)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(engine)
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func documentRawBytes(t *testing.T, r *sav.DocumentRecordData, name string) []byte {
	t.Helper()
	for _, v := range r.Raw {
		if v.Name == name {
			return v.Bytes
		}
	}
	t.Fatalf("%s record holds no %s", r.Class, name)
	return nil
}
