package game

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/sim"
	"againrom/pkg/vfs"
)

// A mod mission is played by a guest party while the party waits. When the
// declared win is recorded and the town would open, the campaign is written
// as the town SAV the one producer writes at that point (the stash) and the
// guest opens the mod's mission alone. When the guest wins, the stash comes
// back through the one town LOAD with the guest as she ended the mission
// recorded beside it, and the town's own grant of that companion takes her
// in that state (DIV-3023).

// guestMission returns the town's guest; nil for a nil town.
func (t *Town) guestMission() *guestMission {
	if t == nil {
		return nil
	}
	return t.guest
}

// guestMission is the session's record of a mod mission and its guest.
type guestMission struct {
	mission mod.Mission
	// stash is the town SAV of the waiting party; nil once the guest has won.
	stash []byte
	// start is the guest as the town would grant her when the mission began.
	start mapload.PartyMember
	// member is the guest as she ended the mission, waiting for the town's
	// grant; nil while the mission runs.
	member *mapload.PartyMember
}

// playing reports that mission n is the guest's mission, still under way.
func (g *guestMission) playing(n int) bool {
	return g != nil && g.stash != nil && g.mission.Number == n
}

// joining is the guest the town of chapter grants as companion npc, as she
// ended her mission, when one is waiting.
func (g *guestMission) joining(chapter, npc int) (mapload.PartyMember, bool) {
	if g == nil || g.member == nil || g.mission.Chapter != chapter || g.mission.Guest != npc {
		return mapload.PartyMember{}, false
	}
	m := mapload.CloneParty([]mapload.PartyMember{*g.member})[0]
	m.StartingHero = false
	return m, true
}

// returnMission is the mission whose return the world map shows after mission
// n: for the guest's mission, the win that started it.
func (g *guestMission) returnMission(n int) int {
	if g != nil && g.mission.Number == n {
		return g.mission.After
	}
	return n
}

func (g *guestMission) clone() *guestMission {
	if g == nil {
		return nil
	}
	c := *g
	c.stash = slices.Clone(g.stash)
	c.start = mapload.CloneParty([]mapload.PartyMember{g.start})[0]
	if g.member != nil {
		m := mapload.CloneParty([]mapload.PartyMember{*g.member})[0]
		c.member = &m
	}
	return &c
}

// SetModMissions records the missions the mods declare and serves each one's
// map and texts under its own mission number on this front end's filesystem:
// a guest mission under a number of its own, a replacing mission over the
// install's campaign mission. A declaration the campaign or the install does
// not bear out, or whose map the map editor's rule table reports an error
// for, is refused with the mod, the file and the line (DIV-3256). The
// warnings of the same rules refuse nothing and come back as lines naming the
// same mod, file and line. It runs once, after SetMods and the other mod
// tables and before any game is opened.
func (f *FrontEnd) SetModMissions(d mod.MissionData) (warnings []string, err error) {
	if d.Empty() {
		return nil, nil
	}
	if f == nil || f.Table == nil || f.Archives == nil || f.Archives.Containers == nil {
		return nil, errors.New("the front end has no definition table or archives to carry the mods' missions")
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
				return nil, bad("%v", err)
			}
			continue
		}
		if err := checkGuest(c, src, m); err != nil {
			return nil, bad("%v", err)
		}
	}
	warnings, err = f.checkModMaps(d)
	if err != nil {
		return nil, err
	}
	files := f.modFileLayer()
	selector := LanguageSelector(src)
	for _, m := range d.Missions {
		serve := func(address string, b []byte) error {
			return files.serveBytes(m.Mod, m.File, address, b)
		}
		address, _ := MissionMap(m.Number)
		if err := serve(address, m.MapBytes); err != nil {
			return nil, err
		}
		dir := mainPrefix + "text/battle/m" + strconv.Itoa(m.Number) + "/"
		for _, name := range slices.Sorted(maps.Keys(m.TextFiles)) {
			if err := serve(dir+name, EncodeInstallTextFile(m.TextFiles[name], selector)); err != nil {
				return nil, err
			}
		}
		if m.Title != "" {
			if err := serve(dir+"title.txt", []byte(EncodeModText(m.Title, selector))); err != nil {
				return nil, err
			}
		}
		if m.Brief != "" {
			if err := serve(dir+"briefing.txt", []byte(EncodeModText(m.Brief, selector))); err != nil {
				return nil, err
			}
		}
	}
	f.Table.Mods.Missions = d
	return warnings, nil
}

// checkGuest refuses a guest mission the campaign c or the install src does
// not bear out: its number is a main-mission number, a campaign mission or an
// installed map, or its after, chapter and guest are not a won main mission,
// the town that win opens and a companion that town grants.
func checkGuest(c Campaign, src *vfs.FS, m mod.Mission) error {
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
	if !slices.Contains(c.Main, m.After) {
		return fmt.Errorf("after %d is not a main mission of this install's campaign", m.After)
	}
	if next, ok := c.AutoAdvance(m.After); ok {
		return fmt.Errorf("mission %d is followed by mission %d, not by a town", m.After, next)
	}
	offer, ok := c.NextMission(m.After)
	if !ok || !offer.Town {
		return fmt.Errorf("the win of mission %d opens no town", m.After)
	}
	if offer.Mission != m.Chapter {
		return fmt.Errorf("the win of mission %d opens the town of chapter %d, not chapter %d", m.After, offer.Mission, m.Chapter)
	}
	if !slices.Contains(c.Chapters[m.Chapter].AddHero, m.Guest) {
		return fmt.Errorf("the town of chapter %d does not grant companion %d", m.Chapter, m.Guest)
	}
	return nil
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
// moment and the guest is built as the town would grant her. It answers the
// mission to open, or zero and the reason none opens.
func (f *FrontEnd) startGuestMission(n int, w *sim.World) (int, string) {
	if f.Table == nil {
		return 0, ""
	}
	m, ok := f.Table.Mods.Missions.AfterWin(n)
	if !ok {
		return 0, ""
	}
	in := f.townInstall()
	start, ok := townCompanion(in, m.Chapter, m.Guest, f.Carried)
	if !ok {
		return 0, fmt.Sprintf("mod %q: companion %d of chapter %d is not built for this party", m.Mod, m.Guest, m.Chapter)
	}
	// She leads her own party: the mission's player character is she.
	start.StartingHero = true
	f.Town.Arrive()
	f.retainReturnCity(in, n, w)
	var s Snapshot
	err := f.captureSave(captureTimer, false, func(c Snapshot, _ string) error {
		s = c
		return nil
	})
	if err == nil {
		var stash []byte
		if stash, err = f.ExportCurrentSave(s, "guest"); err == nil {
			f.Town.guest = &guestMission{mission: m, stash: stash, start: start}
			return m.Number, ""
		}
	}
	return 0, fmt.Sprintf("mod %q: the waiting party could not be written: %v", m.Mod, err)
}

// guestEntry is the party and purse the guest's mission n opens with: the
// guest alone with the declared purse.
func (g *guestMission) guestEntry(n int) ([]mapload.PartyMember, uint32, bool) {
	if !g.playing(n) {
		return nil, 0, false
	}
	return mapload.CloneParty([]mapload.PartyMember{g.start}), uint32(g.mission.Gold), true
}

// finishGuestMission is the win of the guest's mission n. The guest comes
// home through the carry every won mission uses; nothing else of the mission
// carries back. The waiting party comes back through the one town LOAD of the
// stash, which records the guest beside it so the town's grant takes her as
// she is now.
func (f *FrontEnd) finishGuestMission(n int, party []mapload.PartyMember,
	w *sim.World, ids []sim.EntityID, roster map[sim.EntityID]mapload.PartyMember) (int, string) {
	g := f.Town.guestMission()
	if refused := f.carryMissionHome(f.townInstall(), n, party, w, ids, roster); refused != "" {
		return -1, refused
	}
	var member *mapload.PartyMember
	for _, p := range f.Carried {
		if p.CompanionNPC == g.mission.Guest {
			c := mapload.CloneParty([]mapload.PartyMember{p})[0]
			member = &c
			break
		}
	}
	if member == nil {
		return -1, fmt.Sprintf("mission %d won - companion %d did not come home", n, g.mission.Guest)
	}
	back := &guestMission{mission: g.mission, start: g.start, member: member}
	if _, _, err := f.restoreOriginal(g.stash, back); err != nil {
		return -1, fmt.Sprintf("mission %d won - the waiting party did not come back: %v", n, err)
	}
	return 0, fmt.Sprintf("mission %d won - %s returns; the party goes on to the town", n, member.Name)
}

// currentGuest is the supplement's record of a guest: the mod and mission,
// the stash while the mission runs, the guest at its start and as she ended
// it once it is won.
type currentGuest struct {
	Mod    string
	Key    string
	Number int
	Stash  []byte              `json:",omitempty"`
	Start  *currentPartyMember `json:",omitempty"`
	Member *currentPartyMember `json:",omitempty"`
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
	return r
}

// restoreGuest is the guest a save records, checked against the missions the
// active mods declare. A save of a mod mission or of a waiting guest loads
// only under the mod that declares that mission.
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
	if r.Start == nil || (r.Stash == nil) == (r.Member == nil) {
		return nil, fmt.Errorf("invalid current guest record of mod %q", r.Mod)
	}
	if len(r.Stash) > sav.MaxNativeActions {
		return nil, fmt.Errorf("current guest record of mod %q is too large", r.Mod)
	}
	g := &guestMission{mission: m, stash: slices.Clone(r.Stash), start: r.Start.restore()}
	if r.Member != nil {
		member := r.Member.restore()
		if member.CompanionNPC != m.Guest {
			return nil, fmt.Errorf("current guest record of mod %q names companion %d, not %d", r.Mod, member.CompanionNPC, m.Guest)
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
