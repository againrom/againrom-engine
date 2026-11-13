package game

import (
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"slices"
	"strconv"

	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/herokind"
	"againrom/pkg/launch"
	"againrom/pkg/locale"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

// TestPlayModID is the id of the map editor's test mod. Its mods folder is
// TestPlayModID in the editor's player profile, and the mod is the folder of
// the same name inside it.
const TestPlayModID = "testplay"

// testPlayMap and testPlayTexts are where a package or test mod written
// before the place rule (mod.MissionPlace) held the map and the mission's
// text pages; the ownership record still admits them, and Save moves them.
const (
	testPlayMap         = "mission/map.alm"
	testPlayMapExtended = "mission/map" + extendedMapExt
	testPlayTexts       = "mission/text"
)

// TestPlayHost is what the map editor's test play takes from its program: the
// player profile the test mod is written under, the game program and the
// launch that starts it.
type TestPlayHost struct {
	// Profile resolves the editor's player profile when a test play writes.
	Profile func() (RuntimeProfile, error)
	// Game is the game program a test play starts.
	Game string
	// Start launches program with args and returns without waiting.
	Start func(program string, args []string) error
	// ModsFolder is the mods folder the program resolved, the one the starter
	// and the game read; Publish writes there when no folder is typed.
	ModsFolder string
}

// SetTestPlay names the host of the editor's test play.
func (x *MapInspector) SetTestPlay(h TestPlayHost) { x.testPlay = &h }

// TestPlay writes the map as it now stands into the editor's test mod and
// starts the game in that mission. Nothing is written and nothing starts when
// the report holds an error, the install is not the first game's, or the game
// program is missing. It answers the line the window shows.
func (s *mapInspectionEdits) TestPlay() (string, error) { return s.testPlayWith(nil, 0) }

// testPlayWith is TestPlay with the party the test mission declares: party,
// or the open package's own when party is nil. A chosen party plays for the
// hero of kind hero alone and enters its chain as a first link. All of it is
// written into the test mod alone; the open package keeps its own.
func (s *mapInspectionEdits) testPlayWith(party []mod.GuestMember, hero herokind.Kind) (string, error) {
	x := s.inspector
	h := x.testPlay
	if h == nil {
		return "", errors.New("Test play: this editor was started without the game program")
	}
	if x.archives == nil {
		return "", errors.New("Test play: no install is open")
	}
	if err := s.TestPlayRefusal(); err != nil {
		return "", err
	}
	if err := s.gate(false); err != nil {
		return "", err
	}
	b, err := s.model.SaveBytes()
	if err != nil {
		return "", fmt.Errorf("Test play: %w", err)
	}
	if len(b) > mod.MaxFileBytes {
		return "", fmt.Errorf("Test play: the map is %d bytes; a mod map holds at most %d", len(b), mod.MaxFileBytes)
	}
	var pkg *editorPackageSpec
	if s.workspace != nil {
		pkg = &s.workspace.Package
		if party != nil {
			own := *pkg
			defer func() { *pkg = own }()
			pkg.Party, pkg.After = party, s.testPlayFirstLink(pkg.After)
			pkg.Class, pkg.Sex = heroWords(hero)
		}
	}
	m, err := x.testPlayMission(s.source, b, pkg)
	if err != nil {
		return "", fmt.Errorf("Test play: %w", err)
	}
	if pkg == nil && party != nil {
		m.Party, m.Guest, m.Chapter, m.Carry = party, 0, 0, ""
		m.Class, m.Sex = heroWords(hero)
	}
	var plan *missionPackagePlan
	if s.workspace != nil && s.workspace.Project != nil {
		if err := s.unchangedModSource(); err != nil {
			return "", fmt.Errorf("Test play: %w", err)
		}
		files, missions, current, _, e := s.retainedPackageFiles(b)
		if e != nil {
			return "", fmt.Errorf("Test play: %w", e)
		}
		if e := x.checkMissionSet(missions); e != nil {
			return "", fmt.Errorf("Test play: %w", e)
		}
		m = current
		plan = &missionPackagePlan{ID: s.workspace.Package.ID, Record: editorPackageRecord, Files: files, Check: x.admitMissionPackage, Expected: missions, Word: x.word}
		if x.mod != nil {
			plan.Source = x.mod.dir
		}
	} else if s.workspace != nil {
		m.TextFiles = cloneWorkspace(s.workspace).Pages
	}
	if plan == nil {
		if err := s.carryModSpeakers(&m); err != nil {
			return "", fmt.Errorf("Test play: %w", err)
		}
		if err := x.checkMissionRoute(m); err != nil {
			return "", fmt.Errorf("Test play: %w", err)
		}
		// The game loads the test mod through the same rule table; an error there
		// would refuse the mod before the mission opens.
		address, _ := MissionMap(m.Number)
		if rows := x.CheckModMap(m, address); hasError(rows) {
			return "", s.loaderRefusal(rows)
		}
	}
	if st, err := os.Stat(h.Game); err != nil || !st.Mode().IsRegular() {
		return "", fmt.Errorf("Test play: the game program %s is not beside the map editor", h.Game)
	}
	profile, err := h.Profile()
	if err != nil {
		return "", fmt.Errorf("Test play: no player profile: %w", err)
	}
	modsDir, err := writeTestMod(profile, x.archives.Root, m, plan, s.packageProtected()...)
	if err != nil {
		return "", fmt.Errorf("Test play: %w", err)
	}
	id := TestPlayModID
	if plan != nil {
		id = plan.ID
	}
	args, err := TestPlayArgs(x.archives.Root, modsDir, m.Number, id)
	if err != nil {
		return "", fmt.Errorf("Test play: %w", err)
	}
	if err := h.Start(h.Game, args); err != nil {
		return "", fmt.Errorf("Test play: %s did not start: %v", filepath.Base(h.Game), err)
	}
	return fmt.Sprintf("Test play: %s opens mission %d", filepath.Base(h.Game), m.Number), nil
}

func hasError(rows []ui.InspectionFinding) bool {
	errs, _ := ui.CountFindings(rows)
	return errs > 0
}

// TestPlayArgs is the game's command line of a test play: the install root
// and its base, the test mod from modsDir, and the direct start of mission n.
func TestPlayArgs(root, modsDir string, n int, ids ...string) ([]string, error) {
	id := TestPlayModID
	if len(ids) != 0 {
		id = ids[0]
	}
	return launch.Build(func(st launch.Setting) []string {
		switch st.Key {
		case launch.Assets.Key:
			return []string{root}
		case launch.Base.Key:
			if id := BaseID(InspectInstall(root)); base.Nameable(id) {
				return []string{id}
			}
			return nil
		case launch.Mods.Key:
			return []string{id}
		case launch.ModsDir.Key:
			return []string{modsDir}
		case launch.Mission.Key:
			return []string{strconv.Itoa(n)}
		}
		return []string{st.Load("")}
	})
}

// testPlayMission wraps the map and its shared page source in the mission the
// test mod declares: the route of the open mod package pkg when there is one,
// else a guest mission of testPlayGuest.
func (x *MapInspector) testPlayMission(source string, b []byte, pkg *editorPackageSpec) (mod.Mission, error) {
	var m mod.Mission
	if pkg == nil {
		var err error
		if m, err = x.testPlayGuestMission(); err != nil {
			return mod.Mission{}, err
		}
	}
	pages := map[string][]byte{}
	if p := x.editorPages(source); p != nil {
		if p.Err != nil {
			return mod.Mission{}, p.Err
		}
		for name, t := range p.Files() {
			if missionPageName(name) {
				pages[name] = t
			}
		}
	}
	model, err := mapedit.New(b)
	place := missionPlace(source, err == nil && model.Extended())
	if pkg != nil {
		m = pkg.mission(b, pages, place)
		m.Mod = TestPlayModID
		return m, nil
	}
	m.MapBytes, m.TextFiles = b, pages
	m.Map = place.Map
	return m, nil
}

// testPlayGuestMission is testPlayGuest of the install's campaign.
func (x *MapInspector) testPlayGuestMission() (mod.Mission, error) {
	if x.archives == nil {
		return mod.Mission{}, errors.New("no install is open")
	}
	c, err := LoadCampaign(x.archives.Containers)
	if err != nil {
		return mod.Mission{}, err
	}
	return testPlayGuest(c, x.archives.Containers)
}

// testPlayNumber is the mission number test play starts the map of
// workspace w as: the open package's mission, else the campaign's guest
// mission; 0, the standalone start, when test play has no mission for it.
// The guest number is read once.
func (x *MapInspector) testPlayNumber(w *editorWorkspaceState) int {
	if w != nil {
		return w.Package.Number
	}
	if x.guestNumber == nil {
		n := 0
		if m, err := x.testPlayGuestMission(); err == nil {
			n = m.Number
		}
		x.guestNumber = &n
	}
	return *x.guestNumber
}

// testPlayGuest is the guest declaration the test mod plays its map under:
// the first main mission of campaign c whose win opens a town that grants a
// companion, and the lowest number after it that checkGuest admits.
func testPlayGuest(c Campaign, src *vfs.FS) (mod.Mission, error) {
	for _, after := range c.Main {
		offer, ok := c.NextMission(after)
		if !ok || !offer.Town {
			continue
		}
		for _, guest := range c.Chapters[offer.Mission].AddHero {
			m := mod.Mission{Mod: TestPlayModID, File: mod.MissionsFile, Kind: mod.MissionGuest, Key: "map",
				After: after, Guest: guest, Chapter: offer.Mission, Carry: mod.CarryJoin}
			for n := after + 1; n < after+mainMissionStride; n++ {
				m.Number = n
				if checkGuest(c, src, mod.MissionData{Missions: []mod.Mission{m}}, m) == nil {
					return m, nil
				}
			}
		}
	}
	return mod.Mission{}, errors.New("the campaign has no won mission whose town grants a companion, so no guest mission can be declared")
}

// testPlayMissions is the test mod's data/missions.toml for m.
func testPlayMissions(m mod.Mission) []byte {
	b, _ := mod.WriteMissions(nil, nil, []mod.Mission{m})
	return b
}

// writeTestMod validates the resulting mod before publishing it. Its record
// admits replacement only of its own files and removal only of its old pages.
func writeTestMod(profile RuntimeProfile, root string, m mod.Mission, plan *missionPackagePlan, protected ...packageGuard) (string, error) {
	if profile.Directory == "" {
		return "", errors.New("no player profile directory")
	}
	dir, err := filepath.Abs(profile.Directory)
	if err != nil {
		return "", err
	}
	profile.Directory = dir
	modsDir := filepath.Join(profile.Directory, TestPlayModID)
	id, record := TestPlayModID, testPlayRecord
	if plan != nil {
		id, record = plan.ID, plan.Record
	}
	folder := filepath.Join(modsDir, id)
	old, err := missionPackageOwnership(folder, record, protected)
	if err != nil {
		return "", err
	}
	var grant *runtimeProfileAccess
	if access := profile.Saves.profile; access != nil {
		grant = &runtimeProfileAccess{root: access.root, directory: access.root, descendants: true}
	}
	guard, forget := admittedOnce(func(dir string) error {
		if grant != nil {
			allowed, err := grant.allows(dir)
			if err != nil {
				return err
			}
			if allowed {
				return nil
			}
		}
		return fenceEditorWrite(root, dir, "Test play")
	})
	lang := locale.Fallback
	if l, ok := locale.ByBaseSuffix(BaseID(InspectInstall(root))); ok {
		lang = l.Code
	}
	files := missionPackageFiles("Map editor test play", "The map the map editor plays: written by its test play key, replaced by the next one.", m, lang, nil)
	options := missionPackagePlan{}
	if plan != nil {
		files, options = plan.Files, *plan
	}
	if err := testPlayTargets(folder, old, files); err != nil {
		return "", err
	}
	if err := guard(profile.Directory); err != nil {
		return "", err
	}
	if plan != nil && len(old) != 0 {
		if err := filepath.WalkDir(folder, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel, err := filepath.Rel(folder, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if rel != record && !slices.Contains(old, rel) {
				return fmt.Errorf("%s is not a recorded test play file", path)
			}
			return nil
		}); err != nil {
			return "", err
		}
	}
	stage, err := stageMissionPackage(profile.Directory, folder, root, id, record, old, files, guard, options)
	if err != nil {
		return "", err
	}
	modsDir, grant, err = profile.folder(TestPlayModID, root)
	if err != nil {
		_ = os.RemoveAll(stage)
		return "", err
	}
	forget()
	folder = filepath.Join(modsDir, id)
	if err := guard(modsDir); err != nil {
		_ = os.RemoveAll(stage)
		return "", err
	}
	if err := installMissionPackage(stage, folder, id, len(old) != 0); err != nil {
		return "", err
	}
	return modsDir, nil
}

// admittedOnce runs check once per folder spelling until forget: a folder it
// admitted is admitted again without a new install census, and a refusal is
// never remembered. writeTestModFile still checks each folder on a file's path
// for a link or a redirect before it asks the guard.
func admittedOnce(check func(string) error) (guard func(string) error, forget func()) {
	passed := map[string]bool{}
	guard = func(dir string) error {
		if passed[dir] {
			return nil
		}
		if err := check(dir); err != nil {
			return err
		}
		passed[dir] = true
		return nil
	}
	return guard, func() { clear(passed) }
}

// testPlayCompanion is a companion the party window before a test play
// offers: his npc.reg record and the level of the composed row he takes, 0
// for the first.
type testPlayCompanion struct {
	npc   int32
	level int
}

// heroWords are the class and sex words a mission names hero by.
func heroWords(hero herokind.Kind) (class, sex string) {
	class, sex = herokind.Fighter, herokind.Male
	if hero.Mage() {
		class = herokind.Mage
	}
	if hero.Female() {
		sex = herokind.Female
	}
	return class, sex
}

// testPlayHeroes are the heroes the open map's test mission plays for: the
// open package's class and sex, every hero when it names none.
func (s *mapInspectionEdits) testPlayHeroes() herokind.Set {
	if s.workspace == nil {
		return herokind.Every
	}
	return herokind.Named(s.workspace.Package.Class, s.workspace.Package.Sex)
}

// testPlayFirstLink is the mission a chosen party's test mission plays
// after. A mission that follows another mission of the open project starts
// as a first link instead: after the first main mission whose win opens a
// town and that no other mission of the project follows. Any other after
// stands.
func (s *mapInspectionEdits) testPlayFirstLink(after int) int {
	x, w := s.inspector, s.workspace
	if w == nil || w.Project == nil {
		return after
	}
	follows := func(n int) bool {
		for i, m := range w.Project.Missions {
			if i != w.Project.Selected && !m.Off && !m.Campaign() && m.After == n {
				return true
			}
		}
		return false
	}
	if !slices.ContainsFunc(w.Project.Missions, func(m mod.Mission) bool { return !m.Off && m.Number == after }) {
		return after
	}
	c, err := LoadCampaign(x.archives.Containers)
	if err != nil {
		return after
	}
	for _, main := range c.Main {
		_, auto := c.AutoAdvance(main)
		if offer, ok := c.NextMission(main); ok && offer.Town && !auto && !follows(main) {
			return main
		}
	}
	return after
}

// testPlayTakesParty reports that the open map's test mission takes a party
// the window chooses: a guest mission that declares none of its own. A
// mission of the open package that replaces a campaign mission, declares its
// party or is declared by its map file plays as the package says.
func (s *mapInspectionEdits) testPlayTakesParty() bool {
	w := s.workspace
	if w == nil {
		return true
	}
	if (mod.Mission{Kind: w.Package.Kind}).Campaign() || len(w.Package.Party) != 0 {
		return false
	}
	if d := w.Project; d != nil {
		if d.Selected < 0 || d.Selected >= len(d.Missions) {
			return false
		}
		return !d.Missions[d.Selected].Off && !d.Missions[d.Selected].ByFileName()
	}
	return true
}

// townInstall is the install's definitions as the game's member builder
// reads them: the editor's table, body list and name lines.
func (x *MapInspector) townInstall() townInstall {
	in := townInstall{table: x.table, npcName: func(index int, fallback string) string {
		if index < 0 || index >= len(x.table.NPCNames) || x.table.NPCNames[index] == "" {
			return fallback
		}
		return x.table.NPCNames[index]
	}}
	if x.archives != nil {
		in.bodies, _ = ReadBodyList(x.archives.artSource())
	}
	return in
}

// testPlayOffer are the party window's cards for a hero of kind hero and the
// party member each one stands for: the hero, then the campaign's companions
// the install builds. A companion's card is the member the game's builder
// builds beside that hero: his name, whether he casts, and his face as a
// dialogue cuts it. None when test play is closed to the map, its mission
// takes no chosen party or does not play for that hero.
func (s *mapInspectionEdits) testPlayOffer(hero herokind.Kind) ([]ui.TestPlayCard, []mod.GuestMember) {
	x := s.inspector
	if x.testPlay == nil || x.archives == nil || x.table == nil || s.TestPlayRefusal() != nil || !s.testPlayTakesParty() || !s.testPlayHeroes().Holds(hero) {
		return nil, nil
	}
	rows := campaignOf(x.archives.Game()).testPlayParty()
	if len(rows) == 0 {
		return nil, nil
	}
	in, catalog, town, selector := x.townInstall(), installCharacters(x.table), x.testPlayNumber(s.workspace), 0
	if x.font != nil {
		selector = x.font.Selector
	}
	ref := []mapload.PartyMember{{StartingHero: true, PlayerCharacter: true, Mage: hero.Mage(),
		FigureDir: string(data.FigureDirFor(hero.Mage(), hero.Female()))}}
	cards, members := []ui.TestPlayCard{{Hero: true, Caster: hero.Mage()}}, []mod.GuestMember{{Hero: true}}
	for _, r := range rows {
		at := slices.IndexFunc(catalog, func(c mod.Character) bool { return c.NPC == r.npc })
		if at < 0 {
			continue
		}
		g := mod.GuestMember{Character: catalog[at].Key, Level: r.level}
		p, err := in.guestMember(g, town, ref, selector)
		if err != nil {
			continue
		}
		face := composeMemberPortrait(x.archives.artSource(), x.units, 0, mapload.EquipmentFromParty(p), p).Figure
		cards = append(cards, ui.TestPlayCard{Name: x.inspectionUTF8(x.messageUTF8([]byte(p.Name))),
			Portrait: ui.SpeakerPortrait(face, image.Rectangle{}), Caster: p.Mage})
		members = append(members, g)
	}
	return cards, members
}

// TestPlayCards are the characters the party window offers for the hero of
// kind hero, in herokind's order.
func (s *mapInspectionEdits) TestPlayCards(hero int) []ui.TestPlayCard {
	if hero < 0 || hero >= ui.TestPlayHeroes {
		return nil
	}
	cards, _ := s.testPlayOffer(herokind.Kind(hero))
	return cards
}

// TestPlayParty is TestPlay for the hero of kind hero with the cards
// chosen: each card that is on is a member of the test mission's party, with
// the skills and spells keys its boxes set. A card whose character does not
// cast takes no spells key.
func (s *mapInspectionEdits) TestPlayParty(hero int, choice []ui.TestPlayMember) (string, error) {
	if hero < 0 || hero >= ui.TestPlayHeroes {
		return "", errors.New("Test play: the party window names no hero; press F5 again")
	}
	cards, members := s.testPlayOffer(herokind.Kind(hero))
	if len(cards) == 0 || len(choice) != len(cards) {
		return "", errors.New("Test play: the party window does not match this map; press F5 again")
	}
	var party []mod.GuestMember
	for i, c := range choice {
		if !c.On {
			continue
		}
		g := members[i]
		if c.Skills {
			g.Skills = mod.GuestSkills
		}
		if c.Spells && cards[i].Caster {
			g.Spells = mod.GuestSpellsAll
		}
		party = append(party, g)
	}
	if len(party) == 0 {
		return "", errors.New("Test play: nobody is chosen to play the map")
	}
	return s.testPlayWith(party, herokind.Kind(hero))
}

// TestPlayRefusal is the campaign's refusal of every test play, nil when the
// install's campaign allows it or no install is open.
func (s *mapInspectionEdits) TestPlayRefusal() error {
	if s.inspector.archives == nil {
		return nil
	}
	return campaignOf(s.inspector.archives.Game()).testPlayRefusal()
}
