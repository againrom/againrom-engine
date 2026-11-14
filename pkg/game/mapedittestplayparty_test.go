package game

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/reg"
	"againrom/pkg/herokind"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

// partyLines are the fixture's name lines of the four archetypes, in
// herokind's order, and of the two companions who are one row each.
var partyLines = []string{"Danath", "Naira", "Fergard", "Reniesta", "Brian", "Rood Glaen"}

// partyFixture is routeFixture whose install holds the five companions as
// the first game ships them: npc22 to npc24 composed by the hero's class and
// sex over three bands of the four archetype rows, npc25 and npc26 one row
// each, and the name lines both the editor and the game read.
func partyFixture(t *testing.T) (*testPlayFixture, *ui.MapEditor, *mapInspectionEdits) {
	t.Helper()
	f, e, s := routeFixture(t)
	humans := make(dbCollection, 52)
	row := func(at int, name string, mage, female bool) {
		mana, class, sex := int32(0), int32(20), int32(0)
		if mage {
			mana, class = 70, 24
		}
		if female {
			sex = 1
		}
		humans[at] = dbEntry{name: name, params: humansParams(30, mana, 1)}
		humans[at].params[0x18], humans[at].params[0x10], humans[at].params[0x12] = int32(at), class, sex
	}
	for band := range 3 {
		for k, base := range data.ChargenBaseNames() {
			name := base
			if band > 0 {
				name = fmt.Sprintf("%s_%d", base, band+1)
			}
			row(26+4*band+k, name, k >= 2, k%2 == 1)
		}
	}
	row(50, "Man_Brian", false, false)
	row(51, "Man_Rood", true, false)
	record := func(name, flags string, id int32) synth.RegNode {
		return synth.RegNode{Name: name, Kind: 0x01, Children: []synth.RegNode{
			{Name: "Flags", Kind: 0x00, Str: flags}, {Name: "DataBinID", Kind: 0x02, Int: id}}}
	}
	npcReg, err := reg.Parse(synth.Reg(0x11, []synth.RegNode{
		record("npc22", "Hero,Mage,!MySex,Start", 26), record("npc23", "Hero,!Mage,!MySex,Start", 26),
		record("npc24", "Hero,!MyClass,MySex,Start", 26), record("npc25", "Hero", 50), record("npc26", "Hero,Mage", 51)}))
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 26)
	copy(names[20:], partyLines)
	f.x.archives.Containers.Overlay(npcNameLinesSource.at().path, func() ([]byte, error) { return []byte(strings.Join(names, "\n") + "\n"), nil })
	f.x.table = &mapload.Table{Humans: humans, NPC: data.LoadNPCDefs(npcReg), NPCNames: names, Weapons: dbCollection{{}}, Armors: dbCollection{{}},
		Shapes: emptyScale{}, Materials: emptyScale{}, NPCCatalog: &data.NPCCatalog{Entries: []data.NPCEntry{
			{ID: 21, Section: "npc21"}, {ID: 22, Section: "npc22"}, {ID: 23, Section: "npc23"},
			{ID: 24, Section: "npc24"}, {ID: 25, Section: "npc25", Name: "Brian"}, {ID: 26, Section: "npc26"}}}}
	return f, e, s
}

// partyCast are the names and caster flags of the five companions beside a
// hero of each kind, as the first game composes them.
var partyCast = map[herokind.Kind][]string{
	herokind.MaleFighter:   {"Reniesta*", "Naira", "Fergard*", "Brian", "Rood Glaen*"},
	herokind.FemaleFighter: {"Fergard*", "Danath", "Reniesta*", "Brian", "Rood Glaen*"},
	herokind.MaleMage:      {"Reniesta*", "Naira", "Danath", "Brian", "Rood Glaen*"},
	herokind.FemaleMage:    {"Fergard*", "Danath", "Naira", "Brian", "Rood Glaen*"},
}

// A card shows the member the game launches: for every kind of hero the
// window can start, each companion's card carries the name and the caster
// flag of the member the game's guest member builder builds beside that
// hero, and the hero's card casts when he is a mage.
func TestThePartyCardsAreTheMembersTheGameBuilds(t *testing.T) {
	f, _, s := partyFixture(t)
	game := &FrontEnd{InstallResources: InstallResources{Table: f.x.table, Archives: f.x.archives}}
	town := f.x.testPlayNumber(s.workspace)
	for hero, want := range partyCast {
		cards, members := s.testPlayOffer(hero)
		if len(cards) != 6 || len(members) != 6 || !cards[0].Hero || cards[0].Caster != hero.Mage() || !members[0].Hero {
			t.Fatalf("%s: the offer is %+v for %+v", hero, cards, members)
		}
		if !reflect.DeepEqual(cards, s.TestPlayCards(hero.Index())) {
			t.Fatalf("%s: the window's cards differ from the offer", hero)
		}
		ref := []mapload.PartyMember{{StartingHero: true, PlayerCharacter: true, Mage: hero.Mage(),
			FigureDir: string(data.FigureDirFor(hero.Mage(), hero.Female()))}}
		var got []string
		for i, g := range members[1:] {
			p, err := game.guestMember(g, town, ref)
			if err != nil {
				t.Fatalf("%s: %s is not built: %v", hero, g.Character, err)
			}
			card := cards[i+1]
			if card.Name != p.Name || card.Caster != p.Mage || card.Hero {
				t.Errorf("%s: the card of %s is %q, caster %t; the game builds %q, mage %t", hero, g.Character, card.Name, card.Caster, p.Name, p.Mage)
			}
			name := p.Name
			if p.Mage {
				name += "*"
			}
			got = append(got, name)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: the game builds %q, want %q", hero, got, want)
		}
	}
	if s.TestPlayCards(-1) != nil || s.TestPlayCards(ui.TestPlayHeroes) != nil {
		t.Error("a kind no hero is has cards")
	}
}

// The test mission plays for the hero the window showed and for him alone,
// so the game refuses any other hero and the members stay the cards'. A
// package that names its heroes offers the window for those alone.
func TestTestPlayWritesTheHeroTheCardsStandBeside(t *testing.T) {
	f, e, s := partyFixture(t)
	if line, err := s.TestPlayParty(herokind.FemaleMage.Index(), []ui.TestPlayMember{{On: true}, {}, {}, {On: true, Spells: true}, {}, {}}); err != nil {
		t.Fatal(line, err)
	}
	m := loadWorkspaceMod(t, f.root, testModFolder(f))
	if m.Class != herokind.Mage || m.Sex != herokind.Female || !reflect.DeepEqual(m.Party, []mod.GuestMember{{Hero: true}, {Character: "npc24", Level: 3}}) {
		t.Fatalf("the test mod declares class %q sex %q party %+v", m.Class, m.Sex, m.Party)
	}
	if _, err := s.TestPlayParty(ui.TestPlayHeroes, []ui.TestPlayMember{{On: true}, {}, {}, {}, {}, {}}); err == nil || len(f.started) != 1 {
		t.Fatalf("a kind no hero is started: %v, %q", err, f.started)
	}
	setRoute(t, e, map[string]int{"number": 35, "after": 30, "guest": 23, "chapter": 40}, "")
	_ = e.Key("escape")
	s.workspace.Package.Class = herokind.Mage
	for hero := range ui.TestPlayHeroes {
		if got, want := s.TestPlayCards(hero) != nil, herokind.Kind(hero).Mage(); got != want {
			t.Fatalf("a mission for a mage: cards for %s %t", herokind.Kind(hero), got)
		}
	}
	if err := e.Key("testplay"); err != nil || !e.TestPartyOpen() || e.TestPartyHero() != herokind.MaleMage.Index() {
		t.Fatalf("F5 on a mission for a mage: %v, kind %d", err, e.TestPartyHero())
	}
	_ = e.Key("enter")
	m = loadWorkspaceMod(t, f.root, testModFolder(f))
	if m.Class != herokind.Mage || m.Sex != herokind.Male || s.workspace.Package.Sex != "" || s.workspace.Package.Class != herokind.Mage {
		t.Fatalf("the test mod declares class %q sex %q; the package keeps %q %q", m.Class, m.Sex, s.workspace.Package.Class, s.workspace.Package.Sex)
	}
}

// A mission that follows another mission of the open mod starts as a first
// link in the test copy: the hero alone plays it, after the first main
// mission whose town no other mission of the mod takes. The working mod and
// the open package keep the mission's own place.
func TestTestPlayStartsALaterLinkWithTheHeroAlone(t *testing.T) {
	f, e, _ := partyFixture(t)
	table := f.x.table
	dir := writeEditorMod(t, t.TempDir(), "[[mission]]\nkey='first'\nnumber=35\nmap='first.alm'\nafter=30\nparty=[{character='npc22'}]\ngold=500\n"+
		"[[mission]]\nkey='second'\nnumber=36\nmap='second.alm'\nafter=35\nguest=23\nchapter=40\ngold=700\n")
	for _, name := range []string{"first.alm", "second.alm"} {
		writeTestPlayInput(t, filepath.Join(dir, name), e.Document().SourceBytes)
	}
	before := tree(t, dir)
	if err := workspaceField(e, "package", 0, dir); err != nil {
		t.Fatal(err)
	}
	_ = e.Key("escape")
	s := e.Document().Edits.(*mapInspectionEdits)
	f.x.table = table
	s.workspace.Project.Selected = 1
	s.workspace.Package = modPackageSpec(s.workspace.Project.Missions[1], s.workspace.Package)
	own, dirty := s.workspace.Package, s.Dirty()
	if err := e.Key("testplay"); err != nil || !e.TestPartyOpen() {
		t.Fatalf("F5 on the second link did not ask: %v, %q", err, e.Message())
	}
	if err := e.Key("enter"); err != nil || len(f.started) != 1 {
		t.Fatalf("the hero alone did not start: %v, %q, started %q", err, e.Message(), f.started)
	}
	second := loadWorkspaceMod(t, f.root, filepath.Join(f.profile, TestPlayModID, "ambush"), 1)
	if second.Number != 36 || second.After != 20 || second.Gold != 700 || !reflect.DeepEqual(second.Party, []mod.GuestMember{{Hero: true}}) {
		t.Fatalf("the test copy's second link is %+v", second)
	}
	if first := loadWorkspaceMod(t, f.root, filepath.Join(f.profile, TestPlayModID, "ambush"), 0); first.Number != 35 || first.After != 30 {
		t.Fatalf("the test copy's first link is %+v", first)
	}
	if !reflect.DeepEqual(tree(t, dir), before) || !reflect.DeepEqual(s.workspace.Package, own) || s.Dirty() != dirty || own.After != 35 {
		t.Fatal("the working mod, the open package or the dirty mark changed")
	}
}

func partyPress(t *testing.T, e *ui.MapEditor, row, card int) {
	t.Helper()
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPromptGridCell, N: row, K: card})
	if !ok {
		t.Fatalf("row %d of card %d takes no press", row, card)
	}
	p, now := r.Min.Add(r.Size().Div(2)), time.Now()
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, now)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, now)
}

// F5 offers the hero and the campaign's five companions under the game's
// names; Cancel writes nothing, Start writes the test mod whose mission
// declares exactly the chosen party, and the next window starts from it.
func TestTestPlayWritesTheChosenParty(t *testing.T) {
	f, e, s := partyFixture(t)
	cards := s.TestPlayCards(0)
	var names []string
	for _, c := range cards {
		names = append(names, c.Name)
	}
	if want := []string{"", "Reniesta", "Naira", "Fergard", "Brian", "Rood Glaen"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("the cards are %q", names)
	}
	for i, caster := range []bool{false, true, false, true, false, true} {
		if cards[i].Caster != caster || cards[i].Hero != (i == 0) {
			t.Fatalf("card %d: %+v", i, cards[i])
		}
	}
	if err := e.Key("testplay"); err != nil || !e.TestPartyOpen() {
		t.Fatalf("F5 did not ask: %v", err)
	}
	_ = e.Key("escape")
	if e.TestPartyOpen() || len(f.started) != 0 || len(tree(t, f.profile)) != 0 {
		t.Fatalf("Cancel started %q or wrote %v", f.started, tree(t, f.profile))
	}
	_ = e.Key("testplay")
	partyPress(t, e, 0, 1)
	partyPress(t, e, 2, 1)
	partyPress(t, e, 0, 2)
	partyPress(t, e, 1, 0)
	if len(tree(t, f.profile)) != 0 {
		t.Fatal("the open window wrote a file")
	}
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	m := loadWorkspaceMod(t, f.root, testModFolder(f))
	want := []mod.GuestMember{{Hero: true, Skills: mod.GuestSkills}, {Character: "npc22", Spells: mod.GuestSpellsAll}, {Character: "npc23", Level: 2}}
	if m.Kind != mod.MissionGuest || m.Guest != 0 || m.Chapter != 0 || m.After == 0 || !reflect.DeepEqual(m.Party, want) {
		t.Fatalf("the test mod declares %+v", m)
	}
	toml := tree(t, testModFolder(f))[mod.MissionsFile]
	for _, part := range []string{"hero = true", "skills = 100", `character = "npc22"`, `spells = "all"`, `character = "npc23"`, "level = 2"} {
		if !strings.Contains(toml, part) {
			t.Errorf("%s lacks %q:\n%s", mod.MissionsFile, part, toml)
		}
	}
	if len(f.started) != 1 || f.started[0][len(f.started[0])-2] != "-mission" || len(s.workspace.Package.Party) != 0 {
		t.Fatalf("started %q; the open package holds %+v", f.started, s.workspace.Package.Party)
	}
	_ = e.Key("testplay")
	choice := e.TestPartyChoice()
	if !e.TestPartyOpen() || !choice[0].On || !choice[0].Skills || !choice[1].On || !choice[1].Spells || !choice[2].On || choice[3].On {
		t.Fatalf("the second window starts from %+v", choice)
	}
}

// A package's guest route keeps its number, the mission it plays after and
// its gold under the chosen party; a route that replaces a campaign mission
// plays as it says, with no window; one that declares its own party has the switch.
func TestTestPlayPartyAndThePackageRoute(t *testing.T) {
	f, e, _ := partyFixture(t)
	setRoute(t, e, map[string]int{"number": 35, "after": 30, "guest": 23, "chapter": 40, "gold": 500}, "")
	_ = e.Key("escape")
	if err := e.Key("testplay"); err != nil || !e.TestPartyOpen() {
		t.Fatalf("F5 did not ask: %v", err)
	}
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	m := loadWorkspaceMod(t, f.root, testModFolder(f))
	if m.Number != 35 || m.After != 30 || m.Gold != 500 || m.Guest != 0 || !reflect.DeepEqual(m.Party, []mod.GuestMember{{Hero: true}}) {
		t.Fatalf("the test mod declares %+v", m)
	}

	f, e, s := partyFixture(t)
	original := append([]byte(nil), e.Document().SourceBytes...)
	f.x.archives.Containers.Overlay("scenario/20.alm", func() ([]byte, error) { return append([]byte(nil), original...), nil })
	setRoute(t, e, map[string]int{"number": 20}, "replace")
	_ = e.Key("escape")
	if s.TestPlayCards(0) == nil || !s.TestPlayDeclaresParty() {
		t.Fatal("a replacing mission offers the party window with the switch")
	}
	if err := e.Key("testplay"); err != nil || !e.TestPartyOpen() {
		t.Fatalf("F5 on a replacing mission did not ask: %v", err)
	}
	if err := e.Key("enter"); err != nil || len(f.started) != 1 {
		t.Fatalf("the Standard route: %v, started %q", err, f.started)
	}
	if _, ok := startedFlag(f, "hero"); ok {
		t.Fatalf("Standard keeps generation: %q", f.started)
	}
	if m := loadWorkspaceMod(t, f.root, testModFolder(f)); m.Number != 20 || !m.Campaign() {
		t.Fatalf("Standard plays mission %d as declared, campaign %t", m.Number, m.Campaign())
	}
	if _, err := s.TestPlayParty(herokind.FemaleMage.Index(), []ui.TestPlayMember{{On: true, Skills: true, Spells: true}, {On: true}, {}, {}, {}, {}}); err != nil {
		t.Fatal(err)
	}
	if hero, _ := startedFlag(f, "hero"); hero != herokind.FemaleMage.Word() {
		t.Fatalf("Custom skips generation: %q", f.started)
	}
	cm := loadWorkspaceMod(t, f.root, testModFolder(f))
	want := []mod.GuestMember{{Hero: true, Skills: mod.GuestSkills, Spells: mod.GuestSpellsAll}, {Character: "npc22"}}
	if cm.Campaign() || !reflect.DeepEqual(cm.Party, want) || cm.Class != herokind.Mage || cm.Sex != herokind.Female {
		t.Fatalf("Custom declares %+v campaign %t %s %s", cm.Party, cm.Campaign(), cm.Class, cm.Sex)
	}

	_, e, s = partyFixture(t)
	if err := s.prepareWorkspace(); err != nil {
		t.Fatal(err)
	}
	s.workspace.Package.Party = []mod.GuestMember{{Character: "npc24"}}
	if s.TestPlayCards(0) == nil {
		t.Fatal("a mission that declares its party offers the party window")
	}
	if (secondCampaignRules{}).testPlayParty() != nil {
		t.Fatal("the second game offers a party")
	}
}
