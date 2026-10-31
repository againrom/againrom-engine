package game

import (
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

// secondTownHireRow is the Humans row of the hire the town fixture carries:
// above a byte, as the second game's rows run (R2-ENGINE-739).
const secondTownHireRow = 300

// secondTownWideHireTable is the town fixture's table with a Humans row 300
// for a type-3 hire.
func secondTownWideHireTable() *mapload.Table {
	t := secondDollTable()
	rows := make(dbCollection, secondTownHireRow+1)
	params := make([]int32, data.MinHumanRow)
	params[0], params[1], params[2], params[3] = 25, 20, 15, 10
	params[4], params[16], params[17], params[18] = 100, 0x17, 1, 1
	params[21], params[22] = 1, 1
	rows[secondTownHireRow] = dbEntry{name: "NPC03_1", params: params}
	t.Humans = rows
	return t
}

// secondTownHireRecord is the one Human record of doc not named Aldor: the
// hire, whose name is his row's and so is not written.
func secondTownHireRecord(t *testing.T, name string, doc sav.DocumentData) sav.DocumentRecordData {
	t.Helper()
	var found []sav.DocumentRecordData
	for _, r := range doc.Objects {
		if r.Class == "Human" && !slices.ContainsFunc(r.Texts, func(v sav.DocumentTextData) bool { return v.Name == "Name" && v.Value == "Aldor" }) {
			found = append(found, r)
		}
	}
	if len(found) != 1 {
		t.Fatalf("%s: %d Human records name the hire, want 1", name, len(found))
	}
	return found[0]
}

// A second-game town SAVE whose party holds a hire of Humans row 300 writes
// the row whole: the Bsg& record's +0xc holds 300, the engine's own Asg&
// document answers 300 for the hire (its head keeps the low byte, the rows
// leaf the row), and the file LOADs the hire with row 300. Fails while a
// town head refuses a row above a byte.
func TestSecondTownSaveKeepsAHireRowAbove255(t *testing.T) {
	f, _, _ := secondCampaignFixtureFiles(t, false)
	f.Table = secondTownWideHireTable()
	hero := mapload.PartyMember{ID: "hero", Name: "Aldor", PlayerCharacter: true, StartingHero: true}
	hire := mapload.PartyMember{ID: "hire", Name: "NPC03_1", MercenaryType: 3, DefinitionRow: secondTownHireRow}
	f.Carried = []mapload.PartyMember{hero, hire}
	c := secondShopSample()
	c.current, c.available, c.room = partyPlace{2, 1}, []partyPlace{{2, 1}, {1, 10}}, secondTownShop
	c.bank[768] = 10
	f.Town.second, f.Town.gold = c, 500
	s, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatalf("the town SAVE: %v", err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	if row, _ := secondRecordValue(secondTownHireRecord(t, "the Bsg& records", doc), "T0C"); doc.Game != "rom2" || row != secondTownHireRow {
		t.Fatalf("the Bsg& record of the hire holds row %d (game %q), want %d", row, doc.Game, secondTownHireRow)
	}
	engine, err := sav.EngineDocument(raw)
	if err != nil {
		t.Fatal(err)
	}
	inner, err := sav.DecodeDocumentData(engine)
	if err != nil {
		t.Fatal(err)
	}
	if row, _ := secondRecordValue(secondTownHireRecord(t, "the engine document", inner), "T0C"); row != secondTownHireRow {
		t.Fatalf("the engine document answers row %d for the hire, want %d", row, secondTownHireRow)
	}
	cold, _, _ := secondCampaignFixtureFiles(t, false)
	cold.Table = secondTownWideHireTable()
	if _, town, err := cold.RestoreOriginal(raw); err != nil || !town {
		t.Fatal("cold town LOAD", town, err)
	}
	var rows []uint16
	for _, m := range cold.Carried {
		if m.Hired() {
			rows = append(rows, m.DefinitionRow)
		}
	}
	if len(rows) != 1 || rows[0] != secondTownHireRow {
		t.Fatalf("the LOAD's hires hold rows %v, want [%d]", rows, secondTownHireRow)
	}
}

// A writer of the town head alone, with no record to hold the row, still
// refuses a row above a byte rather than write its low byte.
func TestTownHeadAloneRefusesAHireRowAbove255(t *testing.T) {
	table := secondTownWideHireTable()
	hero := mapload.PartyMember{ID: "hero", Name: "Aldor", PlayerCharacter: true, StartingHero: true}
	hire := mapload.PartyMember{ID: "hire", Name: "NPC03_1", MercenaryType: 3, DefinitionRow: secondTownHireRow}
	if _, err := nativeCityData([]mapload.PartyMember{hero}, []mapload.PartyMember{hire}, table, mapload.DifficultyNormal); err == nil {
		t.Fatal("the town head alone wrote the low byte of row 300")
	}
	if _, err := nativeCityDataApproximate([]mapload.PartyMember{hero}, []mapload.PartyMember{hire}, table, mapload.DifficultyNormal); err == nil {
		t.Fatal("the approximate town head alone wrote the low byte of row 300")
	}
}
