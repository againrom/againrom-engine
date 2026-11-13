package mod

import (
	"slices"
	"strings"
	"testing"

	"againrom/pkg/town"
)

// A tavern table takes a chapter floor and won missions, and a shop and a
// school table are read into the same row type by the same parser.
func TestOfferTablesReadTheFloorTheWonListAndTheBuildings(t *testing.T) {
	src := "[[tavern]]\ntown = 1\nfrom = 50\nnpc = 41\nmission = 31\nwon = [51, 30]\n" +
		"[[shop]]\ntown = 2\nchapter = 60\nmission = 61\n" +
		"[[school]]\ntown = 1\nfrom = 40\nmission = 41\nwon = [31]\n" +
		"[[shop]]\ntown = 1\nchapter = 30\nmission = 31\noff = true\n"
	_, rows, err := ParseTowns("m", TownsFile, []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	want := []town.OfferRow{
		{Kind: town.OfferMission, Topic: 31, NPC: 41, Town: 1, From: 50, Won: []int{51, 30}},
		{Kind: town.OfferMission, Topic: 61, Town: 2, Stages: []int{60}, Place: town.PlaceShop},
		{Kind: town.OfferMission, Topic: 41, Town: 1, From: 40, Won: []int{31}, Place: town.PlaceSchool},
		{Kind: town.OfferMission, Topic: 31, Town: 1, Stages: []int{30}, Place: town.PlaceShop},
	}
	keys := []string{"town 1 chapter 50 npc 41", "town 2 chapter 60 mission 61", "town 1 chapter 40 mission 41", "town 1 chapter 30 mission 31"}
	tables := []town.OfferTable{town.TavernTable, town.ShopTable, town.SchoolTable, town.ShopTable}
	if len(rows) != len(want) {
		t.Fatalf("%d rows, want %d", len(rows), len(want))
	}
	for i, r := range rows {
		var got town.OfferRow
		r.Apply(&got)
		if !got.Equal(want[i]) || r.Key().String() != keys[i] || r.Table != tables[i] || r.Table.Catalog() != town.TavernTable || r.Off != (i == 3) {
			t.Fatalf("row %d: %+v key %q table %s off %v, want %+v key %q table %s", i, got, r.Key(), r.Table, r.Off, want[i], keys[i], tables[i])
		}
	}
	// A floor written over a row of one chapter replaces the chapter, and
	// a chapter written over a floor row replaces the floor.
	exact := town.ChapterOffer(1, 50, 41, 0)
	rows[0].Apply(&exact)
	if len(exact.Stages) != 0 || exact.From != 50 {
		t.Fatalf("the floor left the chapter: %+v", exact)
	}
}

// Each new key refuses what it cannot hold with the file and the line.
func TestOfferTableRefusalsOfTheNewKeys(t *testing.T) {
	for _, c := range []struct{ name, src, want string }{
		{"chapter and from", "[[tavern]]\ntown = 1\nchapter = 50\nfrom = 50\nnpc = 41\nmission = 0\n",
			"data/towns.toml:1: [[tavern]] states both chapter and from; a row stands in one chapter or from one on"},
		{"neither", "[[shop]]\ntown = 1\nmission = 31\n", "data/towns.toml:1: [[shop]] has no chapter; town, chapter and mission name the row"},
		{"won not a list", "[[tavern]]\ntown = 1\nchapter = 50\nnpc = 41\nwon = 31\n", "data/towns.toml:5: won must be an array of integers"},
		{"won zero", "[[tavern]]\ntown = 1\nchapter = 50\nnpc = 41\nwon = [0]\n", "data/towns.toml:5: won"},
		{"from zero", "[[school]]\ntown = 1\nfrom = 0\nmission = 31\n", "data/towns.toml:3: from"},
		{"shop names an npc", "[[shop]]\ntown = 1\nchapter = 30\nnpc = 4\nmission = 31\n",
			`data/towns.toml:4: unknown key "npc" in [[shop]] (keys: town, chapter, from, mission, won, off)`},
		{"shop mission zero", "[[shop]]\ntown = 1\nchapter = 30\nmission = 0\n", "data/towns.toml:4: mission"},
		{"inn takes no won", "[[inn]]\ntopic = 7\nnpc = 520\nwon = [31]\n", `data/towns.toml:4: unknown key "won" in [[inn]]`},
		{"shop row twice", "[[shop]]\ntown = 1\nchapter = 30\nmission = 31\n[[shop]]\ntown = 1\nfrom = 30\nmission = 31\n",
			"data/towns.toml:5: shop row town 1 chapter 30 mission 31 is named again; its first table is at line 1"},
		{"unknown table", "[[gate]]\ntown = 1\n", `unknown table "gate" (this file holds [[town]], [[tavern]], [[shop]], [[school]] and [[inn]])`},
	} {
		t.Run(c.name, func(t *testing.T) {
			if _, _, err := ParseTowns("m", TownsFile, []byte(c.src)); err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("%v, want %q", err, c.want)
			}
		})
	}
}

// A written file holds each row under its building's table with only the
// keys the row has a value for, and a row without the new keys is written
// as before them.
func TestWriteTownsWritesEachRowUnderItsTable(t *testing.T) {
	shop := town.ChapterOffer(1, 30, 0, 31)
	shop.Place = town.PlaceShop
	floor := town.OfferRow{Kind: town.OfferMission, Topic: 41, NPC: 23, Town: 1, From: 40, Won: []int{31, 30}}
	out, err := WriteTowns(nil, nil, town.ChapterOffer(1, 30, 22, 30), shop, floor)
	if err != nil {
		t.Fatal(err)
	}
	want := "\n[[tavern]]\ntown        = 1\nchapter     = 30\nnpc         = 22\nmission     = 30\noff         = false\n" +
		"\n[[shop]]\ntown        = 1\nchapter     = 30\nmission     = 31\noff         = false\n" +
		"\n[[tavern]]\ntown        = 1\nfrom        = 40\nnpc         = 23\nmission     = 41\nwon         = [31, 30]\noff         = false\n"
	if string(out) != want {
		t.Fatalf("written\n%s\nwant\n%s", out, want)
	}
	_, rows, err := ParseTowns("", TownsFile, out)
	if err != nil || len(rows) != 3 || !slices.Equal(rows[2].Row.Won, []int{31, 30}) {
		t.Fatalf("read back %+v, %v", rows, err)
	}
}
