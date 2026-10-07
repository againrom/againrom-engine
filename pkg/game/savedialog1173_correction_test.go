package game

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// A current mission before the first town must keep its battlefield and effects.
func saveDialogPreTownBattlefield(t *testing.T) {
	f := releaseFront(t)
	_, raw := groundCorpusFile(t, "2026-08-02/game0007.sav", "a7cb35ea5d87c9c7a9b8cfd70f8a1f61b6927ca089e468a9d6fe983da777156e")
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal(town, err)
	}
	app := f.App("1173 pre-town admission")
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if f.liveMission != 20 || len(f.live.world.ActiveEffects()) != 1 {
		t.Fatal("fixture lacks its authentic mission20 active effect")
	}
	dir := filepath.Join(t.TempDir(), "new destination")
	f.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, nil)
	if err := app.HeadlessKey("f2"); err != nil || app.Screen() != ui.ScreenSave {
		t.Fatal("F2 did not open SAVE", app.Screen(), err)
	}
	before, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveEdit(dir, "Mission20 return", ui.SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveAction("save"); err != nil {
		t.Fatal("pre-town mission SAV", err)
	}
	if got, want := app.HeadlessMessage(), f.Words.SaveAcknowledgement; got != want {
		t.Fatalf("pre-town battlefield SAV carried an unexpected disclosure: %q, want %q", got, want)
	}
	after, _, err := f.Snapshot(true)
	slices.Sort(before.Residue.Commanded)
	slices.Sort(after.Residue.Commanded)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("SAV changed the live effect, World or campaign", err)
	}
	raw2, err := ReadSaveFile(filepath.Join(dir, "Mission20 return.sav"))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw2)
	if err != nil || doc.World == nil || doc.Head.Mission != 20 {
		t.Fatal("pre-town SAV did not store the current battlefield", err)
	}
	g := releaseFront(t)
	resume, town, err := g.RestoreOriginal(raw2)
	if err != nil || town {
		t.Fatal("pre-town battlefield cold LOAD", town, err)
	}
	if err := g.App("1173 pre-town continuation").OpenMission(resume); err != nil {
		t.Fatal(err)
	}
	if len(g.live.world.ActiveEffects()) != 1 {
		t.Fatal("battlefield SAV lost the active effect")
	}
	if f.live.world.Hash() != g.live.world.Hash() {
		currentMenuWorldDiagnostics(t, f.live.world, g.live.world)
	}
	assertCurrentWorldEqual(t, f.live.world, g.live.world, "pre-town dialog LOAD")
	f.LiveAdvance(1)
	g.LiveAdvance(1)
	assertCurrentWorldEqual(t, f.live.world, g.live.world, "pre-town next tick")
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveEdit("", "Mission20 checkpoint", ""); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveAction("save"); err != nil {
		t.Fatal("second mission SAVE", err)
	}
	checkpoint := loadAreaContinuation(t, filepath.Join(dir, "Mission20 checkpoint.sav"))
	assertCurrentWorldEqual(t, f.live.world, checkpoint.live.world, "second pre-town LOAD")
	f.LiveAdvance(1)
	checkpoint.LiveAdvance(1)
	assertCurrentWorldEqual(t, f.live.world, checkpoint.live.world, "second pre-town next tick")
}

func saveDialogRepeatedSuffix(t *testing.T) {
	for _, tc := range []struct{ selected, target string }{
		{"Chapter.sav.sav", "Chapter.sav.sav"},
	} {
		t.Run(tc.selected, func(t *testing.T) {
			f := releaseFront(t)
			f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Selected save", Choices: []int{0, 0, 0}, Stats: []int{30, 30, 30, 30}})
			f.arriveInTown()
			f.addChapterCompanions(f.Town.Chapter())
			s, _, err := f.Snapshot(false)
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			previous := map[string][]byte{}
			for _, name := range []string{tc.selected, tc.target} {
				if _, exists := previous[name]; exists {
					continue
				}
				raw, _, err := f.playerCitySave(s, "Previous town")
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, name), raw, 0600); err != nil {
					t.Fatal(err)
				}
				previous[name] = raw
			}
			app := f.App("selected exact name")
			f.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, nil)
			if err := headlessOpenLoad(app); err != nil {
				t.Fatal(err)
			}
			if err := app.HeadlessActivate("@first"); err != nil || app.Screen() != ui.ScreenTown {
				t.Fatal("seed LOAD", app.Screen(), app.HeadlessMessage(), err)
			}
			f.Town.gold += 31
			if err := app.HeadlessKey("f2"); err != nil || app.Screen() != ui.ScreenSave {
				t.Fatal("town F2", app.Screen(), err)
			}
			state, _ := app.HeadlessSaveState()
			index := -1
			for i, row := range state.Entries {
				if row.Name == tc.selected {
					index = len(state.Directories) + i
				}
			}
			if err := app.HeadlessSaveSelect(index); err != nil {
				t.Fatal("select actual disk row", err)
			}
			if err := app.HeadlessSaveAction("save"); err != nil {
				t.Fatal(err)
			}
			paths := []string{filepath.Join(dir, tc.target)}
			state, ok := app.HeadlessSaveState()
			if !ok || state.Request.Format != ui.SaveSAV || !state.Confirmation || !reflect.DeepEqual(state.Paths, paths) || !reflect.DeepEqual(state.Existing, paths) {
				t.Fatalf("selected %s must confirm exact target %v: %+v", tc.selected, paths, state)
			}
			if err := app.HeadlessSaveAction("back"); err != nil {
				t.Fatal(err)
			}
			for name, before := range previous {
				raw, err := ReadSaveFile(filepath.Join(dir, name))
				if err != nil || !bytes.Equal(raw, before) {
					t.Fatal("cancel changed a file", name, err)
				}
			}
			if err := app.HeadlessSaveAction("save"); err != nil {
				t.Fatal(err)
			}
			if err := app.HeadlessSaveAction("overwrite"); err != nil {
				t.Fatal(err)
			}
			raw, err := ReadSaveFile(paths[0])
			if err != nil {
				t.Fatal(err)
			}
			g := releaseFront(t)
			if _, town, err := g.RestoreOriginal(raw); err != nil || !town || g.Town.Gold() != f.Town.Gold() {
				t.Fatal("selected SAV checkpoint cold LOAD", town, err)
			}
			if tc.selected != tc.target {
				raw, err := ReadSaveFile(filepath.Join(dir, tc.selected))
				if err != nil || !bytes.Equal(raw, previous[tc.selected]) {
					t.Fatal("current SAVE changed legacy input", err)
				}
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != len(previous) {
				t.Fatal("SAVE created a different basename", entries, err)
			}
		})
	}
}

func TestCurrentSaveDialogRejectsLegacyOutputBeforePreparation(t *testing.T) {
	f := &FrontEnd{}
	dir := filepath.Join(t.TempDir(), "not created")
	for _, format := range []ui.SaveFormat{"AGS", "BOTH", "", "invalid"} {
		prepared, err := f.SaveDialogSeams(SaveStore{}, OriginalStore{}).Prepare(ui.SaveRequest{Directory: dir, Name: "checkpoint", Format: format})
		if err == nil || err.Error() != "save format must be SAV" || prepared.Commit != nil || len(prepared.Paths) != 0 {
			t.Fatal("obsolete format reached preparation", format, prepared, err)
		}
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Fatal("invalid format created output directory", err)
		}
	}
}

func TestCurrentSaveDialogMissionConfirmationAndColdContinuation(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	app := f.App("interactive current mission")
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	id := poolEntity(t, f.live.world, 91).ID
	f.LiveDamage(uint32(id), 1)
	f.LiveAdvance(1)
	dir := t.TempDir()
	path := filepath.Join(dir, "Checkpoint.sav")
	old := []byte("existing target")
	if err := os.WriteFile(path, old, 0600); err != nil {
		t.Fatal(err)
	}
	for cycle := 0; cycle < 2; cycle++ {
		f.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, nil)
		before := f.live.world.Hash()
		if err := app.HeadlessKey("f2"); err != nil || app.Screen() != ui.ScreenSave {
			t.Fatal("open actual SAVE", app.Screen(), err)
		}
		state, _ := app.HeadlessSaveState()
		if state.Request.Format != ui.SaveSAV || !state.Request.OnMap {
			t.Fatal("current mission default", state.Request)
		}
		if err := app.HeadlessSaveEdit("", "Checkpoint", ""); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessSaveAction("save"); err != nil {
			t.Fatal(err)
		}
		state, _ = app.HeadlessSaveState()
		if !state.Confirmation || !reflect.DeepEqual(state.Paths, []string{path}) || !reflect.DeepEqual(state.Existing, []string{path}) {
			t.Fatal("actual confirmation targets", state)
		}
		if err := app.HeadlessSaveAction("back"); err != nil {
			t.Fatal(err)
		}
		disk, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(disk, old) || f.live.world.Hash() != before {
			t.Fatal("prepare/cancel mutated current state or disk", err)
		}
		if err := app.HeadlessSaveAction("save"); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessSaveAction("overwrite"); err != nil {
			t.Fatal(err)
		}
		old, err = os.ReadFile(path)
		if err != nil || f.live.world.Hash() != before {
			t.Fatal("confirmation mutated live World", err)
		}
		g := currentPoolFixtureFront(t, 91, 92)
		open, town, err := g.RestoreOriginal(old)
		if err != nil || town {
			t.Fatal("interactive SAV cold LOAD", town, err)
		}
		next := g.App("cold interactive mission")
		if err := next.OpenMission(open); err != nil {
			t.Fatal(err)
		}
		assertCurrentWorldEqual(t, f.live.world, g.live.world, "interactive cold LOAD")
		f.LiveDamage(uint32(id), 1)
		g.LiveDamage(uint32(id), 1)
		for tick := 0; tick < 20; tick++ {
			f.LiveAdvance(1)
			g.LiveAdvance(1)
			assertCurrentWorldEqual(t, f.live.world, g.live.world, "interactive next damage/tick")
		}
		f, app = g, next
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 || entries[0].Name() != "Checkpoint.sav" {
		t.Fatal("interactive SAVE wrote another output", entries, err)
	}
}

func TestCurrentSaveDialogCityPurchaseTrainingAndColdContinuation(t *testing.T) {
	f := currentTrainingCity(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "Checkpoint.sav")
	s, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	old, err := f.ExportCurrentSave(s, "Checkpoint")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, old, 0600); err != nil {
		t.Fatal(err)
	}
	openCity := func(front *FrontEnd) *ui.App {
		app := front.App("interactive current city")
		front.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, nil)
		if err := headlessOpenLoad(app); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessActivate("@first"); err != nil || app.Screen() != ui.ScreenTown {
			t.Fatal("interactive city LOAD", app.Screen(), app.HeadlessMessage(), err)
		}
		return app
	}
	app := openCity(f)
	for cycle := 0; cycle < 2; cycle++ {
		buyTrainingItem(t, f)
		party, graph, gold := mapload.CloneParty(f.Carried), f.Town.cityObjects.Clone(), f.Town.Gold()
		if err := app.HeadlessKey("f2"); err != nil || app.Screen() != ui.ScreenSave {
			t.Fatal("city F2", app.Screen(), err)
		}
		state, _ := app.HeadlessSaveState()
		if state.Request.OnMap || state.Request.Format != ui.SaveSAV {
			t.Fatal("city dialog default", state.Request)
		}
		if err := app.HeadlessSaveEdit("", "Checkpoint", ""); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessSaveAction("save"); err != nil {
			t.Fatal(err)
		}
		state, _ = app.HeadlessSaveState()
		if !state.Confirmation || !reflect.DeepEqual(state.Paths, []string{path}) {
			t.Fatal("city confirmation", state)
		}
		if err := app.HeadlessSaveAction("back"); err != nil {
			t.Fatal(err)
		}
		disk, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(disk, old) {
			t.Fatal("city cancel changed disk", err)
		}
		if err := app.HeadlessSaveAction("save"); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessSaveAction("overwrite"); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(f.Carried, party) || !reflect.DeepEqual(f.Town.cityObjects, graph) || f.Town.Gold() != gold {
			t.Fatal("city prepare/confirmation changed live party, graph or purse")
		}
		old, err = os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		g := &FrontEnd{InstallResources: f.InstallResources, RuntimeServices: f.RuntimeServices}
		next := openCity(g)
		if g.Town.Gold() != gold || !reflect.DeepEqual(g.Town.cityObjects, graph) || len(g.Carried) != len(party) {
			t.Fatal("interactive cold city lost purse, roster or topology")
		}
		for _, before := range party {
			after := trainingPartyMember(t, g, before.ID)
			if before.Hero != after.Hero || before.Book != after.Book || before.KnownSpells != after.KnownSpells || !reflect.DeepEqual(before.Carry, after.Carry) || currentCitySaleHuman(t, before) != currentCitySaleHuman(t, after) {
				t.Fatal("interactive cold city changed current Human, book or Carry", before.ID)
			}
		}
		left, right := train1099(t, f, "hero", 0), train1099(t, g, "hero", 0)
		if !strings.HasPrefix(left, "trained ") || left != right || f.Town.Gold() != g.Town.Gold() || currentCitySaleHuman(t, trainingPartyMember(t, f, "hero")) != currentCitySaleHuman(t, trainingPartyMember(t, g, "hero")) {
			t.Fatal("interactive cold city changed next training", left, right)
		}
		f, app = g, next
	}
}
