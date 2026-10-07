package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func TestGraphics1190PreferencesAndFailureKeepWorld(t *testing.T) {
	m := worldFixtureMap()
	mw := newMapWorld(mapload.FromALM(m), nil, nil, worldFixtureViewer(t, m))
	f := &FrontEnd{CampaignSession: CampaignSession{live: mw}, PersistenceContext: PersistenceContext{Options: OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}}}
	if err := os.WriteFile(f.Options.Path, []byte("FutureOption=preserved\n"), 0644); err != nil {
		t.Fatal(err)
	}
	hash := mw.world.Hash()
	for _, change := range [][2]int{{int(ui.GameOptionSmoothing), 1}, {int(ui.GameOptionAnimation), 0}, {int(ui.GameOptionLighting), 1}} {
		if err := f.setGameOption(true, ui.GameOption(change[0]), change[1]); err != nil {
			t.Fatal(err)
		}
	}
	want := ui.GraphicsOptions{Smoothing: true, StaticObjects: true, DisableLighting: true}
	if f.graphics != want || mw.view.GraphicsOptions() != want || mw.world.Hash() != hash || len(mw.pending) != 0 || mw.applicationState != nil {
		t.Fatal("graphics changed game state or lost coupled flag", f.graphics)
	}
	cold := &FrontEnd{PersistenceContext: PersistenceContext{Options: f.Options}}
	cold.LoadOptions()
	if cold.graphics != want {
		t.Fatal("restart lost graphics choices", cold.graphics)
	}
	values, _ := cold.Options.readAll()
	if values["FutureOption"] != "preserved" || values["Lighting"] != "0" || values["Animation"] != "0" {
		t.Fatal("profile lost unrelated or coupled values", values)
	}
	f.Options.Path = t.TempDir()
	if err := f.setGameOption(true, ui.GameOptionAnimation, 1); err == nil || f.graphics != want || mw.view.GraphicsOptions() != want || mw.world.Hash() != hash {
		t.Fatal("failed write reached renderer/world", err)
	}
}

func TestGraphicsDefaultsAreHighestQualityAndStoredChoiceWins(t *testing.T) {
	best := ui.GraphicsOptions{}
	best.Smoothing = true
	path := filepath.Join(t.TempDir(), "options.txt")

	fresh := &FrontEnd{PersistenceContext: PersistenceContext{Options: OptionsStore{Path: path}}}
	fresh.LoadOptions()
	if fresh.graphics != best {
		t.Fatal("absent profile did not default to the highest-quality graphics", fresh.graphics)
	}
	if on, _ := fresh.Options.TextSmoothing(); !on || fresh.smoothingOff.text {
		t.Fatal("text smoothing default is not on")
	}
	if on, _ := fresh.Options.FrameSmoothing(); !on || fresh.smoothingOff.frame {
		t.Fatal("frame smoothing default is not on")
	}
	var shown ui.GameOptionValues = fresh.gameOptionValues(false)
	for o := ui.GameOptionSmoothing; o <= ui.GameOptionAnimation; o++ {
		if shown[o] != 1 {
			t.Fatal("Game Options screen shows a lower default", o, shown[o])
		}
	}

	if err := os.WriteFile(path, []byte("Shadows=0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	partial := &FrontEnd{PersistenceContext: PersistenceContext{Options: OptionsStore{Path: path}}}
	partial.LoadOptions()
	if want := (ui.GraphicsOptions{Smoothing: true, HideShadows: true}); partial.graphics != want {
		t.Fatal("unset keys did not default beside a stored one", partial.graphics)
	}

	if err := os.WriteFile(path, []byte("Smoothing=0\nShadows=0\nLighting=0\nAnimation=0\nTextSmoothing=0\nFrameSmoothing=0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	saved := &FrontEnd{PersistenceContext: PersistenceContext{Options: OptionsStore{Path: path}}}
	saved.LoadOptions()
	if want := (ui.GraphicsOptions{HideShadows: true, DisableLighting: true, StaticObjects: true}); saved.graphics != want {
		t.Fatal("stored lower graphics choice was overridden", saved.graphics)
	}
	if !saved.smoothingOff.text || !saved.smoothingOff.frame {
		t.Fatal("stored lower smoothing choice was overridden", saved.smoothingOff)
	}
}
