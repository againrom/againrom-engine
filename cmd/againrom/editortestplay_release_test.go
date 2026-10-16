package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/game"
	"againrom/pkg/ui"
)

func TestReleaseEditorTestPlayLaunchEntryWinAndAcknowledgement(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS")
	}
	x, err := game.NewMapInspector(root)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := game.ResolveRuntimeProfile(filepath.Join(t.TempDir(), "saves"), "", "", root)
	if err != nil {
		t.Fatal(err)
	}
	program := filepath.Join(t.TempDir(), "againrom.exe")
	if err := os.WriteFile(program, []byte("launch witness"), 0o600); err != nil {
		t.Fatal(err)
	}
	var started []string
	x.SetTestPlay(game.TestPlayHost{Profile: func() (game.RuntimeProfile, error) { return profile, nil }, Game: program, Start: func(path string, args []string) error {
		if path != program {
			t.Fatal("other game program")
		}
		started = append([]string(nil), args...)
		return nil
	}})
	e := x.Editor()
	if err := e.Open("new/32x32"); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits
	author := s.(ui.InspectionScriptAuthor)
	for _, kind := range []string{"Condition", "Action", "Trigger"} {
		if _, err := author.AddScript(kind); err != nil {
			t.Fatal(err)
		}
	}
	for _, v := range []struct {
		kind, key string
		value     int64
	}{{"Action", "opcode", 4}, {"Trigger", "left0", 1}, {"Trigger", "right0", 1}, {"Trigger", "action0", 1}} {
		if _, err := s.SetField(v.kind, 0, v.key, v.value); err != nil {
			t.Fatal(err)
		}
	}
	_ = e.Key("undo")
	_ = e.Key("redo")
	before := append([]byte(nil), e.Document().SourceBytes...)
	if err := e.Key("testplay"); err != nil {
		t.Fatal(err)
	}
	if len(started) == 0 || !e.Dirty() || !bytes.Equal(before, e.Document().SourceBytes) {
		t.Fatal("F5 did not retain the editor's unsaved document")
	}
	o, err := parse(started)
	if err != nil {
		t.Fatal(err)
	}
	if !o.missionSet || !reflect.DeepEqual(started, game.TestPlayArgs(root, filepath.Join(profile.Directory, game.TestPlayModID), o.mission)) {
		t.Fatalf("launch args %q", started)
	}
	mods, err := resolveMods(o, program)
	if err != nil {
		t.Fatal(err)
	}
	res, err := activateMods(o, mods, root)
	if err != nil {
		t.Fatal(err)
	}
	o.headless = "editor lifecycle witness"
	f, err := frontEnd(root, o, game.OptionsStore{})
	if err != nil {
		t.Fatal(err)
	}
	f.SetDeterministicFrames(true)
	f.SetTipsOff(true)
	if err := f.SetMods(res.Rules, res.Set, false); err != nil {
		t.Fatal(err)
	}
	if err := f.SetModCompanions(res.Companions); err != nil {
		t.Fatal(err)
	}
	if _, err := f.SetModMissions(res.Missions); err != nil {
		t.Fatal(err)
	}
	a := f.App("editor test play lifecycle")
	t.Cleanup(a.StopAudio)
	a.Layout(1024, 768)
	if err := armNewGameDoor(a, f, o); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ui.ScreenChargen {
		t.Fatalf("F5 entry screen=%s", a.Screen())
	}
	scenario := game.HeadlessScenario{Version: game.HeadlessScenarioVersion, Steps: []game.HeadlessStep{{Command: "create_character", Character: &game.HeadlessCharacter{Name: "Author"}}}}
	if err := game.RunHeadlessScenario(f, a, scenario, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	if state := f.HeadlessSnapshot(a.Screen()); state.Screen != "map" || state.Mission != o.mission {
		t.Fatalf("entry %+v", state)
	}
	for i := 0; i < 1000; i++ {
		if _, kind, up := f.LiveNotice(); up {
			if kind == ui.NoticeSuccess {
				break
			}
			if err := a.HeadlessActivate("notice"); err != nil {
				t.Fatal(err)
			}
		}
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if _, kind, up := f.LiveNotice(); !up || kind != ui.NoticeSuccess {
		t.Fatalf("authored result missing: %+v", f.HeadlessSnapshot(a.Screen()))
	}
	if err := a.HeadlessActivate("notice"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4000 && a.Screen() == ui.ScreenMap; i++ {
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	state := f.HeadlessSnapshot(a.Screen())
	t.Logf("F5 mission %d -> chargen -> authored Victory -> acknowledgement -> %s/%s chapter%d members%d", o.mission, state.Screen, state.TownPlace, f.Town.Chapter(), len(state.Members))
	if a.Screen() != ui.ScreenTown || state.TownPlace != "square" || f.Town.Chapter() != res.Missions.Missions[0].Chapter {
		t.Fatalf("F5 victory did not return through the campaign route: screen=%s", a.Screen())
	}
	if !e.Dirty() || !bytes.Equal(before, e.Document().SourceBytes) {
		t.Fatal("test play changed the editor")
	}
	if _, err := os.Stat(filepath.Join(profile.Directory, game.TestPlayModID, game.TestPlayModID, "mod.toml")); err != nil {
		t.Fatal("owned test mod disappeared:", err)
	}
}
