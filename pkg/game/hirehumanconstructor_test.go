package game

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// A first-game town document writes a hired Human with the hire constructor
// values; the hero keeps mask 2 and an empty bag's insert index.
func TestTownDocumentWritesHumanHireConstructorValues(t *testing.T) {
	table := secondTownWideHireTable()
	edition := base.GameROM1.Edition()
	table.Edition = &edition
	hero := mapload.PartyMember{ID: "hero", Name: "Aldor", PlayerCharacter: true, StartingHero: true}
	hire := mapload.PartyMember{ID: "hire", Name: "NPC03_1", MercenaryType: 3, DefinitionRow: secondTownHireRow}
	doc, _, err := cityBaseDocument(table, Snapshot{Party: []mapload.PartyMember{hero, hire}})
	if err != nil {
		t.Fatal(err)
	}
	var humans int
	for _, r := range doc.Objects {
		if r.Class != "Human" {
			continue
		}
		humans++
		mover, err := savedActorRaw(&r, "U154", 180)
		if err != nil {
			t.Fatal(err)
		}
		state, err := savedActorRaw(&r, "U50", 4)
		if err != nil {
			t.Fatal(err)
		}
		head := sim.MoverHeadOf(mover)
		var got [6]uint32
		for i, field := range []string{"T18", "Inventory1C", "U136"} {
			got[i], _ = secondRecordValue(r, field)
		}
		got[3], got[4], got[5] = binary.LittleEndian.Uint32(state), uint32(head.Radius), uint32(head.SiegeRoute)
		want := [6]uint32{0, siegeHireInsertIndex, siegeHireU136, siegeHireStateWord, 5, 0xff}
		if humans == 1 {
			want = [6]uint32{2, 0, got[2], got[3], 0, 0}
		}
		if got != want {
			t.Fatalf("Human %d: mask, insert index, flag word, state word, mover bytes %v, want %v", humans, got, want)
		}
	}
	if humans != 2 {
		t.Fatalf("the document holds %d Human records, want 2", humans)
	}
}
