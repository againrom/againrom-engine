package game

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/scriptprofile"
	"againrom/pkg/ui"
)

// extendedCase is how a test authors one extended-record operation: the
// node a plain map holds, the field edit that turns a fixture node into it
// (opcode 0 where the node's field edit does not offer the operation), and
// the wizard row that writes it.
type extendedCase struct {
	action      bool
	node        scriptNode
	kind        ui.RecordKind
	index       int
	opcode      uint32
	left, right ui.WizardNode
	act         ui.WizardNode
}

var wizardOne = ui.WizardNode{Opcode: 0x10002, Types: [10]uint32{mapload.ScriptInt}, Values: [10]uint32{1}}

var extendedCases = map[string]extendedCase{
	"dialogue": {action: true, node: scriptNode{label: "talk", opcode: 2, id: 1, values: []uint32{5, 1}, tags: []uint32{1, 1}},
		kind: ui.KindAction, index: 1, opcode: 2,
		act: ui.WizardNode{Opcode: 2, Label: "talk", Types: [10]uint32{mapload.ScriptInt, mapload.ScriptInt}, Values: [10]uint32{5, 1}}},
	"leftcell": {node: scriptNode{label: "Departure", opcode: scriptprofile.UnitLeftCellOp, id: 1, values: []uint32{91, 15, 16}, tags: []uint32{4, 5, 6}},
		left: ui.WizardNode{Opcode: scriptprofile.UnitLeftCellOp, Types: [10]uint32{mapload.ScriptUnit, mapload.ScriptX, mapload.ScriptY}, Values: [10]uint32{91, 15, 16}}},
	"global": {node: scriptNode{label: "Global", opcode: scriptprofile.ReadGlobalOp, id: 1, values: []uint32{1}, tags: []uint32{1}},
		left: ui.WizardNode{Opcode: scriptprofile.ReadGlobalOp, Types: [10]uint32{mapload.ScriptInt}, Values: [10]uint32{1}}},
	"teleport": {action: true, node: scriptNode{label: "Pad", opcode: scriptprofile.TeleportCellOp, id: 1, values: []uint32{5, 4, 5, 10}, tags: []uint32{5, 6, 5, 6}},
		kind: ui.KindAction, index: 1, opcode: scriptprofile.TeleportCellOp,
		act: ui.WizardNode{Opcode: scriptprofile.TeleportCellOp, Types: [10]uint32{mapload.ScriptX, mapload.ScriptY, mapload.ScriptX, mapload.ScriptY}, Values: [10]uint32{10, 12, 24, 28}}},
}

// extendedRefusal is the words a host refuses n nodes of row o with.
func extendedRefusal(s *mapInspectionEdits, o scriptprofile.ExtendedOp, n int) string {
	if o.Counted {
		return fmt.Sprintf(s.inspector.word(o.Word), n)
	}
	return s.inspector.word(o.Word)
}

// heldMap is an original map whose script holds c's node and no trigger.
func heldMap(c extendedCase) []byte {
	payload := scriptPayload(nil, []scriptNode{c.node})
	if c.action {
		payload = scriptPayload([]scriptNode{c.node}, nil)
	}
	return synth.ALM(synth.ALMOptions{Width: 40, Height: 40, Units: extendedUnits, Type7Payload: payload})
}

// wantFormNeeded fails t unless err asks to extend the map for words why.
func wantFormNeeded(t *testing.T, step string, err error, why string) {
	t.Helper()
	var need *ui.ScriptFormNeeded
	if !errors.As(err, &need) || need.Fold || need.Why != why {
		t.Errorf("%s answered %v, want the extend question %q", step, err, why)
	}
}

// Every operation of the extended-record table is refused on a map of the
// original record at the first edit that writes it, the node's field edit
// where it offers the operation, with its row's words, and so at the
// wizard, the save and the fold; the map keeps its bytes, and an edit that
// adds no such node is not refused.
func TestEveryExtendedOperationIsRefusedAtTheEditThatWritesIt(t *testing.T) {
	if len(scriptprofile.ExtendedOps) != len(extendedCases) {
		t.Fatalf("the table holds %d rows, the test authors %d", len(scriptprofile.ExtendedOps), len(extendedCases))
	}
	for _, o := range scriptprofile.ExtendedOps {
		c, ok := extendedCases[o.Name]
		if !ok {
			t.Fatalf("row %q has no authoring case", o.Name)
		}
		t.Run(o.Name, func(t *testing.T) {
			plain := formMap(t, 2)
			e, _ := openGoalMap(t, plain)
			s := formHost(e)
			why := extendedRefusal(s, o, 1)
			if c.opcode != 0 {
				_, err := s.SetFields(c.kind, c.index, []ui.InspectionFieldValue{{Key: "opcode", Value: int64(c.opcode)}})
				wantFormNeeded(t, "the field edit", err, why)
			}
			left := wizardOne
			if c.left.Opcode != 0 {
				left = c.left
			}
			act := ui.WizardNode{Opcode: 3, Types: [10]uint32{mapload.ScriptInt, mapload.ScriptInt}, Values: [10]uint32{80, 1}}
			if c.act.Opcode != 0 {
				act = c.act
			}
			_, err := s.WriteTrigger(ui.WizardTrigger{Index: -1, Name: "Probe", Pairs: []ui.WizardPair{{Left: left, Right: wizardOne}}, Actions: []ui.WizardNode{act}})
			wantFormNeeded(t, "the wizard", err, why)
			if !bytes.Equal(s.model.Bytes(), plain) {
				t.Error("a refused edit wrote the map")
			}

			held := heldMap(c)
			e, _ = openGoalMap(t, held)
			s = formHost(e)
			out := filepath.Join(t.TempDir(), "held.alm")
			_, err = s.SaveAs(out)
			wantFormNeeded(t, "the save", err, why)
			if _, err := os.Stat(out); !os.IsNotExist(err) {
				t.Error("a refused save wrote the file", err)
			}
			if _, err := s.Move(ui.KindUnit, 0, image.Pt(13, 14)); err != nil {
				t.Errorf("an edit that adds no %s node was refused: %v", o.Name, err)
			}
			if _, err := s.ExtendScript(); err != nil {
				t.Fatal(err)
			}
			extended := s.model.Bytes()
			if _, err := s.FoldScript(); err == nil || err.Error() != why || !bytes.Equal(s.model.Bytes(), extended) {
				t.Errorf("the fold answered %v, want %q", err, why)
			}
		})
	}
}

// A new extended-record operation is one row of the table: a test-only row
// is refused at the save and the fold with its words and no other code.
func TestATestOnlyExtendedRowIsRefusedTheSameWay(t *testing.T) {
	defer func(rows []scriptprofile.ExtendedOp) { scriptprofile.ExtendedOps = rows }(scriptprofile.ExtendedOps)
	probe := scriptprofile.ExtendedOp{Name: "probe", Ops: []scriptprofile.OpID{{ID: 0x41ff}}, Word: "editor.script.limit.teleport", Counted: true}
	scriptprofile.ExtendedOps = append(slices.Clone(scriptprofile.ExtendedOps), probe)
	c := extendedCase{action: true, node: scriptNode{label: "Probe", opcode: 0x41ff, id: 1, values: []uint32{1}, tags: []uint32{1}}}
	e, _ := openGoalMap(t, heldMap(c))
	s := formHost(e)
	why := extendedRefusal(s, probe, 1)
	_, err := s.SaveAs(filepath.Join(t.TempDir(), "probe.alm"))
	wantFormNeeded(t, "the save", err, why)
	if _, err := s.ExtendScript(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.FoldScript(); err == nil || err.Error() != why {
		t.Errorf("the fold answered %v, want %q", err, why)
	}
	plain := formMap(t, 2)
	e, _ = openGoalMap(t, plain)
	s = formHost(e)
	probeNode := ui.WizardNode{Opcode: 0x41ff, Types: [10]uint32{mapload.ScriptInt}, Values: [10]uint32{1}}
	_, err = s.WriteTrigger(ui.WizardTrigger{Index: -1, Name: "Probe", Pairs: []ui.WizardPair{{Left: wizardOne, Right: wizardOne}}, Actions: []ui.WizardNode{probeNode}})
	wantFormNeeded(t, "the wizard", err, why)
	if !bytes.Equal(s.model.Bytes(), plain) {
		t.Error("a refused wizard row wrote the map")
	}
}

func extendedRowByName(t *testing.T, name string) scriptprofile.ExtendedOp {
	t.Helper()
	for _, o := range scriptprofile.ExtendedOps {
		if o.Name == name {
			return o
		}
	}
	t.Fatalf("extended row %q not found", name)
	return scriptprofile.ExtendedOp{}
}

func TestMessagePageAndWriteTriggerRefuseExtendedOperations(t *testing.T) {
	teleportRow := extendedRowByName(t, "teleport")
	msgNode := ui.WizardNode{Opcode: 2, Types: [10]uint32{mapload.ScriptInt}, Values: [10]uint32{0}}
	tpNode := extendedCases["teleport"].act
	plan := ui.WizardTrigger{
		Index:   -1,
		Name:    "Hop",
		Once:    1,
		Pairs:   []ui.WizardPair{{Left: wizardOne, Right: wizardOne}},
		Actions: []ui.WizardNode{msgNode, tpNode},
	}

	t.Run("ROM1-page-writer", func(t *testing.T) {
		plain := formMap(t, 2)
		e, _ := openGoalMap(t, plain)
		s := formHost(e)
		if err := s.prepareWorkspace(); err != nil {
			t.Fatal(err)
		}
		why := extendedRefusal(s, teleportRow, 1)
		before := bytes.Clone(s.model.Bytes())
		undoBefore := len(s.undoWorkspace)
		pagesBefore := len(s.workspace.Pages)
		owner := ui.InspectionMessageOwner{Kind: ui.KindTrigger, Index: -1, Trigger: &plan, Action: 0}
		parts := []ui.InspectionMessagePart{{Source: -1, Number: 1, Body: "Hello", Speaker: -1}}
		_, _, err := s.CreateMessageParts(owner, parts)
		wantFormNeeded(t, "CreateMessageParts", err, why)
		if !bytes.Equal(s.model.Bytes(), before) || len(s.undoWorkspace) != undoBefore || len(s.workspace.Pages) != pagesBefore || s.CanUndo() {
			t.Fatalf("CreateMessageParts mutated state on refusal: undo %d->%d pages %d->%d", undoBefore, len(s.undoWorkspace), pagesBefore, len(s.workspace.Pages))
		}
		_, err = s.NewMessage(owner)
		wantFormNeeded(t, "NewMessage", err, why)
		if !bytes.Equal(s.model.Bytes(), before) || len(s.undoWorkspace) != undoBefore || len(s.workspace.Pages) != pagesBefore || s.CanUndo() {
			t.Fatalf("NewMessage mutated state on refusal: undo %d->%d pages %d->%d", undoBefore, len(s.undoWorkspace), pagesBefore, len(s.workspace.Pages))
		}
	})

	t.Run("ROM1-write_trigger-command", func(t *testing.T) {
		e := agentEditor(t)
		tool, must := editorMCP(t, e)
		must("new_map", map[string]any{"width": 32, "height": 32})
		s := formHost(e)
		why := extendedRefusal(s, teleportRow, 1)
		before := bytes.Clone(e.Document().SourceBytes)
		undoBefore := len(s.undoWorkspace)
		pagesBefore := len(s.workspace.Pages)
		r := tool("write_trigger", map[string]any{
			"name": "Hop",
			"actions": []any{
				map[string]any{"item": "a.dialog", "parts": []any{map[string]any{"body": "Hello"}}},
				map[string]any{"item": "a.teleport", "params": []any{[]int{5, 5}, []int{10, 10}}},
			},
		})
		if !r.IsError || len(r.Content) == 0 || !strings.Contains(r.Content[0].Text, why) {
			t.Fatalf("write_trigger answered %+v, want error containing %q", r, why)
		}
		if !bytes.Equal(e.Document().SourceBytes, before) || len(s.undoWorkspace) != undoBefore || len(s.workspace.Pages) != pagesBefore {
			t.Fatalf("refused write_trigger mutated map or history: undo %d->%d pages %d->%d", undoBefore, len(s.undoWorkspace), pagesBefore, len(s.workspace.Pages))
		}
	})

	t.Run("ROM2-page-writer", func(t *testing.T) {
		host, _ := secondEventTextHost(t, eventTextFixture)
		why := extendedRefusal(host, teleportRow, 1)
		before := bytes.Clone(host.model.Bytes())
		undoBefore := len(host.undoWorkspace)
		textBefore := string(host.workspace.Pages["main/text/mission10.txt"])
		owner := ui.InspectionMessageOwner{Kind: ui.KindTrigger, Index: -1, Trigger: &plan, Action: 0}
		_, err := host.NewMessage(owner)
		wantFormNeeded(t, "ROM2 NewMessage", err, why)
		if !bytes.Equal(host.model.Bytes(), before) || len(host.undoWorkspace) != undoBefore || string(host.workspace.Pages["main/text/mission10.txt"]) != textBefore {
			t.Fatalf("ROM2 NewMessage mutated state on refusal: undo %d->%d", undoBefore, len(host.undoWorkspace))
		}
	})
}

func TestOriginalSaveDialogueRefusalMatchesCompilerInBothGames(t *testing.T) {
	dialogueRow := extendedRowByName(t, "dialogue")
	cases := []struct {
		name      string
		params    []mapedit.ScriptParam
		wantFirst bool
		wantSec   bool
	}{
		{"third-slot-after-unset", []mapedit.ScriptParam{{Type: 1, Value: 7}, {Type: 0, Value: 0}, {Type: 1, Value: 1}}, true, false},
		{"unset-second-slot", []mapedit.ScriptParam{{Type: 1, Value: 7}, {Type: 0, Value: 1}}, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec, err := mapedit.NewScriptNode("elder", 2, 1, tc.params...)
			if err != nil {
				t.Fatal(err)
			}
			m1, err := mapedit.New(synth.ALM(synth.ALMOptions{Width: 40, Height: 40, Units: extendedUnits, Type7Payload: scriptPayload(nil, nil)}))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := m1.Place(mapedit.KindAction, rec); err != nil {
				t.Fatal(err)
			}
			e1, _ := openGoalMap(t, m1.Bytes())
			s1 := formHost(e1)
			out1 := filepath.Join(t.TempDir(), "rom1.alm")
			_, err = s1.SaveAs(out1)
			if tc.wantFirst {
				wantFormNeeded(t, "ROM1 save", err, extendedRefusal(s1, dialogueRow, 1))
			} else if err != nil {
				t.Fatalf("ROM1 save refused non-dialogue: %v", err)
			}

			m2, err := mapedit.New(secondFixtureMap())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := m2.Place(mapedit.KindAction, rec); err != nil {
				t.Fatal(err)
			}
			d2, err := secondTestInspector(t).inspect(m2.Bytes(), "second.alm")
			if err != nil {
				t.Fatal(err)
			}
			s2 := d2.Edits.(*mapInspectionEdits)
			out2 := filepath.Join(t.TempDir(), "rom2.alm")
			_, err = s2.SaveAs(out2)
			if tc.wantSec {
				wantFormNeeded(t, "ROM2 save", err, extendedRefusal(s2, dialogueRow, 1))
			} else if err != nil {
				t.Fatalf("ROM2 save refused non-dialogue: %v", err)
			}
		})
	}
}

func TestWizardRefusesNewGlobalActionWhenRemovingGlobalCheck(t *testing.T) {
	globalRow := extendedRowByName(t, "global")
	setGlobalAct := ui.WizardNode{
		Opcode: scriptprofile.SetGlobalOp,
		Types:  [10]uint32{mapload.ScriptInt, mapload.ScriptInt},
		Values: [10]uint32{1, 5},
	}
	checkWizard := func(t *testing.T, s *mapInspectionEdits) {
		t.Helper()
		why := extendedRefusal(s, globalRow, 1)
		before := bytes.Clone(s.model.Bytes())
		undoBefore := len(s.undoWorkspace)
		_, err := s.WriteTrigger(ui.WizardTrigger{
			Index:   0,
			Name:    "Swap",
			Once:    1,
			Pairs:   []ui.WizardPair{{Left: wizardOne, Right: wizardOne}},
			Actions: []ui.WizardNode{setGlobalAct},
		})
		wantFormNeeded(t, "wizard swap", err, why)
		if !bytes.Equal(s.model.Bytes(), before) || len(s.undoWorkspace) != undoBefore {
			t.Fatalf("refused wizard swap mutated map or undo: undo %d->%d", undoBefore, len(s.undoWorkspace))
		}
	}

	buildHeld := func(t *testing.T, base []byte) []byte {
		t.Helper()
		m, err := mapedit.New(base)
		if err != nil {
			t.Fatal(err)
		}
		c1, err := mapedit.NewScriptNode("Global", scriptprofile.ReadGlobalOp, 1, mapedit.ScriptParam{Type: mapload.ScriptInt, Value: 1})
		if err != nil {
			t.Fatal(err)
		}
		c2, err := mapedit.NewScriptNode("One", 0x10002, 2, mapedit.ScriptParam{Type: mapload.ScriptInt, Value: 1})
		if err != nil {
			t.Fatal(err)
		}
		a1, err := mapedit.NewScriptNode("Set", 3, 1, mapedit.ScriptParam{Type: mapload.ScriptInt, Value: 80}, mapedit.ScriptParam{Type: mapload.ScriptInt, Value: 1})
		if err != nil {
			t.Fatal(err)
		}
		tr, err := m.NewTrigger(alm.ScriptTrigger{Name: "G", Once: 1, Left: []uint32{1}, Right: []uint32{2}, Cmp: []uint32{0}, Acts: []uint32{1}})
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range []struct {
			k mapedit.Kind
			b []byte
		}{{mapedit.KindCondition, c1}, {mapedit.KindCondition, c2}, {mapedit.KindAction, a1}, {mapedit.KindTrigger, tr}} {
			if _, err := m.Place(p.k, p.b); err != nil {
				t.Fatal(err)
			}
		}
		return m.Bytes()
	}

	t.Run("ROM1", func(t *testing.T) {
		raw := buildHeld(t, synth.ALM(synth.ALMOptions{Width: 40, Height: 40, Units: extendedUnits, Type7Payload: scriptPayload(nil, nil)}))
		e, _ := openGoalMap(t, raw)
		checkWizard(t, formHost(e))
	})

	t.Run("ROM2", func(t *testing.T) {
		raw := buildHeld(t, secondFixtureMap())
		d, err := secondTestInspector(t).inspect(raw, "second.alm")
		if err != nil {
			t.Fatal(err)
		}
		checkWizard(t, d.Edits.(*mapInspectionEdits))
	})
}
