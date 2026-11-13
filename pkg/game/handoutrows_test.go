package game

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// handoutFixture switches off the tavern talk that gives main mission 30
// and gives the mission to NPC record 41 once side mission 31 is won.
const handoutFixture = `[[tavern]]
town    = 1
chapter = 30
npc     = 22
off     = true

[[tavern]]
town    = 1
chapter = 30
npc     = 41
mission = 30
won     = [31]
`

var handoutLeaves = []string{"main/text/inn/npc/npc41m30.txt"}

// handoutFront is the layer fixture's front for base with the mod's rows
// set, standing in the town of chapter 30.
func handoutFront(t *testing.T, dir, base string) *FrontEnd {
	t.Helper()
	f := layerFront(t)
	rows := make(dbCollection, 30)
	rows[28].name, rows[29].name = "PC_Leader", "PC_Companion"
	f.Table = &mapload.Table{Humans: rows}
	f.Carried = mapload.CloneParty(currentTrainingCity(t).Carried)
	tavernLeaves(f, handoutLeaves...)
	if _, err := f.SetModTowns(runTownModsFor(t, base, dir, "handout").Towns); err != nil {
		t.Fatal(err)
	}
	f.Town = NewTown(f.Campaign.Value())
	f.Town.Arrive()
	return f
}

// handoutReload saves f's town as a SAV and loads it into a fresh front of
// the same mod.
func handoutReload(t *testing.T, f *FrontEnd, dir, base string) *FrontEnd {
	t.Helper()
	s, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	g := handoutFront(t, dir, base)
	if _, inTown, err := g.RestoreOriginal(raw); err != nil || !inTown {
		t.Fatalf("LOAD of the town SAV: town %v, %v", inTown, err)
	}
	return g
}

// A mod's tavern talk waits for a named mission: it is absent before the
// win, also after a SAVE and LOAD in the town, shows after the win, also
// after a SAVE and LOAD, and hearing it by the tavern's own route opens its
// mission. The mod loads the same for the English and the Russian base.
func TestAConditionedTavernTalkWaitsForAWinThroughSaveAndLoad(t *testing.T) {
	for _, base := range []string{"rom1-en", "rom1-ru"} {
		t.Run(base, func(t *testing.T) {
			dir := t.TempDir()
			writeTownMod(t, dir, "handout", handoutFixture)
			f := handoutFront(t, dir, base)
			waiting := []TownOffer{{Index: 0, NPC: 90}}
			if got := f.Town.Offers(TownTavern); !reflect.DeepEqual(got, waiting) {
				t.Fatalf("before the win the tavern offers %+v, want %+v", got, waiting)
			}
			f = handoutReload(t, f, dir, base)
			if got := f.Town.Offers(TownTavern); !reflect.DeepEqual(got, waiting) {
				t.Fatalf("after SAVE and LOAD before the win the tavern offers %+v, want %+v", got, waiting)
			}
			if got := f.Town.TavernRoster(); slices.ContainsFunc(got, func(o TownOffer) bool { return o.NPC == 41 }) {
				t.Fatalf("the waiting talker sits in the tavern: %+v", got)
			}
			shop := f.Town.Offers(TownShop)
			if len(shop) != 1 || shop[0].Mission != 31 {
				t.Fatalf("the shop offers %+v, want mission 31", shop)
			}
			if m, ok := f.Town.Take(TownShop, shop[0].Index); !ok || m != 31 {
				t.Fatalf("the shop gave %d %v", m, ok)
			}
			if side, ok := f.Town.Won(31); !ok || !side {
				t.Fatalf("the win of 31: side %v accepted %v", side, ok)
			}
			talk := func(f *FrontEnd) TownOffer {
				t.Helper()
				for _, o := range f.Town.Offers(TownTavern) {
					if o.NPC == 41 && o.Mission == 30 {
						return o
					}
				}
				t.Fatalf("after the win the tavern offers %+v, want NPC 41 with mission 30", f.Town.Offers(TownTavern))
				return TownOffer{}
			}
			talk(f)
			f = handoutReload(t, f, dir, base)
			talk(f)
			if slices.Contains(f.Town.Available(), 30) {
				t.Fatal("mission 30 is open before its talk is heard")
			}
			s := f.TownScreen().(*townScreen)
			app := f.App("handout")
			app.Layout(640, 480)
			app.SetSaveSeams(nil, func() []ui.SaveEntry { return []ui.SaveEntry{{Name: "town.ags", Label: "Town"}} },
				func(string) (ui.MapOpener, bool, error) { s.resetForNewGame(); return nil, true, nil })
			for _, step := range []string{"load", "@first"} {
				var err error
				if step == "load" {
					err = app.HeadlessKey(step)
				} else {
					err = app.HeadlessActivate(step)
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			s.CloseTip()
			if err := app.HeadlessActivate("TAVERN"); err != nil {
				t.Fatal(err)
			}
			if err := app.HeadlessActivate("NPC 41"); err != nil {
				t.Fatal(err)
			}
			for n := 0; s.room == roomTalk && n < 8; n++ {
				if err := app.HeadlessKey("enter"); err != nil {
					t.Fatal(err)
				}
			}
			if err := app.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if !slices.Contains(f.Town.Available(), 30) {
				t.Fatalf("hearing the talk opened no mission: available %v, room %d", f.Town.Available(), s.room)
			}
		})
	}
}

// A shop's and a school's hand-out are rows of the same catalog: a mod
// switches the shop's mission off, gives one to the school, and a row
// stands from a chapter on until its mission is won.
func TestShopAndSchoolRowsAndTheChapterFloor(t *testing.T) {
	dir := t.TempDir()
	writeTownMod(t, dir, "m", "[[shop]]\ntown = 1\nchapter = 30\nmission = 31\noff = true\n"+
		"[[school]]\ntown = 1\nfrom = 30\nmission = 31\n"+
		"[[school]]\ntown = 1\nchapter = 40\nmission = 41\nwon = [31]\n")
	f, _, err := tavernLayerFront(t, dir, nil, "m")
	if err != nil {
		t.Fatal(err)
	}
	c := f.Campaign.Value()
	for _, want := range []struct {
		chapter      int
		shop, school []int
	}{{30, nil, []int{31}}, {40, []int{40}, []int{41, 31}}} {
		ch := c.chapter(want.chapter)
		if !slices.Equal(ch.Shop, want.shop) || !slices.Equal(ch.School, want.school) {
			t.Fatalf("chapter %d: shop %v school %v, want %v %v", want.chapter, ch.Shop, ch.School, want.shop, want.school)
		}
		if raw := c.Chapters[want.chapter]; !slices.Equal(ch.Inn, raw.Inn) || !slices.Equal(ch.InnNPC, raw.InnNPC) {
			t.Fatalf("chapter %d's tavern changed: %+v", want.chapter, ch)
		}
	}
	tw := NewTown(c)
	tw.Arrive()
	if got := offerMissions(tw.Offers(TownSchool)); !slices.Equal(got, []int{31}) || len(tw.Offers(TownShop)) != 0 {
		t.Fatalf("chapter 30: school %v shop %+v", got, tw.Offers(TownShop))
	}
	tw.Won(30)
	if got := offerMissions(tw.Offers(TownSchool)); tw.Chapter() != 40 || !slices.Equal(got, []int{31}) {
		t.Fatalf("chapter %d before the win of 31: school %v, want the floor row's 31 alone", tw.Chapter(), got)
	}
	school := tw.Offers(TownSchool)
	if m, ok := tw.Take(TownSchool, school[0].Index); !ok || m != 31 {
		t.Fatalf("the school gave %d %v", m, ok)
	}
	tw.Won(31)
	if got := offerMissions(tw.Offers(TownSchool)); !slices.Equal(got, []int{41}) {
		t.Fatalf("after the win of 31 the school offers %v, want 41", got)
	}
}

// The check refuses a condition or a building row it cannot hold, naming
// the mod, the file, the line and the row.
func TestHandoutRowRefusals(t *testing.T) {
	for _, c := range []struct{ name, towns, want string }{
		{"won names no mission", "[[tavern]]\ntown = 1\nchapter = 30\nnpc = 22\nwon = [77]\n",
			`mod "m": data/towns.toml:1: tavern row town 1 chapter 30 npc 22: won names mission 77, which is not a mission section of the campaign`},
		{"won names its own mission", "[[tavern]]\ntown = 1\nchapter = 30\nnpc = 22\nwon = [30]\n",
			`mod "m": data/towns.toml:1: town catalog rom1: tavern row town 1 chapter 30 npc 22: won names mission 30, the mission the row itself gives`},
		{"won twice", "[[shop]]\ntown = 1\nchapter = 30\nmission = 31\nwon = [41, 41]\n",
			`mod "m": data/towns.toml:1: town catalog rom1: shop row town 1 chapter 30 mission 31: won names mission 41 twice`},
		{"shop mission absent", "[[shop]]\ntown = 1\nchapter = 30\nmission = 77\n",
			`mod "m": data/towns.toml:1: shop row town 1 chapter 30 mission 77: mission 77 is not a mission section of the campaign`},
		{"floor not a chapter", "[[school]]\ntown = 1\nfrom = 31\nmission = 31\n",
			`mod "m": data/towns.toml:1: school row town 1 chapter 31 mission 31: chapter 31 is not a main mission of the campaign`},
		{"floor meets a chapter's talk", "[[tavern]]\ntown = 1\nfrom = 30\nnpc = 23\nmission = 31\n",
			"npc 23 already speaks in chapter 40, in the row town 1 chapter 40 npc 23"},
		{"floor shop meets the install's shop", "[[shop]]\ntown = 1\nfrom = 30\nmission = 40\n",
			`mod "m": data/towns.toml:1: town catalog rom1: shop row town 1 chapter 30 mission 40: the shop of town 1 already hands out mission 40 in chapter 40, in the row town 1 chapter 40 mission 40`},
		{"floor school meets a school row", "[[school]]\ntown = 1\nchapter = 40\nmission = 41\n\n[[school]]\ntown = 1\nfrom = 30\nmission = 41\n",
			`data/towns.toml:6: town catalog rom1: school row town 1 chapter 30 mission 41: the school of town 1 already hands out mission 41 in chapter 40`},
		{"floor talk without text", "[[tavern]]\ntown = 1\nfrom = 30\nnpc = 41\n",
			"tavern row town 1 chapter 30 npc 41: its talk text main/text/inn/npc/npc41m40.txt is neither"},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			writeTownMod(t, dir, "m", c.towns)
			f := layerFront(t)
			tavernLeaves(f, "main/text/inn/npc/npc41m30.txt", "main/text/inn/npc/npc23m31.txt")
			if err := setTownModsErr(f, dir, "m"); err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("%v, want %q", err, c.want)
			}
		})
	}
}

// Layer 0 holds the scenario's shop and school lists as rows after each
// chapter's talks, and a layer that names none leaves every list as the
// scenario states it.
func TestScenarioShopAndSchoolAreLayerZeroRows(t *testing.T) {
	c := townCampaign(t)
	rows := scenarioTavern(c, townBooks["rom1"].chapterTowns())
	var got []string
	for _, r := range rows {
		got = append(got, town.TavernTable.Key(r).String())
	}
	want := []string{"town 1 chapter 30 npc 22", "town 1 chapter 30 npc 90", "town 1 chapter 30 mission 31",
		"town 1 chapter 40 npc 23", "town 1 chapter 40 mission 40", "town 1 chapter 40 mission 41"}
	if !slices.Equal(got, want) {
		t.Fatalf("layer 0 rows %q, want %q", got, want)
	}
	dir := t.TempDir()
	writeTownMod(t, dir, "forty", "[[tavern]]\ntown = 1\nchapter = 40\nnpc = 23\nmission = 41\n")
	f, _, err := tavernLayerFront(t, dir, []string{"main/text/inn/npc/npc23m41.txt"}, "forty")
	if err != nil {
		t.Fatal(err)
	}
	c = f.Campaign.Value()
	for n, ch := range c.Chapters {
		if got := c.chapter(n); !slices.Equal(got.Shop, ch.Shop) || !slices.Equal(got.School, ch.School) {
			t.Fatalf("chapter %d: shop %v school %v, want the scenario's %v %v", n, got.Shop, got.School, ch.Shop, ch.School)
		}
	}
}
