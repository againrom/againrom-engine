package game

import (
	"bytes"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

// installedEditorMap is the installed map the editor's byte test opens. It
// holds a record of every kind the editor edits (units, structures, ground
// loot, script triggers, conditions and actions, players and enchantments)
// and is the installed map with the most units of those that hold them all.
const installedEditorMap = "scenario/90.alm"

// TestReleaseMapEditorSavesAnInstalledMapKeepingItsUntouchedBytes opens one
// installed map, saves it untouched, then edits one field of a unit, a
// structure and a player and saves again. The untouched save is the installed
// bytes; the edited save is the bytes mapedit writes for the same fields on the
// installed bytes, and each edit moves no byte outside its field.
func TestReleaseMapEditorSavesAnInstalledMapKeepingItsUntouchedBytes(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	keys, maps := installedMapBytes(t, x)
	var raw []byte
	for n, key := range keys {
		if key == installedEditorMap {
			raw = maps[n]
		}
	}
	if raw == nil {
		t.Fatalf("%s is not an installed map", installedEditorMap)
	}
	m, err := alm.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	script, err := m.Script()
	if err != nil {
		t.Fatal(err)
	}
	for kind, count := range map[string]int{
		"units": len(m.Units), "structures": len(m.Objects), "loot": len(m.LootSection.Body),
		"triggers": len(script.Triggers), "conditions": len(script.Conditions), "actions": len(script.Actions),
		"players": len(m.Groups), "enchantments": len(m.Enchantments),
	} {
		if count == 0 {
			t.Fatalf("%s holds no %s: pick another installed map", installedEditorMap, kind)
		}
	}

	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open(installedEditorMap); err != nil {
		t.Fatal(err)
	}
	closeMapPanel(t, e)
	if e.Dirty() || !bytes.Equal(e.Document().SourceBytes, raw) {
		t.Fatal("opening the map changed it")
	}
	if _, _, saved := areaSaveAs(t, e, "untouched.alm"); !bytes.Equal(saved, raw) {
		t.Fatal("Save As changed the untouched installed bytes")
	}

	witness, err := mapedit.New(raw)
	if err != nil {
		t.Fatal(err)
	}
	edits := objectEdits(t, e)
	current := raw
	for _, c := range []struct {
		shown ui.RecordKind
		kind  mapedit.Kind
		count int
	}{{ui.KindUnit, mapedit.KindUnit, len(m.Units)}, {ui.KindStructure, mapedit.KindStructure, len(m.Objects)}, {ui.KindPlayer, mapedit.KindPlayer, len(m.Groups)}} {
		done := false
		for i := 0; i < c.count && !done; i++ {
			fields, err := witness.Fields(c.kind, i)
			if err != nil {
				t.Fatal(err)
			}
			for _, f := range fields {
				for _, v := range []int64{f.Min, f.Min + 1} {
					d, err := edits.SetField(c.shown, i, f.Key, v)
					if err != nil || bytes.Equal(current, d.SourceBytes) {
						continue
					}
					if err := witness.SetField(c.kind, i, f.Key, v); err != nil {
						t.Fatalf("%s %s: the editor wrote %d, mapedit refuses it: %v", c.shown, f.Key, v, err)
					}
					changed := 0
					for n := range current {
						if current[n] != d.SourceBytes[n] {
							changed++
						}
					}
					if len(current) != len(d.SourceBytes) || changed > f.Width {
						t.Fatalf("%s %s: moved %d bytes of %d, want at most %d", c.shown, f.Key, changed, len(current), f.Width)
					}
					current = d.SourceBytes
					done = true
					break
				}
				if done {
					break
				}
			}
		}
		if !done {
			t.Fatalf("no %s record took an edit", c.shown)
		}
	}
	if !bytes.Equal(current, witness.Bytes()) {
		t.Fatal("the editor's bytes differ from mapedit's for the same three fields")
	}
	_, _, edited := areaSaveAs(t, e, "edited.alm")
	if !bytes.Equal(edited, witness.Bytes()) {
		t.Fatal("the saved file differs from the edited bytes")
	}
}
