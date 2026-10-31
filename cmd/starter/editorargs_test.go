package main

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	basepkg "againrom/pkg/base"
)

// The map editor's arguments name the asset root alone, the map list check
// adds -list, and a detected base or enabled mods, which the game's arguments
// carry, reach neither.
func TestEditorArgumentsNameTheAssetRootAlone(t *testing.T) {
	for _, c := range []struct {
		name, ini string
		base      bool
	}{
		{"no base, no mods", "[starter]\nlast-base = en\n[bases]\nen = {root}\n", false},
		{"base and mods", "[starter]\nlast-base = en\n[bases]\nen = {root}\n[mods]\nenabled = alpha\n", true},
	} {
		r := newRig(t, c.ini)
		if c.base {
			writeModFolder(t, filepath.Join(r.dir, "mods"), "alpha", goodMod("alpha"))
			r.fakes.validRoots[r.root] = profileInfo(basepkg.ROM1EN, true)
		}
		exe := writeEditor(t, r)
		a := r.app()
		got, start, check, err := a.mapEditorPlan()
		if err != nil || got != exe {
			t.Fatalf("%s: plan %q, %v", c.name, got, err)
		}
		if want := []string{"-assets", r.root}; !reflect.DeepEqual(start, want) {
			t.Errorf("%s: editor arguments %q, want %q", c.name, start, want)
		}
		if want := []string{"-assets", r.root, "-list"}; !reflect.DeepEqual(check, want) {
			t.Errorf("%s: map list arguments %q, want %q", c.name, check, want)
		}
		if c.base {
			line, err := a.command()
			if err != nil || !strings.Contains(line, "-base rom1-en") || !strings.Contains(line, "-mods alpha") {
				t.Errorf("%s: the game's command line %q (%v) lacks the base or the mods", c.name, line, err)
			}
		}
	}
}
