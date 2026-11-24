//go:build release

package game

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
				host := d.Edits.(*mapInspectionEdits)
				key, want := "value"+string(rune('0'+slot)), int64(n.Value[slot])+1
				if scriptAddressesVariable(condition, n.Opcode) && slot == 0 {
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

// A trigger's name, once flag and first comparison and a node's label edit,
// save and reload with only those changed; a slot of type 9 and an opcode
// change are refused with their one line; a trigger moves in the stored
// order.
func TestReleaseSecondMapTriggerFieldsAndLabelsEdit(t *testing.T) {
	x := installedInspector(t)
	sources, raws := installedSecondMaps(t, x)
	edited, unread, moved := 0, 0, 0
	for i, source := range sources {
		sc := secondScriptOf(t, raws[i])
		d, err := x.Open(source)
		if err != nil {
			t.Fatal(err)
		}
		host := d.Edits.(*mapInspectionEdits)
		for index, n := range sc.Conditions {
			if n.Type[0] == 9 {
				want := fmt.Sprintf(x.word("editor.map.secondtype"), 9)
				if _, err := host.SetField(ui.KindCondition, index, "value0", int64(n.Value[0])+1); err == nil || err.Error() != want {
					t.Fatalf("%s: type-9 slot not refused with its line: %v", source, err)
				}
				if _, err := host.SetField(ui.KindCondition, index, "opcode", 5); err == nil || err.Error() != x.word("editor.map.secondscript") {
					t.Fatalf("%s: opcode change not refused with its line: %v", source, err)
				}
				unread++
			}
		}
		if len(sc.Triggers) > 0 && len(sc.Actions) > 0 && edited < 3 {
			tr := sc.Triggers[0]
			edit := []ui.InspectionFieldValue{{Key: "name", Text: true, TextValue: "Edited trigger"}, {Key: "once", Value: int64(1 - min(tr.Once, 1))},
				{Key: scriptprofile.CmpKey(0), Value: int64((tr.Cmp[0] + 1) % 6)}}
			if _, err := host.SetFields(ui.KindTrigger, 0, edit); err != nil {
				t.Fatalf("%s: trigger edit: %v", source, err)
			}
			if _, err := host.SetFields(ui.KindAction, 0, []ui.InspectionFieldValue{{Key: "name", Text: true, TextValue: "Edited node"}}); err != nil {
				t.Fatalf("%s: node label: %v", source, err)
			}
			target, err := host.SaveAs(filepath.Join(t.TempDir(), "edited.alm"))
			if err != nil {
				t.Fatal(source, err)
			}
			saved, err := os.ReadFile(target)
			if err != nil {
				t.Fatal(err)
			}
			got := secondScriptOf(t, saved)
			want := tr
			want.Name, want.Once, want.Cmp[0] = "Edited trigger", uint32(1-min(tr.Once, 1)), (tr.Cmp[0]+1)%6
			wantAction := sc.Actions[0]
			wantAction.Label = "Edited node"
			if !reflect.DeepEqual(got.Triggers[0], want) || !reflect.DeepEqual(got.Triggers[1:], sc.Triggers[1:]) || !reflect.DeepEqual(got.Actions[0], wantAction) ||
				!reflect.DeepEqual(got.Actions[1:], sc.Actions[1:]) || !reflect.DeepEqual(got.Conditions, sc.Conditions) || len(saved) != len(raws[i]) {
				t.Fatalf("%s: the edit did not reload as made: %+v", source, got.Triggers[0])
			}
			edited++
			continue
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
	if edited == 0 || unread == 0 || moved == 0 {
		t.Fatalf("edited %d, type-9 refusals %d, moved %d", edited, unread, moved)
	}
	t.Logf("PASS: %d trigger and label edits saved and reloaded, %d type-9 refusals, %d trigger moves (%s)", edited, unread, moved, installLanguage(x))
}

// Every operation number the installed maps hold is shown in words, and on
// its first node every typed slot but type 9 takes a write; a number slot
// written one higher saves and reloads with only that word changed.
func TestReleaseSecondMapEveryOperationFamilyEdits(t *testing.T) {
	x := installedInspector(t)
	sources, raws := installedSecondMaps(t, x)
	type family struct {
		condition bool
		opcode    uint32
	}
	done := map[family]bool{}
	for i, source := range sources {
		sc := secondScriptOf(t, raws[i])
		for _, condition := range []bool{true, false} {
			nodes, kind := sc.Actions, ui.KindAction
			if condition {
				nodes, kind = sc.Conditions, ui.KindCondition
			}
			for index, n := range nodes {
				f := family{condition, n.Opcode}
				if done[f] {
					continue
				}
				done[f] = true
				t.Run(fmt.Sprintf("%v-%d", condition, n.Opcode), func(t *testing.T) {
					d, err := x.Open(source)
					if err != nil {
						t.Fatal(err)
					}
					shown := d.Script.Actions
					if condition {
						shown = d.Script.Conditions
					}
					if line := d.Records[shown[index].Record].Lines[0].Text; strings.Contains(line, "not decoded") {
						t.Errorf("%s node %d: %q", source, index, line)
					}
					host, number := d.Edits.(*mapInspectionEdits), -1
					for s, typ := range n.Type {
						if typ == 0 {
							continue
						}
						_, err := host.SetField(kind, index, fmt.Sprintf("value%d", s), int64(n.Value[s]))
						if (err == nil) == (typ == 9) {
							t.Fatalf("%s node %d slot %d type %d: write error %v", source, index, s, typ, err)
						}
						if typ == mapload.ScriptInt && number < 0 && !scriptAddressesVariable(condition, n.Opcode) {
							number = s
						}
					}
					if number < 0 {
						return
					}
					if _, err := host.SetField(kind, index, fmt.Sprintf("value%d", number), int64(n.Value[number])+1); err != nil {
						t.Fatalf("%s node %d slot %d: %v", source, index, number, err)
					}
					target, err := host.SaveAs(filepath.Join(t.TempDir(), "edited.alm"))
					if err != nil {
						t.Fatal(err)
					}
					saved, err := os.ReadFile(target)
					if err != nil {
						t.Fatal(err)
					}
					got, want := secondScriptOf(t, saved), n
					gotNodes := got.Actions
					if condition {
						gotNodes = got.Conditions
					}
					want.Value[number]++
					if !reflect.DeepEqual(gotNodes[index], want) || differs(saved, raws[i]) > 4 {
						t.Fatalf("%s node %d: reloaded %+v, %d bytes differ", source, index, gotNodes[index], differs(saved, raws[i]))
					}
				})
			}
		}
	}
	for f := range done {
		if !mapload.ScriptOperationDecoded(scriptprofile.OpsSecond, f.condition, f.opcode) {
			t.Errorf("operation %+v is not decoded", f)
		}
	}
	t.Logf("PASS: %d operation families (%s)", len(done), installLanguage(x))
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
