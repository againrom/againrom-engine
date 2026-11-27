package ui

import (
	"slices"
	"testing"

	"againrom/pkg/scriptprofile"
)

func TestWizardReadsOperationWordAndSlots(t *testing.T) {
	for _, ops := range []scriptprofile.Ops{scriptprofile.OpsFirst, scriptprofile.OpsSecond} {
		for _, id := range []uint32{10, 23} {
			op, ok := scriptprofile.OpRow(ops, false, id)
			if !ok {
				t.Fatalf("vocabulary %v has no action %d", ops, id)
			}
			op.Word = "changed action"
			at := len(op.Signature)
			op.Signature = append(slices.Clone(op.Signature), scriptprofile.Slot{Type: scriptStructure})
			items := opItems([]scriptprofile.Op{op})
			if len(items) != 1 {
				t.Fatalf("vocabulary %v action %d built %d items", ops, id, len(items))
			}
			it := items[0]
			types, _, _ := it.itemSlots(false)
			if it.labelWord != op.Word || types[at] != scriptStructure || len(it.params) != at+1 {
				t.Errorf("vocabulary %v action %d built word %q, types %v and %d fields", ops, id, it.labelWord, types, len(it.params))
			}
		}
	}
}

func TestWizardPresetsJoinOperationRows(t *testing.T) {
	for k, condition := range []bool{false, true} {
		items := wizardTables[scriptprofile.OpsFirst][k]
		if want := [2]int{35, 25}[k]; len(items) != want {
			t.Fatalf("condition %t: got %d items, want %d", condition, len(items), want)
		}
		for _, op := range scriptprofile.OpRows(scriptprofile.OpsFirst, condition) {
			for _, preset := range op.Presets {
				i := preset.Order - 1
				word := preset.Word
				if word == "" {
					word = op.Word
				}
				if i < 0 || i >= len(items) || items[i].opcode != op.ID || items[i].labelWord != word {
					t.Errorf("condition %t: missing %q (opcode %d) at index %d", condition, word, op.ID, i)
				}
			}
		}
	}
}

func TestUnitChangesItsCellIsOfferedForBothGames(t *testing.T) {
	for _, ops := range []scriptprofile.Ops{scriptprofile.OpsFirst, scriptprofile.OpsSecond} {
		at := slices.IndexFunc(wizardTables[ops][1], func(it wizItem) bool { return it.opcode == 0x4105 })
		if at < 0 {
			t.Fatalf("operation vocabulary %v does not offer departure", ops)
		}
		it := wizardTables[ops][1][at]
		if it.labelWord != "editor.wizard.c.leftcell" || len(it.params) != 1 || it.params[0].kind != wpUnit {
			t.Errorf("operation vocabulary %v departure row %q, %d fields", ops, it.labelWord, len(it.params))
		}
		teleport := slices.ContainsFunc(wizardTables[ops][0], func(it wizItem) bool { return it.opcode == 0x4106 })
		if teleport != (ops == scriptprofile.OpsFirst) {
			t.Errorf("operation vocabulary %v offers the cell teleport: %t", ops, teleport)
		}
	}
}

func TestWizardDialogueNodeUsesMapOperations(t *testing.T) {
	e, _ := wizardChoiceEditor(t)
	action := WizardNode{ID: 31}
	for _, tc := range []struct {
		file bool
		name string
		want WizardNode
	}{
		{true, "meeting", WizardNode{ID: 31, Opcode: 2, Label: "meeting", Types: [10]uint32{scriptInt, scriptInt}, Values: [10]uint32{0, 1}}},
		{false, "", WizardNode{ID: 31, Opcode: 2, Types: [10]uint32{scriptInt}}},
	} {
		if err := e.setDialogueNode(&action, tc.file, tc.name); err != nil || action != tc.want {
			t.Fatalf("dialogue file %v: action %+v, error %v", tc.file, action, err)
		}
	}
	e.doc.Script.Limits = scriptprofile.Second
	before := action
	if err := e.setDialogueNode(&action, true, "meeting"); err == nil || action != before {
		t.Errorf("converted the second game's event to %+v, error %v", action, err)
	}
}
