package game

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/town"
)

// tavernLeaves serves an empty talk text at each path on f's install.
func tavernLeaves(f *FrontEnd, paths ...string) {
	for _, p := range paths {
		f.Archives.Containers.Overlay(p, func() ([]byte, error) { return []byte("<part=1>\r\nwords"), nil })
	}
}

// tavernLayerFront is layerFront with the talk texts at paths and the town
// layers of the mods ids of dir set.
func tavernLayerFront(t *testing.T, dir string, paths []string, ids ...string) (*FrontEnd, []string, error) {
	t.Helper()
	f := layerFront(t)
	tavernLeaves(f, paths...)
	overrides, err := f.SetModTowns(runTownMods(t, dir, ids...).Towns)
	return f, overrides, err
}

const tavernRowsFixture = `[[tavern]]
town    = 1
chapter = 30
npc     = 41
mission = 0

[[tavern]]
town    = 1
chapter = 30
npc     = 52
mission = 31

[[tavern]]
town    = 1
chapter = 40
npc     = 23
mission = 41

[[tavern]]
town    = 1
chapter = 30
npc     = 90
off     = true
`

var tavernRowsLeaves = []string{"main/text/inn/npc/npc41m30.txt", "main/text/inn/npc/npc52m31.txt", "main/text/inn/npc/npc23m41.txt"}

// A mod's rows add a talk and a quest to a chapter, change a row's mission
// and switch a row off; the chapter's offer data, the tavern's offers and
// an accepted quest read the merged rows.
func TestTavernLayerAddsChangesAndSwitchesOff(t *testing.T) {
	dir := t.TempDir()
	writeTownMod(t, dir, "tavern", tavernRowsFixture)
	f, overrides, err := tavernLayerFront(t, dir, tavernRowsLeaves, "tavern")
	if err != nil || len(overrides) != 0 {
		t.Fatalf("layer: %v, overrides %v", err, overrides)
	}
	c := f.Campaign.Value()
	for _, want := range []struct {
		chapter     int
		npc, offers []int
	}{{30, []int{22, 41, 52}, []int{30, 0, 31}}, {40, []int{23}, []int{41}}} {
		ch := c.chapter(want.chapter)
		if !slices.Equal(ch.InnNPC, want.npc) || !slices.Equal(ch.Inn, want.offers) || ch.Mission != want.chapter {
			t.Fatalf("chapter %d: npc %v missions %v, want %v %v", want.chapter, ch.InnNPC, ch.Inn, want.npc, want.offers)
		}
		if raw := c.Chapters[want.chapter]; !slices.Equal(ch.Shop, raw.Shop) || !slices.Equal(ch.School, raw.School) || ch.Payment != raw.Payment {
			t.Fatalf("chapter %d's other offer data changed: %+v", want.chapter, ch)
		}
	}
	tw := f.Town
	tw.Arrive()
	want := []TownOffer{{Index: 0, Mission: 30, NPC: 22}, {Index: 1, NPC: 41}, {Index: 2, Mission: 31, NPC: 52}}
	if got := tw.Offers(TownTavern); !reflect.DeepEqual(got, want) {
		t.Fatalf("tavern offers %+v, want %+v", got, want)
	}
	if m, ok := tw.Take(TownTavern, 2); !ok || m != 31 || !slices.Contains(tw.Available(), 31) {
		t.Fatalf("the mod quest: took %d %v, available %v", m, ok, tw.Available())
	}
	if got := tw.TavernRoster(); len(got) != 3 || got[2].Index >= 0 || got[2].NPC != 52 {
		t.Fatalf("the roster keeps the mod quest's speaker: %+v", got)
	}
}

// With no tavern row in any mod the catalog states none and every chapter
// is the scenario's own; a tavern layer leaves every chapter it does not
// name equal to the scenario's.
func TestTavernLayerLeavesTheScenarioChapters(t *testing.T) {
	dir := t.TempDir()
	writeTownMod(t, dir, "named", "[[town]]\nid = 1\nname = \"West\"\n")
	writeTownMod(t, dir, "forty", "[[tavern]]\ntown = 1\nchapter = 40\nnpc = 23\nmission = 41\n")
	f, _, err := tavernLayerFront(t, dir, nil, "named")
	if err != nil {
		t.Fatal(err)
	}
	c := f.Campaign.Value()
	if _, ok := c.Towns.tavernRows(); ok {
		t.Fatal("a towns layer without tavern rows states rows")
	}
	for n, ch := range c.Chapters {
		if !reflect.DeepEqual(c.chapter(n), ch) {
			t.Fatalf("chapter %d: %+v, want the scenario's %+v", n, c.chapter(n), ch)
		}
	}
	g, _, err := tavernLayerFront(t, dir, []string{"main/text/inn/npc/npc23m41.txt"}, "forty")
	if err != nil {
		t.Fatal(err)
	}
	c = g.Campaign.Value()
	for n, ch := range c.Chapters {
		got := c.chapter(n)
		if n == 40 {
			continue
		}
		if !slices.Equal(got.InnNPC, ch.InnNPC) || !slices.Equal(got.Inn, ch.Inn) {
			t.Fatalf("chapter %d: npc %v missions %v, want the scenario's %v %v", n, got.InnNPC, got.Inn, ch.InnNPC, ch.Inn)
		}
	}
	rows, _ := c.Towns.tavernRows()
	if want := (town.TavernRow{Town: 1, Chapter: 30, NPC: 22, Mission: 30}); len(rows) != 3 || rows[0] != want {
		t.Fatalf("rows %+v, want layer 0 first", rows)
	}
}

// A refusal names the mod, the file, the line and the row.
func TestTavernLayerRefusals(t *testing.T) {
	for _, c := range []struct {
		name, towns string
		leaves      []string
		npcs        []int32
		want        string
	}{
		{"unknown town", "[[tavern]]\ntown = 5\nchapter = 30\nnpc = 41\nmission = 0\n", []string{"main/text/inn/npc/npc41m30.txt"}, nil,
			`mod "m": data/towns.toml:1: town catalog rom1: tavern row town 5 chapter 30 npc 41: town 5 is not a town of the catalog`},
		{"unknown row", "[[tavern]]\ntown = 1\nchapter = 30\nnpc = 99\noff = true\n", nil, nil,
			`mod "m": data/towns.toml:1: tavern row town 1 chapter 30 npc 99 is switched off, but no layer holds it`},
		{"key not settable", "[[tavern]]\ntown = 1\nchapter = 30\nnpc = 22\nspeaker = \"Ann\"\n", nil, nil,
			`unknown key "speaker" in [[tavern]]`},
		{"side chapter", "[[tavern]]\ntown = 1\nchapter = 31\nnpc = 41\nmission = 0\n", []string{"main/text/inn/npc/npc41m31.txt"}, nil,
			`mod "m": data/towns.toml:1: tavern row town 1 chapter 31 npc 41: chapter 31 is not a main mission of the campaign`},
		{"no such mission", "[[tavern]]\ntown = 1\nchapter = 30\nnpc = 41\nmission = 77\n", []string{"main/text/inn/npc/npc41m77.txt"}, nil,
			"tavern row town 1 chapter 30 npc 41: mission 77 is not a mission section of the campaign"},
		{"no talk text", "[[tavern]]\ntown = 1\nchapter = 30\nnpc = 41\nmission = 0\n", nil, nil,
			"tavern row town 1 chapter 30 npc 41: its talk text main/text/inn/npc/npc41m30.txt is neither in the install nor served by a loaded mod"},
		{"changed row without text", "[[tavern]]\ntown = 1\nchapter = 30\nnpc = 22\nmission = 31\n", nil, nil,
			"its talk text main/text/inn/npc/npc22m31.txt is neither"},
		{"no npc record", "[[tavern]]\ntown = 1\nchapter = 30\nnpc = 41\nmission = 0\n", []string{"main/text/inn/npc/npc41m30.txt"}, []int32{22, 90},
			"tavern row town 1 chapter 30 npc 41: npc 41 is not a record of the install's NPC table"},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			writeTownMod(t, dir, "m", c.towns)
			f := layerFront(t)
			tavernLeaves(f, c.leaves...)
			if c.npcs != nil {
				f.NPCFaces = map[int32]data.NPCFace{}
				for _, n := range c.npcs {
					f.NPCFaces[n] = data.NPCFace{}
				}
			}
			err := setTownModsErr(f, dir, "m")
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("%v, want %q", err, c.want)
			}
		})
	}
}

// setTownModsErr runs the mods ids of dir and sets their town layers on f,
// answering the first refusal.
func setTownModsErr(f *FrontEnd, dir string, ids ...string) error {
	entries, err := mod.Resolve(dir, ids)
	if err != nil {
		return err
	}
	res, err := modrt.Load(entries, "rom1-en", nil, modrt.Options{})
	if err != nil {
		return err
	}
	_, err = f.SetModTowns(res.Towns)
	return err
}

// Two mods that set one row's mission are refused without an order; with
// load-after the later wins and the override names the row and the field.
func TestTavernLayerTwoModsOnOneRow(t *testing.T) {
	dir := t.TempDir()
	row := "[[tavern]]\ntown = 1\nchapter = 30\nnpc = 22\nmission = %d\n"
	writeTownMod(t, dir, "a", strings.ReplaceAll(row, "%d", "31"))
	writeTownMod(t, dir, "b", strings.ReplaceAll(row, "%d", "0"))
	writeTownMod(t, dir, "c", strings.ReplaceAll(row, "%d", "0"), "a")
	leaves := []string{"main/text/inn/npc/npc22m31.txt", "main/text/inn/npc/npc22m30.txt"}
	f := layerFront(t)
	tavernLeaves(f, leaves...)
	if err := setTownModsErr(f, dir, "a", "b"); err == nil ||
		!strings.Contains(err.Error(), `tavern row town 1 chapter 30 npc 22 mission is set by mod "a" and by mod "b", and neither loads after the other`) {
		t.Fatalf("unordered: %v", err)
	}
	f, overrides, err := tavernLayerFront(t, dir, leaves, "a", "c")
	if err != nil || len(overrides) != 1 ||
		!strings.HasPrefix(overrides[0], `tavern row town 1 chapter 30 npc 22 mission: mod "c" (data/towns.toml:5) overrides mod "a"`) {
		t.Fatalf("ordered: %v, overrides %q", err, overrides)
	}
	if ch := f.Campaign.Value().chapter(30); !slices.Equal(ch.Inn, []int{0, 0}) {
		t.Fatalf("the later mod's mission does not stand: %v", ch.Inn)
	}
}

// A row a mod adds to East shows in East's tavern alone; the rows of layer 0
// keep the town that offers their mission. A quest stands in a town that
// offers its mission, and a chapter names an NPC record once over every
// town.
func TestTavernLayerOfAModTown(t *testing.T) {
	dir := minimalEastMods(t, nil)
	writeTownMod(t, dir, "east-tavern", "[[tavern]]\ntown = 2\nchapter = 30\nnpc = 30\nmission = 0\n"+
		"[[tavern]]\ntown = 2\nchapter = 30\nnpc = 52\nmission = 31\n", "east-town")
	leaves := []string{"main/text/inn/npc/npc30m30.txt", "main/text/inn/npc/npc52m31.txt"}
	f, _, err := tavernLayerFront(t, dir, leaves, "east-town", "east-tavern")
	if err != nil {
		t.Fatal(err)
	}
	tw := f.Town
	tw.Arrive()
	west := []TownOffer{{Index: 0, Mission: 30, NPC: 22}, {Index: 1, NPC: 90}}
	if got := tw.Offers(TownTavern); !reflect.DeepEqual(got, west) {
		t.Fatalf("West's tavern %+v, want %+v", got, west)
	}
	tw.Won(20)
	if !tw.travelTo(2) {
		t.Fatal("the party cannot travel to East")
	}
	east := []TownOffer{{Index: 2, NPC: 30}, {Index: 3, Mission: 31, NPC: 52}}
	if got := tw.Offers(TownTavern); tw.townID() != 2 || !reflect.DeepEqual(got, east) {
		t.Fatalf("East's tavern %+v, want %+v", got, east)
	}
	for _, c := range []struct{ name, towns, want string }{
		{"quest in a town that does not offer it", "[[tavern]]\ntown = 1\nchapter = 30\nnpc = 52\nmission = 31\n",
			"tavern row town 1 chapter 30 npc 52: town 1 does not offer mission 31"},
		{"npc twice", "[[tavern]]\ntown = 2\nchapter = 30\nnpc = 22\nmission = 0\n",
			"tavern row town 2 chapter 30 npc 22: npc 22 already speaks in chapter 30, in the row town 1 chapter 30 npc 22"},
	} {
		t.Run(c.name, func(t *testing.T) {
			writeTownMod(t, dir, "bad", c.towns, "east-town")
			g := layerFront(t)
			tavernLeaves(g, "main/text/inn/npc/npc52m31.txt", "main/text/inn/npc/npc22m30.txt")
			if err := setTownModsErr(g, dir, "east-town", "bad"); err == nil || !strings.Contains(err.Error(), `mod "bad": data/towns.toml:1: `) ||
				!strings.Contains(err.Error(), c.want) {
				t.Fatalf("%v, want %q", err, c.want)
			}
		})
	}
}

// A towns layer that switches off the talk a companion join waits for is
// refused, naming the towns mod, its file and row and the companions mod;
// one that changes the talk's mission keeps the talk and is accepted.
func TestTavernLayerRefusesSwitchingOffAJoinTalk(t *testing.T) {
	for _, c := range []struct{ name, towns, want string }{
		{"switched off", "[[tavern]]\ntown = 1\nchapter = 30\nnpc = 90\noff = true\n",
			`mod "m": data/towns.toml:1: tavern row town 1 chapter 30 npc 90 is switched off, but mod "c" (` + mod.CompanionsFile + ` line 2) makes companion 22 join on this talk in chapter 30`},
		{"changed", "[[tavern]]\ntown = 1\nchapter = 30\nnpc = 90\nmission = 31\n", ""},
		{"another talk switched off", "[[tavern]]\ntown = 1\nchapter = 30\nnpc = 22\noff = true\n", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := layerFront(t)
			camp := f.Campaign.Value()
			ch := camp.Chapters[30]
			ch.AddHero = []int{22}
			camp.Chapters[30] = ch
			f.Campaign = resolved(camp, nil)
			if f.Table == nil {
				f.Table = &mapload.Table{}
			}
			j := joinOf(30, 22, 90)
			j.Mod = "c"
			if err := f.SetModCompanions(mod.CompanionData{Joins: []mod.CompanionJoin{j}}); err != nil {
				t.Fatalf("join: %v", err)
			}
			tavernLeaves(f, "main/text/inn/npc/npc90m31.txt")
			dir := t.TempDir()
			writeTownMod(t, dir, "m", c.towns)
			err := setTownModsErr(f, dir, "m")
			if c.want == "" {
				if err != nil {
					t.Fatalf("%v, want the layer accepted", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("%v, want %q", err, c.want)
			}
		})
	}
}

// A chapter load writes the live arrays from the merged rows, with a side
// record for the mod's quest.
func TestTavernLayerChapterLoadTakesMergedRows(t *testing.T) {
	dir := t.TempDir()
	writeTownMod(t, dir, "tavern", tavernRowsFixture)
	f, _, err := tavernLayerFront(t, dir, tavernRowsLeaves, "tavern")
	if err != nil {
		t.Fatal(err)
	}
	p := &campaignProgress{}
	p.loadMain(f.Campaign.Value(), 30, false)
	if !slices.Equal(p.innNPC, []int{22, 41, 52}) || !slices.Equal(p.innMission, []int{30, 0, 31}) {
		t.Fatalf("live arrays npc %v missions %v, want npc [22 41 52] missions [30 0 31]", p.innNPC, p.innMission)
	}
	if p.record(31) == nil {
		t.Fatal("no side record for the mod's quest 31")
	}
}
