package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"
	"strconv"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

var spellChoiceArms = []struct {
	opcode uint32
	slot   int
	types  []uint32
}{
	{21, 4, []uint32{5, 6, 5, 6, 1, 1}},
	{24, 3, []uint32{5, 6, 4, 1, 1}},
	{25, 0, []uint32{1, 1, 5, 6}},
	{29, 2, []uint32{5, 6, 1, 1}},
	{30, 1, []uint32{4, 1, 1}},
}

func spellChoiceEditor(t *testing.T, opcode, current uint32) (*ui.MapEditor, int) {
	t.Helper()
	for _, arm := range spellChoiceArms {
		if arm.opcode != opcode {
			continue
		}
		values := make([]uint32, len(arm.types))
		values[arm.slot] = current
		b := built(t, synth.ALMOptions{Width: 40, Height: 40, Type7Payload: goalScript([]scriptNode{
			{label: "Spell", opcode: opcode, id: 1, tags: arm.types, values: values},
		}, nil, nil)}, nil, nil).Bytes()
		e, path := openGoalMap(t, b)
		x := objectEdits(t, e).inspector
		rows := make(dbCollection, 31)
		for i := range rows {
			rows[i].name = fmt.Sprintf("Spell name %d", i)
		}
		x.table = &mapload.Table{Spells: rows}
		x.recordOptions = nil
		e = x.Editor()
		if err := e.Open(path); err != nil {
			t.Fatal(err)
		}
		e.Layout(1280, 800)
		closeMapPanel(t, e)
		e.Select(trigRecord(t, e, "Action", 0), false)
		return e, arm.slot
	}
	t.Fatal("unknown arm", opcode)
	return nil, 0
}

func spellFieldChoices(t *testing.T, e *ui.MapEditor, slot int) []ui.InspectionChoice {
	t.Helper()
	for _, f := range e.Document().Edits.Fields("Action", 0) {
		if f.Key == fmt.Sprintf("value%d", slot) {
			return f.Choices
		}
	}
	t.Fatal("missing spell field", slot)
	return nil
}

func TestScriptSpellChoicesRowsAndExclusions(t *testing.T) {
	for _, arm := range spellChoiceArms {
		t.Run(strconv.Itoa(int(arm.opcode)), func(t *testing.T) {
			e, slot := spellChoiceEditor(t, arm.opcode, 3)
			var want []ui.InspectionChoice
			for i := 1; i <= 28; i++ {
				if arm.opcode == 25 && i == 26 {
					continue
				}
				if arm.opcode == 29 && !slices.Contains([]int{3, 7, 8, 12, 17, 19}, i) {
					continue
				}
				want = append(want, ui.InspectionChoice{Value: int64(i), Label: fmt.Sprintf("%d: Spell name %d", i, i)})
			}
			got := spellFieldChoices(t, e, slot)
			if !slices.Equal(got, want) {
				t.Fatalf("spell rows = %v, want %v", got, want)
			}
			n := e.Document().Script.Actions[0]
			r, ok := e.FieldRect(parameterField(n, slot))
			trigClick(t, e, r, ok, "spell field")
			var labels []string
			for _, row := range want {
				labels = append(labels, row.Label)
			}
			if !slices.Equal(e.DropdownChoices(), labels) {
				t.Fatal("popup and E differ", e.DropdownChoices(), labels)
			}
			catalog := objectEdits(t, e).inspector.recordCatalog("spell")
			for _, row := range want {
				if !slices.Contains(catalog, row) {
					t.Fatal("recipe catalog differs", row)
				}
			}
		})
	}
}

func TestScriptSpellParameterFollowsPackedScalars(t *testing.T) {
	for _, tc := range []struct {
		opcode uint32
		types  [10]uint32
		slot   int
		want   uint32
	}{
		{21, [10]uint32{4, 5, 6, 5, 6, 7, 1}, 5, ui.ScriptSpellParameter},
		{24, [10]uint32{4, 5, 6, 7, 1}, 3, ui.ScriptSpellParameter},
		{25, [10]uint32{3, 4, 1, 1, 5, 6}, 2, ui.ScriptEntrySpellParameter},
		{29, [10]uint32{2, 5, 6, 7, 1}, 3, ui.ScriptAreaSpellParameter},
		{30, [10]uint32{4, 1, 1}, 1, ui.ScriptSpellParameter},
		{30, [10]uint32{4, 1, 1}, 0, 4},
		{30, [10]uint32{4, 1, 1}, 2, 1},
		{2, [10]uint32{1}, 0, ui.ScriptMessageParameter},
	} {
		if got := scriptParameterKind(false, tc.opcode, tc.types, tc.slot); got != tc.want {
			t.Errorf("action=%d slot=%d kind=%d want=%d", tc.opcode, tc.slot, got, tc.want)
		}
		if got := scriptParameterKind(true, tc.opcode, tc.types, tc.slot); got != tc.types[tc.slot] {
			t.Errorf("condition=%d slot=%d kind=%d", tc.opcode, tc.slot, got)
		}
	}
}

func TestScriptSpellUnknownNumberPreserved(t *testing.T) {
	for _, arm := range spellChoiceArms {
		for _, current := range []uint32{0, 29, 255, 257, 0xffffffff, 26} {
			t.Run(fmt.Sprintf("%d/%d", arm.opcode, current), func(t *testing.T) {
				e, slot := spellChoiceEditor(t, arm.opcode, current)
				before := bytes.Clone(e.Document().SourceBytes)
				n := e.Document().Script.Actions[0]
				kind := scriptParameterKind(false, n.Opcode, n.Types, slot)
				if current > 0x7fffffff {
					trigChoose(t, e, parameterField(n, slot), strconv.FormatInt(int64(int32(current)), 10))
				} else {
					chooseParameterValue(t, e, parameterField(n, slot), current, e.Document().Script.Targets[kind])
				}
				r, ok := e.PopupButtonRect(0)
				trigClick(t, e, r, ok, "Apply unchanged")
				if !bytes.Equal(before, e.Document().SourceBytes) || e.Dirty() {
					t.Fatal("current value changed bytes")
				}
				_ = e.Key("escape")
				scriptWindowSet(t, e, "Action", 0, fmt.Sprintf("value%d", slot), strconv.FormatInt(int64(int32(current)), 10))
				if !bytes.Equal(before, e.Document().SourceBytes) {
					t.Fatal("E current value changed bytes")
				}
			})
		}
	}
}

func TestScriptSpellPickWritesOnlyValueAndUndo(t *testing.T) {
	for _, arm := range spellChoiceArms {
		t.Run(strconv.Itoa(int(arm.opcode)), func(t *testing.T) {
			e, slot := spellChoiceEditor(t, arm.opcode, 3)
			before := bytes.Clone(e.Document().SourceBytes)
			n := e.Document().Script.Actions[0]
			trigChoose(t, e, parameterField(n, slot), "19: Spell name 19")
			if !bytes.Equal(before, e.Document().SourceBytes) {
				t.Fatal("pick wrote before Apply")
			}
			r, ok := e.PopupButtonRect(0)
			trigClick(t, e, r, ok, "Apply")
			after := bytes.Clone(e.Document().SourceBytes)
			if len(before) != len(after) || bytes.Equal(before, after) {
				t.Fatal("pick size or no write")
			}
			first := -1
			for i := range before {
				if before[i] != after[i] {
					first = i
					break
				}
			}
			if first < 0 || binary.LittleEndian.Uint32(before[first:]) != 3 || binary.LittleEndian.Uint32(after[first:]) != 19 {
				t.Fatal("value word", first)
			}
			want := bytes.Clone(before)
			binary.LittleEndian.PutUint32(want[first:], 19)
			if !bytes.Equal(want, after) {
				t.Fatal("bytes outside value changed")
			}
			if err := e.Key("undo"); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, e.Document().SourceBytes) {
				t.Fatal("Undo bytes")
			}
			scriptWindowSet(t, e, "Action", 0, fmt.Sprintf("value%d", slot), "19")
			if !bytes.Equal(after, e.Document().SourceBytes) {
				t.Fatal("E and popup write different bytes")
			}
		})
	}
}
