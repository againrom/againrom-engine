package game

import (
	"path/filepath"
	"testing"

	"againrom/pkg/ui"
)

// The loot-dot option is on in a profile that never stored it, persists when
// switched off, and survives a restart: a second front end reading the same
// profile file starts with it off. Loss controls: the fresh profile reads on,
// and switching it back on restores the stored default.
func TestLootDotsOptionPersistsAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "options.txt")
	first := &FrontEnd{PersistenceContext: PersistenceContext{Options: OptionsStore{Path: path}}}
	first.LoadOptions()
	if first.graphics.HideLootDots || first.gameOptionValues(false)[ui.GameOptionLootDots] != 1 {
		t.Fatal("a fresh profile does not show loot dots")
	}
	if err := first.setGameOption(false, ui.GameOptionLootDots, 0); err != nil {
		t.Fatal(err)
	}
	if !first.graphics.HideLootDots {
		t.Fatal("the running front end ignores the switch")
	}
	second := &FrontEnd{PersistenceContext: PersistenceContext{Options: OptionsStore{Path: path}}}
	second.LoadOptions()
	if !second.graphics.HideLootDots || second.gameOptionValues(false)[ui.GameOptionLootDots] != 0 {
		t.Fatal("the restarted front end lost the switch")
	}
	if err := second.setGameOption(false, ui.GameOptionLootDots, 1); err != nil {
		t.Fatal(err)
	}
	third := &FrontEnd{PersistenceContext: PersistenceContext{Options: OptionsStore{Path: path}}}
	third.LoadOptions()
	if third.graphics.HideLootDots {
		t.Fatal("switching the option back on did not persist")
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
