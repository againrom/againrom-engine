package game

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestGameOptions1186StoredValuesPreserveUnrelatedKeys(t *testing.T) {
	s := OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	if err := os.WriteFile(s.Path, []byte("FutureOption=unchanged\nShowTimeFlow=bad\nFormationMode=9\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, present, err := s.gameOptions()
	if err != nil || present != ([len(gameOptionKeys)]bool{}) {
		t.Fatal("malformed or missing values became choices", present, err)
	}
	want := ui.GameOptionValues{0, 1, 0, 2, 1}
	for i, v := range want {
		if err := s.setGameOption(ui.GameOption(i), v); err != nil {
			t.Fatal(err)
		}
	}
	values, present, err := (OptionsStore{Path: s.Path}).gameOptions()
	var allPresent [len(gameOptionKeys)]bool
	for i := range allPresent {
		allPresent[i] = true
	}
	if err != nil || values != want || present != allPresent {
		t.Fatal("cold preference read", values, present, err)
	}
	before, _ := os.ReadFile(s.Path)
	if err := s.setGameOption(ui.GameOption(255), 0); err == nil {
		t.Fatal("invalid option accepted")
	}
	if err := s.setGameOption(ui.GameOptionHealth, 2); err == nil {
		t.Fatal("invalid boolean accepted")
	}
	after, _ := os.ReadFile(s.Path)
	m, _ := s.readAll()
	if string(before) != string(after) || m["FutureOption"] != "unchanged" {
		t.Fatal("invalid write or unrelated key changed")
	}
}

func TestGameOptions1186QueueAndFailureKeepStoppedWorld(t *testing.T) {
	m := worldFixtureMap()
	mw := newMapWorld(mapload.FromALM(m), nil, nil, worldFixtureViewer(t, m))
	f := &FrontEnd{CampaignSession: CampaignSession{live: mw}, PersistenceContext: PersistenceContext{Options: OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}}}
	before := mw.world.Hash()
	for i, v := range (ui.GameOptionValues{0, 0, 0, 2, 2}) {
		if err := f.setGameOption(true, ui.GameOption(i), v); err != nil {
			t.Fatal(err)
		}
	}
	if mw.world.Hash() != before || f.gameOptionValues(true) != (ui.GameOptionValues{0, 0, 0, 2, 2}) {
		t.Fatal("settings stepped world or did not include queued selection")
	}
	want := []sim.Command{
		{Kind: sim.KindPlayerParameter, Player: sim.SelfSlot, X: int32(sim.PlayerParameterFormation), Y: 2},
		{Kind: sim.KindPlayerParameter, Player: sim.SelfSlot, X: int32(sim.PlayerParameterRetreat), Y: 2},
		{Kind: sim.KindPlayerParameter, Player: sim.SelfSlot, X: int32(sim.PlayerParameterAutoHealing), Y: 0},
	}
	if !reflect.DeepEqual(mw.pending, want) {
		t.Fatal("menu bypassed canonical command queue", mw.pending)
	}
	application, err := f.captureApplicationState(mw)
	if err != nil || application == nil || application.LocalOnly || application.View.ShowHealth || application.View.TimeFlow || application.View.FlyingHP || application.Original.Wimpy != 2 {
		t.Fatal("current application lost its settings", application, err)
	}
	back := newMapWorld(mapload.FromALM(m), nil, nil, worldFixtureViewer(t, m))
	wantView := back.view.SaveApplication()
	wantView.ShowHealth, wantView.FlyingHP, wantView.TimeFlow = false, false, false
	wantView.PeriodUS, wantView.Unpaced = application.View.PeriodUS, application.View.Unpaced
	application.View.ViewX += 9
	application.View.ViewY += 7
	application.View.PressedSpell = 16
	application.View.InventoryOpen = !wantView.InventoryOpen
	application.View.SpellBookOpen = !wantView.SpellBookOpen
	wantView = application.View
	if err := back.restoreApplicationState(application, false); err != nil || back.view.HealthBarsShown() || back.view.TimeFlow() {
		t.Fatal("local application did not restore", err)
	}
	gotView := back.view.SaveApplication()
	if !slices.Equal(gotView.Selection, wantView.Selection) {
		t.Fatal("current selection did not restore")
	}
	gotView.Selection, wantView.Selection = nil, nil
	if !reflect.DeepEqual(gotView, wantView) {
		t.Fatal("current application fields did not restore", gotView, wantView)
	}
	// A directory is a portable read failure. Never replace unreadable profile
	// contents with a partial option file, or apply the unpersisted setting.
	f.Options.Path = t.TempDir()
	for i := range (ui.GameOptionValues{}) {
		if err := f.setGameOption(true, ui.GameOption(i), 1); err == nil {
			t.Fatal("unreadable profile accepted")
		}
	}
	if !reflect.DeepEqual(mw.pending, want) || f.gameOptionValues(true) != (ui.GameOptionValues{0, 0, 0, 2, 2}) || mw.world.Hash() != before {
		t.Fatal("failed writer changed runtime state")
	}
	if n := mw.tick(); n != 3 || mw.world.FormationMode(sim.SelfSlot) != 1 {
		t.Fatal("resume did not apply selected formation", n)
	}
}

func TestGameOptions1186TownDoesNotChangeRetainedMission(t *testing.T) {
	m := worldFixtureMap()
	mw := newMapWorld(mapload.FromALM(m), nil, nil, worldFixtureViewer(t, m))
	f := &FrontEnd{CampaignSession: CampaignSession{live: mw}, PersistenceContext: PersistenceContext{Options: OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}}}
	before := mw.world.Hash()
	for i, v := range (ui.GameOptionValues{0, 0, 0, 0, 2}) {
		if err := f.setGameOption(false, ui.GameOption(i), v); err != nil {
			t.Fatal(err)
		}
	}
	if f.gameOptionValues(false) != (ui.GameOptionValues{0, 0, 0, 0, 2}) || len(mw.pending) != 0 || mw.world.Hash() != before || !mw.view.HealthBarsShown() || !mw.view.TimeFlow() {
		t.Fatal("town settings mutated the retained mission or read its old values")
	}
}

func TestGameOptions1186RejectMalformedSavedCommands(t *testing.T) {
	for _, orders := range [][]PendingGameOption{
		{{ui.GameOptionHealth, 1}}, {{ui.GameOptionFormation, 3}}, {{ui.GameOptionRetreat, -1}},
		make([]PendingGameOption, 4097),
	} {
		s := Snapshot{Residue: SnapshotResidue{PendingGameOptions: orders}}
		if _, err := EncodeSave(s, "invalid options"); err == nil {
			t.Fatal("invalid options encoded")
		}
		f := &FrontEnd{}
		if _, _, err := f.Restore(s); err == nil {
			t.Fatal("invalid options accepted by direct restore")
		}
	}
}
