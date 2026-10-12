package game

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// autoCastLevelMission opens a mission on either game through the ordinary
// App route and returns it settled.
func autoCastLevelMission(t *testing.T, f *FrontEnd) *ui.App {
	t.Helper()
	f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	f.LoadOptions()
	if f.Archives.Game() == base.GameROM2 {
		app := f.App("automatic-casting level")
		app.Layout(1024, 768)
		enterSecondCampaignMission(t, app)
		for range 20 {
			if err := app.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
		}
		dismissSecondMissionDialogue(t, app, f.live)
		return app
	}
	party := f.ChargenParty(ui.ChargenResult{Name: "Autocast witness", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	app := f.App("automatic-casting level")
	app.SetCutscenes(nil)
	if err := app.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	app.Layout(640, 480)
	return app
}

func autoCastLevelFront(t *testing.T) *FrontEnd {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	if f.Archives.Game() == base.GameROM2 {
		f.MusicPlayer = &secondTestMusic{}
	}
	return f
}

// autoCastColdLoad opens raw in a fresh process through the original-SAV
// restore route.
func autoCastColdLoad(t *testing.T, raw []byte) *FrontEnd {
	t.Helper()
	cold := autoCastLevelFront(t)
	cold.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	cold.LoadOptions()
	open, town, err := cold.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal("cold LOAD", town, err)
	}
	app := cold.App("automatic-casting cold LOAD")
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	return cold
}

// autoCastWithoutCarrier is raw with the settings removed from the action
// continuation: the loss control for the SAV carrier.
func autoCastWithoutCarrier(t *testing.T, raw []byte) []byte {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a == nil || len(a.Actions.AutoCast) == 0 {
		t.Fatal("SAV carries no automatic-casting setting", err)
	}
	a.Actions.AutoCast = nil
	payload, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeActions(&doc.State, payload); err != nil {
		t.Fatal(err)
	}
	out, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// TestReleaseAutoCastLevelsSurviveSAVEAndColdLOAD is the release witness of
// the one automatic-casting setting on each game's root: the Game Options
// row shows the install's or the engine's caption, each level set through it
// reaches the World, and an ordinary SAV and a cold LOAD keep the setting with
// the World hash and the next 32 ticks equal. Removing the setting from the
// SAV's action continuation loses it (loss control).
func TestReleaseAutoCastLevelsSurviveSAVEAndColdLOAD(t *testing.T) {
	f := autoCastLevelFront(t)
	app := autoCastLevelMission(t, f)
	openMissionGameMenu(t, app)
	if err := app.HeadlessGameMenuAction("game-options"); err != nil {
		t.Fatal(err)
	}
	words := app.GameOptionWords()
	label, choice := words.Labels[ui.GameOptionAutoCast], words.Choices[ui.GameOptionAutoCast][f.gameOptionValues(true)[ui.GameOptionAutoCast]]
	caption := ""
	for _, row := range app.HeadlessRows() {
		if row.Text == label+": "+choice {
			caption = row.Text
		}
	}
	if label == "" || choice == "" || caption == "" || strings.Contains(caption, "::") {
		t.Fatalf("Game Options has no automatic-casting row %q: %q", label, caption)
	}
	ascii := strings.IndexFunc(label, func(r rune) bool { return r > 0x7e }) < 0
	if f.Archives.Game() != base.GameROM2 && ascii && label != "AutoCast mode" {
		t.Fatalf("first game's caption %q is not the engine word", label)
	}
	if f.Archives.Game() == base.GameROM2 {
		table := LoadTextTable(f.Archives.Containers, DialogsTextPath, f.textCode())
		for i, want := range []string{words.Labels[ui.GameOptionAutoCast], words.Choices[ui.GameOptionAutoCast][1], words.Choices[ui.GameOptionAutoCast][2], words.Choices[ui.GameOptionAutoCast][3]} {
			line, ok := table.At(170 + i)
			if i == 0 {
				line = strings.TrimSuffix(strings.TrimSpace(line), ":")
			}
			if !ok || line != want || line == "" {
				t.Fatalf("caption %d is %q, the install's dialogs line %d is %q", i, want, 170+i, line)
			}
		}
	}
	column := f.optionColumn()
	if start := f.gameOptionValues(true)[ui.GameOptionAutoCast]; start != ui.GameOptionAutoCast.Default(column) || f.live.world.AutoCastStored(sim.SelfSlot) {
		t.Fatalf("a new game starts at choice %d, stored %v", start, f.live.world.AutoCastStored(sim.SelfSlot))
	}
	t.Logf("%s root: row %q, captions %q", f.Archives.Game(), caption, words.Choices[ui.GameOptionAutoCast])
	if err := app.HeadlessGameMenuAction("page-return"); err != nil {
		t.Fatal(err)
	}
	for tries := 0; app.Screen() != ui.ScreenMap && tries < 4; tries++ {
		if err := app.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
	}
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	// Each offered level away from the start, then the start again: Off on
	// the first game, Average on the second.
	var levels []int
	for _, level := range ui.GameOptionAutoCast.Offers(column) {
		if level != ui.GameOptionAutoCast.Default(column) {
			levels = append(levels, level)
		}
	}
	levels = append(levels, ui.GameOptionAutoCast.Default(column))
	for _, level := range levels {
		if err := f.setGameOption(true, ui.GameOptionAutoCast, level); err != nil {
			t.Fatal(err)
		}
		f.live.tick()
		want := f.live.world.AutoCast(sim.SelfSlot)
		if want.Level() != sim.AutoCastLevel(level) || f.gameOptionValues(true)[ui.GameOptionAutoCast] != level {
			t.Fatalf("level %d applied as %#x", level, want)
		}
		name, err := save(true)
		if err != nil || filepath.Ext(name) != ".sav" {
			t.Fatal("ordinary SAV", name, err)
		}
		raw, err := store.Read(name)
		if err != nil {
			t.Fatal(err)
		}
		cold := autoCastColdLoad(t, raw)
		if got := cold.live.world.AutoCast(sim.SelfSlot); got != want || cold.live.world.Hash() != f.live.world.Hash() || cold.gameOptionValues(true)[ui.GameOptionAutoCast] != level {
			t.Fatalf("level %d: cold LOAD reads %#x, want %#x; hash equal %v", level, got, want, cold.live.world.Hash() == f.live.world.Hash())
		}
		for i := range 32 {
			f.live.tick()
			cold.live.tick()
			if f.live.world.Hash() != cold.live.world.Hash() {
				t.Fatalf("level %d: tick %d after cold LOAD differs", level, i)
			}
		}
		if !f.live.world.AutoCastStored(sim.SelfSlot) {
			t.Logf("level %d: setting %#x unstored; SAV %d bytes; cold LOAD and 32 ticks equal", level, want, len(raw))
			continue
		}
		loss := autoCastColdLoad(t, autoCastWithoutCarrier(t, raw))
		if loss.live.world.AutoCastStored(sim.SelfSlot) {
			t.Fatalf("level %d: the loss control still carries the setting", level)
		}
		t.Logf("level %d: setting %#x; SAV %d bytes; cold LOAD and 32 ticks equal; without the carrier the cold World reads %#x", level, want, len(raw), loss.live.world.AutoCast(sim.SelfSlot))
	}
}
