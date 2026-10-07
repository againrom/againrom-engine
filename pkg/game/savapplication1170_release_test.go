package game

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// A controlled application fixture over an authentic source graph. Only the
// explicitly named YA1 fields are changed in memory; no original process runs
// and neither the lawful install nor the source save is written.
func releaseApplicationSource1170(t *testing.T) []byte {
	t.Helper()
	_, input := groundCorpusFile(t, "2026-08-15/game0016.sav", "5e67d1282398076498867ac0124046d5c0e7f1acd2d0ffc1e66888c5296e0345")
	file, err := sav.Open(input)
	if err != nil {
		t.Fatal(err)
	}
	fog, present, err := file.Fog()
	if err != nil || !present {
		t.Fatalf("source Fog: %v", err)
	}
	for _, leaf := range []struct {
		section, key string
		value        int32
	}{
		{"GameOptions", "Wimpy", 7}, {"GameOptions", "ShowHP", -7}, {"GameOptions", "FlyingHP", 9},
		{"GameOptions", "Formation", 31}, {"GameOptions", "Speed", 99}, {"GameOptions", "ShowTimeFlow", -3},
		{"Inventory", "IsOpen", 6}, {"SpellBook", "IsOpen", 5}, {"SpellBook", "Pressed", -1},
		{"View", "X", 12}, {"View", "Y", 24}, {"Fog", "FirstState", 0},
	} {
		if err := file.SetStoreInt(leaf.section, leaf.key, leaf.value); err != nil {
			t.Fatal(err)
		}
	}
	if err := file.SetStoreIntArray("Objects", "Selection", nil); err != nil {
		t.Fatal(err)
	}
	if err := file.SetStoreIntArray("Fog", "Data", []int32{int32(len(fog.Cells))}); err != nil {
		t.Fatal(err)
	}
	return file.Marshal()
}

func TestReleaseApplicationPointerPanWritesNearestSAVCamera(t *testing.T) {
	t.Run("original", func(t *testing.T) { applicationPointerPanWitness(t, false, false) })
	t.Run("legacy", func(t *testing.T) { applicationPointerPanWitness(t, true, true) })
	t.Run("legacy-no-options", func(t *testing.T) { applicationPointerPanWitness(t, true, false) })
}

func applicationPointerPanWitness(t *testing.T, legacy, toggleOption bool) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	if path := os.Getenv("AGAINROM_APPLICATION_SAV_INPUT"); path != "" {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		open, town, err := f.RestoreOriginal(raw)
		if err != nil || town {
			t.Fatalf("fresh-process SAV LOAD: town=%t %v", town, err)
		}
		if err := f.App("fresh projected application").OpenMission(open); err != nil {
			t.Fatal(err)
		}
		view := f.live.view.SaveApplication()
		if view.ViewX != 11.53125 || view.ViewY != 23.65625 || view.Zoom != 2 || view.PeriodUS != 250000 || view.Unpaced || len(view.Selection) != 2 {
			t.Fatalf("fresh projected application = %+v", view)
		}
		f.live.view.ToggleShowHealth()
		snapshot, label, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		out, err := f.ExportCurrentWorldSave(snapshot, label)
		if err != nil {
			t.Fatal("SAV after the next application action", err)
		}
		if bytes.Equal(raw, out) {
			t.Fatal("next application action did not change SAV")
		}
		return
	}
	open, town, err := f.RestoreOriginal(releaseApplicationSource1170(t))
	if err != nil || town {
		t.Fatal(err)
	}
	if legacy {
		if _, _, _, _, _, _, _, _, _, _, err := open(); err != nil {
			t.Fatal(err)
		}
		snapshot, _, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		snapshot.ApplicationState = nil
		raw, err := EncodeSave(snapshot, "legacy application")
		if err != nil {
			t.Fatal(err)
		}
		snapshot, _, err = DecodeSave(raw)
		if err != nil {
			t.Fatal(err)
		}
		f = releaseFront(t)
		f.Options = OptionsStore{}
		f.SetDeterministicFrames(true)
		open, town, err = f.Restore(snapshot)
		if err != nil || town {
			t.Fatal(err)
		}
	}
	app := f.App("fractional pointer pan")
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	app.Layout(1024, 768)
	for _, key := range []string{"0", "select-all"} {
		if err := app.HeadlessKey(key); err != nil {
			t.Fatal(err)
		}
	}
	// The established cell origin is 12,24 at 32 pixels/cell and zoom 1.
	// A real right-button drag moves it by -15,-11 map pixels.
	wantX, wantY := 12.-15./32, 24.-11./32
	for _, event := range []struct {
		action string
		x, y   int
	}{{"right-press", 400, 300}, {"right-move", 415, 311}, {"right-release", 415, 311}} {
		if err := app.HeadlessPointer(event.action, event.x, event.y); err != nil {
			t.Fatal(err)
		}
	}
	view := f.live.view.SaveApplication()
	view.Zoom, view.DollOpen, view.WornOpen, view.MinimapOpen = 2, false, false, false
	if err := f.live.view.RestoreSaveApplication(view); err != nil {
		t.Fatal(err)
	}
	if toggleOption {
		for _, action := range []string{"escape", "game-options", "show-health", "page-return"} {
			var err error
			if action == "escape" {
				err = app.HeadlessKey(action)
			} else {
				err = app.HeadlessGameMenuAction(action)
			}
			if err != nil {
				t.Fatal(action, err)
			}
		}
		if f.live.applicationState == nil || !f.live.applicationState.LocalOnly {
			t.Fatal("legacy option toggle did not create the local-only checkpoint")
		}
		if err := app.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
	}
	// Speeds slower than the dialog's slider reach go through the map key.
	for i := 0; i < 9 && f.live.clock.Period() < 250000; i++ {
		if err := app.HeadlessKey("numpad-minus"); err != nil {
			t.Fatal(err)
		}
	}
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	if s.ApplicationState == nil {
		t.Fatal("live capture did not record current application state")
	}
	want := s.ApplicationState.View
	if want.ViewX != wantX || want.ViewY != wantY || want.PeriodUS != 250000 || len(want.Selection) != 2 {
		t.Fatalf("pointer pan/current selection = %+v; want origin %v,%v and 2 actors", want, wantX, wantY)
	}
	if _, err := f.ExportCurrentWorldSave(s, "fractional view"); err != nil {
		t.Fatalf("fractional camera, local panels and extended speed refused SAV: %v", err)
	}
	store := SaveStore{Dir: t.TempDir()}
	app.SetSaveSeams(f.SaveSeams(store, OriginalStore{}, nil))
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(store.Dir)
	if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Name()) != ".sav" {
		t.Fatalf("fractional ordinary SAVE files %v: %v", entries, err)
	}
	raw, err := os.ReadFile(filepath.Join(store.Dir, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	projected, err := readOriginalApplicationState(&doc)
	if err != nil || projected.ViewX != 12 || projected.ViewY != 24 || projected.Speed != 0 {
		t.Fatalf("SAV did not project camera and speed: %+v %v", projected, err)
	}
	if got := f.live.view.SaveApplication(); !reflect.DeepEqual(got, want) {
		t.Fatalf("SAV changed the live application: got %+v want %+v", got, want)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^"+t.Name()+"$", "-test.v")
	cmd.Env = append(os.Environ(), "AGAINROM_APPLICATION_SAV_INPUT="+filepath.Join(store.Dir, entries[0].Name()))
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fresh-process LOAD and next action: %v\n%s", err, output)
	}
	native, err := EncodeSave(s, "explicit native checkpoint")
	if err != nil {
		t.Fatal(err)
	}
	saved, _, err := DecodeSave(native)
	if err != nil || !reflect.DeepEqual(saved.ApplicationState.View, want) {
		t.Fatal("ordinary native SAVE lost fractional current application", err)
	}
	cold := releaseFront(t)
	coldOpen, coldTown, err := cold.Restore(saved)
	if err != nil || coldTown {
		t.Fatal(err)
	}
	if err := cold.App("cold pointer pan").OpenMission(coldOpen); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cold.live.view.SaveApplication(), want) {
		t.Fatal("cold native LOAD changed fractional camera or selection")
	}
}

func checkRawApplication1170(t *testing.T, raw []byte, want OriginalStateData) {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	got, err := readOriginalApplicationState(&doc)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("raw application = %+v, want %+v: %v", got, want, err)
	}
}

func TestReleaseApplication1170CurrentFogSelectionOptionsAndColdSave(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{} // Explicitly no profile writer in this witness.
	f.SetDeterministicFrames(true)
	input := releaseApplicationSource1170(t)
	// The raw expected values are independent of both LOAD and either exporter.
	wantRaw := OriginalStateData{Wimpy: 7, ShowHP: -7, FlyingHP: 9, Formation: 31, Speed: 99,
		ShowTimeFlow: -3, InventoryOpen: 6, BookOpen: 5, Pressed: -1, ViewX: 12, ViewY: 24,
		Selection: []uint32{}}
	// Both conversion directions must preserve values outside the native UI
	// subset without a window adoption or a process preference overwrite.
	// AGS is a retired write target, so the legacy fixture the surviving
	// AGS->SAV leg below reads is authored directly from converter's own
	// RestoreOriginal+Snapshot+EncodeSave, the same steps ConvertSave's
	// removed "-to ags" direction once ran.
	converter := releaseFront(t)
	open, sourceTown, err := converter.RestoreOriginal(input)
	if err != nil {
		t.Fatal("SAV to AGS", err)
	}
	if !sourceTown {
		if _, _, _, _, _, _, _, _, _, _, err = open(); err != nil {
			t.Fatal("SAV to AGS", err)
		}
	}
	sourceSnapshot, _, err := converter.Snapshot(!sourceTown)
	if err != nil {
		t.Fatal("SAV to AGS", err)
	}
	agnostic, err := EncodeSave(sourceSnapshot, "agnostic")
	if err != nil {
		t.Fatal("SAV to AGS", err)
	}
	converted, _, err := releaseFront(t).ConvertSave(agnostic, "sav", nil)
	if err != nil {
		t.Fatal("AGS to SAV", err)
	}
	checkRawApplication1170(t, converted, wantRaw)
	open, town, err := f.RestoreOriginal(input)
	if err != nil || town {
		t.Fatalf("original application prepare: %v", err)
	}
	app := f.App("current application witness")
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	app.Layout(1024, 768)
	loadedCameraY := f.live.view.Camera().Y
	first, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	if first.ApplicationState.View.PeriodUS != 31000 || f.wimpyMode != 0 {
		t.Fatal("source option interpretation was overwritten during adoption")
	}
	unchanged, err := f.ExportCurrentWorldSave(first, "unchanged application")
	if err != nil {
		t.Fatal(err)
	}
	checkRawApplication1170(t, unchanged, wantRaw)
	key := func(name string) {
		t.Helper()
		if err := app.HeadlessKey(name); err != nil {
			t.Fatal(err)
		}
	}
	key("0")
	if !f.live.stopped {
		t.Fatal("witness did not pause the simulation")
	}
	wantSelection := []uint32{}
	wantRuntime := []uint32{}
	for _, actor := range f.live.world.Entities() {
		if actor.Owner == sim.SelfSlot && actor.HP > 0 {
			wantSelection = append(wantSelection, uint32(actor.ID))
		}
	}
	slices.Sort(wantSelection)
	for _, id := range wantSelection {
		for _, actor := range f.live.world.Entities() {
			if uint32(actor.ID) == id {
				wantRuntime = append(wantRuntime, actor.SourceBinding.RuntimeID)
			}
		}
	}
	if len(wantSelection) < 2 {
		t.Fatalf("authentic fixture needs multiple selectable owned actors, got %d", len(wantSelection))
	}
	key("select-all")
	if !slices.Equal(app.HeadlessSelection(), wantSelection) {
		t.Fatalf("select-all = %v, want %v", app.HeadlessSelection(), wantSelection)
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	// Hover assignment works even for a currently unavailable source book
	// spell; F5 then sets the separate current spell through its real key path.
	x, y, err := app.HeadlessSpellPoint(23)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessPointer("hover", x, y); err != nil {
		t.Fatal(err)
	}
	key("ctrl-f5")
	key("f5")
	if slots, current, _ := f.live.view.QuickSpellState(); slots != [4]uint32{23} || current != 23 {
		t.Fatalf("actual quick binding/current = %v/%d", slots, current)
	}
	if err := app.HeadlessPointer("hover", 400, 300); err != nil {
		t.Fatal(err)
	}
	key("i")
	key("b")
	f.live.view.ToggleShowHealth()
	f.live.view.ToggleDamageNumerals()
	f.live.view.ToggleTimeFlow()
	key("escape")
	for _, action := range []string{"game-options", "speed-up", "speed-up", "speed-up", "speed-up", "speed-up", "speed-up", "speed-up", "speed-up", "page-return", "return"} {
		if err := app.HeadlessGameMenuAction(action); err != nil {
			t.Fatal(err)
		}
	}
	// Book-edge pointer gestures may pan the map. Set the intended current
	// camera through its public API after those gestures, relative to the
	// verified loaded origin, so the expected cell coordinates stay explicit.
	f.live.view.Camera().X = 13 * 32
	f.live.view.Camera().Y = loadedCameraY + 64
	// The expected plane is read from current simulation sight before the Fog
	// producer runs. The input plane is independently all-unseen. This creates
	// an unseen-to-visible transition and leaves both unseen and explored cells.
	wantFog := make([]byte, len(f.live.fog.explored))
	for i, lit := range f.live.world.Sight(sim.SelfSlot) {
		if lit != 0 {
			wantFog[i] = 1
		}
	}
	if bytes.Count(wantFog, []byte{1}) == 0 || bytes.Count(wantFog, []byte{0}) == 0 {
		t.Fatal("current sight fixture is not mixed")
	}
	f.live.fog.refresh(f.live.world, sim.SelfSlot)
	f.live.push()
	wantRaw.Selection = wantRuntime
	wantRaw.ShowHP, wantRaw.FlyingHP, wantRaw.ShowTimeFlow = 0, 0, 0
	wantRaw.InventoryOpen, wantRaw.BookOpen, wantRaw.Pressed, wantRaw.Speed = 0, 0, 5, 7
	wantRaw.ViewX, wantRaw.ViewY = 13, 26
	// The Game Options OK above sends the formation command again, which
	// rewrites the loaded word with the current mode.
	wantRaw.Formation = 1
	wantView := ui.SaveApplicationState{Selection: wantSelection, ViewX: 13, ViewY: 26, Zoom: 1,
		InventoryOpen: false, SpellBookOpen: false, DollOpen: true, WornOpen: true, MinimapOpen: true,
		ShowHealth: false, FlyingHP: false, TimeFlow: false, PressedSpell: 23, PeriodUS: 35000}
	if got := f.live.view.SaveApplication(); !reflect.DeepEqual(got, wantView) {
		t.Fatalf("live UI = %+v, want %+v", got, wantView)
	}
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(s.Residue.FogExplored, wantFog) || !bytes.Equal(s.Residue.FogVisible, wantFog) {
		t.Fatal("current Fog was not captured")
	}
	native, err := EncodeSave(s, "current application native checkpoint")
	if err != nil {
		t.Fatal(err)
	}
	nativeSnapshot, _, err := DecodeSave(native)
	if err != nil {
		t.Fatal(err)
	}
	nativeFront := releaseFront(t)
	nativeOpen, nativeTown, err := nativeFront.Restore(nativeSnapshot)
	if err != nil || nativeTown {
		t.Fatal("current application native restore", err)
	}
	if err := nativeFront.App("native application witness").OpenMission(nativeOpen); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(nativeFront.live.view.SaveApplication(), wantView) || !bytes.Equal(nativeFront.live.fog.explored, wantFog) || !bytes.Equal(nativeFront.live.fog.visible, wantFog) {
		t.Fatal("native cold restore changed current application or sampled visibility")
	}
	currentConverted, _, err := releaseFront(t).ConvertSave(native, "sav", nil)
	if err != nil {
		t.Fatal("current application AGS to SAV", err)
	}
	checkRawApplication1170(t, currentConverted, wantRaw)
	store := SaveStore{Dir: t.TempDir()}
	app.SetSaveSeams(f.SaveSeams(store, OriginalStore{}, nil))
	key("escape")
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(store.Dir)
	if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Name()) != ".sav" {
		t.Fatalf("ordinary SAVE files %v: %v", entries, err)
	}
	saved, err := os.ReadFile(filepath.Join(store.Dir, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	checkRawApplication1170(t, saved, wantRaw)
	file, err := sav.Open(saved)
	if err != nil {
		t.Fatal(err)
	}
	fog, _, err := file.Fog()
	if err != nil || !bytes.Equal(fog.Cells, wantFog) {
		t.Fatal("ordinary SAVE kept historical Fog")
	}
	if bytes.Equal(first.Residue.FogExplored, wantFog) || slices.Equal(first.ApplicationState.View.Selection, wantSelection) {
		t.Fatal("frozen-source loss controls did not distinguish current state")
	}
	clear(input)
	clear(agnostic)
	clear(converted)
	clear(native)
	clear(currentConverted)
	for generation := 0; generation < 2; generation++ {
		fresh := releaseFront(t)
		open, town, err := fresh.RestoreOriginal(saved)
		if err != nil || town {
			t.Fatalf("cold generation %d: %v", generation, err)
		}
		cold := fresh.App("cold application witness")
		if err := cold.OpenMission(open); err != nil {
			t.Fatal(err)
		}
		cold.Layout(1024, 768)
		got := fresh.live.view.SaveApplication()
		// Runtime IDs survive output graph reminting; native IDs are resolved
		// afresh, so verify their relation rather than relying on incidental IDs.
		wantCold := wantView
		wantCold.Selection = nil
		for _, runtime := range wantRuntime {
			for _, actor := range fresh.live.world.Entities() {
				if actor.SourceBinding.RuntimeID == runtime {
					wantCold.Selection = append(wantCold.Selection, uint32(actor.ID))
				}
			}
		}
		slices.Sort(wantCold.Selection)
		if !reflect.DeepEqual(got, wantCold) || !bytes.Equal(fresh.live.fog.explored, wantFog) {
			t.Fatalf("cold application generation %d = %+v, want %+v", generation, got, wantCold)
		}
		next, _, err := fresh.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		saved, err = fresh.ExportCurrentWorldSave(next, "second current application")
		if err != nil {
			t.Fatal(err)
		}
		checkRawApplication1170(t, saved, wantRaw)
	}
	t.Logf("current app: %d exact actor bindings, %d/%d explored cells, current panels/current spell/speed; two cold SAV generations", len(wantSelection), bytes.Count(wantFog, []byte{1}), len(wantFog))
}
