package game

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

// secondNewNodes is the host of the fixture second-game map with one new
// check (65538, a number 0) and one new action (2, a number 1).
func secondNewNodes(t *testing.T) *mapInspectionEdits {
	t.Helper()
	host := secondAuthorHost(t)
	for _, kind := range []ui.RecordKind{ui.KindCondition, ui.KindAction} {
		if _, err := host.AddRecord(ui.InspectionNewRecord{Kind: kind}); err != nil {
			t.Fatalf("a new %v: %v", kind, err)
		}
	}
	return host
}

// A second-game node's operation changes as the trigger window sends it:
// the label, the slot types, the values, the operation last. Check 21 takes
// a type-9 identity of a type-4 object of the map (R2-ENGINE-1725); action
// 24 takes the vector 5, 6, 4, 1, 1: the caster's cell, the unit, the spell
// and the power (R2-ENGINE-1742). Both decode back as written, keep their id
// and zero in word +0x48 and the texts (R2-ENGINE-1728).
func TestSecondMapNodeOperationChangesToCheckAndActionOfTheClaims(t *testing.T) {
	host := secondNewNodes(t)
	check := []ui.InspectionFieldValue{{Key: "type0", Value: 9}, {Key: "value0", Value: 1}, {Key: "opcode", Value: 21}}
	if _, err := host.SetFields(ui.KindCondition, 0, check); err != nil {
		t.Fatalf("check 65538 to check 21: %v", err)
	}
	cast := []ui.InspectionFieldValue{{Key: "name", Text: true, TextValue: "Curse"},
		{Key: "type0", Value: 5}, {Key: "type1", Value: 6}, {Key: "type2", Value: 4}, {Key: "type3", Value: 1}, {Key: "type4", Value: 1},
		{Key: "value0", Value: 3}, {Key: "value1", Value: 4}, {Key: "value2", Value: 10001}, {Key: "value3", Value: 6}, {Key: "value4", Value: 100},
		{Key: "opcode", Value: 24}}
	if _, err := host.SetFields(ui.KindAction, 0, cast); err != nil {
		t.Fatalf("action 2 to action 24: %v", err)
	}
	sc, m := secondAuthorScript(t, host)
	c, a := sc.Conditions[0], sc.Actions[0]
	if c.Opcode != 21 || c.ID != 1 || c.Type != [10]uint32{9} || c.Value != [10]uint32{1} {
		t.Fatalf("the check reads %+v", c)
	}
	if a.Opcode != 24 || a.ID != 1 || a.Label != "Curse" || a.Type != [10]uint32{5, 6, 4, 1, 1} || a.Value != [10]uint32{3, 4, 10001, 6, 100} {
		t.Fatalf("the action reads %+v", a)
	}
	secondNewNodeZeroes(t, m, mapedit.KindCondition, 0)
	secondNewNodeZeroes(t, m, mapedit.KindAction, 0)

	before := string(host.model.Bytes())
	for _, c := range []struct {
		name   string
		kind   ui.RecordKind
		values []ui.InspectionFieldValue
	}{
		{"an object identity the map does not hold", ui.KindCondition, []ui.InspectionFieldValue{{Key: "value0", Value: 65535}}},
		{"a type vector that is not the signature", ui.KindAction, []ui.InspectionFieldValue{{Key: "type2", Value: 1}, {Key: "value2", Value: 7}, {Key: "opcode", Value: 21}}},
		{"a slot of a new type with no value", ui.KindCondition, []ui.InspectionFieldValue{{Key: "type0", Value: 4}, {Key: "opcode", Value: 5}}},
		{"an operation the table does not hold", ui.KindAction, []ui.InspectionFieldValue{{Key: "opcode", Value: 7}}},
	} {
		if _, err := host.SetFields(c.kind, 0, c.values); err == nil {
			t.Errorf("%s is written", c.name)
		}
	}
	if string(host.model.Bytes()) != before {
		t.Fatal("a refused operation change moved the map")
	}
}

// A second-game node does not change to an operation whose slots the table
// does not state, check 27 or check 65538: the change is refused in the
// second game's words and the map stays as it was. A node of such an
// operation left as it is still takes its label.
func TestSecondMapNodeOperationRefusesAnOperationWithNoStatedSlots(t *testing.T) {
	host := secondNewNodes(t)
	if _, err := host.SetFields(ui.KindCondition, 0, []ui.InspectionFieldValue{{Key: "type0", Value: 9}, {Key: "value0", Value: 1}, {Key: "opcode", Value: 21}}); err != nil {
		t.Fatalf("check 65538 to check 21: %v", err)
	}
	before := string(host.model.Bytes())
	for _, opcode := range []int64{27, 0x10002} {
		_, err := host.SetFields(ui.KindCondition, 0, []ui.InspectionFieldValue{{Key: "opcode", Value: opcode}})
		if want := fmt.Sprintf(host.inspector.word("editor.map.secondunstated"), opcode); err == nil || err.Error() != want {
			t.Errorf("check 21 to %d: %v, want %q", opcode, err, want)
		}
	}
	if string(host.model.Bytes()) != before {
		t.Fatal("a refused operation change moved the map")
	}
	fresh := secondNewNodes(t)
	if _, err := fresh.SetFields(ui.KindCondition, 0, []ui.InspectionFieldValue{{Key: "name", Text: true, TextValue: "Limit"}}); err != nil {
		t.Fatalf("the label of a check 65538: %v", err)
	}
	if sc, _ := secondAuthorScript(t, fresh); sc.Conditions[0].Opcode != 0x10002 || sc.Conditions[0].Label != "Limit" || sc.Conditions[0].Type != [10]uint32{1} {
		t.Fatalf("the check reads %+v", sc.Conditions[0])
	}
}

// The trigger wizard's row writes a map register in the trigger window's
// words: a register past the second game's 1000, or below 0, of action 3,
// action 8 or check 19 is refused as the trigger window refuses it, and the
// map stays as it was.
func TestSecondMapWizardRegisterTakesTheTriggerWindowCheck(t *testing.T) {
	host := secondNewNodes(t)
	limit := ui.WizardNode{Opcode: 0x10002, Types: [10]uint32{1}, Values: [10]uint32{0}}
	object := ui.WizardPair{Left: ui.WizardNode{Opcode: 21, Types: [10]uint32{9}, Values: [10]uint32{1}}, Right: limit, Cmp: 5}
	set := func(opcode, register uint32) ui.WizardNode {
		if opcode == 8 {
			return ui.WizardNode{Opcode: 8, Types: [10]uint32{1}, Values: [10]uint32{register}}
		}
		return ui.WizardNode{Opcode: opcode, Types: [10]uint32{1, 1}, Values: [10]uint32{register, 7}}
	}
	register := func(value uint32) ui.WizardPair {
		return ui.WizardPair{Left: ui.WizardNode{Opcode: 19, Types: [10]uint32{1}, Values: [10]uint32{value}}, Right: limit, Cmp: 5}
	}
	before := string(host.model.Bytes())
	minus3 := uint32(0xFFFFFFFD)
	for _, c := range []struct {
		name  string
		pair  ui.WizardPair
		act   ui.WizardNode
		value int64
	}{
		{"action 3 register 5000", object, set(3, 5000), 5000},
		{"action 3 register -3", object, set(3, minus3), -3},
		{"action 8 register 1000", object, set(8, 1000), 1000},
		{"check 19 register 4000", register(4000), set(3, 0), 4000},
	} {
		_, err := host.WriteTrigger(ui.WizardTrigger{Index: -1, Name: "Register", Once: 1, Pairs: []ui.WizardPair{c.pair}, Actions: []ui.WizardNode{c.act}})
		// The wizard names the row before the trigger window's words.
		if want := fmt.Sprintf(host.inspector.word("editor.script.variableout"), c.value, 999); err == nil || !strings.HasSuffix(err.Error(), ": "+want) {
			t.Errorf("%s: %v, want %q", c.name, err, want)
		}
	}
	if string(host.model.Bytes()) != before {
		t.Fatal("a refused register moved the map")
	}
	// The trigger window refuses the same value in the same words.
	if _, err := host.SetFields(ui.KindAction, 0, []ui.InspectionFieldValue{{Key: "type1", Value: 1}, {Key: "value0", Value: 5000}, {Key: "value1", Value: 7}, {Key: "opcode", Value: 3}}); err == nil ||
		err.Error() != fmt.Sprintf(host.inspector.word("editor.script.variableout"), 5000, 999) {
		t.Errorf("the trigger window's action 3 register 5000: %v", err)
	}
	if _, err := host.WriteTrigger(ui.WizardTrigger{Index: -1, Name: "Register", Once: 1,
		Pairs: []ui.WizardPair{register(999)}, Actions: []ui.WizardNode{set(8, 999)}}); err != nil {
		t.Fatalf("register 999: %v", err)
	}
}

// A new check whose register a variable uses names the second game's 1000
// slots, not the first game's 100.
func TestSecondMapRegisterTakenNamesTheSecondGameSlots(t *testing.T) {
	host := secondAuthorHost(t)
	want := fmt.Sprintf(host.inspector.word("editor.wizard.registers"), 5, 1000)
	if err := host.checkWords(registerTaken{5}); err == nil || err.Error() != want {
		t.Fatalf("the taken register: %v, want %q", err, want)
	}
}

// A new check past the second game's 1000 result registers is refused in
// words naming 1000, not the first game's 100, and asks no change of the
// script's form: the second game has no Againrom script record.
func TestSecondMapChecksFullNamesTheSecondGameLimit(t *testing.T) {
	host := secondAuthorHost(t)
	m, err := mapedit.New(host.model.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	for i := range 1000 {
		rec, err := mapedit.NewScriptNode("", 0x10002, uint32(i+1), mapedit.ScriptParam{Type: 1})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := m.Place(mapedit.KindCondition, rec); err != nil {
			t.Fatal(err)
		}
	}
	d, err := secondTestInspector(t).inspect(m.Bytes(), "second.alm")
	if err != nil {
		t.Fatal(err)
	}
	full := d.Edits.(*mapInspectionEdits)
	_, err = full.AddRecord(ui.InspectionNewRecord{Kind: ui.KindCondition})
	var need *ui.ScriptFormNeeded
	if want := fmt.Sprintf(full.inspector.word("editor.wizard.checksfull"), 1000); err == nil || err.Error() != want || errors.As(err, &need) {
		t.Fatalf("the 1001st check: %v, want %q", err, want)
	}
}
