package game

import (
	"bytes"
	"image"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

// wizField is the index of the shown wizard's field key.
func wizField(t *testing.T, e *ui.MapEditor, key string) int {
	t.Helper()
	n := e.WizardField(key)
	if n < 0 {
		t.Fatalf("the wizard shows no field %s:\n%s", key, trigLines(e))
	}
	return n
}

// wizType replaces the text of wizard field key with s.
func wizType(t *testing.T, e *ui.MapEditor, key, s string) {
	t.Helper()
	r, ok := e.FieldRect(wizField(t, e, key))
	// A click puts the caret where it lands; at the field's right end it
	// follows the text, which the backspaces then remove.
	r.Min.X = max(r.Min.X, r.Max.X-4)
	trigClick(t, e, r, ok, key)
	for i := 0; i < 24; i++ {
		_ = e.Key("backspace")
	}
	e.TextInput(s)
}

// wizPress presses button k of the row that edits wizard field key; an
// empty key presses the Add condition row's button, "+action" the Add
// action row's.
func wizPress(t *testing.T, e *ui.MapEditor, key string, k int) {
	t.Helper()
	var r image.Rectangle
	var ok bool
	switch key {
	case "":
		r, ok = e.RowButtonRect(-1, 0, false)
	case "+action":
		r, ok = e.RowButtonRect(-1, 0, true)
	default:
		r, ok = e.RowButtonRect(wizField(t, e, key), k, false)
	}
	trigClick(t, e, r, ok, "row button of "+key)
}

// wizApply presses Apply.
func wizApply(t *testing.T, e *ui.MapEditor) {
	t.Helper()
	r, ok := e.PopupButtonRect(0)
	trigClick(t, e, r, ok, "Apply")
	if strings.Contains(trigLines(e), "Apply or cancel") {
		t.Fatalf("Apply was refused:\n%s", trigLines(e))
	}
}

// wizScript is the edited map's script.
func wizScript(t *testing.T, b []byte) alm.Script {
	t.Helper()
	m, err := alm.Open(b)
	if err != nil {
		t.Fatal(err)
	}
	sc, err := m.Script()
	if err != nil {
		t.Fatal(err)
	}
	return sc
}

// wizNode is the node with id, failing when there is none.
func wizNode(t *testing.T, nodes []alm.ScriptNode, id uint32) alm.ScriptNode {
	t.Helper()
	for _, n := range nodes {
		if n.ID == id {
			return n
		}
	}
	t.Fatalf("no node with ID %d", id)
	return alm.ScriptNode{}
}

// wizNewTrigger runs Tools > New trigger and names the trigger name.
func wizNewTrigger(t *testing.T, e *ui.MapEditor, name string) {
	t.Helper()
	r, ok := e.BarRect("tools")
	trigClick(t, e, r, ok, "Tools")
	n, ok := e.MenuRow("tools", "newtrigger")
	if !ok {
		t.Fatal("the Tools menu has no New trigger")
	}
	r, ok = e.MenuRowRect(n)
	trigClick(t, e, r, ok, "New trigger")
	addSteps(t, e, map[string]string{"name": name})
	if !e.WizardShown() {
		t.Fatalf("the new trigger does not open the wizard:\n%s", trigLines(e))
	}
}

// emptyScriptMap is a size by size map with no unit and an empty script, as
// the editor's New map writes it.
func emptyScriptMap(t *testing.T, size int) []byte {
	t.Helper()
	return built(t, synth.ALMOptions{Width: size, Height: size, Type7Payload: goalScript(nil, nil, nil)}, nil, nil).Bytes()
}

// heroAreaMap is a 40x40 map with no script, a drop zone at 12,12 added with
// the drop tool, and the owner's example built in the wizard: when the hero
// enters the box box, give player 1 500 gold and win.
func heroAreaMap(t *testing.T, box [4]int) (*ui.MapEditor, string) {
	t.Helper()
	e, path := openGoalMap(t, emptyScriptMap(t, 40))
	e.Fit()
	_ = e.Key("drop")
	clickCell(e, image.Pt(12, 12))
	_ = e.Key("escape")
	wizNewTrigger(t, e, "Treasure")
	wizPress(t, e, "", 0)
	wizPress(t, e, "c0.p1", 0) // Draw on map
	if !e.AreaDrawArmed() {
		t.Fatalf("Draw on map armed no drawing:\n%s", trigLines(e))
	}
	dragCell(e, image.Pt(box[0], box[1]), image.Pt(box[2], box[3]))
	if e.AreaDrawArmed() || !e.WizardShown() {
		t.Fatal("the drawing did not return to the wizard")
	}
	wizPress(t, e, "+action", 0)
	wizPress(t, e, "+action", 0)
	trigChoose(t, e, wizField(t, e, "a1.item"), "Win the mission")
	wizApply(t, e)
	return e, path
}

// The owner's example from an empty map: the wizard writes the box check,
// its Constant, the pair and both actions, which the Details view reads as
// the pair it would need, and the saved map reads back to the same trigger.
func TestTriggerWizardBuildsTheHeroAreaExample(t *testing.T) {
	e, path := heroAreaMap(t, [4]int{10, 12, 14, 15})
	sc := wizScript(t, e.Document().SourceBytes)
	if len(sc.Triggers) != 2 {
		t.Fatalf("%d triggers, want the drop zone's and the wizard's", len(sc.Triggers))
	}
	tr := sc.Triggers[1]
	if tr.Name != "Treasure" || tr.Cmp[0] != 0 || tr.Left[1] != 0 || tr.Acts[2] != 0 || tr.Once != 1 {
		t.Fatalf("trigger reads %+v", tr)
	}
	left, right := wizNode(t, sc.Conditions, tr.Left[0]), wizNode(t, sc.Conditions, tr.Right[0])
	if left.Opcode != 2 || left.Type != [10]uint32{4, 5, 6, 5, 6} || left.Value != [10]uint32{10001, 10, 12, 14, 15} {
		t.Fatalf("the box check reads %+v", left)
	}
	if right.Opcode != mapload.ScriptConstant || right.Type != [10]uint32{1} || right.Value != [10]uint32{1} {
		t.Fatalf("the constant reads %+v", right)
	}
	gold, win := wizNode(t, sc.Actions, tr.Acts[0]), wizNode(t, sc.Actions, tr.Acts[1])
	if gold.Opcode != 23 || gold.Type != [10]uint32{3, 1} || gold.Value != [10]uint32{1, 500} || win.Opcode != 4 || win.Type != [10]uint32{} {
		t.Fatalf("the actions read %+v and %+v", gold, win)
	}
	r, ok := e.PopupButtonRect(4)
	trigClick(t, e, r, ok, "Details")
	got := trigLines(e)
	for _, want := range []string{"Pair 1", "Unit Hero 1: Main hero, Corner 10,12, Opposite corner 14,15", "Compared with", "Value 1", "Action 1", "Action 2", "#" + strconv.Itoa(int(win.ID)) + " Win"} {
		if !strings.Contains(got, want) {
			t.Fatalf("the Details view lacks %q:\n%s", want, got)
		}
	}
	if _, err := e.Document().Edits.SaveAs(path + ".out.alm"); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(path + ".out.alm")
	if err != nil || !bytes.Equal(saved, e.Document().SourceBytes) {
		t.Fatalf("the saved map differs: %v", err)
	}
	back := wizScript(t, saved)
	if back.Triggers[1] != tr || len(back.Conditions) != len(sc.Conditions) || len(back.Actions) != len(sc.Actions) {
		t.Fatal("the saved map reads another trigger")
	}
	reopened, _ := openGoalMap(t, saved)
	reopened.Select(trigRecord(t, reopened, "Trigger", 1), true)
	if rows, kept := reopened.WizardRowCounts(); rows != 3 || kept != 0 {
		t.Fatalf("the reopened trigger shows %d rows, %d kept:\n%s", rows, kept, trigLines(reopened))
	}
	got = trigLines(reopened)
	for _, want := range []string{"When", "Hero enters area", "Area 10,12,14,15", "Do", "Give gold", "Gold 500", "Win the mission"} {
		if !strings.Contains(got, want) {
			t.Fatalf("the reopened wizard lacks %q:\n%s", want, got)
		}
	}
}

// A row the wizard reads edits its own node in place; a row of another
// trigger's node gets a node of its own; removing rows deletes what no
// trigger names and keeps what another trigger names.
func TestTriggerWizardEditsAndRemovesRows(t *testing.T) {
	e, _ := openGoalMap(t, triggerMap(t))
	e.Select(trigRecord(t, e, "Trigger", 0), true)
	if rows, kept := e.WizardRowCounts(); rows != 2 || kept != 0 {
		t.Fatalf("Near shows %d rows, %d kept:\n%s", rows, kept, trigLines(e))
	}
	got := trigLines(e)
	for _, want := range []string{"Unit near a point", "Point 5,6", "Radius 2", "Start dialog"} {
		if !strings.Contains(got, want) {
			t.Fatalf("Near lacks %q:\n%s", want, got)
		}
	}
	before := e.Document().SourceBytes
	wizType(t, e, "c0.p2", "6")
	wizApply(t, e)
	sc := wizScript(t, e.Document().SourceBytes)
	if n := sc.Conditions[0]; n.ID != 1 || n.Value != [10]uint32{10, 5, 6, 6} || len(sc.Conditions) != 4 {
		t.Fatalf("the radius edit wrote %+v over %d conditions", n, len(sc.Conditions))
	}
	if err := e.Key("undo"); err != nil || !bytes.Equal(e.Document().SourceBytes, before) {
		t.Fatal("one Undo did not restore the bytes", err)
	}
	_ = e.Key("redo")

	// Far keeps its pair and its cast as they stand.
	e.Select(trigRecord(t, e, "Trigger", 1), true)
	if rows, kept := e.WizardRowCounts(); rows != 3 || kept != 2 {
		t.Fatalf("Far shows %d rows, %d kept:\n%s", rows, kept, trigLines(e))
	}
	if !strings.Contains(trigLines(e), "Other: ") || !strings.Contains(trigLines(e), "Win the mission") {
		t.Fatalf("Far's rows:\n%s", trigLines(e))
	}

	// Removing Near's message deletes the action; removing its condition
	// deletes the check and its constant. Far's nodes stay.
	e.Select(trigRecord(t, e, "Trigger", 0), true)
	wizPress(t, e, "a0.item", 1) // Remove, after Configure
	wizPress(t, e, "c0.item", 0)
	wizApply(t, e)
	sc = wizScript(t, e.Document().SourceBytes)
	near, far := sc.Triggers[0], sc.Triggers[1]
	if near.Left != [3]uint32{} || near.Acts != [4]uint32{} || len(sc.Conditions) != 2 || len(sc.Actions) != 3 {
		t.Fatalf("Near %+v over %d conditions and %d actions", near, len(sc.Conditions), len(sc.Actions))
	}
	if far.Left[0] != 3 || far.Right[0] != 4 || far.Acts[0] != 2 || far.Acts[1] != 3 {
		t.Fatalf("Far changed: %+v", far)
	}
	wizNode(t, sc.Conditions, 3)
	wizNode(t, sc.Conditions, 4)
}

// sharedMap holds two triggers that name the same check and constant: any
// unit of player 1 within 4 of cell 20,20.
func sharedMap(t *testing.T) []byte {
	t.Helper()
	conditions := []scriptNode{
		{label: "far", opcode: 15, id: 1, values: []uint32{1, 20, 20}, tags: []uint32{3, 5, 6}},
		{label: "four", opcode: 0x10002, id: 2, values: []uint32{4}, tags: []uint32{1}},
	}
	actions := []scriptNode{{label: "win", opcode: 4, id: 1}, {label: "lose", opcode: 5, id: 2}}
	o := synth.ALMOptions{Width: 40, Height: 40, Type7Payload: goalScript(actions, conditions, []goalTrigger{
		{name: "A", left: 1, right: 2, cmp: 5, actions: []uint32{1}},
		{name: "B", left: 1, right: 2, cmp: 5, actions: []uint32{2}},
	})}
	return built(t, o, nil, nil).Bytes()
}

// An edit of a row whose nodes another trigger names writes new nodes for
// this trigger; removing the row deletes nothing the other names.
func TestTriggerWizardLeavesSharedNodesAlone(t *testing.T) {
	e, _ := openGoalMap(t, sharedMap(t))
	e.Select(trigRecord(t, e, "Trigger", 0), true)
	if rows, kept := e.WizardRowCounts(); rows != 2 || kept != 0 {
		t.Fatalf("A shows %d rows, %d kept:\n%s", rows, kept, trigLines(e))
	}
	if got := trigLines(e); !strings.Contains(got, "Any unit of a player near a point") || !strings.Contains(got, "Radius 4") {
		t.Fatalf("A's rows:\n%s", got)
	}
	wizType(t, e, "c0.p2", "7")
	wizApply(t, e)
	sc := wizScript(t, e.Document().SourceBytes)
	a, b := sc.Triggers[0], sc.Triggers[1]
	if b.Left[0] != 1 || b.Right[0] != 2 || wizNode(t, sc.Conditions, 2).Value[0] != 4 {
		t.Fatalf("B changed: %+v", b)
	}
	if a.Left[0] != 1 || a.Right[0] == 2 || wizNode(t, sc.Conditions, a.Right[0]).Value[0] != 7 || a.Cmp[0] != 5 {
		t.Fatalf("A reads %+v", a)
	}
	n := len(sc.Conditions)
	wizPress(t, e, "c0.item", 0)
	wizApply(t, e)
	sc = wizScript(t, e.Document().SourceBytes)
	if len(sc.Conditions) != n-1 || sc.Triggers[1].Left[0] != 1 || wizNode(t, sc.Conditions, 1).Opcode != 15 || wizNode(t, sc.Conditions, 2).Value[0] != 4 {
		t.Fatalf("removing A's row reached B: %d conditions, B %+v", len(sc.Conditions), sc.Triggers[1])
	}
}

// The wizard's choices of unit include placed units, and a click on the map
// picks one: the condition is not wired to hero 1.
func TestTriggerWizardPicksAUnitOnTheMap(t *testing.T) {
	e, _ := openGoalMap(t, triggerMap(t))
	e.Fit()
	e.Select(trigRecord(t, e, "Trigger", 0), true)
	wizPress(t, e, "c0.p0", 0)
	clickCell(e, image.Pt(18, 14))
	if !e.WizardShown() {
		t.Fatal("the unit pick did not return to the wizard")
	}
	wizApply(t, e)
	sc := wizScript(t, e.Document().SourceBytes)
	if n := wizNode(t, sc.Conditions, sc.Triggers[0].Left[0]); n.Value[0] != 12 {
		t.Fatalf("the picked unit wrote %+v", n)
	}
}

// startEdited starts mission 10 from map bytes b with a one-hero party.
func startEdited(t *testing.T, b []byte) *Mission {
	t.Helper()
	return startEditedParty(t, b, MissionParty(nil, data.BodyList{}, nil))
}

// startEditedParty starts mission 10 from map bytes b with party.
func startEditedParty(t *testing.T, b []byte, party []mapload.PartyMember) *Mission {
	t.Helper()
	dir := t.TempDir()
	archive := filepath.Join(dir, ScenarioArchive)
	if err := os.WriteFile(archive, synth.Archive([]synth.File{{Path: "10.alm", Data: b}}), 0o644); err != nil {
		t.Fatal(err)
	}
	fsys, err := vfs.Open([]string{archive}, nil)
	if err != nil {
		t.Fatal(err)
	}
	m, err := StartMission(fsys, 10, nil, mapload.DifficultyNormal, party)
	if err != nil {
		t.Fatal(err)
	}
	if m.RaiseErr != nil {
		t.Fatalf("the script did not compile: %v", m.RaiseErr)
	}
	return m
}

// A map the wizard wrote starts through the game's reader: the party drops
// on the tool's zone and the box check is bound to the hero. The walk itself
// needs the installed unit table (TestReleaseTriggerWizardMapPlaysTheTrigger).
func TestTriggerWizardMapStartsInTheGame(t *testing.T) {
	e, _ := heroAreaMap(t, [4]int{10, 4, 12, 6})
	m := startEdited(t, e.Document().SourceBytes)
	if m.Start.Cells[0] != (mapload.Cell{X: 12, Y: 12}) {
		t.Fatalf("the hero starts at %v, the drop zone is 12,12", m.Start.Cells[0])
	}
	c := m.World.Script().Checks()
	if len(c) != 2 || c[0].Op != 2 || !c[0].HasUnit || c[0].Unit != m.Start.IDs[0] || [4]int32(c[0].Args[:4]) != [4]int32{10, 4, 12, 6} {
		t.Fatalf("the compiled checks %+v", c)
	}
	if got := wizSteps(m.World, 3); got != sim.OutcomeUndecided {
		t.Fatalf("the trigger fired with the hero outside the area: %d", got)
	}
}

// wizSteps runs n script cycles of 16 ticks and reports the outcome.
func wizSteps(w *sim.World, n int) sim.Outcome {
	for i := 0; i < 16*n && w.Outcome() == sim.OutcomeUndecided; i++ {
		sim.Step(w, nil)
	}
	return w.Outcome()
}
