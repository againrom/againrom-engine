package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/game"
	"againrom/pkg/herokind"
	"againrom/pkg/ui"
)

func TestReleaseEditorTestPlayLaunchEntryWinAndAcknowledgement(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
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
	author := s.(interface {
		AddScript(kind ui.RecordKind) (*ui.InspectionDocument, error)
	})
	for _, kind := range []ui.RecordKind{"Condition", "Action", "Trigger"} {
		if _, err := author.AddScript(kind); err != nil {
			t.Fatal(err)
		}
	}
	// The new map's drop action is action 0 with id 1; the added action is
	// action 1 with id 2.
	for _, v := range []struct {
		kind  string
		index int
		key   string
		value int64
	}{{"Action", 1, "opcode", 4}, {"Trigger", 0, "left0", 1}, {"Trigger", 0, "right0", 1}, {"Trigger", 0, "action0", 2}} {
		if _, err := s.SetField(ui.RecordKind(v.kind), v.index, v.key, v.value); err != nil {
			t.Fatal(err)
		}
	}
	_ = e.Key("undo")
	_ = e.Key("redo")
	before := append([]byte(nil), e.Document().SourceBytes...)
	if err := e.Key("testplay"); err != nil {
		t.Fatal(err)
	}
	if e.TestPartyOpen() {
		_ = e.Key("enter")
	}
	if len(started) == 0 || !e.Dirty() || !bytes.Equal(before, e.Document().SourceBytes) {
		t.Fatal("F5 did not retain the editor's unsaved document")
	}
	o, err := parse(started)
	if err != nil {
		t.Fatal(err)
	}
	want, err := game.TestPlayArgs(root, filepath.Join(profile.Directory, game.TestPlayModID), o.mission, herokind.MaleFighter.Word())
	if err != nil || !o.missionSet || !o.heroSet || !reflect.DeepEqual(started, want) {
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
	// The party window chose the hero, so the game opens the map with no
	// generation screen.
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
	t.Logf("F5 mission %d -> authored Victory -> acknowledgement -> %s/%s chapter%d members%d", o.mission, state.Screen, state.TownPlace, f.Town.Chapter(), len(state.Members))
	// The party window's mission declares its party and no guest, so it names
	// no chapter: the hero waits in the campaign's first town.
	first, ok := f.Campaign.Value().TownBegins()
	if m := res.Missions.Missions[0]; !ok || len(m.Party) != 1 || !m.Party[0].Hero || m.Guest != 0 || m.Chapter != 0 {
		t.Fatalf("the test mission is not the party window's: first town %d %t, mission %+v", first, ok, m)
	}
	if a.Screen() != ui.ScreenTown || state.TownPlace != "square" || f.Town.Chapter() != first {
		t.Fatalf("F5 victory did not return to the campaign town: screen=%s place=%s chapter=%d, want chapter %d",
			a.Screen(), state.TownPlace, f.Town.Chapter(), first)
	}
	if !e.Dirty() || !bytes.Equal(before, e.Document().SourceBytes) {
		t.Fatal("test play changed the editor")
	}
	if _, err := os.Stat(filepath.Join(profile.Directory, game.TestPlayModID, game.TestPlayModID, "mod.toml")); err != nil {
		t.Fatal("owned test mod disappeared:", err)
	}
}
