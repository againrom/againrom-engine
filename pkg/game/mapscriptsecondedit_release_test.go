//go:build release

package game

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/scriptprofile"
	"againrom/pkg/ui"
)

func secondScriptOf(t *testing.T, data []byte) alm.Script {
	t.Helper()
	m, err := alm.OpenROM2(data)
	if err != nil {
		t.Fatal(err)
	}
	sc, err := m.Script()
	if err != nil {
		t.Fatal(err)
	}
	return sc
}

// differs counts the bytes where a and b disagree.
func differs(a, b []byte) (n int) {
	for i := range a {
		if a[i] != b[i] {
			n++
		}
	}
	return n
}

// One integer parameter of a decoded action and one of a decoded check are
// edited, saved, reloaded and compared; only that parameter's word changes.
func TestReleaseSecondMapParameterEditSavesAndReloads(t *testing.T) {
	x := installedInspector(t)
	sources, raws := installedSecondMaps(t, x)
	tried := map[bool]int{}
	for i, source := range sources {
		sc := secondScriptOf(t, raws[i])
		for _, condition := range []bool{false, true} {
			if tried[condition] >= 3 {
				continue
			}
			nodes, kind := sc.Actions, ui.KindAction
			if condition {
				nodes, kind = sc.Conditions, ui.KindCondition
			}
			for index, n := range nodes {
				slot := -1
				for s, tag := range n.Type {
					if tag == mapload.ScriptInt && int32(n.Value[s]) >= 0 && n.Value[s] < 400 {
						slot = s
						break
					}
				}
				if slot < 0 || n.Opcode == mapload.ScriptConstant || !mapload.ScriptOperationDecoded(scriptprofile.OpsSecond, condition, n.Opcode) {
					continue
				}
				d, err := x.Open(source)
				if err != nil {
					t.Fatal(err)
				}
				host := d.Edits.(secondEdits)
				key, want := "value"+string(rune('0'+slot)), int64(n.Value[slot])+1
				if registerOp := secondRegisterOps[[2]uint32{map[bool]uint32{false: 0, true: 1}[condition], n.Opcode}]; registerOp && slot == 0 {
					if _, err := host.SetField(kind, index, key, scriptprofile.SecondRegisters); err == nil {
						t.Fatalf("%s node %d: register 1000 accepted", source, index)
					}
				}
				if _, err := host.SetField(kind, index, key, want); err != nil {
					t.Fatalf("%s node %d: %v", source, index, err)
				}
				if !host.Dirty() || !host.CanUndo() {
					t.Fatalf("%s: edit left no history", source)
				}
				target, err := host.SaveAs(filepath.Join(t.TempDir(), "edited.alm"))
				if err != nil {
					t.Fatal(source, err)
				}
				saved, err := os.ReadFile(target)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := x.Open(target); err != nil {
					t.Fatalf("%s: the saved map does not reload: %v", source, err)
				}
				got := secondScriptOf(t, saved)
				gotNodes := got.Actions
				if condition {
					gotNodes = got.Conditions
				}
				if int64(gotNodes[index].Value[slot]) != want {
					t.Fatalf("%s: edit not in the saved map", source)
				}
				if len(saved) != len(raws[i]) || differs(raws[i], saved) > 2 {
					t.Fatalf("%s: %d bytes differ, want the parameter word only", source, differs(raws[i], saved))
				}
				tried[condition]++
				break
			}
		}
	}
	if tried[false] == 0 || tried[true] == 0 {
		t.Fatalf("edits tried: actions %d, checks %d", tried[false], tried[true])
	}
	t.Logf("PASS: %d action and %d check parameter edits saved, reloaded and compared (%s)", tried[false], tried[true], installLanguage(x))
}

// A not-decoded operation, a rename, an opcode change and an added node are
// refused with their one line; a trigger moves in the stored order.
func TestReleaseSecondMapOtherScriptEditsAreRefused(t *testing.T) {
	x := installedInspector(t)
	sources, raws := installedSecondMaps(t, x)
	undecoded, moved := 0, 0
	for i, source := range sources {
		sc := secondScriptOf(t, raws[i])
		d, err := x.Open(source)
		if err != nil {
			t.Fatal(err)
		}
		host := d.Edits.(secondEdits)
		if len(sc.Triggers) > 0 {
			if _, err := host.SetFields(ui.KindTrigger, 0, []ui.InspectionFieldValue{{Key: "name", Text: true, TextValue: "x"}}); err == nil || err.Error() != x.word("editor.map.secondscript") {
				t.Fatalf("%s: rename not refused with its line: %v", source, err)
			}
		}
		for index, n := range sc.Conditions {
			if mapload.ScriptOperationDecoded(scriptprofile.OpsSecond, true, n.Opcode) {
				if _, err := host.SetField(ui.KindCondition, index, "opcode", 5); err == nil || err.Error() != x.word("editor.map.secondscript") {
					t.Fatalf("%s: opcode change not refused with its line: %v", source, err)
				}
				continue
			}
			if undecoded < 3 {
				if _, err := host.SetField(ui.KindCondition, index, "value0", 1); err == nil {
					t.Fatalf("%s: undecoded check %d took an edit", source, index)
				}
				undecoded++
			}
		}
		if len(sc.Triggers) >= 2 && moved < 3 {
			if _, err := host.SetField(ui.KindTrigger, 0, "order", 1); err != nil {
				t.Fatalf("%s: %v", source, err)
			}
			target, err := host.SaveAs(filepath.Join(t.TempDir(), "moved.alm"))
			if err != nil {
				t.Fatal(source, err)
			}
			saved, err := os.ReadFile(target)
			if err != nil {
				t.Fatal(err)
			}
			got := secondScriptOf(t, saved)
			if !reflect.DeepEqual(got.Triggers[1], sc.Triggers[0]) || !reflect.DeepEqual(got.Triggers[0], sc.Triggers[1]) || !reflect.DeepEqual(got.Actions, sc.Actions) {
				t.Fatalf("%s: the trigger did not move whole", source)
			}
			moved++
		}
	}
	if undecoded == 0 || moved == 0 {
		t.Fatalf("undecoded %d, moved %d", undecoded, moved)
	}
	t.Logf("PASS: %d undecoded refusals, %d trigger moves (%s)", undecoded, moved, installLanguage(x))
}

// The first game's trigger save is as before.
func TestReleaseFirstMapTriggerEditSaves(t *testing.T) {
	x := installedInspector(t)
	if x.installGame() == base.GameROM2 {
		t.Skip("requires a first-game install")
	}
	for _, entry := range x.Maps {
		source := ui.MapSourceKey(ui.SourceLoose, entry.Source)
		if entry.FromArchive {
			source = ui.MapSourceKey(ui.SourceScenario, entry.Source)
		}
		d, err := x.Open(source)
		if err != nil || d.Script == nil || len(d.Script.Triggers) == 0 {
			continue
		}
		tr := d.Script.Triggers[0]
		want := int64(1 - tr.Once)
		if _, err := d.Edits.SetField(ui.KindTrigger, 0, "once", want); err != nil {
			continue
		}
		target, err := d.Edits.SaveAs(filepath.Join(t.TempDir(), "edited.alm"))
		if err != nil {
			t.Fatal(source, err)
		}
		saved, err := os.ReadFile(target)
		if err != nil {
			t.Fatal(err)
		}
		m, err := alm.Open(saved)
		if err != nil {
			t.Fatal(err)
		}
		sc, err := m.Script()
		if err != nil || int64(sc.Triggers[0].Once) != want {
			t.Fatalf("%s: trigger edit not in the saved map: %v", source, err)
		}
		t.Logf("PASS: first-game trigger edit saved and reloaded on %s", source)
		return
	}
	t.Fatal("no first-game map took a trigger edit")
}
