package game

import (
	"image"
	"math/bits"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	"againrom/pkg/data"
	"againrom/pkg/herokind"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

// startedFlag is the value the last start passed for flag, and whether it
// passed it.
func startedFlag(f *testPlayFixture, flag string) (string, bool) {
	args := f.started[len(f.started)-1]
	at := slices.Index(args, "-"+flag)
	if at < 0 || at+1 >= len(args) {
		return "", false
	}
	return args[at+1], true
}

// partyPressAt presses the middle of r.
func partyPressAt(e *ui.MapEditor, r image.Rectangle) {
	p, now := r.Min.Add(r.Size().Div(2)), time.Now()
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, now)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, now)
}

// F5 through the party window starts the game with the window's hero, so
// the game opens no generation screen: the default choice passes a man
// fighter, a mage woman with two companions passes hers, and the mission
// declares the party, the skills and the spells the boxes set. A project
// mission that declares its own party offers the window's switch: Standard
// plays that party with generation, Custom the window's choice without it.
func TestTestPlayPassesTheWindowsHero(t *testing.T) {
	f, e, s := partyFixture(t)
	if err := e.Key("testplay"); err != nil || !e.TestPartyOpen() {
		t.Fatalf("F5 did not ask: %v", err)
	}
	_ = e.Key("enter")
	if hero, ok := startedFlag(f, "hero"); !ok || hero != herokind.MaleFighter.Word() {
		t.Fatalf("the default choice starts %q", f.started)
	}
	if _, ok := startedFlag(f, "mission"); !ok {
		t.Fatalf("no mission in %q", f.started)
	}
	if line, err := s.TestPlayParty(herokind.FemaleMage.Index(), []ui.TestPlayMember{{On: true, Skills: true, Spells: true}, {On: true, Spells: true}, {}, {}, {On: true, Skills: true}, {}}); err != nil {
		t.Fatal(line, err)
	}
	if hero, _ := startedFlag(f, "hero"); hero != herokind.FemaleMage.Word() {
		t.Fatalf("a mage woman starts %q", f.started)
	}
	m := loadWorkspaceMod(t, f.root, testModFolder(f))
	want := []mod.GuestMember{{Hero: true, Skills: mod.GuestSkills, Spells: mod.GuestSpellsAll}, {Character: "npc22", Spells: mod.GuestSpellsAll}, {Character: "npc25", Skills: mod.GuestSkills}}
	if m.Class != herokind.Mage || m.Sex != herokind.Female || !reflect.DeepEqual(m.Party, want) {
		t.Fatalf("the test mod declares %s %s %+v", m.Class, m.Sex, m.Party)
	}

	f, e, s = partyFixture(t)
	if err := s.prepareWorkspace(); err != nil {
		t.Fatal(err)
	}
	s.workspace.Package.Party = []mod.GuestMember{{Hero: true}, {Character: "npc24"}}
	declared := []mod.GuestMember{{Hero: true}, {Character: "npc24"}}
	if !s.TestPlayDeclaresParty() || len(s.TestPlayCards(0)) == 0 {
		t.Fatal("a mission with its own party offers cards and says so")
	}
	if _, err := s.TestPlay(); err != nil || len(f.started) != 1 {
		t.Fatalf("Standard: %v, started %q", err, f.started)
	}
	if _, ok := startedFlag(f, "hero"); ok {
		t.Fatalf("Standard keeps generation: %q", f.started)
	}
	if m := loadWorkspaceMod(t, f.root, testModFolder(f)); !reflect.DeepEqual(m.Party, declared) {
		t.Fatalf("Standard declares %+v", m.Party)
	}
	if _, err := s.TestPlayParty(herokind.MaleMage.Index(), []ui.TestPlayMember{{On: true, Skills: true}, {}, {}, {}, {On: true}, {}}); err != nil {
		t.Fatal(err)
	}
	if hero, _ := startedFlag(f, "hero"); hero != herokind.MaleMage.Word() {
		t.Fatalf("Custom skips generation: %q", f.started)
	}
	if m := loadWorkspaceMod(t, f.root, testModFolder(f)); !reflect.DeepEqual(m.Party, []mod.GuestMember{{Hero: true, Skills: mod.GuestSkills}, {Character: "npc25"}}) || !reflect.DeepEqual(s.workspace.Package.Party, declared) {
		t.Fatalf("Custom declares %+v, the package keeps %+v", m.Party, s.workspace.Package.Party)
	}
}

// The hero the skipped screen makes for a mage woman plays the mission the
// party window wrote: the hero and the two companions, each with the skills
// and the spell book the boxes set, and SAVE and LOAD keep them.
func TestHeroWithoutGenerationPlaysTheWindowsParty(t *testing.T) {
	const party = `[{ hero = true, skills = 100, spells = "all" }, { character = "npc22", skills = 100, spells = "all" }, { character = "sword" }]` +
		"\nclass = \"mage\"\nsex = \"female\""
	start := func() *FrontEnd {
		f := grantedPartyFront(t, party)
		c := ui.NewChargen(f.ChargenSetup())
		if !c.ChooseHero(true, true) {
			t.Fatal("the screen did not take a mage woman")
		}
		res, ok := c.Result()
		if !ok {
			t.Fatal("the unchanged screen is not legal")
		}
		open, err := f.NewGameBegin(grantedPartyMission)(res)
		if err != nil {
			t.Fatal(err)
		}
		if _, _, _, _, _, _, _, _, _, _, err := open(); err != nil {
			t.Fatal(err)
		}
		return f
	}
	f := start()
	limit := mapload.TableRules(f.Table).SkillCap()
	check := func(f *FrontEnd, stage string) {
		t.Helper()
		if f.liveMission != grantedPartyMission {
			t.Fatalf("%s: plays %d", stage, f.liveMission)
		}
		got := f.LiveParty()
		if ids := grantedPartyIDs(got); len(ids) != 3 || ids[1] != "npc:22" || ids[2] != "char:sword" {
			t.Fatalf("%s: the party is %v", stage, ids)
		}
		hero := got[0]
		if !hero.StartingHero || !hero.Mage || !data.FigureDir(hero.FigureDir).Female() {
			t.Fatalf("%s: the hero is %+v", stage, hero)
		}
		for i, p := range got[:2] {
			if bits.OnesCount32(p.KnownSpells) != 3 {
				t.Fatalf("%s: member %d knows %#x, want the table's three spells", stage, i, p.KnownSpells)
			}
			for slot := 1; slot < data.SkillSlots; slot++ {
				if p.Hero.Skill[slot] != limit {
					t.Fatalf("%s: member %d skill %d is %d, want %d", stage, i, slot, p.Hero.Skill[slot], limit)
				}
			}
		}
		if got[2].Hero.Skill[1] >= limit {
			t.Fatalf("%s: the member without the box took the skills", stage)
		}
	}
	check(f, "start")
	saved := guestMissionSave(t, f)
	g := grantedPartyFront(t, party)
	open, town, err := g.RestoreOriginal(saved)
	if err != nil || town {
		t.Fatalf("LOAD: %t %v", town, err)
	}
	if _, _, _, _, _, _, _, _, _, _, err = open(); err != nil {
		t.Fatal(err)
	}
	check(g, "after LOAD")
	for i, p := range g.LiveParty() {
		q := f.LiveParty()[i]
		if p.ID != q.ID || p.Name != q.Name || p.Mage != q.Mage || p.FigureDir != q.FigureDir || p.KnownSpells != q.KnownSpells || p.Hero.Skill != q.Hero.Skill {
			t.Fatalf("LOAD changed member %d:\n%+v\n%+v", i, p, q)
		}
	}
}

// The party window's choice lives in the editor's profile file: a new
// editor over the same profile folder opens the window with the kind and
// the boxes the last one chose, and F5 there starts that hero.
func TestThePartyChoiceOutlivesTheEditor(t *testing.T) {
	f, e, _ := partyFixture(t)
	dir := t.TempDir()
	file := filepath.Join(dir, ui.PanelStateFile)
	profile := func(e *ui.MapEditor) {
		data, err := os.ReadFile(file)
		e.SetPanelProfile(data, err, func(b []byte) error { return ui.WritePanelState(file, b) })
	}
	profile(e)
	_ = e.Key("testplay")
	for range 3 {
		r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPromptButton, N: 1})
		if !ok {
			t.Fatal("the kind button takes no press")
		}
		partyPressAt(e, r)
	}
	partyPress(t, e, 0, 1)
	partyPress(t, e, 2, 1)
	partyPress(t, e, 1, 0)
	want := e.TestPartyChoice()
	if e.TestPartyHero() != herokind.FemaleMage.Index() {
		t.Fatalf("three presses show kind %d", e.TestPartyHero())
	}
	_ = e.Key("enter")
	if hero, _ := startedFlag(f, "hero"); hero != herokind.FemaleMage.Word() {
		t.Fatalf("started %q", f.started)
	}
	next := f.x.Editor()
	if err := next.Key("new"); err != nil {
		t.Fatal(err)
	}
	_ = next.Key("clear")
	next.TextInput("40x56")
	_ = next.Key("enter")
	_ = next.Key("enter")
	profile(next)
	if err := next.Key("testplay"); err != nil || !next.TestPartyOpen() {
		t.Fatalf("F5 in the new editor: %v", err)
	}
	if next.TestPartyHero() != herokind.FemaleMage.Index() || !slices.Equal(next.TestPartyChoice(), want) {
		t.Fatalf("the new editor opens kind %d with %+v, want %+v", next.TestPartyHero(), next.TestPartyChoice(), want)
	}
	_ = next.Key("enter")
	if hero, _ := startedFlag(f, "hero"); hero != herokind.FemaleMage.Word() || len(f.started) != 2 {
		t.Fatalf("started %q", f.started)
	}
}
