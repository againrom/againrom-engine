package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func saveDialogMission(t *testing.T, f *FrontEnd, app *ui.App, out string) {
	t.Helper()
	f.SetDeterministicFrames(true)
	var hero sim.EntityID
	found := false
	for i, member := range f.live.mission.party {
		if member.StartingHero {
			hero, found = f.live.mission.ids[i], true
		}
	}
	if !found {
		t.Fatal("mission lacks its actual starting hero")
	}
	f.LiveDamage(uint32(hero), 7)
	actor, _ := f.live.entity(hero)
	f.LiveOrder(uint32(hero), int(actor.X)+3, int(actor.Y))
	f.LiveAdvance(1)
	actor, found = f.live.entity(hero)
	if !found || !actor.Alive() || actor.HP >= actor.MaxHP {
		t.Fatal("pool witness needs an injured live hero")
	}
	f.ConfigureSaveSeams(app, SaveStore{Dir: out}, OriginalStore{}, nil)
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	before, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil || app.Screen() != ui.ScreenSave {
		t.Fatal("ordinary save dialog", app.Screen(), err)
	}
	state, ok := app.HeadlessSaveState()
	if !ok || state.Request.Format != ui.SaveSAV || !state.Request.OnMap {
		t.Fatal("mission dialog did not default to SAV", state)
	}
	if err := app.HeadlessSaveEdit(out, "Current expedition", ""); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveAction("save"); err != nil {
		t.Fatal("current mission SAV", err)
	}
	if msg := app.HeadlessMessage(); strings.Contains(msg, "DIV-1334") || strings.Contains(msg, "last visited") {
		t.Fatal("mission SAVE changed the save point", msg)
	}
	after, _, err := f.Snapshot(true)
	slices.Sort(before.Residue.Commanded)
	slices.Sort(after.Residue.Commanded)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("SAVE changed live mission, campaign, items or application state", err)
	}
	if _, err := os.Stat(filepath.Join(out, "Current expedition.ags")); !os.IsNotExist(err) {
		t.Fatal("dialog produced AGS", err)
	}
	path := filepath.Join(out, "Current expedition.sav")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	saveDialogCurrentMission(t, f, path, raw, hero)
}

func saveDialogCurrentMission(t *testing.T, f *FrontEnd, path string, raw []byte, hero sim.EntityID) {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil || doc.World == nil || doc.Head.Mission != uint32(f.liveMission) {
		t.Fatal("dialog SAV did not retain the current mission", err)
	}
	actor, found := f.live.entity(hero)
	if !found {
		t.Fatal("current mission hero disappeared")
	}
	cold := releaseFront(t)
	cold.Options = OptionsStore{}
	cold.SetDeterministicFrames(true)
	open, town, err := cold.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal("dialog cold mission LOAD", town, err)
	}
	app := cold.App("cold dialog continuation")
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if cold.liveMission != f.liveMission {
		t.Fatal("cold SAVE changed mission")
	}
	assertCurrentWorldEqual(t, f.live.world, cold.live.world, "dialog cold LOAD")
	p := currentActionProof{Runtime: actor.SourceBinding.RuntimeID, Actor: actor.ID, Mode: "dialog"}
	for step := 0; step <= 64; step++ {
		if p.Runtime != 0 {
			p.Samples = append(p.Samples, currentActionSample(t, f, p.Runtime))
		}
		assertCurrentWorldEqual(t, f.live.world, cold.live.world, fmt.Sprintf("dialog continuation %d", step))
		if step < 64 {
			f.live.tick()
			cold.live.tick()
		}
	}
	if p.Runtime != 0 {
		proof, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path+".json", proof, 0600); err != nil {
			t.Fatal(err)
		}
		runSpellWitnessChild(t, path, "AGAINROM_CURRENT_ACTION_INPUT")
	}
	cold.ConfigureSaveSeams(app, SaveStore{Dir: filepath.Dir(path)}, OriginalStore{}, nil)
	if err := app.HeadlessKey("f2"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenSave {
		// A shipped script notice owns this boundary in the direct imported
		// mission. The fresh-process continuation above is its witness; the
		// other cases exercise the second interactive SAVE cycle.
		return
	}
	if err := app.HeadlessSaveEdit("", "Second expedition", ""); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveAction("save"); err != nil {
		t.Fatal("second interactive SAVE", err)
	}
	second := loadAreaContinuation(t, filepath.Join(filepath.Dir(path), "Second expedition.sav"))
	assertCurrentWorldEqual(t, cold.live.world, second.live.world, "second dialog cold LOAD")
	actor, _ = cold.live.entity(hero)
	cold.LiveOrder(uint32(hero), int(actor.X), int(actor.Y)+2)
	second.LiveOrder(uint32(hero), int(actor.X), int(actor.Y)+2)
	for step := 0; step < 3; step++ {
		cold.LiveAdvance(1)
		second.LiveAdvance(1)
		assertCurrentWorldEqual(t, cold.live.world, second.live.world, "second dialog next move")
	}
}

func TestReleaseSaveDialog1173MissionCity(t *testing.T) {
	if path := os.Getenv("AGAINROM_CURRENT_ACTION_INPUT"); path != "" {
		currentActionCold(t, path)
		return
	}
	t.Run("original_mission_return", saveOriginalMissionReturn)
	t.Run("pre_town_battlefield", saveDialogPreTownBattlefield)
	t.Run("native", func(t *testing.T) {
		f := releaseFront(t)
		f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Native explorer", Choices: []int{0, 0, 0}, Stats: []int{30, 30, 30, 30}})
		f.arriveInTown()
		f.addChapterCompanions(f.Town.Chapter())
		app := f.App("1173 native")
		if err := app.OpenMission(f.MissionOpenerWith(f.Town.Chapter(), f.NextParty())); err != nil {
			t.Fatal(err)
		}
		saveDialogMission(t, f, app, t.TempDir())
	})
	t.Run("imported_city_loot", func(t *testing.T) {
		f, _ := townReturnImported1168(t)
		app := f.App("1173 imported")
		if err := app.OpenMission(f.MissionOpenerWith(30, f.NextParty())); err != nil {
			t.Fatal(err)
		}
		hero, _ := returnIDs1169(t, f)
		if err := f.live.world.HeadlessPlace(hero, 32, 31); err != nil {
			t.Fatal(err)
		}
		liveTakeAt(t, f.live, hero, 32, 31)
		saveDialogMission(t, f, app, t.TempDir())
	})
	t.Run("direct_original_mission", func(t *testing.T) {
		f := releaseFront(t)
		_, raw := groundCorpusFile(t, "2026-08-15/game0016.sav", "5e67d1282398076498867ac0124046d5c0e7f1acd2d0ffc1e66888c5296e0345")
		open, town, err := f.RestoreOriginal(raw)
		if err != nil || town {
			t.Fatal(town, err)
		}
		app := f.App("1173 original World")
		if err := app.OpenMission(open); err != nil {
			t.Fatal(err)
		}
		for i, member := range f.live.mission.party {
			if !member.StartingHero {
				continue
			}
			id := f.live.mission.ids[i]
			actor, ok := f.live.entity(id)
			if !ok {
				t.Fatal("starting hero is absent")
			}
			f.live.pending = append(f.live.pending, sim.DropCarried(id, 0, sim.CellPoint{X: actor.X, Y: actor.Y}))
			f.live.tick()
			liveTakeAt(t, f.live, id, actor.X, actor.Y)
			picked, _ := f.live.entity(id)
			if picked.SourceBinding.GroupIndex == 0 || picked.SourceBinding.GroupSelector != picked.CommandGroup {
				t.Fatalf("pickup left the source-bound selector at %d, command Group at %d", picked.SourceBinding.GroupSelector, picked.CommandGroup)
			}
		}
		saveDialogMission(t, f, app, t.TempDir())
	})
	if os.Getenv("AGAINROM_PLAYED_SAVE1173") != "" {
	}
}

func saveOriginalMissionReturn(t *testing.T) {
	for _, tc := range []struct{ name, path, hash string }{
		{"side_mission", "2026-08-15/game0016.sav", "5e67d1282398076498867ac0124046d5c0e7f1acd2d0ffc1e66888c5296e0345"},
		{"mission111", "2026-09-09/game0076.sav", "ca6f2980986859fb19c7b602a00b92b0e3ae95b1d00adc6c757152d596ed556c"},
	} {
		t.Run(tc.name, func(t *testing.T) { saveOriginalMissionReturnFile(t, tc.path, tc.hash) })
	}
}

func saveOriginalMissionReturnFile(t *testing.T, path, hash string) {
	f := releaseFront(t)
	_, raw := groundCorpusFile(t, path, hash)
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal(town, err)
	}
	a := f.App("original mission return")
	if err := a.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	n, line, err := f.LiveCompleteCampaign()
	if err != nil || n != 0 || !f.Town.Open() {
		t.Fatal("mission return", n, line, err)
	}
	s, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _, err := f.playerCitySave(s, "Returned from original mission")
	if err != nil {
		t.Fatal("ordinary SAV after returning from original mission", err)
	}
	g := releaseFront(t)
	if _, town, err := g.RestoreOriginal(encoded); err != nil || !town {
		t.Fatal("cold SAV", town, err)
	}
	if g.Town.Gold() != f.Town.Gold() || g.Town.Chapter() != f.Town.Chapter() || len(g.Carried) != len(f.Carried) {
		t.Fatal("city round trip changed campaign or roster")
	}
	for i, want := range f.Carried {
		got := g.Carried[i]
		if got.ID != want.ID || got.Hero != want.Hero || got.KnownSpells != want.KnownSpells || got.Book != want.Book {
			t.Fatalf("current character/book changed for %s", want.ID)
		}
		// SAV stores the loaded speed word. The native unladen speed is an
		// import projection; all source operands and the current raw speed
		// must survive without being recomputed by the writer.
		if got.Carry.LiveLoad.Inventory != want.Carry.LiveLoad.Inventory ||
			got.Carry.LiveLoad.Movement.RawSpeed != want.Carry.LiveLoad.Movement.RawSpeed ||
			!reflect.DeepEqual(returnItems(mapload.MemberCarriedItems(got, nil)), returnItems(mapload.MemberCarriedItems(want, nil))) {
			t.Fatalf("current Human or pack changed for %s", want.ID)
		}
	}
	ags, err := EncodeSave(s, "Returned AGS")
	if err != nil {
		t.Fatal(err)
	}
	cold, _, err := DecodeSave(ags)
	if err != nil {
		t.Fatal(err)
	}
	h := releaseFront(t)
	if _, town, err := h.Restore(cold); err != nil || !town {
		t.Fatal("cold AGS", town, err)
	}
	if !reflect.DeepEqual(h.Carried, f.Carried) {
		t.Fatal("AGS changed the complete carried party")
	}
	next, _, err := h.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := h.playerCitySave(next, "Cold AGS to SAV"); err != nil {
		t.Fatal("town AGS lost SAV continuity", err)
	}
}

func TestReleaseSaveDialog1173TownPairOverwriteAndRefusal(t *testing.T) {
	t.Run("selected_repeated_suffix", saveDialogRepeatedSuffix)
	f := releaseFront(t)
	f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Town traveler", Choices: []int{0, 0, 0}, Stats: []int{30, 30, 30, 30}})
	f.arriveInTown()
	f.addChapterCompanions(f.Town.Chapter())
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	if _, err := save(false); err != nil {
		t.Fatal(err)
	}
	app := f.App("1173 town")
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	if err := headlessOpenLoad(app); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("@first"); err != nil || app.Screen() != ui.ScreenTown {
		t.Fatal("ordinary LOAD to town", err)
	}
	dir := filepath.Join(store.Dir, "Chosen folder")
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveEdit(dir, "My town", ui.SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveAction("save"); err != nil {
		t.Fatal(err)
	}
	paths := []string{filepath.Join(dir, "My town.sav")}
	old := make([][]byte, len(paths))
	for i, path := range paths {
		var err error
		old[i], err = os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
	}
	// LOAD immediately lists the chosen directory, with the current SAV row.
	if err := headlessOpenLoad(app); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("My town"); err != nil || app.Screen() != ui.ScreenTown {
		t.Fatal("LOAD did not follow chosen directory", app.Screen(), err)
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveEdit(dir, "My town", ui.SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveAction("save"); err != nil {
		t.Fatal(err)
	}
	state, ok := app.HeadlessSaveState()
	if !ok || !state.Confirmation || !reflect.DeepEqual(state.Paths, paths) || !reflect.DeepEqual(state.Existing, paths) {
		t.Fatal("overwrite did not confirm exact target", state)
	}
	if err := app.HeadlessSaveAction("back"); err != nil {
		t.Fatal(err)
	}
	for i, path := range paths {
		current, _ := os.ReadFile(path)
		if !bytes.Equal(current, old[i]) {
			t.Fatal("cancel replaced file", path)
		}
	}
	f.Town.gold += 19
	if err := app.HeadlessSaveAction("save"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveAction("overwrite"); err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		g := releaseFront(t)
		if IsOriginal(path) {
			if _, town, err := g.RestoreOriginal(raw); err != nil || !town {
				t.Fatal("overwritten SAV cold LOAD", err)
			}
		} else {
			s, _, err := DecodeSave(raw)
			if err != nil {
				t.Fatal(err)
			}
			if _, town, err := g.Restore(s); err != nil || !town {
				t.Fatal("overwritten AGS cold LOAD", err)
			}
		}
		if g.Town.Gold() != f.Town.Gold() {
			t.Fatal("SAVE changed current purse")
		}
	}
	gold := f.Town.gold
	f.Town.gold = -1
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveEdit(dir, "Malformed purse", ui.SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveAction("save"); err == nil {
		t.Fatal("invalid current purse was saved")
	}
	for _, ext := range []string{".ags", ".sav"} {
		if _, err := os.Stat(filepath.Join(dir, "Malformed purse"+ext)); !os.IsNotExist(err) {
			t.Fatal("failed preparation published output", err)
		}
	}
	if f.Town.gold != -1 {
		t.Fatal("failed preparation changed current purse")
	}
	f.Town.gold = gold
	if err := app.HeadlessSaveAction("save"); err != nil {
		t.Fatal("valid current SAV retry", err)
	}
	unicodeName := "\u0421\u043e\u0445\u0440\u0430\u043d\u0435\u043d\u0438\u0435"
	unicodeDir := filepath.Join(dir, "\u041f\u0430\u043f\u043a\u0430")
	// A name Windows-1251 cannot represent at all still refuses outright and
	// publishes nothing partial: Arabic has no Windows-1251 byte at any
	// selector.
	unrepresentable := "\u0645\u0631\u062d\u0628\u0627"
	unrepresentableDir := filepath.Join(store.Dir, "Unrepresentable SAV")
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveEdit(unrepresentableDir, unrepresentable, ui.SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveAction("save"); err == nil || !strings.Contains(err.Error(), "cannot represent") {
		t.Fatal("unrepresentable original label lacked an explicit representation error", err)
	}
	if _, err := os.Stat(filepath.Join(unrepresentableDir, unrepresentable+".sav")); !os.IsNotExist(err) {
		t.Fatal("unrepresentable label published SAV", err)
	}
	state, ok = app.HeadlessSaveState()
	if !ok || state.Request.Name != unrepresentable || state.Request.Directory != unrepresentableDir || state.Request.Format != ui.SaveSAV {
		t.Fatal("invalid label changed the typed request", state)
	}
	if err := app.HeadlessSaveEdit("", "Representable retry", ""); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveAction("save"); err != nil {
		t.Fatal("corrected label retry", err)
	}
	cyrillicDir := filepath.Join(store.Dir, "Cyrillic SAV")
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveEdit(cyrillicDir, unicodeName, ui.SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveAction("save"); err != nil {
		t.Fatal("a representable Cyrillic original label must be written, not refused", err)
	}
	cyrillicRaw, err := ReadSaveFile(filepath.Join(cyrillicDir, unicodeName+".sav"))
	if err != nil {
		t.Fatal(err)
	}
	// OriginalSaveLabel appends a known suffix (originalsave.go): a prefix
	// check would still pass on trailing garbage, so assert the exact
	// string. This save is a town save (arriveInTown, Mission 0).
	wantCyrillicDecoded := unicodeName + " - between missions"
	cyrillicDecoded, err := OriginalSaveLabel(cyrillicRaw, f.textSelector())
	if err != nil || cyrillicDecoded != wantCyrillicDecoded {
		t.Fatalf("Cyrillic SAV label did not round trip: OriginalSaveLabel = %q, %v; want %q", cyrillicDecoded, err, wantCyrillicDecoded)
	}
	if err := headlessOpenLoad(app); err != nil {
		t.Fatal(err)
	}
	cyrillicRows := app.HeadlessRows()
	if len(cyrillicRows) != 1 || cyrillicRows[0].Text != cyrillicDecoded {
		t.Fatalf("the LOAD window disagrees with the SAVE browser about this file's label: LOAD shows %v; want one row %q", cyrillicRows, cyrillicDecoded)
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	// The successful Cyrillic SAV write above closed the dialog; reopen it
	// and return to unicodeDir/unicodeName for the Unicode path case that follows.
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveEdit(unicodeDir, unicodeName, ui.SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveAction("save"); err != nil {
		t.Fatal("Unicode SAV name", err)
	}
	raw, err := ReadSaveFile(filepath.Join(unicodeDir, unicodeName+".sav"))
	if err != nil {
		t.Fatal(err)
	}
	label, err := OriginalSaveLabel(raw, f.textSelector())
	if err != nil || label != wantCyrillicDecoded {
		t.Fatal("Unicode SAV label changed", label, err)
	}
	if err := headlessOpenLoad(app); err != nil {
		t.Fatal(err)
	}
	rows := app.HeadlessRows()
	if len(rows) != 1 || rows[0].Text != wantCyrillicDecoded {
		t.Fatal("LOAD did not retain Unicode SAV label until font rendering", rows)
	}
	if err := app.HeadlessActivate("@first"); err != nil || app.Screen() != ui.ScreenTown {
		t.Fatal("Unicode filename cold LOAD", err)
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveEdit(unicodeDir, "Latin label", ui.SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveAction("save"); err != nil {
		t.Fatal("Unicode directory with representable SAV label", err)
	}
	raw, err = ReadSaveFile(filepath.Join(unicodeDir, "Latin label.sav"))
	if err != nil {
		t.Fatal(err)
	}
	original, err := sav.Open(raw)
	if err != nil || !bytes.HasPrefix(original.Label, []byte("Latin label")) {
		t.Fatal("SAV label encoding changed", err)
	}
}

func saveDialogArrivedTown(t *testing.T, dir string) (*FrontEnd, *ui.App) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("controlled victory and actual town arrival")
	f.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, nil)
	entry, err := ReadHeadlessScenario(filepath.Join("..", "..", "scenarios", "1173-save-dialog.json"))
	if err != nil {
		t.Fatal(err)
	}
	entry.Steps = entry.Steps[:3]
	entry.Steps[1].Target = "Mission 20: 20.alm"
	if err := RunHeadlessScenario(f, app, entry, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	// Control only the terminal prerequisite; the notice and homeward route
	// must perform the actual arrival before any city SAVE is offered.
	w := f.live.world
	clock, hasClock := w.SessionClock()
	session := sim.OriginalSession{HasClock: hasClock, Clock: clock, Registers: w.ScriptRegisters(),
		RawHead: w.RawSessionHead(), RawMid: w.RawSessionMid(), TriggerLatches: make([]byte, 1000),
		Diplomacy: make([]byte, 2500), Won: 1, Outcome: sim.OutcomeWon}
	for i := range session.TriggerLatches {
		if w.ScriptLatched(int32(i)) {
			session.TriggerLatches[i] = 1
		}
	}
	relations := w.Relations()
	for i := range 50 {
		for j := range 50 {
			session.Diplomacy[i*50+j] = relations.Byte(uint32(i), uint32(j))
		}
	}
	if err := w.ImportOriginalSession(session); err != nil {
		t.Fatal(err)
	}
	for frame := 0; frame < 180; frame++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		if _, kind, open := f.LiveNotice(); open {
			if kind == ui.NoticeSuccess {
				campaignReturn(t, f, app)
				s, _, err := f.Snapshot(false)
				if err != nil || f.Town.Chapter() != 30 || s.CampaignState && !s.Campaign.FirstMapPoint || s.WorldMapReturn != nil {
					t.Fatalf("city home: chapter=%d campaign=%v first=%v return=%v error=%v", f.Town.Chapter(), s.CampaignState, s.Campaign.FirstMapPoint, s.WorldMapReturn, err)
				}
				return f, app
			}
			if err := app.HeadlessActivate("notice"); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Fatal("controlled terminal state did not display its victory")
	return nil, nil
}

func TestReleaseSaveDialog1173NewGameTownArrival(t *testing.T) {
	f := releaseFront(t)
	scenario, err := ReadHeadlessScenario(filepath.Join("..", "..", "scenarios", "1173-save-dialog.json"))
	if err != nil {
		t.Fatal(err)
	}
	app := f.App("1173 ordinary New Game")
	dir := t.TempDir()
	f.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, nil)
	if err := RunHeadlessScenario(f, app, scenario, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	raw, err := ReadSaveFile(filepath.Join(dir, "Expedition checkpoint.sav"))
	if err != nil {
		t.Fatal(err)
	}
	checkpoint, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	party, err := checkpoint.Party()
	checkpointDoc, docErr := sav.DecodeDocumentData(raw)
	if err != nil || docErr != nil || len(party) != 1 || checkpointDoc.Head.Mission != 30 {
		t.Fatal("real New Game checkpoint did not have exactly one hero", err)
	}
	raw, err = ReadSaveFile(filepath.Join(dir, "Expedition mission.sav"))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil || doc.World == nil || doc.Head.Mission != 30 {
		t.Fatal("ordinary mission SAV became a town", err)
	}
	f, app = saveDialogArrivedTown(t, dir)
	takeCampaignOffer(t, f, 30)
	townDialog := HeadlessScenario{Version: HeadlessScenarioVersion, Steps: []HeadlessStep{
		{Command: "save_open"},
		{Command: "save_edit", Name: "Expedition city", Target: "SAV"},
		{Command: "save_action", Target: "save"},
		{Command: "load", Target: "@first"},
		{Command: "save_open"},
		{Command: "save_edit", Name: "Town pair", Target: "SAV"},
		{Command: "save_action", Target: "save"},
		{Command: "save_open"},
		{Command: "save_edit", Name: "Town pair", Target: "SAV"},
		{Command: "save_action", Target: "save"},
		{Command: "save_action", Target: "back"},
		{Command: "save_action", Target: "save"},
		{Command: "save_action", Target: "overwrite"},
		{Command: "save_open"},
		{Command: "save_action", Target: "cancel"},
		{Command: "key", Key: "escape"},
	}}
	if err := RunHeadlessScenario(f, app, townDialog, io.Discard, io.Discard); err != nil || app.Screen() != ui.ScreenTown {
		t.Fatal("arrived town SAVE/overwrite/cancel", app.Screen(), err)
	}
	for _, name := range []string{"Expedition city.sav", "Town pair.sav"} {
		raw, err := ReadSaveFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil || doc.World != nil || doc.Campaign.Scalars[4] != 1 {
			t.Fatal("arrived city SAV lost its home position", name, err)
		}
		g := releaseFront(t)
		if _, town, err := g.RestoreOriginal(raw); err != nil || !town {
			t.Fatal(name, err)
		}
		if len(g.Carried) != 2 || g.Town.won[30] || !slices.Contains(g.Town.Available(), 30) {
			t.Fatal("ordinary pending companion is missing or duplicated, or current mission completed", name, g.Town.Available(), len(g.Carried))
		}
		if pending := g.Town.takeAddHeroes(30); len(pending) != 0 {
			t.Fatal("saved arrival grant remained pending", pending)
		}
	}
}

func TestReleaseSaveDialog1173NewGameMission30RepairsPermanentMercenaryList(t *testing.T) {
	f := releaseFront(t)
	scenario, err := ReadHeadlessScenario(filepath.Join("..", "..", "scenarios", "1173-save-dialog.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(scenario.Steps) < 12 {
		t.Fatalf("scenario has %d steps, want at least 12 (NEW GAME through the Expedition mission SAV attempt)", len(scenario.Steps))
	}
	if scenario.Steps[10].Name != "Expedition mission" || scenario.Steps[10].Target != "SAV" {
		t.Fatalf("scenario step 10 = %+v, want the Expedition mission SAV save_edit this witness depends on", scenario.Steps[10])
	}
	app := f.App("1202 New Game mission 30 repair")
	dir := t.TempDir()
	f.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, nil)
	route := scenario
	route.Steps = scenario.Steps[:12]
	if err := RunHeadlessScenario(f, app, route, io.Discard, io.Discard); err != nil {
		t.Fatalf("the writer must repair chapter 30's permanent mercenary list from the campaign's own implied prologue, not refuse: %v", err)
	}

	begins, ok := f.Campaign.Value().TownBegins()
	if !ok || begins != 30 {
		t.Fatalf("this witness assumes the shipped campaign's first town chapter is 30, got %d ok=%v", begins, ok)
	}
	implied := impliedMercenaryUnlocks(f.Campaign.Value(), begins)
	if len(implied) == 0 {
		t.Fatal("impliedMercenaryUnlocks found nothing for the campaign's own first town chapter")
	}
	var want [16]bool
	for _, m := range implied {
		for _, typ := range f.Campaign.Value().Chapters[m].EnableMercenary {
			if typ > 0 && typ < len(want) {
				want[typ] = true
			}
		}
	}
	wantList := mercEnabledU16s(f.Campaign.Value(), nil, want)
	if len(wantList) == 0 {
		t.Fatal("the implied missions carry no EnableMercenary keys; this witness cannot tell the repair from a no-op")
	}

	raw, err := ReadSaveFile(filepath.Join(dir, "Expedition mission.sav"))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	proj, ok, err := parsed.Campaign()
	if err != nil || !ok {
		t.Fatalf("decoded SAV has no campaign projection: ok=%v err=%v", ok, err)
	}
	if !slices.Equal(proj.PermanentMercenaries, wantList) {
		t.Fatalf("PermanentMercenaries = %v, want %v (mission 10's own EnableMercenary keys; mission 20 grants none)", proj.PermanentMercenaries, wantList)
	}
	literalWant := []uint16{14, 6, 13, 4, 7, 3, 2, 9, 12}
	if !slices.Equal(proj.PermanentMercenaries, literalWant) {
		t.Fatalf("PermanentMercenaries = %v, want the literal %v", proj.PermanentMercenaries, literalWant)
	}
}
