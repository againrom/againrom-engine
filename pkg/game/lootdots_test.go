package game

import (
	"path/filepath"
	"testing"

	"againrom/pkg/ui"
)

// The loot-dot option is off in a fresh profile (DIV-2954); switched on it
// survives a restart, and switched back off it reads off again.
func TestLootDotsOptionPersistsAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "options.txt")
	first := &FrontEnd{PersistenceContext: PersistenceContext{Options: OptionsStore{Path: path}}}
	first.LoadOptions()
	if !first.graphics.HideLootDots || first.gameOptionValues(false)[ui.GameOptionLootDots] != 0 {
		t.Fatal("a fresh profile shows loot dots")
	}
	if err := first.setGameOption(false, ui.GameOptionLootDots, 1); err != nil {
		t.Fatal(err)
	}
	if first.graphics.HideLootDots {
		t.Fatal("the running front end ignores the switch")
	}
	second := &FrontEnd{PersistenceContext: PersistenceContext{Options: OptionsStore{Path: path}}}
	second.LoadOptions()
	if second.graphics.HideLootDots || second.gameOptionValues(false)[ui.GameOptionLootDots] != 1 {
		t.Fatal("the restarted front end lost the switch")
	}
	if err := second.setGameOption(false, ui.GameOptionLootDots, 0); err != nil {
		t.Fatal(err)
	}
	third := &FrontEnd{PersistenceContext: PersistenceContext{Options: OptionsStore{Path: path}}}
	third.LoadOptions()
	if !third.graphics.HideLootDots {
		t.Fatal("switching the option back off did not persist")
	}
}

// A fresh profile's minimap draws no sack mark and keeps the original's
// marks only; a graphics change that writes the options file keeps it off.
func TestLootDotsAreOffByDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "options.txt")
	f := &FrontEnd{PersistenceContext: PersistenceContext{Options: OptionsStore{Path: path}}}
	f.LoadOptions()
	if !f.graphics.HideLootDots {
		t.Fatal("a fresh profile shows loot dots on the minimap")
	}
	if err := f.setGameOption(false, ui.GameOptionShadows, 0); err != nil {
		t.Fatal(err)
	}
	g := &FrontEnd{PersistenceContext: PersistenceContext{Options: OptionsStore{Path: path}}}
	g.LoadOptions()
	if !g.graphics.HideLootDots {
		t.Fatal("a graphics write switched loot dots on")
	}
}

// Changing a graphics option writes the stored loot-dot value unchanged.
func TestLootDotsOptionSurvivesAGraphicsWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "options.txt")
	f := &FrontEnd{PersistenceContext: PersistenceContext{Options: OptionsStore{Path: path}}}
	f.LoadOptions()
	if err := f.setGameOption(false, ui.GameOptionLootDots, 0); err != nil {
		t.Fatal(err)
	}
	if err := f.setGameOption(false, ui.GameOptionShadows, 0); err != nil {
		t.Fatal(err)
	}
	g := &FrontEnd{PersistenceContext: PersistenceContext{Options: OptionsStore{Path: path}}}
	g.LoadOptions()
	if !g.graphics.HideLootDots || !g.graphics.HideShadows {
		t.Fatalf("graphics after restart = %+v", g.graphics)
	}
}
