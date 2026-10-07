package game

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/render/frame"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const abandonModDir = "../modrt/testdata/mods"

// abandonFront is a front end on the lawful install under the mods of dir named
// by ids, with their screens and actions registered on a fresh App.
func abandonFront(t *testing.T, dir string, ids ...string) (*FrontEnd, *ui.App, mod.ScreenData) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	var screens mod.ScreenData
	if len(ids) > 0 {
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
		if err := f.SetModItems(res.Items); err != nil {
			t.Fatal(err)
		}
		screens = res.Screens
	}
	app := f.App("mission leave")
	app.Layout(1024, 768)
	if err := app.SetModScreens(ModScreens(screens)); err != nil {
		t.Fatal(err)
	}
	return f, app, screens
}

// abandonTown is the player's route to the town square of chapter 30 under the
// mods: a town SAV made after mission 20, loaded through the load window. The
// tavern, shop and school offers are taken so mission 30 is offered.
func abandonTown(t *testing.T, dir string, ids ...string) (*FrontEnd, *ui.App, mod.ScreenData, *townScreen) {
	t.Helper()
	f, a, screens := abandonFront(t, dir, ids...)
	store := SaveStore{Dir: t.TempDir()}
	n := 0
	save, list, load := f.SaveSeams(store, OriginalStore{}, func() time.Time {
		n++
		return time.Date(2001, 1, 1, 0, 0, n, 0, time.UTC)
	})
	a.SetSaveSeams(save, list, load)
	f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Mission leave", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	f.FinishMission(20, f.Carried, nil, nil)
	for _, building := range []TownBuilding{TownTavern, TownShop, TownSchool} {
		for _, offer := range f.Town.Offers(building) {
			if offer.Mission > 0 {
				if _, ok := f.Town.Take(building, offer.Index); !ok {
					t.Fatalf("could not accept installed offer %+v", offer)
				}
			}
		}
	}
	if !containsMission(f.Town.Available(), 30) {
		t.Fatalf("the installed chapter did not offer mission 30: %v", f.Town.Available())
	}
	if _, err := save(false); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"load", "enter"} {
		if err := a.HeadlessKey(k); err != nil {
			t.Fatalf("key %q: %v", k, err)
		}
	}
	s := f.TownScreen().(*townScreen)
	if a.Screen() != ui.ScreenTown || !s.AtTownSquare() {
		t.Fatalf("the town load left screen %s, square %v", a.Screen(), s.AtTownSquare())
	}
	return f, a, screens, s
}

// abandonEnter takes the production route from the town square to mission n:
// the Gates, the mission's scroll, the party's travel and the map screen.
func abandonEnter(t *testing.T, f *FrontEnd, a *ui.App, s *townScreen, n int) {
	t.Helper()
	if err := a.HeadlessActivate("GATES"); err != nil {
		t.Fatal(err)
	}
	x, y, err := a.HeadlessWorldMapMissionPoint(n)
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"press", "release"} {
		if err := a.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 10000 && a.Screen() == ui.ScreenTown; i++ {
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if a.Screen() != ui.ScreenMap || f.live == nil || f.live.mission.number != n {
		t.Fatalf("the scroll of mission %d did not open it: screen %s", n, a.Screen())
	}
}

func abandonSteps(t *testing.T, a *ui.App, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
}

// abandonTownState is what a town holds that a mission left must not change.
type abandonTownState struct {
	party     []mapload.PartyMember
	gold      int
	finished  int
	available []int
	done30    bool
}

func captureAbandonTown(f *FrontEnd) abandonTownState {
	return abandonTownState{
		party: mapload.CloneParty(f.Carried), gold: f.Town.Gold(), finished: f.Town.finishedCount(),
		available: slices.Clone(f.Town.Available()), done30: f.Town.Done(30),
	}
}

func requireAbandonTown(t *testing.T, what string, f *FrontEnd, want abandonTownState) {
	t.Helper()
	got := captureAbandonTown(f)
	if got.gold != want.gold || got.finished != want.finished || got.done30 || !reflect.DeepEqual(got.available, want.available) {
		t.Fatalf("%s: gold %d/%d finished %d/%d available %v/%v done30 %v", what, got.gold, want.gold, got.finished, want.finished, got.available, want.available, got.done30)
	}
	if !reflect.DeepEqual(got.party, want.party) {
		t.Fatalf("%s: the party differs from the party that entered the mission", what)
	}
}

// abandonLabel is a mod entry's text as the install's font draws it.
func abandonLabel(f *FrontEnd, s string) string {
	return EncodeInstallText(s, f.Font.Value().Selector)
}

func writeAbandonShot(t *testing.T, name string, panel *image.RGBA) {
	t.Helper()
	shot := image.NewRGBA(image.Rect(0, 0, frame.W, frame.H))
	draw.Draw(shot, shot.Bounds(), &image.Uniform{C: color.RGBA{A: 0xff}}, image.Point{}, draw.Src)
	draw.Draw(shot, shot.Bounds(), panel, image.Point{}, draw.Over)
	writeModShot(t, name, shot)
}

// A party that enters mission 30 from the town square, plays on, and abandons
// through the in-game menu returns to the town as it entered: the same party,
// purse, offers and finished count, with mission 30 still offered and no
// completion paid. The town SAV written after the party is home is read back by
// a cold LOAD under the mod and refused without it; the mission is entered again.
func TestReleaseModAbandonReturnsToTheTownAsEntered(t *testing.T) {
	f, a, screens, s := abandonTown(t, abandonModDir, "mission-abandon")
	screen := screens.Screens[0]
	if screen.Kind != mod.ActionAbandon {
		t.Fatalf("the example mod declares %+v", screen)
	}
	entry := captureAbandonTown(f)
	entrySav := currentTownSave(t, f)
	abandonEnter(t, f, a, s, 30)
	driver := f.live
	if driver.world.Purse(sim.SelfSlot) != uint32(entry.gold) {
		t.Fatalf("the mission purse is %d, the town gold %d", driver.world.Purse(sim.SelfSlot), entry.gold)
	}

	// The mission is played on and made richer, so leaving it has something to
	// lose: the purse is carried out only by a completion.
	abandonSteps(t, a, 60)
	driver.world.SetPurse(sim.SelfSlot, uint32(entry.gold)+777)
	entryQuick := f.quickSpells
	f.quickSpells[1] ^= 0x33
	if err := a.HeadlessKey("escape"); err != nil || a.Screen() != ui.ScreenGameMenu {
		t.Fatalf("Escape in the mission: %v on %s", err, a.Screen())
	}
	label, confirm := abandonLabel(f, screen.MenuLabel), abandonLabel(f, screen.Title)
	rows := a.HeadlessRows()
	if len(rows) != 8 || rows[7].Text != label || !rows[7].Choosable {
		t.Fatalf("the menu rows over the mission: %d, last %+v, want 8 ending %q", len(rows), rows[len(rows)-1], label)
	}
	panel := a.GameMenuPanel()
	requireTemplate(t, "the entry label", panel, modTemplate(t, f, screen.MenuLabel), image.Rect(100, 300, 440, 345))
	writeAbandonShot(t, "game-menu-with-abandon", panel)

	// The entry asks first; Return gives the mission back with nothing asked.
	if err := a.HeadlessActivate(label); err != nil {
		t.Fatal(err)
	}
	asked := a.HeadlessRows()
	if a.Screen() != ui.ScreenGameMenu || len(asked) != 2 || asked[0].Text != confirm || !asked[0].Choosable {
		t.Fatalf("the confirming page is %+v on %s", asked, a.Screen())
	}
	askPanel := a.GameMenuPanel()
	requireTemplate(t, "the confirming row", askPanel, modTemplate(t, f, screen.Title), image.Rect(100, 100, 440, 200))
	writeAbandonShot(t, "abandon-confirm", askPanel)
	if err := a.HeadlessActivate(asked[1].Text); err != nil || a.Screen() != ui.ScreenGameMenu || len(a.HeadlessRows()) != 8 {
		t.Fatalf("Return from the confirming page: %v, %d rows on %s", err, len(a.HeadlessRows()), a.Screen())
	}
	if f.live != driver || a.Screen() != ui.ScreenGameMenu {
		t.Fatal("asking and returning changed the mission")
	}
	if err := a.HeadlessActivate(label); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessActivate(confirm); err != nil {
		t.Fatal(err)
	}

	// The party is on the road home: no completion was paid, the mission is
	// offered, and the purse the mission gained is gone.
	view := s.WorldMapView()
	if a.Screen() != ui.ScreenTown || !s.AtWorldMap() || !view.Returning || !view.HideScrolls {
		t.Fatalf("after abandoning: screen %s gates %v view %+v", a.Screen(), s.AtWorldMap(), view)
	}
	requireAbandonTown(t, "on the road home", f, entry)
	if f.quickSpells != entryQuick {
		t.Fatalf("the quick-spell bindings are %v, the mission was entered with %v", f.quickSpells, entryQuick)
	}
	if f.live != driver {
		t.Fatal("the front end replaced its driver on abandoning")
	}
	for i := 0; i < 10000 && !s.AtTownSquare(); i++ {
		if a.Screen() != ui.ScreenTown || f.live != driver {
			t.Fatal("homeward travel reopened a mission")
		}
		abandonSteps(t, a, 1)
	}
	if !s.AtTownSquare() {
		t.Fatal("the party never reached the town square")
	}
	requireAbandonTown(t, "at the town square", f, entry)
	if pix, _, err := a.HeadlessFrame(); err != nil {
		t.Fatal(err)
	} else {
		writeModShot(t, "town-after-abandon", pix)
	}

	// SAVE and cold LOAD. The leaf names the mod set; the town it holds is the
	// town that entered the mission.
	raw := currentTownSave(t, f)
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil || doc.Head.Mission != 0 || doc.World != nil || !bytes.Contains(raw, modMarkName) {
		t.Fatalf("the town SAV: mission %d, world %v, mark %v, err %v", doc.Head.Mission, doc.World != nil, bytes.Contains(raw, modMarkName), err)
	}
	entryDoc, err := sav.DecodeDocumentData(entrySav)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Objects) != len(entryDoc.Objects) {
		t.Fatalf("the town SAV after abandoning holds %d objects, the one before the mission %d", len(doc.Objects), len(entryDoc.Objects))
	}
	if !bytes.Equal(entrySav, raw) {
		t.Fatalf("the town SAV after abandoning (%d bytes) differs from the one written before the mission (%d bytes); objects equal %v",
			len(raw), len(entrySav), reflect.DeepEqual(doc.Objects, entryDoc.Objects))
	}
	g := abandonReload(t, raw, false)
	requireAbandonTown(t, "after the cold LOAD", g, entry)
	if _, err := abandonReloadErr(t, raw, true); err == nil || !errors.Is(err, ErrModMark) {
		t.Fatalf("a LOAD without the mod: %v", err)
	}

	// Loss control: a completion of the same mission carries the purse out, so
	// the equality above is the abandon's doing and not a town that never moves.
	h, ha, _, hs := abandonTown(t, abandonModDir, "mission-abandon")
	abandonEnter(t, h, ha, hs, 30)
	h.live.world.SetPurse(sim.SelfSlot, uint32(entry.gold)+777)
	if _, _, err := h.LiveCompleteCampaign(); err != nil {
		t.Fatal(err)
	}
	if h.Town.Gold() < entry.gold+777 || !h.Town.Done(30) {
		t.Fatalf("a completion left gold %d and done %v", h.Town.Gold(), h.Town.Done(30))
	}

	// The mission is taken again from the same town: its scroll is on the map
	// and the production route opens it.
	abandonEnter(t, f, a, s, 30)
	if f.live == driver || f.live.mission.number != 30 {
		t.Fatalf("mission 30 was not entered again: same driver %v", f.live == driver)
	}
	if got := f.live.world.Purse(sim.SelfSlot); got != uint32(entry.gold) {
		t.Fatalf("the second entry starts with purse %d, want %d", got, entry.gold)
	}
}

func abandonReloadErr(t *testing.T, raw []byte, plain bool) (*FrontEnd, error) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "town.sav"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	var g *FrontEnd
	if plain {
		g = releaseFront(t)
	} else {
		g, _, _ = abandonFront(t, abandonModDir, "mission-abandon")
	}
	_, _, load := g.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: dir}, nil)
	_, town, err := load("town.sav")
	if err == nil && !town {
		t.Fatal("the town SAV did not open a town")
	}
	return g, err
}

func abandonReload(t *testing.T, raw []byte, plain bool) *FrontEnd {
	t.Helper()
	g, err := abandonReloadErr(t, raw, plain)
	if err != nil {
		t.Fatalf("cold LOAD: %v", err)
	}
	return g
}

// Without the mod the menu has the shipped seven rows and none of the entry's
// words; with the mod over a mission that was not entered from a town the menu
// is the same seven rows, frame for frame.
func TestReleaseUnmoddedMissionMenuHasNoAbandonEntry(t *testing.T) {
	var panels [2]*image.RGBA
	for i, withMod := range []bool{false, true} {
		var f *FrontEnd
		var a *ui.App
		var screens mod.ScreenData
		if withMod {
			f, a, screens = abandonFront(t, abandonModDir, "mission-abandon")
		} else {
			f, a, screens = abandonFront(t, abandonModDir)
		}
		party := f.ChargenParty(ui.ChargenResult{Name: "No leave", Choices: []int{1, 0, 0}, Stats: []int{31, 27, 24, 29}})
		if err := a.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
			t.Fatalf("open mission 10: %v", err)
		}
		if err := a.HeadlessKey("escape"); err != nil || a.Screen() != ui.ScreenGameMenu {
			t.Fatalf("Escape in the mission: %v on %s", err, a.Screen())
		}
		if rows := a.HeadlessRows(); len(rows) != 7 {
			t.Fatalf("mod %v: the menu has %d rows over a mission that has no town, want 7", withMod, len(rows))
		}
		panels[i] = a.GameMenuPanel()
		if withMod {
			if len(screens.Screens) != 1 {
				t.Fatalf("the mod declares %+v", screens.Screens)
			}
			requireNoTemplate(t, "the entry", panels[i], modTemplate(t, f, screens.Screens[0].MenuLabel), image.Rect(100, 60, 440, 420))
		}
	}
	if !bytes.Equal(panels[0].Pix, panels[1].Pix) {
		box, _ := diffBounds(panels[0], panels[1])
		t.Fatalf("the modded menu over a mission with no town differs from the shipped one inside %v", box)
	}
}

// writeLeaveMod writes a mod folder that declares both actions.
func writeLeaveMod(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	root := filepath.Join(dir, "mission-leave")
	files := map[string]string{
		"mod.toml": "id = \"mission-leave\"\ntitle = \"Mission leave\"\nversion = \"1.0.0\"\nauthor = \"Againrom\"\n" +
			"description = \"Abandon and restart.\"\napi = 1\napplies-to = [\"rom1\"]\nentry = \"main.star\"\n",
		"main.star":            "def init(game, settings):\n    game.data.add(\"data/screens.toml\")\n",
		"data/screens.toml":    "[[action]]\nkey = \"abandon\"\naction = \"abandon\"\nmenu = \"abandon.menu\"\n\n[[action]]\nkey = \"restart\"\naction = \"restart\"\nmenu = \"restart.menu\"\nconfirm = \"restart.confirm\"\n",
		"text/en/strings.toml": "\"abandon.menu\" = \"Abandon mission\"\n\"restart.menu\" = \"Restart mission\"\n\"restart.confirm\" = \"Restart it\"\n",
	}
	for name, body := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// Restart reopens the mission from its entry point with the party and purse it
// was entered with: the mission world is the world of the first entry again, and
// what the first run changed is gone.
func TestReleaseModRestartReopensTheMissionFromItsEntry(t *testing.T) {
	dir := writeLeaveMod(t)
	f, a, screens, s := abandonTown(t, dir, "mission-leave")
	if len(screens.Screens) != 2 || screens.Screens[1].Kind != mod.ActionRestart {
		t.Fatalf("the mod declares %+v", screens.Screens)
	}
	entry := captureAbandonTown(f)
	abandonEnter(t, f, a, s, 30)
	first := f.live
	entryHash, entryTick := first.world.Hash(), first.world.Tick()
	abandonSteps(t, a, 80)
	first.world.SetPurse(sim.SelfSlot, uint32(entry.gold)+500)
	if first.world.Hash() == entryHash {
		t.Fatal("the mission did not change, so a restart proves nothing")
	}
	if err := a.HeadlessKey("escape"); err != nil || a.Screen() != ui.ScreenGameMenu {
		t.Fatalf("Escape in the mission: %v on %s", err, a.Screen())
	}
	rows := a.HeadlessRows()
	restart := abandonLabel(f, screens.Screens[1].MenuLabel)
	if len(rows) != 9 || rows[8].Text != restart {
		t.Fatalf("the menu rows: %d, last %+v", len(rows), rows[len(rows)-1])
	}
	writeAbandonShot(t, "game-menu-with-restart", a.GameMenuPanel())
	if err := a.HeadlessActivate(restart); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessActivate(abandonLabel(f, screens.Screens[1].Title)); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ui.ScreenMap || f.live == first || f.live.mission.number != 30 {
		t.Fatalf("restart left screen %s, same driver %v", a.Screen(), f.live == first)
	}
	if f.live.world.Tick() != entryTick || f.live.world.Hash() != entryHash {
		t.Fatalf("the restarted mission is at tick %d hash %x, the first entry was tick %d hash %x",
			f.live.world.Tick(), f.live.world.Hash(), entryTick, entryHash)
	}
	if got := f.live.world.Purse(sim.SelfSlot); got != uint32(entry.gold) {
		t.Fatalf("the restarted mission has purse %d, want %d", got, entry.gold)
	}
	requireAbandonTown(t, "after restart", f, entry)
	// The restarted mission is a mission the menu can leave again.
	if err := a.HeadlessKey("escape"); err != nil || len(a.HeadlessRows()) != 9 {
		t.Fatalf("the restarted mission's menu: %v, %d rows", err, len(a.HeadlessRows()))
	}
	if err := a.HeadlessActivate(abandonLabel(f, screens.Screens[0].MenuLabel)); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessActivate(abandonLabel(f, screens.Screens[0].Title)); err != nil || a.Screen() != ui.ScreenTown || !s.AtWorldMap() {
		t.Fatalf("abandoning the restarted mission: %v on %s", err, a.Screen())
	}
	requireAbandonTown(t, "after restart and abandon", f, entry)
}
