package game

import (
	"bytes"
	"encoding/binary"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// goalTrigger is one trigger of a synthetic script: the condition ids of its
// first pair and the action ids it runs.
type goalTrigger struct {
	name        string
	left, right uint32
	actions     []uint32
	cmp         uint32         // the first pair's comparison
	raw         map[int]uint32 // words written at record offsets last
}

func goalScript(actions, conditions []scriptNode, triggers []goalTrigger) []byte {
	b := scriptPayload(actions, conditions)
	b = b[:len(b)-4]
	b = binary.LittleEndian.AppendUint32(b, uint32(len(triggers)))
	for _, t := range triggers {
		rec := make([]byte, 184)
		copy(rec, t.name)
		binary.LittleEndian.PutUint32(rec[0x80:], t.left)
		binary.LittleEndian.PutUint32(rec[0x84:], t.right)
		binary.LittleEndian.PutUint32(rec[0xa8:], t.cmp)
		for i, id := range t.actions {
			binary.LittleEndian.PutUint32(rec[0x98+4*i:], id)
		}
		binary.LittleEndian.PutUint32(rec[0xb4:], 1)
		for at, v := range t.raw {
			binary.LittleEndian.PutUint32(rec[at:], v)
		}
		b = append(b, rec...)
	}
	return b
}

// goalMap is a 32x32 map of four units with unit ids 10..13 and group ids
// 20..23, and a script with a drop cell, a win trigger on unit 10 or group 22,
// a lose trigger on unit 12 and a constant.
func goalMap(t *testing.T, withDrop bool) []byte {
	t.Helper()
	actions := []scriptNode{{label: "win", opcode: 4, id: 2}, {label: "lose", opcode: 5, id: 3}}
	if withDrop {
		actions = append(actions, scriptNode{label: "drop", opcode: 0x10002, id: 1, values: []uint32{5, 6}, tags: []uint32{5, 6}})
	}
	conditions := []scriptNode{
		{label: "dead", opcode: 5, id: 1, values: []uint32{10}, tags: []uint32{4}},
		{label: "zero", opcode: 0x10002, id: 2, values: []uint32{0}, tags: []uint32{7}},
		{label: "group", opcode: 1, id: 3, values: []uint32{22}, tags: []uint32{2}},
		{label: "dead2", opcode: 5, id: 4, values: []uint32{12}, tags: []uint32{4}},
	}
	o := synth.ALMOptions{Width: 32, Height: 32, Type7Payload: goalScript(actions, conditions, []goalTrigger{
		{name: "Win", left: 1, right: 2, actions: []uint32{2}},
		{name: "Win group", left: 3, right: 2, actions: []uint32{2}},
		{name: "Lose", left: 4, right: 2, actions: []uint32{3}},
	})}
	for i := 0; i < 4; i++ {
		o.Units = append(o.Units, synth.ALMUnit{X: uint32(12+3*i)*256 + 128, Y: 14*256 + 128, ClassID: 40, ClassSubID: 2})
	}
	return built(t, o, nil, nil).Bytes()
}

func openGoalMap(t *testing.T, b []byte) (*ui.MapEditor, string) {
	t.Helper()
	x := &MapInspector{tiles: &terrain.Tileset{}, rules: []mapRule{}}
	path := filepath.Join(t.TempDir(), "goal.alm")
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	e := x.Editor()
	if err := e.Open(path); err != nil {
		t.Fatal(err)
	}
	e.Layout(1280, 800)
	e.Document().Viewer.SetFlat(true)
	closeMapPanel(t, e)
	return e, path
}

func scriptOf(t *testing.T, e *ui.MapEditor) ui.InspectionScript {
	t.Helper()
	s, ok := e.Document().Edits.(ui.InspectionScript)
	if !ok {
		t.Fatal("the host edits no script")
	}
	return s
}

func TestGoalsListTheWinLoseAndProtectArms(t *testing.T) {
	e, _ := openGoalMap(t, goalMap(t, true))
	goals := scriptOf(t, e).Goals()
	var arms []string
	for _, g := range goals {
		arms = append(arms, g.Arm+"/"+g.Pick)
	}
	want := []string{"Win/Unit", "Win/Unit", "Lose/Unit", "Protect/Unit"}
	if len(goals) != 4 || strings.Join(arms, " ") != strings.Join(want, " ") {
		t.Fatalf("arms %v, want %v: %+v", arms, want, goals)
	}
	if !strings.Contains(goals[0].Text, "unit 10 at 12,14") || !goals[0].HasCell || goals[0].Cell != image.Pt(12, 14) {
		t.Fatalf("the win arm reads %+v", goals[0])
	}
	if goals[3].Index != -1 || goals[1].Pick != "Unit" {
		t.Fatalf("the add row: %+v", goals[3])
	}
}

func TestSetGoalTargetRetargetsTheWinArmAndUndoRestoresTheBytes(t *testing.T) {
	b := goalMap(t, true)
	e, _ := openGoalMap(t, b)
	s := scriptOf(t, e)
	unit := recordOf(e.Document(), "Unit", 3)
	if unit < 0 {
		t.Fatal("no unit record")
	}
	next, err := s.SetGoalTarget(0, "Unit", 3)
	if err != nil || next == nil {
		t.Fatalf("retarget: %v", err)
	}
	after := next.SourceBytes
	m, err := alm.Open(after)
	if err != nil {
		t.Fatal(err)
	}
	sc, _ := m.Script()
	if sc.Conditions[0].Value[0] != 13 || sc.Conditions[0].Opcode != 5 || len(after) != len(b) {
		t.Fatalf("condition 0 names %d", sc.Conditions[0].Value[0])
	}
	// Only the one value word changed.
	changed := 0
	for i := range b {
		if b[i] != after[i] {
			changed++
		}
	}
	if changed != 1 {
		t.Fatalf("%d bytes changed, want the one the target number differs in", changed)
	}
	if !e.Dirty() {
		t.Fatal("the edit is not dirty")
	}
	_ = e.Key("undo")
	if e.Dirty() || !bytes.Equal(e.Document().SourceBytes, b) {
		t.Fatal("undo did not restore the bytes")
	}
	_ = e.Key("redo")
	if !bytes.Equal(e.Document().SourceBytes, after) {
		t.Fatal("redo did not restore the edit")
	}
}

func TestSetGoalTargetRefusesWhatTheScriptCannotName(t *testing.T) {
	b := goalMap(t, true)
	e, _ := openGoalMap(t, b)
	s := scriptOf(t, e)
	if _, err := s.SetGoalTarget(0, "Structure", 0); err == nil {
		t.Error("a structure was accepted for a unit target")
	}
	if _, err := s.SetGoalTarget(9, "Unit", 0); err == nil {
		t.Error("an arm that does not exist was accepted")
	}
	if next, err := s.SetGoalTarget(0, "Unit", 0); err != nil || next != nil || e.Dirty() {
		t.Errorf("the arm's own target: %v %v", next, err)
	}
	if !bytes.Equal(e.Document().SourceBytes, b) {
		t.Fatal("a refused edit changed the bytes")
	}
}

func TestSetGoalTargetPointsAGroupArmAtAUnitsGroup(t *testing.T) {
	e, _ := openGoalMap(t, goalMap(t, true))
	s := scriptOf(t, e)
	// Arm 1 is the group-count check on group 22.
	next, err := s.SetGoalTarget(1, "Unit", 0)
	if err != nil || next == nil {
		t.Fatalf("retarget: %v", err)
	}
	m, _ := alm.Open(next.SourceBytes)
	sc, _ := m.Script()
	if sc.Conditions[2].Value[0] != 20 || sc.Conditions[2].Type[0] != 2 {
		t.Fatalf("the group check now names group %d", sc.Conditions[2].Value[0])
	}
}

func TestAddingAProtectCheckAppendsOneNodeAndUndoRemovesIt(t *testing.T) {
	b := goalMap(t, true)
	e, _ := openGoalMap(t, b)
	s := scriptOf(t, e)
	last := len(s.Goals()) - 1
	next, err := s.SetGoalTarget(last, "Unit", 1)
	if err != nil || next == nil {
		t.Fatalf("add: %v", err)
	}
	if len(next.SourceBytes) != len(b)+796 {
		t.Fatalf("length %d, want %d", len(next.SourceBytes), len(b)+796)
	}
	m, _ := alm.Open(next.SourceBytes)
	sc, _ := m.Script()
	n := sc.Conditions[len(sc.Conditions)-1]
	if len(sc.Conditions) != 5 || n.Opcode != 18 || n.ID != 5 || n.Type[0] != 4 || n.Value[0] != 11 {
		t.Fatalf("the new check: %+v of %d conditions", n, len(sc.Conditions))
	}
	if _, err := e.Document().Edits.(ui.InspectionScript).SetGoalTarget(len(s.Goals())-1, "Unit", 1); err == nil {
		t.Fatal("a unit was protected twice")
	}
	_ = e.Key("undo")
	if !bytes.Equal(e.Document().SourceBytes, b) || e.Dirty() {
		t.Fatal("undo did not remove the check")
	}
}

func TestAProtectCheckIsRefusedWhereAVariableTakesItsRegister(t *testing.T) {
	actions := []scriptNode{{label: "set", opcode: 3, id: 1, values: []uint32{1, 5}, tags: []uint32{1, 1}}}
	conditions := []scriptNode{{label: "c", opcode: 5, id: 1, values: []uint32{10}, tags: []uint32{4}}}
	o := synth.ALMOptions{Width: 32, Height: 32, Type7Payload: goalScript(actions, conditions, nil),
		Units: []synth.ALMUnit{{X: 12*256 + 128, Y: 14*256 + 128, ClassID: 40}, {X: 15*256 + 128, Y: 14*256 + 128, ClassID: 40}}}
	e, _ := openGoalMap(t, built(t, o, nil, nil).Bytes())
	s := scriptOf(t, e)
	if _, err := s.SetGoalTarget(len(s.Goals())-1, "Unit", 1); err == nil || !strings.Contains(err.Error(), "register") {
		t.Fatalf("variable 1 against the new check's register 1: %v", err)
	}
}

func TestSetDropCellMovesTheCellAndMarksIt(t *testing.T) {
	b := goalMap(t, true)
	e, _ := openGoalMap(t, b)
	if d := e.Document().Drops; len(d) != 1 || d[0] != image.Pt(5, 6) {
		t.Fatalf("drop marks %v", d)
	}
	s := scriptOf(t, e)
	next, err := s.SetDropCell(image.Pt(20, 21))
	if err != nil || next == nil {
		t.Fatalf("move: %v", err)
	}
	m, _ := alm.Open(next.SourceBytes)
	if cells := mapload.DropCells(m); len(cells) != 1 || cells[0] != (mapload.Cell{X: 20, Y: 21}) || len(next.SourceBytes) != len(b) {
		t.Fatalf("drop cells %v", cells)
	}
	if len(next.Drops) != 1 || next.Drops[0] != image.Pt(20, 21) {
		t.Fatalf("marks %v", next.Drops)
	}
	if again, err := s.SetDropCell(image.Pt(20, 21)); err != nil || again != nil {
		t.Fatalf("the same cell: %v %v", again, err)
	}
	if _, err := s.SetDropCell(image.Pt(40, 2)); err == nil {
		t.Fatal("a cell outside the map was accepted")
	}
	_ = e.Key("undo")
	if e.Dirty() || !bytes.Equal(e.Document().SourceBytes, b) || e.Document().Edits.CanUndo() {
		t.Fatal("one undo did not restore the drop cell")
	}
}

func TestSetDropCellAddsADropActionToAMapWithout(t *testing.T) {
	b := goalMap(t, false)
	e, _ := openGoalMap(t, b)
	if len(e.Document().Drops) != 0 {
		t.Fatal("the map has a drop cell")
	}
	next, err := scriptOf(t, e).SetDropCell(image.Pt(7, 9))
	if err != nil || next == nil || len(next.SourceBytes) != len(b)+796 {
		t.Fatalf("add: %v", err)
	}
	m, _ := alm.Open(next.SourceBytes)
	if cells := mapload.DropCells(m); len(cells) != 1 || cells[0] != (mapload.Cell{X: 7, Y: 9}) {
		t.Fatalf("drop cells %v", cells)
	}
}

func TestWindowKeysMoveTheDropCellAndRetargetAnArm(t *testing.T) {
	b := goalMap(t, true)
	e, _ := openGoalMap(t, b)
	_ = e.Key("drop")
	dragCell(e, e.Document().Drops[0], image.Pt(22, 20))
	_ = e.Key("drop")
	if d := e.Document().Drops; !e.Dirty() || len(d) != 1 || d[0] != image.Pt(22, 20) {
		t.Fatalf("drop cell after the click: %v", d)
	}
	// G lists the arms; Enter arms the pick; a click on a unit retargets.
	_ = e.Key("goals")
	rows, open := e.ReportOpen()
	if !open || len(rows) != 5 || e.ReportSelected() != 0 {
		t.Fatalf("the script panel: open %v, %d rows", open, len(rows))
	}
	_ = e.Key("enter")
	clickCell(e, image.Pt(21, 14)) // unit 3, id 13
	m, err := alm.Open(e.Document().SourceBytes)
	if err != nil {
		t.Fatal(err)
	}
	sc, _ := m.Script()
	if sc.Conditions[0].Value[0] != 13 {
		t.Fatalf("the win arm names unit %d after the click", sc.Conditions[0].Value[0])
	}
	if _, open := e.ReportOpen(); !open || e.ReportSelected() != 0 {
		t.Fatal("the script panel did not stay open on the arm")
	}
	// A click on empty ground names no target and leaves the bytes.
	before := e.Document().SourceBytes
	_ = e.Key("enter")
	clickCell(e, image.Pt(16, 20)) // empty ground clear of the script panel
	if !bytes.Equal(e.Document().SourceBytes, before) || e.Message() == "" {
		t.Fatalf("a click on no unit: message %q", e.Message())
	}
	_ = e.Key("escape")
	_ = e.Key("undo")
	_ = e.Key("undo")
	if e.Dirty() || !bytes.Equal(e.Document().SourceBytes, b) {
		t.Fatal("two undos did not restore the installed bytes")
	}
}

func TestAnUneditedScriptWritesItsBytes(t *testing.T) {
	b := goalMap(t, true)
	e, path := openGoalMap(t, b)
	target := filepath.Join(filepath.Dir(path), "saved.alm")
	_ = e.Key("goals")
	_ = e.Key("goals")
	_ = e.Key("saveas")
	e.TextInput(target)
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(target); err != nil || !bytes.Equal(got, b) {
		t.Fatalf("saved bytes differ: %v", err)
	}
}

// reportedEdits is a checked map (the rule table of checkInspector) whose
// script holds a drop cell on open ground and a win arm on unit 10.
func reportedEdits(t *testing.T) *mapInspectionEdits {
	t.Helper()
	actions := []scriptNode{{label: "win", opcode: 4, id: 2}, {label: "drop", opcode: 0x10002, id: 1, values: []uint32{16, 16}, tags: []uint32{5, 6}}}
	conditions := []scriptNode{{label: "dead", opcode: 5, id: 1, values: []uint32{10}, tags: []uint32{4}}}
	o := cleanOptions()
	o.Type7Payload = goalScript(actions, conditions, []goalTrigger{{name: "Win", left: 1, right: 1, actions: []uint32{2}}})
	m := built(t, o, nil, nil)
	return &mapInspectionEdits{inspector: checkInspector(), model: m, source: "x", saved: m.Bytes()}
}

func TestAnEditThatBreaksTheDropCellOrAReferenceIsReported(t *testing.T) {
	s := reportedEdits(t)
	if rows := s.Report(); len(rows) != 0 {
		t.Fatalf("the unedited script reports %+v", rows)
	}
	// A drop cell moved off the map by the node's own fields is a warning.
	if err := s.model.SetDropCell(40, 16); err != nil {
		t.Fatal(err)
	}
	rows := ofRule(s.Report(), "drop-cell")
	if len(rows) != 1 || rows[0].Error || !strings.Contains(rows[0].Text, "outside the 32x32 map") {
		t.Fatalf("drop cell off the map: %+v", s.Report())
	}
	// A target number the map does not hold is an error that refuses a save.
	if err := s.model.SetField(mapedit.KindCondition, 0, "value0", 99); err != nil {
		t.Fatal(err)
	}
	refs := ofRule(s.Report(), "script-reference")
	if len(refs) != 1 || !refs[0].Error || refs[0].Kind != "Condition" || !strings.Contains(refs[0].Text, "unit 99") {
		t.Fatalf("dangling target: %+v", s.Report())
	}
	if _, err := s.SaveAs(filepath.Join(t.TempDir(), "x.alm")); err == nil {
		t.Fatal("a dangling script reference was saved")
	}
}

func TestSettingATargetByFieldChecksItAgainstTheMap(t *testing.T) {
	e, _ := openGoalMap(t, goalMap(t, true))
	s := e.Document().Edits
	nodes := map[string]int{}
	for i, r := range e.Document().Records {
		if r.Kind == "Condition" {
			nodes[r.Label] = i
		}
	}
	e.Select(nodes["dead"], false)
	b := e.Document().SourceBytes
	if next, err := s.SetField("Condition", 0, "value0", 99); err == nil || next != nil {
		t.Fatalf("a unit the map lacks: %v", err)
	}
	if next, err := s.SetField("Condition", 0, "value0", 11); err != nil || next == nil {
		t.Fatalf("a unit the map holds: %v", err)
	}
	next, err := s.SetField("Condition", 0, "value0", 10500)
	if err != nil || next == nil {
		t.Fatalf("a hero-band value: %v", err)
	}
	if bytes.Equal(next.SourceBytes, b) {
		t.Fatal("no edit reached the bytes")
	}
	if next, err := s.SetField("Action", 2, "value0", 400); err == nil || next != nil {
		t.Fatalf("a drop X over 255: %v", err)
	}
}
