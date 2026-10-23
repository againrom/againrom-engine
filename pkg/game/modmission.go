package game

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/sim"
	"againrom/pkg/vfs"
)

// A mod mission is played by a guest party while the party waits. When the
// declared win is recorded and the town would open, the campaign is written
// as the town SAV the one producer writes at that point (the stash) and the
// guest party opens the mod's mission alone. A guest mission may follow a
// guest mission's win with no town between, as the original's AutoGetMission
// opens the next mission (REG-SCN-063); the links make one chain. When the
// last link is won, the stash comes back through the one town LOAD, with the
// guest who joins recorded beside it as she ended her mission, and the town's
// own grant of that companion takes her in that state (DIV-3023, DIV-4001).

// guestMission returns the town's guest; nil for a nil town.
func (t *Town) guestMission() *guestMission {
	if t == nil {
		return nil
	}
	return t.guest
}

// guestMission is the session's record of a chain of mod missions and its
// guest party.
type guestMission struct {
	// mission is the link under way, or the link of the guest who joins once
	// the chain is won.
	mission mod.Mission
	// stash is the town SAV of the waiting party; nil once the chain is won.
	stash []byte
	// start is the party's first member as the link began; rest the others.
	start mapload.PartyMember
	rest  []mapload.PartyMember
	// member is the guest who joins, as she ended her link, waiting for the
	// town's grant; nil while her link runs.
	member *mapload.PartyMember
	// chain is the links from the first to mission; empty is mission alone.
	chain []mod.Mission
	// hero is the waiting party's starting hero, which a later link's npc
	// records resolve against; nil for a chain of one link.
	hero *guestHero
}

// guestHero is the class and figure of the waiting party's starting hero.
type guestHero struct {
	Mage      bool
	FigureDir string
}

// playing reports that mission n is the guest's mission, still under way.
func (g *guestMission) playing(n int) bool {
	return g != nil && g.stash != nil && g.mission.Number == n
}

// links are the chain's links from the first through the current one.
func (g *guestMission) links() []mod.Mission {
	if len(g.chain) == 0 {
		return []mod.Mission{g.mission}
	}
	return g.chain
}

// joiner is the link of the chain whose guest joins the town: the one link
// in the guest form.
func (g *guestMission) joiner() (mod.Mission, bool) {
	for _, m := range g.links() {
		if m.Guest != 0 {
			return m, true
		}
	}
	return mod.Mission{}, false
}

// party is the guest party the current link began with.
func (g *guestMission) party() []mapload.PartyMember {
	return mapload.CloneParty(append([]mapload.PartyMember{g.start}, g.rest...))
}

// joining is the guest the town of chapter grants as companion npc, as she
// ended her mission, when one is waiting.
func (g *guestMission) joining(chapter, npc int) (mapload.PartyMember, bool) {
	if g == nil || g.member == nil {
		return mapload.PartyMember{}, false
	}
	if j, ok := g.joiner(); !ok || j.Chapter != chapter || j.Guest != npc {
		return mapload.PartyMember{}, false
	}
	m := mapload.CloneParty([]mapload.PartyMember{*g.member})[0]
	m.StartingHero = false
	return m, true
}

// returnMission is the mission whose return the world map shows after mission
// n: for a link of the guest's chain, the win that started the chain.
func (g *guestMission) returnMission(n int) int {
	if g != nil && slices.ContainsFunc(g.links(), func(m mod.Mission) bool { return m.Number == n }) {
		return g.links()[0].After
	}
	return n
}

// heroParty is the party a link's npc records resolve against: the waiting
// party's starting hero, else fallback.
func (g *guestMission) heroParty(fallback []mapload.PartyMember) []mapload.PartyMember {
	if g == nil || g.hero == nil {
		return fallback
	}
	return []mapload.PartyMember{{StartingHero: true, PlayerCharacter: true, Mage: g.hero.Mage, FigureDir: g.hero.FigureDir}}
}

func (g *guestMission) clone() *guestMission {
	if g == nil {
		return nil
	}
	c := *g
	c.stash = slices.Clone(g.stash)
	c.start = mapload.CloneParty([]mapload.PartyMember{g.start})[0]
	c.rest = mapload.CloneParty(g.rest)
	c.chain = slices.Clone(g.chain)
	if g.member != nil {
		m := mapload.CloneParty([]mapload.PartyMember{*g.member})[0]
		c.member = &m
	}
	if g.hero != nil {
		h := *g.hero
		c.hero = &h
	}
	return &c
}

// SetModMissions records the missions the mods declare and serves each one's
// map and texts under its own mission number on this front end's filesystem:
// a guest mission under a number of its own, a replacing mission over the
// install's campaign mission. A declaration the campaign, the install or the
// character catalog does not bear out, or whose map the map editor's rule
// table reports an error for, is refused with the mod, the file and the line
// (DIV-3256). The warnings of the same rules refuse nothing and come back as
// lines naming the same mod, file and line. It runs once, after SetMods and
// the other mod tables and before any game is opened.
func (f *FrontEnd) SetModMissions(d mod.MissionData) (warnings []string, err error) {
	_, warnings, err = f.setModMissions(d)
	return warnings, err
}

// setModMissions is SetModMissions answering the missions a later mod
// replaced: the records merge through mod.MergeMissions first.
func (f *FrontEnd) setModMissions(declared mod.MissionData) ([]mod.Override, []string, error) {
	if declared.Empty() {
		return nil, nil, nil
	}
	if f == nil || f.Table == nil || f.Archives == nil || f.Archives.Containers == nil {
		return nil, nil, errors.New("the front end has no definition table or archives to carry the mods' missions")
	}
	d, overrides, err := mod.MergeMissions(declared)
	if err != nil {
		return nil, nil, err
	}
	c := f.Campaign.Value()
	src := f.Archives.Containers
	for _, m := range d.Missions {
		bad := func(format string, args ...any) error {
			return itemFileError(m.Mod, m.File, m.Line, format, args...)
		}
		address, _ := MissionMap(m.Number)
		if m.Kind == mod.MissionReplace {
			if err := f.checkReplacing(m, address); err != nil {
				return nil, nil, bad("%v", err)
			}
			continue
		}
		if err := checkGuest(c, src, d, m); err != nil {
			return nil, nil, bad("%v", err)
		}
		town, _ := c.NextMission(d.Chain(m)[0].After)
		if err := f.checkGuestParty(m, town.Mission); err != nil {
			return nil, nil, itemFileError(m.Mod, m.File, m.PartyLine, "%v", err)
		}
	}
	warnings, err := f.checkModMaps(d)
	if err != nil {
		return nil, nil, err
	}
	files := f.modFileLayer()
	selector := LanguageSelector(src)
	for _, m := range d.Missions {
		serve := func(address string, b []byte) error {
			return files.serveBytes(m.Mod, m.File, address, b)
		}
		address, _ := MissionMap(m.Number)
		if err := serve(address, m.MapBytes); err != nil {
			return nil, nil, err
		}
		dir := mainPrefix + "text/battle/m" + strconv.Itoa(m.Number) + "/"
		for _, name := range slices.Sorted(maps.Keys(m.TextFiles)) {
			if err := serve(dir+name, EncodeInstallTextFile(m.TextFiles[name], selector)); err != nil {
				return nil, nil, err
			}
		}
		if m.Title != "" {
			if err := serve(dir+"title.txt", []byte(EncodeModText(m.Title, selector))); err != nil {
				return nil, nil, err
			}
		}
		if m.Brief != "" {
			if err := serve(dir+"briefing.txt", []byte(EncodeModText(m.Brief, selector))); err != nil {
				return nil, nil, err
			}
		}
	}
	f.Table.Mods.Missions = d
	return overrides, warnings, nil
}

// checkGuest refuses a guest mission the campaign c, the install src or the
// merged missions d do not bear out: its number is a main-mission number, a
// campaign mission or an installed map; its after is neither a won main
// mission whose win opens a town nor a guest mission; its chain holds two
// guests who join; or its guest and chapter are not a companion the chain's
// town grants and that town.
func checkGuest(c Campaign, src *vfs.FS, d mod.MissionData, m mod.Mission) error {
	address, _ := MissionMap(m.Number)
	switch {
	case m.Number%mainMissionStride == 0:
		return fmt.Errorf("number %d is a multiple of %d, the numbers of the campaign's main missions", m.Number, mainMissionStride)
	case slices.Contains(c.Side, m.Number) || slices.Contains(c.Offered, m.Number):
		return fmt.Errorf("number %d is a mission of this install's campaign", m.Number)
	}
	if _, err := src.ReadFile(address); err == nil {
		return fmt.Errorf("number %d is a map of this install (%s)", m.Number, address)
	}
	chain := d.Chain(m)
	root := chain[0]
	if !slices.Contains(c.Main, root.After) {
		return fmt.Errorf("after %d is not a main mission of this install's campaign, nor a guest mission of the active mods", root.After)
	}
	if next, ok := c.AutoAdvance(root.After); ok {
		return fmt.Errorf("mission %d is followed by mission %d, not by a town", root.After, next)
	}
	offer, ok := c.NextMission(root.After)
	if !ok || !offer.Town {
		return fmt.Errorf("the win of mission %d opens no town", root.After)
	}
	joiners := 0
	for _, link := range chain {
		if link.Guest != 0 {
			joiners++
		}
	}
	if joiners > 1 {
		return fmt.Errorf("the chain of missions %s holds %d guests who join; one guest of a chain joins the town", chainText(chain), joiners)
	}
	if m.Guest == 0 {
		return nil
	}
	if offer.Mission != m.Chapter {
		return fmt.Errorf("the win of mission %d opens the town of chapter %d, not chapter %d", root.After, offer.Mission, m.Chapter)
	}
	if !slices.Contains(c.Chapters[m.Chapter].AddHero, m.Guest) {
		return fmt.Errorf("the town of chapter %d does not grant companion %d", m.Chapter, m.Guest)
	}
	return nil
}

// chainText names the links of a chain in order.
func chainText(chain []mod.Mission) string {
	parts := make([]string, len(chain))
	for i, m := range chain {
		parts[i] = strconv.Itoa(m.Number)
	}
	return strings.Join(parts, ", ")
}

// checkGuestParty refuses a party member of guest mission m, whose chain ends
// in the town of chapter town, that the character catalog does not hold or
// that the one builder cannot build: a level the character has no row for,
// an item no cell of its kind holds, a name the install cannot draw.
func (f *FrontEnd) checkGuestParty(m mod.Mission, town int) error {
	ref := []mapload.PartyMember{{StartingHero: true, PlayerCharacter: true}}
	for _, g := range m.Party {
		if _, err := f.guestMember(g, town, ref); err != nil {
			return err
		}
	}
	return nil
}

// guestLevelMissions is the width of one band of the composed npc rows: a
// composed record takes the row of band mission/40 (HERO-JOIN-120), so level
// L is the band of mission (L-1)*40.
const guestLevelMissions = 40

// guestMemberID is the party identity of a guest member: the npc record's
// own for a record of npc.reg, else the mod character's key.
func guestMemberID(c mod.Character) string {
	if c.NPC != 0 {
		return fmt.Sprintf("npc:%d", c.NPC)
	}
	return "char:" + c.Key
}

// guestMember builds member g of a guest party whose chain ends in the town
// of chapter town, resolved against ref, through the one builder of a
// persistent party member (DIV-4002).
func (f *FrontEnd) guestMember(g mod.GuestMember, town int, ref []mapload.PartyMember) (mapload.PartyMember, error) {
	t := f.Table
	c, ok := catalogCharacter(t, g.Character)
	if !ok {
		return mapload.PartyMember{}, fmt.Errorf("character %q is not in the character catalog (npc<n> of npc.reg, or the key of a mod's [[character]])", g.Character)
	}
	s, err := guestSpec(t, c, g, town)
	if err != nil {
		return mapload.PartyMember{}, err
	}
	name := ""
	if c.NPC == 0 && c.Name != "" {
		if name, err = encodeSaveLabel(c.Name, f.textSelector()); err != nil {
			return mapload.PartyMember{}, fmt.Errorf("character %q: the name %q cannot be drawn by this install: %v", g.Character, c.Name, err)
		}
	}
	if c.NPC != 0 && s.Row > 0 {
		e, _ := t.NPCCatalog.Entry(c.NPC)
		name = e.Name
		if name == "" {
			name = f.townInstall().localizedNPC(int(c.NPC)-1, c.Key)
		}
	}
	member, ok := f.townInstall().character(s, name, ref)
	if !ok {
		return mapload.PartyMember{}, fmt.Errorf("character %q has no valid Humans definition with a unique server id and resolvable equipment", g.Character)
	}
	member.ID = guestMemberID(c)
	return member, nil
}

// guestSpec is what the builder builds for member g, character c of the
// catalog: an npc record resolved at its level's band, or by default for the
// town's chapter as the town grants it; a mod character's row, or the row of
// its family at the level, by default the band of the town's chapter; with
// the face c sets and the cells g sets.
func guestSpec(t *mapload.Table, c mod.Character, g mod.GuestMember, town int) (mapload.CharacterSpec, error) {
	s := mapload.CharacterSpec{NPC: c.NPC, Face: c.Face, Cells: guestCells(g)}
	if t == nil || t.Humans == nil {
		return s, fmt.Errorf("character %q has no Humans collection", g.Character)
	}
	if c.NPC != 0 {
		s.Mission = town
		if g.Level > 0 {
			if _, direct := t.NPC.ServerID(c.NPC); direct {
				return s, fmt.Errorf("character %q is one definition row; it takes no level", g.Character)
			}
			s.Mission = (g.Level - 1) * guestLevelMissions
		}
		_, direct := t.NPC.ServerID(c.NPC)
		_, _, composed := t.NPC.ComposedArchetype(c.NPC, false, false)
		if !direct && !composed {
			portrait := t.NPCCatalog.Portraits()[c.NPC]
			selector := 0
			if portrait.Tokens.Has(data.NPCTokenFemale) {
				selector++
			}
			if portrait.Tokens.Has(data.NPCTokenMage) {
				selector += 2
			}
			// DIV-4003: a dialogue-only record borrows an archetype's row.
			level := g.Level
			if level == 0 {
				level = min(town/guestLevelMissions+1, mod.MaxGuestLevel)
			}
			name := data.ChargenBaseNames()[selector]
			if level > 1 {
				name = fmt.Sprintf("%s_%d", name, level)
			}
			row, err := levelRow(t.Humans, name, 0, town)
			if err != nil {
				return s, fmt.Errorf("character %q has no DataBinID; default: %v", g.Character, err)
			}
			s.Row = row
			if s.Face == 0 && portrait.Kind == data.NPCFigure {
				s.Face = int32(portrait.Face)
			}
		}
	} else {
		row, err := levelRow(t.Humans, c.Row, g.Level, town)
		if err != nil {
			return s, fmt.Errorf("character %q: %v", g.Character, err)
		}
		s.Row = row
	}
	if err := mapload.CheckCharacterCells(t, s.Cells); err != nil {
		return s, fmt.Errorf("character %q: no item of its cell is %v", g.Character, err)
	}
	return s, nil
}

// guestCells are the equipment cells g sets: the weapon, the shield and, when
// armour is set, all eight armour cells, those it does not name emptied.
func guestCells(g mod.GuestMember) map[int]string {
	cells := map[int]string{}
	if g.Weapon != nil {
		cells[data.HumanCellWeapon] = *g.Weapon
	}
	if g.Shield != nil {
		cells[data.HumanCellShield] = *g.Shield
	}
	if g.Armour != nil {
		for cell := data.HumanCellArmour; cell < data.HumanCellCount; cell++ {
			cells[cell] = ""
			if i := cell - data.HumanCellArmour; i < len(g.Armour) {
				cells[cell] = g.Armour[i]
			}
		}
	}
	if len(cells) == 0 {
		return nil
	}
	return cells
}

// levelRow is the row name names at level: the row of that name when no level
// is set, else the row of its family named family_level, the family's band
// for the town's chapter when no level is set and name is a family.
func levelRow(c data.Collection, name string, level, town int) (int, error) {
	rows := characterRows(c, name)
	if len(rows) == 0 {
		return 0, fmt.Errorf("no definition row is named %q, nor is %q a family of tier rows", name, name)
	}
	family := name
	if c.EntryName(rows[0]) == name {
		if level == 0 {
			return rows[0], nil
		}
		at := strings.LastIndexByte(name, '_')
		if at <= 0 || !allDigits(name[at+1:]) {
			return 0, fmt.Errorf("the row %q is no tier of a family; it takes no level", name)
		}
		family = name[:at]
	}
	if level == 0 {
		level = min(town/guestLevelMissions+1, mod.MaxGuestLevel)
	}
	want := fmt.Sprintf("%s_%d", family, level)
	for _, i := range characterRows(c, want) {
		if c.EntryName(i) == want {
			return i, nil
		}
	}
	return 0, fmt.Errorf("no definition row is named %q, the row of level %d of %q", want, level, family)
}

// guestParty builds the party link m opens with in a chain that ends in the
// town of chapter town, resolved against ref. A member of continuing, the
// party the previous link ended with, whom m names again comes as she ended
// that link. The first member leads: the mission's player character is she.
func (f *FrontEnd) guestParty(m mod.Mission, town int, ref, continuing []mapload.PartyMember) ([]mapload.PartyMember, error) {
	again := func(id string) (mapload.PartyMember, bool) {
		for _, p := range continuing {
			if p.ID == id {
				return mapload.CloneParty([]mapload.PartyMember{p})[0], true
			}
		}
		return mapload.PartyMember{}, false
	}
	var party []mapload.PartyMember
	if len(m.Party) == 0 {
		member, ok := again(fmt.Sprintf("npc:%d", m.Guest))
		if !ok {
			member, ok = townCompanion(f.townInstall(), m.Chapter, m.Guest, ref)
		}
		if !ok {
			return nil, fmt.Errorf("companion %d of chapter %d is not built for this party", m.Guest, m.Chapter)
		}
		party = append(party, member)
	}
	for _, g := range m.Party {
		c, _ := catalogCharacter(f.Table, g.Character)
		member, ok := again(guestMemberID(c))
		if !ok {
			var err error
			if member, err = f.guestMember(g, town, ref); err != nil {
				return nil, err
			}
		}
		party = append(party, member)
	}
	for i := range party {
		party[i].StartingHero = i == 0
		party[i].PlayerCharacter = true
	}
	return party, nil
}

// modMapInspector is the map editor's inspector over this front end's tables,
// the rule table's one reader at the mod load.
func (f *FrontEnd) modMapInspector() (*MapInspector, error) {
	structures, err := LoadStructures(f.Archives.Containers)
	if errors.Is(err, fs.ErrNotExist) {
		structures, err = nil, nil // the structure-kind rule has no table to hold the map against
	}
	if err != nil {
		return nil, err
	}
	return &MapInspector{archives: f.Archives, table: f.Table, structures: structures}, nil
}

// checkModMaps runs the map editor's rule table over the map of every mission
// the mods declare, with the texts that mission ships. An error finding
// refuses the mission; the lines of the warnings come back.
func (f *FrontEnd) checkModMaps(d mod.MissionData) ([]string, error) {
	x, err := f.modMapInspector()
	if err != nil {
		return nil, err
	}
	var warnings []string
	for _, m := range d.Missions {
		address, _ := MissionMap(m.Number)
		var refusals []string
		for _, finding := range x.CheckModMap(m, address) {
			if finding.Error {
				refusals = append(refusals, finding.Line())
				continue
			}
			warnings = append(warnings, fmt.Sprintf("mod %q: %s:%d: map %s: %s", m.Mod, m.File, m.Line, m.Map, finding.Line()))
		}
		if len(refusals) > 0 {
			return nil, itemFileError(m.Mod, m.File, m.Line, "map %s: %s", m.Map, strings.Join(refusals, "; "))
		}
	}
	return warnings, nil
}

// checkReplacing refuses a replacing mission whose number is no mission of
// this install's campaign or whose map the mission build refuses.
func (f *FrontEnd) checkReplacing(m mod.Mission, address string) error {
	c := f.Campaign.Value()
	if !slices.Contains(c.Main, m.Number) && !slices.Contains(c.Side, m.Number) && !slices.Contains(c.Offered, m.Number) {
		return fmt.Errorf("number %d is no mission of this install's campaign", m.Number)
	}
	if _, err := f.Archives.Containers.ReadFile(address); err != nil {
		return fmt.Errorf("the install holds no map for mission %d (%s)", m.Number, address)
	}
	if _, err := MapOpener(f.Archives.Game())(m.MapBytes); err != nil {
		return fmt.Errorf("map %s: %v", m.Map, err)
	}
	return nil
}

// startGuestMission is the guest half of the win of mission n: when a mod
// mission plays after it, the campaign is written as the town SAV of this
// moment and the guest party is built as the town would grant its members.
// It answers the mission to open, or zero and the reason none opens.
func (f *FrontEnd) startGuestMission(n int, w *sim.World) (int, string) {
	if f.Table == nil {
		return 0, ""
	}
	missions := f.Table.Mods.Missions
	m, ok := missions.AfterWin(n)
	if !ok {
		return 0, ""
	}
	in := f.townInstall()
	town, _ := in.campaign.NextMission(n)
	party, err := f.guestParty(m, town.Mission, f.Carried, nil)
	if err != nil {
		return 0, fmt.Sprintf("mod %q: %v", m.Mod, err)
	}
	g := &guestMission{mission: m, start: party[0], rest: party[1:], chain: missions.Chain(m)}
	if _, more := missions.AfterWin(m.Number); more {
		hero := guestHero{}
		if len(f.Carried) > 0 {
			p := f.Carried[0]
			for _, c := range f.Carried {
				if c.StartingHero {
					p = c
					break
				}
			}
			hero = guestHero{Mage: p.Mage, FigureDir: p.FigureDir}
		}
		g.hero = &hero
	}
	f.Town.Arrive()
	f.retainReturnCity(in, n, w)
	var s Snapshot
	err = f.captureSave(captureTimer, false, func(c Snapshot, _ string) error {
		s = c
		return nil
	})
	if err == nil {
		if g.stash, err = f.ExportCurrentSave(s, "guest"); err == nil {
			f.Town.guest = g
			return m.Number, ""
		}
	}
	return 0, fmt.Sprintf("mod %q: the waiting party could not be written: %v", m.Mod, err)
}

// guestEntry is the party and purse the guest's mission n opens with: the
// guest party alone with the declared purse.
func (g *guestMission) guestEntry(n int) ([]mapload.PartyMember, uint32, bool) {
	if !g.playing(n) {
		return nil, 0, false
	}
	return g.party(), uint32(g.mission.Gold), true
}

// finishGuestMission is the win of the guest's mission n. The guest party
// comes home through the carry every won mission uses. When a guest mission
// plays after n, the party opens it with no town between, each member it
// names again as she ended n. Otherwise the waiting party comes back through
// the one town LOAD of the stash, which records the guest who joins beside
// it so the town's grant takes her as she is now; nothing else of the chain
// carries back.
func (f *FrontEnd) finishGuestMission(n int, party []mapload.PartyMember,
	w *sim.World, ids []sim.EntityID, roster map[sim.EntityID]mapload.PartyMember) (int, string) {
	g := f.Town.guestMission()
	if refused := f.carryMissionHome(f.townInstall(), n, party, w, ids, roster); refused != "" {
		return -1, refused
	}
	member := g.member
	joiner, joins := g.joiner()
	if joins {
		in := func(p mapload.PartyMember) bool { return p.CompanionNPC == joiner.Guest }
		if i := slices.IndexFunc(f.Carried, in); i >= 0 {
			c := mapload.CloneParty([]mapload.PartyMember{f.Carried[i]})[0]
			member = &c
		} else if slices.ContainsFunc(g.party(), in) {
			return -1, fmt.Sprintf("mission %d won - companion %d did not come home", n, joiner.Guest)
		}
	}
	missions := f.Table.Mods.Missions
	if next, ok := missions.AfterWin(n); ok {
		town, _ := f.townInstall().campaign.NextMission(g.links()[0].After)
		start, err := f.guestParty(next, town.Mission, g.heroParty(f.Carried), f.Carried)
		if err != nil {
			return -1, fmt.Sprintf("mission %d won - mod %q: %v", n, next.Mod, err)
		}
		f.Town.guest = &guestMission{mission: next, stash: g.stash, start: start[0], rest: start[1:],
			member: member, chain: missions.Chain(next), hero: g.hero}
		return next.Number, fmt.Sprintf("mission %d won - the guest party goes on to mission %d", n, next.Number)
	}
	var back *guestMission
	if member != nil {
		start := *member
		if joiner.Number == g.mission.Number {
			start = g.start
		}
		back = &guestMission{mission: joiner, start: start, member: member}
	}
	if _, _, err := f.restoreOriginal(g.stash, back); err != nil {
		return -1, fmt.Sprintf("mission %d won - the waiting party did not come back: %v", n, err)
	}
	if member == nil {
		return 0, fmt.Sprintf("mission %d won - the party goes on to the town", n)
	}
	return 0, fmt.Sprintf("mission %d won - %s returns; the party goes on to the town", n, member.Name)
}

// currentGuest is the supplement's record of a guest party: the mod and the
// link under way, the stash while the chain runs, the party at the link's
// start (Start and Party), the guest who joins as she ended her link once it
// is won, and the waiting party's starting hero for a chain of more links.
// A save of a chain of one link writes neither Party nor Hero.
type currentGuest struct {
	Mod    string
	Key    string
	Number int
	Stash  []byte               `json:",omitempty"`
	Start  *currentPartyMember  `json:",omitempty"`
	Member *currentPartyMember  `json:",omitempty"`
	Party  []currentPartyMember `json:",omitempty"`
	Hero   *guestHero           `json:",omitempty"`
}

func (g *guestMission) record() *currentGuest {
	if g == nil {
		return nil
	}
	start := captureCurrentParty(0, g.start)
	r := &currentGuest{Mod: g.mission.Mod, Key: g.mission.Key, Number: g.mission.Number, Stash: slices.Clone(g.stash), Start: &start}
	if g.member != nil {
		m := captureCurrentParty(0, *g.member)
		r.Member = &m
	}
	for _, p := range g.rest {
		r.Party = append(r.Party, captureCurrentParty(0, p))
	}
	if g.hero != nil {
		h := *g.hero
		r.Hero = &h
	}
	return r
}

// restoreGuest is the guest a save records, checked against the missions the
// active mods declare. A save of a mod mission or of a waiting guest loads
// only under the mod that declares that mission; the chain is the declared
// chain that mission ends.
func restoreGuest(s *currentSessionData, mods mapload.ModContext) (*guestMission, error) {
	if s == nil || s.Guest == nil {
		return nil, nil
	}
	r := s.Guest
	m, ok := mods.Missions.Find(r.Mod, r.Key)
	if !ok {
		return nil, modMarkRefusal("this save plays mission %q of mod %q, which no active mod declares; start with that mod to load it", r.Key, r.Mod)
	}
	if m.Number != r.Number {
		return nil, modMarkRefusal("this save plays mission %q of mod %q as mission %d; the active mod declares mission %d", r.Key, r.Mod, r.Number, m.Number)
	}
	g := &guestMission{mission: m, chain: mods.Missions.Chain(m)}
	joiner, joins := g.joiner()
	switch {
	case r.Start == nil || r.Stash == nil && r.Member == nil,
		r.Member != nil && (!joins || (r.Stash == nil) != (joiner.Number == m.Number)),
		r.Stash == nil && (len(r.Party) != 0 || r.Hero != nil),
		len(r.Party) >= mod.MaxGuestParty:
		return nil, fmt.Errorf("invalid current guest record of mod %q", r.Mod)
	}
	if len(r.Stash) > sav.MaxNativeActions {
		return nil, fmt.Errorf("current guest record of mod %q is too large", r.Mod)
	}
	g.stash, g.start = slices.Clone(r.Stash), r.Start.restore()
	for _, p := range r.Party {
		g.rest = append(g.rest, p.restore())
	}
	if r.Hero != nil {
		h := *r.Hero
		g.hero = &h
	}
	if r.Member != nil {
		member := r.Member.restore()
		if member.CompanionNPC != joiner.Guest {
			return nil, fmt.Errorf("current guest record of mod %q names companion %d, not %d", r.Mod, member.CompanionNPC, joiner.Guest)
		}
		g.member = &member
	}
	return g, nil
}

// refuseUnmarkedReplaced refuses a save inside mission n, written without a
// mod mark, while an active mod replaces that mission's map: its world was
// built from another map. A marked save passed the mod mark, so the mod that
// replaces the map now replaced it when the save was written.
func refuseUnmarkedReplaced(saved []byte, n int, mods mapload.ModContext) error {
	m, ok := mods.Missions.Replacing(n)
	if !ok || carriesModMark(saved) {
		return nil
	}
	return modMarkRefusal("this save plays mission %d without a mod mark, and mod %q replaces that mission's map; start without that mod to load it", n, m.Mod)
}

// carriesModMark reports that saved holds a readable mod mark.
func carriesModMark(saved []byte) bool {
	if !sav.MayCarryNativeMods(saved) {
		return false
	}
	doc, err := sav.DecodeDocumentData(saved)
	if err != nil {
		return false
	}
	_, ok, err := readModMark(doc)
	return ok && err == nil
}
