package game

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const (
	guestModID     = "reniesta"
	guestMissionNo = 25
	guestCompanion = 22
)

// guestMods copies the fixture mods ids into a fresh mods folder and assembles
// the reniesta mod's map and texts from the install under test.
func guestMods(t *testing.T, ids ...string) string {
	t.Helper()
	return assembledMods(t, guestModID, ids...)
}

// assembledMods copies the fixture mods ids into a fresh mods folder and
// assembles the map and texts of the mod into from the install under test.
func assembledMods(t *testing.T, into string, ids ...string) string {
	t.Helper()
	if os.Getenv("AGAINROM_ASSETS") == "" {
		t.Skip("no AGAINROM_ASSETS: the guest mission needs a lawful install")
	}
	root := t.TempDir()
	for _, id := range ids {
		from := filepath.Join(modItemsDir, id)
		err := filepath.WalkDir(from, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel, _ := filepath.Rel(from, p)
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			dst := filepath.Join(root, id, rel)
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return err
			}
			return os.WriteFile(dst, b, 0o644)
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	install := os.Getenv("AGAINROM_ASSETS")
	lang := modrt.LanguageFor(BaseID(InspectInstall(install)))
	written, err := AssembleModMission(install, lang, 10, filepath.Join(root, into))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(written, "mission/"+lang+"/map.alm") || len(written) < 2 {
		t.Fatalf("assembled %v", written)
	}
	return root
}

// guestFront is a front end on the install under the mods ids of the folder
// dir, set up through the launcher's own steps.
func guestFront(t *testing.T, dir string, ids ...string) *FrontEnd {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	if len(ids) == 0 {
		return f
	}
	entries, err := mod.Resolve(dir, ids)
	if err != nil {
		t.Fatal(err)
	}
	res, err := modrt.Load(entries, BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS"))), nil, modrt.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.SetMods(res.Rules, res.Set, false); err != nil {
		t.Fatal(err)
	}
	if err := f.SetModCompanions(res.Companions); err != nil {
		t.Fatal(err)
	}
	if _, err := f.SetModMissions(res.Missions); err != nil {
		t.Fatal(err)
	}
	return f
}

// guestRoute plays the campaign's first two missions through the production
// completion path with a hero of the given choices and returns the answer of
// mission 20's win.
func guestRoute(t *testing.T, f *FrontEnd, choices []int) (*ui.App, int) {
	t.Helper()
	party := f.ChargenParty(ui.ChargenResult{Name: "Guest", Choices: choices, Stats: []int{31, 27, 24, 29}})
	app := f.App("guest mission")
	next := 0
	for _, n := range []int{10, 20} {
		open := f.MissionOpenerWith(n, party)
		if n != 10 {
			open = f.MissionOpener(n)
		}
		if err := app.OpenMission(open); err != nil {
			t.Fatalf("open mission %d: %v", n, err)
		}
		var err error
		if next, _, err = f.LiveCompleteCampaign(); err != nil {
			t.Fatalf("finish mission %d: %v", n, err)
		}
	}
	return app, next
}

// guestPlay opens the guest's mission and has the guest kill the nearest
// living enemy through the ordinary attack order, healed each tick, so she
// ends the mission with experience she did not start with. It returns her
// experience at the end.
func guestPlay(t *testing.T, f *FrontEnd, app *ui.App) [6]int32 {
	t.Helper()
	if err := app.OpenMission(f.MissionOpener(guestMissionNo)); err != nil {
		t.Fatalf("open the guest mission: %v", err)
	}
	party := f.LiveParty()
	if len(party) != 1 || party[0].CompanionNPC != guestCompanion || f.live.mission.state.Address != "scenario/25.alm" {
		t.Fatalf("the guest mission opened %q with %v", f.live.mission.state.Address, party)
	}
	if got := f.live.world.Purse(sim.SelfSlot); got != 0 {
		t.Fatalf("the guest's purse is %d", got)
	}
	live, w := f.live, f.live.world
	hero := live.mission.ids[0]
	h, _ := live.entity(hero)
	before := h.SkillXP
	victim, best := sim.EntityID(0), int64(-1)
	relations := w.Relations()
	for _, e := range w.Entities() {
		if !e.Alive() || e.HP <= 0 || !relations.Hostile(sim.SelfSlot, e.Owner) {
			continue
		}
		d := int64(e.X-h.X)*int64(e.X-h.X) + int64(e.Y-h.Y)*int64(e.Y-h.Y)
		if best < 0 || d < best {
			victim, best = e.ID, d
		}
	}
	if best < 0 {
		t.Fatal("the guest mission has no enemy to fight")
	}
	v, _ := live.entity(victim)
	if err := w.HeadlessPlace(hero, v.X-1, v.Y); err != nil {
		t.Fatal(err)
	}
	for i := 0; ; i++ {
		if i == 30000 {
			t.Fatalf("the guest did not kill entity %d", victim)
		}
		if i%300 == 0 {
			live.pending = append(live.pending, sim.Attack(hero, victim))
		}
		if h, _ := live.entity(hero); h.HP < h.MaxHP {
			_ = w.HeadlessHeal(hero)
		}
		live.tick()
		if e, ok := live.entity(victim); !ok || e.HP < 0 {
			break
		}
	}
	h, _ = live.entity(hero)
	if h.SkillXP == before {
		t.Fatalf("the guest gained no experience: %v", h.SkillXP)
	}
	return h.SkillXP
}

// guestWin acknowledges the live mission's win through the map driver's own
// continuation and returns where it leads.
func guestWin(t *testing.T, f *FrontEnd) (ui.NoticeDest, string) {
	t.Helper()
	f.live.mission.outcome = sim.OutcomeWon
	list := func(...ui.NoticeAction) (ui.NoticeDest, string, ui.MapOpener) { return ui.NoticeToMapList, "", nil }
	dest, line, _ := continueMission(frontTransitions{f}, f.liveMission, f.live.mission.state, list)()
	return dest, line
}

// guestDigest is the campaign the town holds, apart from the guest companion:
// every other member whole, the purse, the progress, the hired pools, the
// documents, the fame and the session's choices. A fresh town records a
// consumed grant by chapter and a loaded one in its AddHero array, so the
// pending grants are read from the town SAV the one producer writes.
func guestDigest(t *testing.T, f *FrontEnd) string {
	t.Helper()
	var pending []int
	for _, n := range modJoinSavedAddHero(t, currentTownSave(t, f)) {
		pending = append(pending, int(n))
	}
	s, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	var party []mapload.PartyMember
	for _, p := range s.Party {
		if p.CompanionNPC != guestCompanion {
			party = append(party, p)
		}
	}
	b, err := json.Marshal(struct {
		Party                                     []mapload.PartyMember
		Gold, Main, Selected, Offered, Documented int
		Won, Available, PendingGrants             []int
		Difficulty                                mapload.Difficulty
		QuickSpells                               [4]uint32
		Fame                                      *SnapshotFame
		Pool                                      [16]int
		Enabled, Hired                            [16]bool
		Documents                                 []SnapshotDocument
		Taken                                     []SnapshotOffer
		LastMap                                   string
		Chapter                                   int
	}{party, s.Gold, s.MainMission, s.SelectedMission, s.Offered, s.DocumentMission,
		s.Won, s.Available, pending, s.Difficulty, s.QuickSpells, s.Fame, s.MercenaryPool,
		s.MercenaryEnabled, s.MercenaryHired, s.Documents, s.Taken, s.LastMap, f.Town.Chapter()})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func guestMember(f *FrontEnd) (mapload.PartyMember, bool) {
	for _, p := range f.Carried {
		if p.CompanionNPC == guestCompanion {
			return p, true
		}
	}
	return mapload.PartyMember{}, false
}

func guestMemberJSON(t *testing.T, p mapload.PartyMember) string {
	t.Helper()
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// guestMissionSave is a SAVE taken on the map of the running mission.
func guestMissionSave(t *testing.T, f *FrontEnd) []byte {
	t.Helper()
	s, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	b, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatalf("mission SAVE: %v", err)
	}
	return b
}

// TestReleaseModMissionGuestPlaysAndJoins proves, on the lawful install, the
// guest route: mission 20's win opens the mod's mission with companion 22
// alone, its map read from the mod; her win brings back the party exactly as
// the town would have received it without the mod, and she joins it there as
// she ended her mission. A SAVE and cold LOAD inside the mission gives the
// same town.
func TestReleaseModMissionGuestPlaysAndJoins(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	dir := guestMods(t, guestModID)

	// Control: no mod. Mission 20's win opens the town with her definition row.
	plain := guestFront(t, dir)
	if _, next := guestRoute(t, plain, []int{0, 0, 0}); next != 0 || plain.Town.Chapter() != 30 {
		t.Fatalf("unmodded win of mission 20 answered %d in chapter %d", next, plain.Town.Chapter())
	}
	row, ok := guestMember(plain)
	if !ok || !row.Mage {
		t.Fatalf("the unmodded town granted %v", modJoinRoster(plain))
	}
	want := guestDigest(t, plain)

	// The guest route without a SAVE in the middle.
	f := guestFront(t, dir, guestModID)
	app, next := guestRoute(t, f, []int{0, 0, 0})
	if next != guestMissionNo || f.Town.guest == nil || len(f.Town.guest.stash) == 0 || f.campaign().route(f, 20, next) != winMission {
		t.Fatalf("mission 20's win answered %d (guest %v)", next, f.Town.guest != nil)
	}
	if len(f.Town.guest.stash) > 1<<20 {
		t.Fatalf("the stash is %d bytes", len(f.Town.guest.stash))
	}
	if _, has := guestMember(f); has {
		t.Fatal("the waiting party already holds the guest")
	}
	if f.Town.guest.start.Name != row.Name || f.Town.guest.start.Mage != row.Mage {
		t.Fatalf("the guest starts as %q, the town grants %q", f.Town.guest.start.Name, row.Name)
	}
	gained := guestPlay(t, f, app)
	if dest, line := guestWin(t, f); dest != ui.NoticeToTown {
		t.Fatalf("the guest's win led to %v: %q", dest, line)
	}
	// Mission 20 has no world-map marker, so its return, like the ordinary
	// one, stands at the town's home point with no travel to show.
	if wm := f.TownScreen().(*townScreen).worldMap; wm != nil {
		t.Fatalf("the return from the guest mission travels the world map from %d", wm.returnMission)
	}
	joined, ok := guestMember(f)
	if !ok || f.Town.guest != nil || f.Town.Chapter() != 30 {
		t.Fatalf("after the guest's win: roster %v chapter %d", modJoinRoster(f), f.Town.Chapter())
	}
	if got := guestDigest(t, f); got != want {
		t.Fatalf("the town after the guest differs from the town without the mod:\n got %s\nwant %s", got, want)
	}
	if joined.Carry == nil || joined.Carry.SkillXP != gained {
		t.Fatalf("the guest joined with experience %v, she ended her mission with %v", joined.Carry, gained)
	}
	if row.Carry != nil && row.Carry.SkillXP == gained {
		t.Fatal("the town's definition row already holds the guest's gains")
	}
	if !f.Town.Open() {
		t.Fatal("the town is not open after the guest's win")
	}

	// The same route with a SAVE and a cold LOAD inside the guest's mission.
	g := guestFront(t, dir, guestModID)
	gapp, _ := guestRoute(t, g, []int{0, 0, 0})
	guestPlay(t, g, gapp)
	saved := guestMissionSave(t, g)
	h := guestFront(t, dir, guestModID)
	open, town, err := h.RestoreOriginal(saved)
	if err != nil || town {
		t.Fatalf("cold LOAD of the guest mission: town %v %v", town, err)
	}
	happ := h.App("guest mission load")
	if err := happ.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if !h.Town.guest.playing(guestMissionNo) || !bytes.Equal(h.Town.guest.stash, g.Town.guest.stash) {
		t.Fatal("the cold LOAD lost the waiting party")
	}
	if _, _, err := h.LiveCompleteCampaign(); err != nil {
		t.Fatal(err)
	}
	if got := guestDigest(t, h); got != want {
		t.Fatalf("the town after a LOAD inside the guest mission differs:\n got %s\nwant %s", got, want)
	}
	reloaded, ok := guestMember(h)
	if !ok || guestMemberJSON(t, reloaded) != guestMemberJSON(t, joined) {
		t.Fatalf("the guest joined differently after the LOAD")
	}

	// Loss controls: a changed stash fails the digest; a LOAD without the mod,
	// or of a SAV whose guest record is gone, refuses.
	k := guestFront(t, dir, guestModID)
	kapp, _ := guestRoute(t, k, []int{0, 0, 0})
	m := guestFront(t, dir, guestModID)
	if _, _, err := m.RestoreOriginal(k.Town.guest.stash); err != nil {
		t.Fatal(err)
	}
	m.Town.gold++
	s, label, err := m.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	if k.Town.guest.stash, err = m.ExportCurrentSave(s, label); err != nil {
		t.Fatal(err)
	}
	guestPlay(t, k, kapp)
	if _, _, err := k.LiveCompleteCampaign(); err != nil {
		t.Fatal(err)
	}
	if guestDigest(t, k) == want {
		t.Fatal("the digest did not see a changed stash")
	}
	if _, _, err := guestFront(t, dir).RestoreOriginal(saved); err == nil || !strings.Contains(err.Error(), guestModID) {
		t.Fatalf("a LOAD without the mod: %v", err)
	}
	stripped, err := withoutGuestRecord(saved)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := guestFront(t, dir, guestModID).RestoreOriginal(stripped); err == nil || !strings.Contains(err.Error(), "mission 25") {
		t.Fatalf("a LOAD of the guest mission without its guest record: %v", err)
	}
}

// TestReleaseModMissionReadsTheMapFromTheMod proves the guest's mission is
// built from the mod's copy of the map: a changed altitude byte in that copy
// reaches the built map, and the install's mission 10 map keeps its own.
func TestReleaseModMissionReadsTheMapFromTheMod(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	dir := guestMods(t, guestModID)
	lang := modrt.LanguageFor(BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS"))))
	path := filepath.Join(dir, guestModID, "mission", lang, "map.alm")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m, err := alm.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	plane := m.Altitudes
	at := bytes.Index(raw, plane)
	if at < 0 || bytes.Index(raw[at+1:], plane) >= 0 {
		t.Fatal("the altitude plane is not found once in the map")
	}
	raw[at] ^= 0x01
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	f := guestFront(t, dir, guestModID)
	app, next := guestRoute(t, f, []int{0, 0, 0})
	if next != guestMissionNo {
		t.Fatalf("mission 20's win answered %d", next)
	}
	if err := app.OpenMission(f.MissionOpener(guestMissionNo)); err != nil {
		t.Fatal(err)
	}
	built := f.live.mission.state
	if built.Address != "scenario/25.alm" || built.Map.Altitudes[0] != plane[0]^0x01 {
		t.Fatalf("the guest mission was built from %s with altitude %d", built.Address, built.Map.Altitudes[0])
	}
	installed, err := f.Archives.Containers.ReadFile("scenario/10.alm")
	if err != nil {
		t.Fatal(err)
	}
	if ten, err := alm.Open(installed); err != nil || ten.Altitudes[0] != plane[0] {
		t.Fatalf("the install's mission 10 map changed: %v", err)
	}
}

// TestReleaseModMissionGuestWaitsForAHeldGrant proves the guest meets a join
// condition of another mod unchanged: with a female hero, the counterpart the
// town grants as companion 22 plays the mission; after her win she waits in
// the town as she ended it until her tavern conversation opens, a town SAVE
// and cold LOAD keep her waiting, and the talk joins her in that state.
func TestReleaseModMissionGuestWaitsForAHeldGrant(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	dir := guestMods(t, guestModID, modJoinID)
	female := []int{1, 0, 0}

	plain := guestFront(t, dir)
	guestRoute(t, plain, female)
	row, ok := guestMember(plain)
	if !ok {
		t.Fatalf("the unmodded town granted %v", modJoinRoster(plain))
	}

	f := guestFront(t, dir, guestModID, modJoinID)
	app, next := guestRoute(t, f, female)
	if next != guestMissionNo || f.Town.guest.start.Name != row.Name || f.Town.guest.start.FigureDir != row.FigureDir {
		t.Fatalf("mission 20's win answered %d with guest %q, the town grants %q", next, f.Town.guest.start.Name, row.Name)
	}
	gained := guestPlay(t, f, app)
	if _, _, err := f.LiveCompleteCampaign(); err != nil {
		t.Fatal(err)
	}
	if _, has := guestMember(f); has || f.Town.guest == nil || f.Town.guest.member == nil || f.Town.guest.member.Carry.SkillXP != gained {
		t.Fatalf("after the guest's win under a held grant: roster %v", modJoinRoster(f))
	}
	waiting := *f.Town.guest.member

	// The waiting guest crosses a town SAVE and a cold LOAD.
	raw := currentTownSave(t, f)
	if got := modJoinSavedAddHero(t, raw); len(got) != 1 || got[0] != guestCompanion {
		t.Fatalf("the town SAV holds AddHero %v, want the held grant", got)
	}
	if _, err := modItemReload(t, raw, true); err == nil {
		t.Fatal("a town SAV with a waiting guest loaded without the mods")
	}
	store := SaveStore{Dir: t.TempDir()}
	if err := os.WriteFile(filepath.Join(store.Dir, "town.sav"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	g := guestFront(t, dir, guestModID, modJoinID)
	tapp, s := releaseTavernApp(t, g, store)
	if g.Town.guest == nil || g.Town.guest.member == nil || guestMemberJSON(t, *g.Town.guest.member) != guestMemberJSON(t, waiting) {
		t.Fatal("the cold LOAD lost the waiting guest")
	}
	if _, has := guestMember(g); has {
		t.Fatal("the guest joined on the cold LOAD before her conversation")
	}
	modJoinHearCompanion(t, tapp, s, "")
	joined, ok := guestMember(g)
	if !ok || g.Town.guest != nil || joined.StartingHero || joined.Carry == nil || joined.Carry.SkillXP != gained {
		t.Fatalf("after the talk: roster %v", modJoinRoster(g))
	}
	waiting.StartingHero = false
	if guestMemberJSON(t, joined) != guestMemberJSON(t, waiting) {
		t.Fatal("the guest joined in another state than she waited in")
	}
}

// withoutGuestRecord is saved with the guest record removed from its supplement.
func withoutGuestRecord(saved []byte) ([]byte, error) {
	doc, err := sav.DecodeDocumentData(saved)
	if err != nil {
		return nil, err
	}
	a, err := readCurrentActions(&doc)
	if err != nil {
		return nil, err
	}
	if a == nil || a.Session == nil {
		return nil, errors.New("the SAV has no session record")
	}
	a.Session.Guest = nil
	b, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	if err := sav.SetNativeActions(&doc.State, b); err != nil {
		return nil, err
	}
	return sav.EncodeDocumentData(doc)
}
