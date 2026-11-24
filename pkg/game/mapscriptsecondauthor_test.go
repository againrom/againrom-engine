package game

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

// secondAuthorHost is the editor host of the fixture second-game map, whose
// type-7 section holds no node and no trigger.
func secondAuthorHost(t *testing.T) *mapInspectionEdits {
	t.Helper()
	d, err := secondTestInspector(t).inspect(secondFixtureMap(), "second.alm")
	if err != nil {
		t.Fatal(err)
	}
	return d.Edits.(*mapInspectionEdits)
}

// secondAuthorScript is the host map's script and its editor.
func secondAuthorScript(t *testing.T, host *mapInspectionEdits) (alm.Script, *mapedit.Editor) {
	t.Helper()
	m, err := mapedit.New(host.model.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	view, err := m.Map()
	if err != nil {
		t.Fatal(err)
	}
	sc, err := view.Script()
	if err != nil {
		t.Fatal(err)
	}
	return sc, m
}

// secondNewNodeZeroes checks that node i of kind k holds zero in word +0x48
// and in its ten text fields, the words no compiler read names
// (R2-ENGINE-1728).
func secondNewNodeZeroes(t *testing.T, m *mapedit.Editor, k mapedit.Kind, i int) {
	t.Helper()
	rec, err := m.Record(k, i)
	if err != nil {
		t.Fatal(err)
	}
	if w := binary.LittleEndian.Uint32(rec[0x48:]); w != 0 {
		t.Fatalf("new node %d word +0x48 is %d, want 0", i, w)
	}
	for b, v := range rec[alm.ScriptNodeTypes+4*alm.ScriptParams:] {
		if v != 0 {
			t.Fatalf("new node %d text byte +%#x is %d, want 0", i, alm.ScriptNodeTypes+4*alm.ScriptParams+b, v)
		}
	}
}

// A new trigger, action and check on a second-game map are written under
// the second game's names with the fields the claims state, and the type-0
// word +0x28 stays the sum of the three counts (R2-ASSET-255).
func TestSecondMapNewTriggerAndNodesWriteTheFieldsTheClaimsState(t *testing.T) {
	host := secondAuthorHost(t)
	for _, kind := range []ui.RecordKind{ui.KindTrigger, ui.KindAction, ui.KindCondition} {
		if _, err := host.AddRecord(ui.InspectionNewRecord{Kind: kind}); err != nil {
			t.Fatalf("a new %v on a second-game map: %v", kind, err)
		}
	}
	sc, m := secondAuthorScript(t, host)
	if len(sc.Triggers) != 1 || sc.Triggers[0].Name != "Trigger" || sc.Triggers[0].Once != 1 {
		t.Fatalf("the new trigger reads %+v", sc.Triggers)
	}
	if len(sc.Actions) != 1 || sc.Actions[0].Label != "Send event" || sc.Actions[0].Opcode != 2 || sc.Actions[0].ID != 1 ||
		sc.Actions[0].Type[0] != 1 || sc.Actions[0].Value[0] != 1 {
		t.Fatalf("the new action reads %+v", sc.Actions)
	}
	if len(sc.Conditions) != 1 || sc.Conditions[0].Label != "Compiler parameter" || sc.Conditions[0].Opcode != 0x10002 || sc.Conditions[0].ID != 1 {
		t.Fatalf("the new check reads %+v", sc.Conditions)
	}
	secondNewNodeZeroes(t, m, mapedit.KindAction, 0)
	secondNewNodeZeroes(t, m, mapedit.KindCondition, 0)
	if got := binary.LittleEndian.Uint32(host.model.Bytes()[2*alm.RecordHeaderSize+0x28:]); got != 3 {
		t.Fatalf("the type-0 word +0x28 is %d, want 3", got)
	}
}

// The wizard writes a new second-game trigger with new rows: a check 21
// naming the map's type-4 object 1 (R2-ENGINE-1725), a constant, and an
// action 21 casting spell 6 from (3, 4) at (5, 6) with power 100
// (R2-ENGINE-1741).
func TestSecondMapWizardWritesANewTriggerWithNewRows(t *testing.T) {
	host := secondAuthorHost(t)
	check := ui.WizardNode{Opcode: 21, Types: [10]uint32{9}, Values: [10]uint32{1}}
	limit := ui.WizardNode{Opcode: 0x10002, Types: [10]uint32{1}, Values: [10]uint32{0}}
	cast := ui.WizardNode{Opcode: 21, Types: [10]uint32{5, 6, 5, 6, 1, 1}, Values: [10]uint32{3, 4, 5, 6, 6, 100}}
	absent := ui.WizardNode{Opcode: 21, Types: [10]uint32{9}, Values: [10]uint32{65535}}
	if _, err := host.WriteTrigger(ui.WizardTrigger{Index: -1, Name: "Absent", Once: 1,
		Pairs: []ui.WizardPair{{Left: absent, Right: limit, Cmp: 5}}, Actions: []ui.WizardNode{cast}}); err == nil {
		t.Fatal("a check naming an object identity the map does not hold is written")
	}
	if _, err := host.WriteTrigger(ui.WizardTrigger{Index: -1, Name: "Cast", Once: 1,
		Pairs: []ui.WizardPair{{Left: check, Right: limit, Cmp: 5}}, Actions: []ui.WizardNode{cast}}); err != nil {
		t.Fatalf("the wizard's new second-game trigger: %v", err)
	}
	sc, m := secondAuthorScript(t, host)
	if len(sc.Triggers) != 1 || len(sc.Conditions) != 2 || len(sc.Actions) != 1 {
		t.Fatalf("the map holds %d triggers, %d checks, %d actions, want 1, 2, 1", len(sc.Triggers), len(sc.Conditions), len(sc.Actions))
	}
	tr := sc.Triggers[0]
	c, a := sc.Conditions[0], sc.Actions[0]
	if c.Opcode != 21 || c.Type[0] != 9 || c.Value[0] != 1 || c.Label != "Word 0x42 of the object with a type-4 id" {
		t.Fatalf("the new check reads %+v", c)
	}
	if a.Opcode != 21 || a.Type != cast.Types || a.Value != cast.Values || a.Label != "Cast spell from cell at cell" {
		t.Fatalf("the new action reads %+v", a)
	}
	if tr.Left[0] != c.ID || tr.Right[0] != sc.Conditions[1].ID || tr.Cmp[0] != 5 || tr.Acts[0] != a.ID || tr.Once != 1 {
		t.Fatalf("the new trigger reads %+v", tr)
	}
	secondNewNodeZeroes(t, m, mapedit.KindCondition, 0)
	secondNewNodeZeroes(t, m, mapedit.KindAction, 0)
	if got := binary.LittleEndian.Uint32(host.model.Bytes()[2*alm.RecordHeaderSize+0x28:]); got != 4 {
		t.Fatalf("the type-0 word +0x28 is %d, want 4", got)
	}
}
