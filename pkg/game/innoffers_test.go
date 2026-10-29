package game

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// A row added, switched off or gated in the data appears, disappears or
// follows its gate in the one offer builder: town, stages, the stage floor
// and each bank gate, in row order, the hire bank's options after them.
func TestInnOffersFollowTheRows(t *testing.T) {
	rows := []town.OfferRow{
		{Kind: 0, Topic: 5, NPC: 7, Town: 2, Stages: []int{30}},
		{Kind: 3, Topic: 6, NPC: 8, Zero: []int{900}},
		{Kind: 3, Topic: 7, NPC: 9, Any: []int{901, 902}},
		{Kind: 0, Topic: 8, NPC: 10, One: []int{903}, Nonzero: []int{904}},
		{Kind: 0, Topic: 9, NPC: 11, After: 40},
	}
	keys := func(s innState) []int {
		var out []int
		for _, o := range innOffers(rows, s) {
			out = append(out, o.topic)
		}
		return out
	}
	var b [innBankSlots]int32
	at := func(town, stage int) innState { return innState{town: town, stage: stage, bank: &b} }
	if got := keys(at(2, 30)); !slices.Equal(got, []int{5, 6}) {
		t.Fatalf("town 2 stage 30: %v", got)
	}
	if got := keys(at(3, 30)); !slices.Equal(got, []int{6}) {
		t.Fatalf("town 3 holds town 2's row: %v", got)
	}
	if got := keys(innState{every: true, stage: 30, bank: &b}); !slices.Equal(got, []int{5, 6}) {
		t.Fatalf("every town: %v", got)
	}
	b[900], b[902], b[903], b[904] = 1, 5, 1, -1
	if got := keys(at(2, 41)); !slices.Equal(got, []int{7, 8, 9}) {
		t.Fatalf("gates set at stage 41: %v", got)
	}
	b[903] = 2
	if got := keys(at(2, 40)); !slices.Equal(got, []int{7}) {
		t.Fatalf("slot 903 at 2 and stage 40: %v", got)
	}
	if got := keys(innState{town: 2, stage: 30}); !slices.Equal(got, []int{5, 6}) {
		t.Fatalf("no bank reads zero: %v", got)
	}
	b[secondHireAvailability+1], b[secondHireTown+1], b[secondHireSelection+1] = 1, 2, 1
	got := innOffers(rows[:1], innState{town: 2, stage: 30, bank: &b, hire: true})
	if want := []innOffer{{kind: 0, topic: 5, npc: 7}, {kind: 1, npc: 2, high: true}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("rows then hires: %+v, want %+v", got, want)
	}
	if got := innOffers(nil, innState{town: 2, stage: 30, bank: &b}); got != nil {
		t.Fatalf("a state without the hire bank hires: %+v", got)
	}
}

// The second game's inn reads the rows of its book: a row added to the
// data shows, a row switched off goes, and with none the hire actors alone
// remain.
func TestSecondInnReadsItsBookRows(t *testing.T) {
	c := innAt(2, 30, nil)
	shipped := c.innOptions()
	c.offers = append(slices.Clone(townBooks["rom2"].offerRows()), town.OfferRow{Kind: 0, Topic: 38, NPC: 2110, Town: 2, Stages: []int{30}})
	if got := c.innOptions(); len(got) != len(shipped)+1 || got[len(got)-1] != (innOffer{topic: 38, npc: 2110}) {
		t.Fatalf("added row: %+v", got)
	}
	c.offers = slices.DeleteFunc(slices.Clone(townBooks["rom2"].offerRows()), func(r town.OfferRow) bool { return r.Topic == 39 })
	if got := c.innOptions(); len(got) != len(shipped)-1 || slices.Contains(got, innOffer{topic: 39, npc: 2110}) {
		t.Fatalf("switched off row: %+v", got)
	}
	c.offers = []town.OfferRow{}
	c.bank[secondHireAvailability], c.bank[secondHireTown] = 2, 2
	if got := c.innOptions(); !reflect.DeepEqual(got, []innOffer{{kind: 2, npc: 1}}) {
		t.Fatalf("no rows: %+v", got)
	}
}

// The builder admits a tavern row at its chapter only.
func TestTavernRowsAreInnRows(t *testing.T) {
	offers := []town.OfferRow{town.ChapterOffer(1, 30, 22, 30), town.ChapterOffer(2, 30, 41, 0), town.ChapterOffer(1, 40, 23, 41)}
	if !offers[1].Equal(town.OfferRow{Kind: town.OfferTalk, NPC: 41, Town: 2, Stages: []int{30}}) {
		t.Fatalf("talk row: %+v", offers[1])
	}
	got := innOffers(offers, innState{every: true, stage: 30})
	if want := []innOffer{{kind: 3, topic: 30, npc: 22}, {kind: 0, npc: 41}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("chapter 30: %+v, want %+v", got, want)
	}
	if got := innOffers(offers[:1], innState{every: true, stage: 30}); len(got) != 1 {
		t.Fatalf("a dropped row stays: %+v", got)
	}
}

// The first game's catalog states the mercenary rule the tavern reads: 15
// stock types, unit id phase 6, the price ladder from mission 30 in steps of
// 10, and levels 2, 3 and 4 from missions 60, 100 and 130 (TAVERN-ORDER-015,
// DIV-1410). The expectations restate that rule independently of the data.
func TestShippedMercenaryRule(t *testing.T) {
	if tavernRule == nil || tavernRule.StockTypes != 15 || tavernRule.StockIDPhase != 6 {
		t.Fatalf("shipped rule %+v", tavernRule)
	}
	ladder := []int{10, 15, 20, 40, 60, 80, 100, 600, 800, 1000, 6000, 8000, 10000}
	for m := -20; m <= 200; m++ {
		price := 0
		if i := m/10 - 3; m >= 0 && i >= 0 && i < len(ladder) {
			price = ladder[i]
		}
		level := 1
		switch {
		case m >= 130:
			level = 4
		case m >= 100:
			level = 3
		case m >= 60:
			level = 2
		}
		if got := mercenaryUnitPrice(m); got != price {
			t.Errorf("mission %d: unit price %d, want %d", m, got, price)
		}
		if got := mercenaryLevel(m); got != level {
			t.Errorf("mission %d: level %d, want %d", m, got, level)
		}
	}
}

// writeInnMod writes a second-game mod id under dir whose towns file is
// towns.
func writeInnMod(t *testing.T, dir, id, towns string) {
	t.Helper()
	files := map[string]string{
		"mod.toml":        "id = \"" + id + "\"\ntitle = \"" + id + "\"\nversion = \"1.0\"\napplies-to = [\"common\"]\nentry = \"main.star\"\n",
		"main.star":       "def init(game, settings):\n    game.data.add(\"data/towns.toml\")\n",
		"data/towns.toml": towns,
	}
	for name, s := range files {
		path := filepath.Join(dir, id, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

const innRowsFixture = `[[inn]]
topic  = 38
npc    = 2110
kind   = 0
town   = 2
stages = [30]

[[inn]]
topic   = 31
npc     = 2108
nonzero = [533]

[[inn]]
topic = 39
npc   = 2110
off   = true
`

// A second-game mod edits an inn talk: it adds a talk, gates a shipped
// one and switches one off, and the inn's options follow; the first game
// refuses an [[inn]] table, the second a [[town]] table, and a gate outside
// the bank is refused at its line.
func TestSecondGameModEditsAnInnTalk(t *testing.T) {
	dir := t.TempDir()
	writeInnMod(t, dir, "inn", innRowsFixture)
	res := runTownModsFor(t, "rom2-en", dir, "inn")
	book, overrides, err := layerTownBook(townBooks["rom2"], res.Towns, Campaign{}, nil, nil, modTownInstall{})
	if err != nil || len(overrides) != 0 {
		t.Fatalf("layer: %v, overrides %v", err, overrides)
	}
	if townBooks["rom2"].catalog.Offers[4].Nonzero != nil {
		t.Fatal("the layer changed the shipped rows")
	}
	c := innAt(2, 30, nil)
	c.offers = book.offerRows()
	want := []innOffer{{kind: 3, topic: 30, npc: 22}, {kind: 3, topic: 31, npc: 2108}, {topic: 38, npc: 2110}}
	if got := c.innOptions(); !reflect.DeepEqual(got, []innOffer{want[0], want[2]}) {
		t.Fatalf("slot 533 clear: %+v", got)
	}
	c.bank[533] = 1
	if got := c.innOptions(); !reflect.DeepEqual(got[:3], want) {
		t.Fatalf("slot 533 set: %+v", got)
	}
	for _, bad := range []struct{ base, towns, want string }{
		{"rom2-en", "[[inn]]\ntopic = 40\nnpc = 22\nzero = [1024]\n", "data/towns.toml:1: inn row topic 40 npc 22: bank slot 1024 is outside the bank"},
		{"rom2-en", "[[inn]]\ntopic = 40\nnpc = 22\nkind = 1\n", "kind 1 is neither a talk"},
		{"rom2-en", "[[inn]]\ntopic = 40\nnpc = 22\ntown = 9\n", "town 9 is not a town of the catalog"},
		{"rom2-en", "[[town]]\nid = 2\nname = \"East\"\n", "catalog rom2 takes [[inn]] offer rows only"},
		{"rom2-en", "[[tavern]]\ntown = 2\nchapter = 30\nnpc = 22\nmission = 31\n", "catalog rom2 takes [[inn]] offer rows, not [[tavern]] ones"},
	} {
		d := t.TempDir()
		writeInnMod(t, d, "bad", bad.towns)
		res := runTownModsFor(t, bad.base, d, "bad")
		if _, _, err := layerTownBook(townBooks["rom2"], res.Towns, Campaign{}, nil, nil, modTownInstall{}); err == nil || !strings.Contains(err.Error(), bad.want) {
			t.Errorf("%q: %v, want %q", bad.towns, err, bad.want)
		}
	}
	d := t.TempDir()
	writeInnMod(t, d, "first", innRowsFixture)
	res = runTownModsFor(t, "rom1-en", d, "first")
	if _, _, err := layerTownBook(townBooks["rom1"], res.Towns, Campaign{}, nil, nil, modTownInstall{}); err == nil || !strings.Contains(err.Error(), "catalog rom1 takes [[tavern]] offer rows, not [[inn]] ones") {
		t.Fatalf("the first game took an inn row: %v", err)
	}
}

// secondModFront is a second-game install without scenario.reg, with two
// talk sections and NPC records npcs.
func secondModFront(t *testing.T, npcs ...int32) (*FrontEnd, *ui.App, *townScreen) {
	t.Helper()
	text := synth.File{Path: "text/town.txt", Data: []byte("#npc2108talk31\r\n<NPC=2108,PART=1>\r\nThirty-one.\r\n#npc4321talk99\r\n<NPC=4321,PART=1>\r\nNinety-nine.\r\n#other\r\n")}
	f, app, screen := secondCampaignFixtureFiles(t, false, text)
	f.Campaign = resolved(Campaign{}, errors.New("read scenario/scenario.reg: file does not exist"))
	f.NPCFaces = map[int32]data.NPCFace{}
	for _, n := range npcs {
		f.NPCFaces[n] = data.NPCFace{}
	}
	return f, app, screen
}

const innProbeFixture = `[[inn]]
topic  = 99
npc    = 4321
kind   = 0
town   = 2
stages = [30]

[[inn]]
topic = 39
npc   = 2110
off   = true
`

// A second-game mod loads through SetModTowns without a scenario: the inn
// lists and opens the added row and drops the one switched off.
func TestSecondGameModLoadsWithoutAScenario(t *testing.T) {
	f, app, screen := secondModFront(t, 4321)
	dir := t.TempDir()
	writeInnMod(t, dir, "probe", innProbeFixture)
	if _, err := f.SetModTowns(runTownModsFor(t, "rom2-en", dir, "probe").Towns); err != nil {
		t.Fatal(err)
	}
	if f.Campaign.Err() == nil || f.Campaign.Value().Towns.catalog.OfferTable != town.InnTable {
		t.Fatalf("campaign after the layer: %v", f.Campaign.Err())
	}
	c := innAt(2, 30, nil)
	f.Town.second = c
	app.SetTown(screen)
	screen.Choose(0)
	want := []ui.TownRow{{Text: "TALK 22", Choosable: true}, {Text: "TALK 2108", Choosable: true}, {Text: "TALK 4321", Choosable: true}, {Text: "EXIT", Choosable: true}}
	if c.room != secondTownInn || !reflect.DeepEqual(screen.Rows(), want) {
		t.Fatal("inn rows", screen.Rows())
	}
	screen.Choose(2)
	if body, ok := screen.dialogueBody(); !ok || !strings.Contains(body, "Ninety-nine.") {
		t.Fatalf("TALK 4321 shows %q", body)
	}
}

// A second-game row naming a missing NPC record, talk section or mission
// is refused at its line, as a tavern row is.
func TestSecondGameModRefusesMissingReferences(t *testing.T) {
	for _, c := range []struct{ name, towns, want string }{
		{"no npc record", "[[inn]]\ntopic = 99\nnpc = 4322\ntown = 2\nstages = [30]\n",
			`mod "m": data/towns.toml:1: inn row topic 99 npc 4322: npc 4322 is not a record of the install's NPC table`},
		{"no talk text", "[[inn]]\ntopic = 98\nnpc = 4321\ntown = 2\nstages = [30]\n",
			`mod "m": data/towns.toml:1: inn row topic 98 npc 4321: its talk text section npc4321talk98 of main/text/town.txt is neither in the install nor served by a loaded mod`},
		{"no mission", "\n[[inn]]\ntopic = 999\nnpc = 4321\nkind = 3\n",
			`mod "m": data/towns.toml:2: inn row topic 999 npc 4321: mission 999 is not a mission record of the world map`},
		{"changed row without its talk", "[[inn]]\ntopic = 39\nnpc = 2110\nstages = [40]\n",
			"inn row topic 39 npc 2110: its talk text section npc2110talk39 of main/text/town.txt is neither"},
	} {
		t.Run(c.name, func(t *testing.T) {
			f, _, _ := secondModFront(t, 4321, 2110)
			dir := t.TempDir()
			writeInnMod(t, dir, "m", c.towns)
			if _, err := f.SetModTowns(runTownModsFor(t, "rom2-en", dir, "m").Towns); err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("%v, want %q", err, c.want)
			}
		})
	}
}
