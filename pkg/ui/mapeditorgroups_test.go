package ui

import (
	"fmt"
	"image"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
)

// groupsUnit is one unit of the groups probe: its owner, group and cell.
type groupsUnit struct {
	player, group int
	cell          image.Point
}

// groupsProbe is the full host with units that carry players and groups, a
// trigger whose action names group 1, and the group seam.
type groupsProbe struct {
	modesFull
	units []groupsUnit
}

func (p *groupsProbe) SetUnitsGroup(refs []RecordRef, group int) (*InspectionDocument, error) {
	p.calls.add("SetUnitsGroup %v %d", refs, group)
	for _, r := range refs {
		p.units[r.Index].group = group
	}
	p.dirty = true
	return p.doc(p.self), nil
}

func (p *groupsProbe) FreeGroup() int {
	n := 1
	for slices.ContainsFunc(p.units, func(u groupsUnit) bool { return u.group == n }) {
		n++
	}
	return n
}

// groupsEditor opens the probe's map: units 0 and 1 of player 1 in group 1,
// unit 2 of player 1 in none, unit 3 of player 2 in group 2, unit 4 of
// player 2 in none; record 5 a structure, 6 an action naming group 1 and 7
// the trigger that runs it.
func groupsEditor(t *testing.T) (*MapEditor, *groupsProbe) {
	t.Helper()
	p := &groupsProbe{units: []groupsUnit{{1, 1, image.Pt(10, 10)}, {1, 1, image.Pt(12, 10)}, {1, 0, image.Pt(14, 10)}, {2, 2, image.Pt(16, 10)}, {2, 0, image.Pt(18, 10)}}}
	p.calls, p.self, p.players = &modesCalls{}, p, []string{"Self", "Enemy"}
	build := func(source string) *InspectionDocument {
		v, err := NewViewer("groups", terrain.Grid{Width: 50, Height: 50, Tiles: make([]uint16, 2500)}, &terrain.Tileset{})
		if err != nil {
			t.Fatal(err)
		}
		d := &InspectionDocument{Viewer: v, Source: source, Width: 50, Height: 50, SourceBytes: []byte("map"), Edits: p}
		members := map[int]int{}
		for i, u := range p.units {
			d.Records = append(d.Records, InspectionRecord{Kind: "Unit", Index: i, Section: 6, Cell: u.cell, Spatial: true, Size: image.Pt(1, 1), Label: fmt.Sprintf("unit%d", i), Player: u.player, Group: u.group})
			members[u.group]++
		}
		d.Records = append(d.Records,
			InspectionRecord{Kind: "Structure", Index: 0, Section: 4, Cell: image.Pt(30, 30), Spatial: true, Size: image.Pt(2, 2), Label: "hall"},
			InspectionRecord{Kind: "Action", Index: 0, Section: 7},
			InspectionRecord{Kind: "Trigger", Index: 0, Section: 7})
		var targets []InspectionScriptTarget
		for g := 1; g <= 3; g++ {
			t := InspectionScriptTarget{Value: uint32(g), Record: -1, Members: members[g]}
			if g == 1 {
				t.Uses = []InspectionScriptUse{{Node: 6, Slot: 0, Triggers: []int{7}}}
			}
			targets = append(targets, t)
		}
		d.Script = &InspectionScriptModel{
			Actions:  []InspectionScriptNode{{Record: 6, ID: 1, Opcode: 32, Label: "off map", Types: [10]uint32{scriptGroup}, Values: [10]uint32{1}}},
			Triggers: []InspectionTrigger{{Record: 7, Name: "Hide them", Acts: [4]uint32{1}}},
			Targets:  map[uint32][]InspectionScriptTarget{scriptGroup: targets},
		}
		return d
	}
	p.doc = func(InspectionEdits) *InspectionDocument { return build("edited") }
	e := NewMapEditor(panelFont(), []EditorMap{{Key: "map0", Label: "Map 0"}}, func(source string) (*InspectionDocument, error) { return build(source), nil })
	e.Layout(1024, 768)
	if err := e.Open("map0"); err != nil {
		t.Fatal(err)
	}
	e.listAt, e.listPlaced = image.Pt(8, e.barHeight()+8), true
	e.objectAt, e.objPlaced = image.Pt(e.window.X, e.window.Y), true
	return e, p
}

// dragOnMap presses at cell a, moves to cell b with the button held and
// releases there, as the pointer does.
func dragOnMap(e *MapEditor, a, b image.Point, mid ...image.Point) {
	now := time.Unix(1, 0)
	pa := e.CellPoint(a)
	e.Pointer(Input{CursorX: pa.X, CursorY: pa.Y, PrimaryDown: true}, now)
	for _, c := range append(mid, b) {
		pc := e.CellPoint(c)
		e.Pointer(Input{CursorX: pc.X, CursorY: pc.Y, PrimaryDown: true}, now)
	}
	pb := e.CellPoint(b)
	e.Pointer(Input{CursorX: pb.X, CursorY: pb.Y}, now)
}

func refsText(refs []RecordRef) string { return fmt.Sprint(refs) }

// The Units tab's picture grid: a press on a cell opens the Add window on
// that class with the cell pick armed and shows the cell pressed; each click
// on the map adds one copy, a held drag adds one per cell it crosses, and a
// second press on the pressed cell ends the placing.
func TestMapEditorPlaceGridChoosesAndPlacesCopies(t *testing.T) {
	e, calls := newModesEditor(t, "full")
	e.listAt = image.Pt(8, e.barHeight()+8)
	chromeClick(e, barPoint(t, e, "units"))
	if n := e.PlaceGridLen(); n != 2 {
		t.Fatalf("the Units grid holds %d cells, want the host's 2 classes", n)
	}
	r, ok := e.PlaceGridCellRect(1)
	if !ok {
		t.Fatal("grid cell 1 is not shown")
	}
	chromeClick(e, center(r))
	run, ok := e.addRunning()
	if !ok || !run.repeat || run.kind != "Unit" || e.pick != pickAddCell {
		t.Fatalf("the grid press opened no placing Add window: %+v pick %q", run, e.pick)
	}
	g, note, _ := e.placeGridSpec()
	if g.Cells[0].Pressed || !g.Cells[1].Pressed || !strings.Contains(note, "2: Two") {
		t.Fatalf("pressed cells %v %v, note %q", g.Cells[0].Pressed, g.Cells[1].Pressed, note)
	}
	e.Fit()
	modesClick(e, e.CellPoint(image.Pt(40, 20)))
	if len(calls.lines) != 1 || !strings.Contains(calls.lines[0], "AddRecord Unit (40,20) true [{class 2") {
		t.Fatalf("a click placed %v", calls.lines)
	}
	if _, ok := e.addRunning(); !ok || e.pick != pickAddCell {
		t.Fatal("the placing ended after one copy")
	}
	dragOnMap(e, image.Pt(40, 30), image.Pt(42, 30), image.Pt(41, 30))
	var cells []string
	for _, l := range calls.lines[1:] {
		if !strings.HasPrefix(l, "AddRecord Unit ") {
			t.Fatalf("the held drag called %q", l)
		}
		cells = append(cells, strings.Fields(l)[2])
	}
	if strings.Join(cells, " ") != "(40,30) (41,30) (42,30)" {
		t.Fatalf("the held drag placed on %v, want one copy per cell", cells)
	}
	if run, _ := e.addRunning(); run.placed != 4 {
		t.Fatalf("the window counts %d copies", run.placed)
	}
	r, _ = e.PlaceGridCellRect(1)
	chromeClick(e, center(r))
	if _, ok := e.addRunning(); ok || e.pick != pickNone {
		t.Fatal("a press on the pressed cell did not end the placing")
	}
	// Loss control: the Add button still adds one record and closes.
	before := len(calls.lines)
	s, _ := e.listSpec()
	chromeClick(e, center(layoutPopup(s, e.font, e.bodyFont()).Buttons[0]))
	if run, ok := e.addRunning(); !ok || run.repeat {
		t.Fatal("the Add button opened a placing window")
	}
	if len(calls.lines) != before {
		t.Fatal("opening the Add window wrote")
	}
}

// A left drag on the map with no tool draws a box that selects the records
// inside it, only the open tab's kind on the Units and Structures tabs; a
// click that does not move selects as a click does.
func TestMapEditorBoxSelectsTheOpenTabsRecords(t *testing.T) {
	e, calls := newModesEditor(t, "full")
	e.Fit()
	dragOnMap(e, image.Pt(5, 5), image.Pt(35, 35))
	if got := refsText(e.selection.refs); got != "[{Unit 0} {Unit 1} {Structure 0} {Sack 0}]" {
		t.Fatalf("a box with no tab selected %s", got)
	}
	e.showRecords(1)
	e.listAt = image.Pt(e.window.X, e.window.Y)
	e.Fit()
	dragOnMap(e, image.Pt(5, 5), image.Pt(35, 35))
	if got := refsText(e.selection.refs); got != "[{Unit 0} {Unit 1}]" {
		t.Fatalf("a box on the Units tab selected %s", got)
	}
	e.showRecords(2)
	e.Fit()
	dragOnMap(e, image.Pt(5, 5), image.Pt(35, 35))
	if e.selected < 0 || e.doc.Records[e.selected].Kind != "Structure" || len(e.selection.refs) != 1 {
		t.Fatalf("a box on the Structures tab selected %v (record %d)", e.selection.refs, e.selected)
	}
	// A click with no move on bare ground clears the selection; on a unit it
	// selects the unit.
	modesClick(e, e.CellPoint(image.Pt(25, 25)))
	if e.selected >= 0 || len(e.selection.refs) != 0 {
		t.Fatal("a click on bare ground kept a selection")
	}
	modesClick(e, e.CellPoint(image.Pt(10, 10)))
	if e.selected != 0 {
		t.Fatalf("a click on unit 0 selected %d", e.selected)
	}
	if len(calls.lines) != 0 {
		t.Fatal("selecting wrote", calls.lines)
	}
}

// Dragging a selected group shows its records' silhouettes at the cells the
// release would put them on, writes nothing until the release, and moves the
// whole selection as one edit.
func TestMapEditorMovedSelectionShowsSilhouettesUntilRelease(t *testing.T) {
	e, calls := newModesEditor(t, "full")
	e.Fit()
	dragOnMap(e, image.Pt(5, 5), image.Pt(25, 15))
	if got := refsText(e.selection.refs); got != "[{Unit 0} {Unit 1}]" {
		t.Fatalf("the box selected %s", got)
	}
	now := time.Unix(1, 0)
	a, b := e.CellPoint(image.Pt(10, 10)), e.CellPoint(image.Pt(13, 12))
	e.Pointer(Input{CursorX: a.X, CursorY: a.Y, PrimaryDown: true}, now)
	e.Pointer(Input{CursorX: b.X, CursorY: b.Y, PrimaryDown: true}, now)
	f, ok := e.dragPlacement()
	if !ok || !f.Dragged || f.Cell != image.Pt(13, 12) || len(calls.lines) != 0 {
		t.Fatalf("no silhouettes during the move: %+v %v, wrote %v", f, ok, calls.lines)
	}
	if !e.recordDrag.group {
		t.Fatal("the drag does not hold the selection")
	}
	e.Pointer(Input{CursorX: b.X, CursorY: b.Y}, now)
	if _, ok := e.dragPlacement(); ok {
		t.Fatal("silhouettes stay after the release")
	}
	if len(calls.lines) != 1 || calls.lines[0] != "MoveRecords [{Unit 0} {Unit 1}] (3,2)" {
		t.Fatalf("the release wrote %v", calls.lines)
	}
}

// The selection's popup acts on every selected record: Delete deletes them
// as one edit.
func TestMapEditorSelectionPopupDeletesTheSelection(t *testing.T) {
	e, calls := newModesEditor(t, "full")
	e.objectAt = image.Pt(400, 100)
	e.Fit()
	dragOnMap(e, image.Pt(5, 5), image.Pt(35, 35))
	r, ok := e.SelectionButtonRect(2)
	if !ok {
		t.Fatal("no selection popup")
	}
	chromeClick(e, center(r))
	if len(calls.lines) != 1 || calls.lines[0] != "DeleteRecords [{Unit 0} {Unit 1} {Structure 0} {Sack 0}]" {
		t.Fatalf("Delete wrote %v", calls.lines)
	}
}

// The Groups window lists each player's groups with number, member count,
// member pictures and the triggers that name them.
func TestMapEditorGroupsWindowRows(t *testing.T) {
	e, _ := groupsEditor(t)
	chromeClick(e, barPoint(t, e, "groups"))
	got := strings.Join(e.GroupsText(), "|")
	for _, want := range []string{"Player 1|Group 1 2 units|Used by Trigger 1  Hide them: Action 1  off map|Player 2|Group 2 1 units", "Selected units: 0"} {
		if !strings.Contains(got, want) {
			t.Fatalf("the Groups window reads %q, lacks %q", got, want)
		}
	}
	// The palette lists no group; a group's pictures and its delete button
	// stand in the chosen group's window.
	if s, _ := e.listSpec(); s.List != nil || len(s.Rows) != 0 {
		t.Fatalf("the Groups palette lists rows: %v", s.Rows)
	}
	r, ok := e.GroupsRowRect(1, 1, false)
	if !ok {
		t.Fatal("no tree row of group 1")
	}
	chromeClick(e, center(r))
	s, _, _ := e.objectSpec()
	if len(s.Rows) < 2 || s.Rows[1].Value != "2 units" || len(s.Rows[1].Icons) != len(e.memberIcons(e.groupSel)) || len(s.Buttons) == 0 || s.Buttons[0] != "Delete group" {
		t.Fatalf("the group window lacks its pictures or Delete group: %+v %v", s.Rows, s.Buttons)
	}
}

// A click on a group's row selects its members, marks them with the outline
// and the number badge, and centres the view on them.
func TestMapEditorGroupsRowChoosesAndFramesTheGroup(t *testing.T) {
	e, _ := groupsEditor(t)
	chromeClick(e, barPoint(t, e, "groups"))
	camX, camY := e.doc.Viewer.cam.X, e.doc.Viewer.cam.Y
	r, ok := e.GroupsRowRect(1, 1, false)
	if !ok {
		t.Fatal("no row of group 1")
	}
	chromeClick(e, center(r))
	if got := fmt.Sprint(e.GroupMembers()) + " " + refsText(e.selection.refs); got != "[0 1] [{Unit 0} {Unit 1}]" {
		t.Fatalf("the row chose %s", got)
	}
	if n := e.GroupBadges(); n != 2 {
		t.Fatalf("%d badges, want 2", n)
	}
	if len(e.groupMarkerLines()) != 8 {
		t.Fatal("the members carry no outline")
	}
	if e.doc.Viewer.cam.X == camX && e.doc.Viewer.cam.Y == camY {
		t.Fatal("the view did not move to the group")
	}
	// The number keys choose a group too. Control: a number
	// no group carries chooses nothing.
	_ = e.Key("group2")
	if got := fmt.Sprint(e.GroupMembers()); got != "[3]" {
		t.Fatalf("key 2 chose %s", got)
	}
	_ = e.Key("group7")
	if got := fmt.Sprint(e.GroupMembers()); got != "[3]" || !strings.Contains(e.selection.note, "no group 7") {
		t.Fatalf("key 7 chose %s, note %q", got, e.selection.note)
	}
}

// To group... puts the selected units in a new group, the first free
// number, or a group of their player, and skips another player's units with
// a note.
func TestMapEditorToGroupNewAndExistingSkipsOtherPlayers(t *testing.T) {
	e, p := groupsEditor(t)
	chromeClick(e, barPoint(t, e, "groups"))
	e.setSelection([]RecordRef{{"Unit", 2}, {"Unit", 4}})
	b, ok := e.GroupsButtonRect(0)
	if !ok {
		t.Fatal("no To group button")
	}
	chromeClick(e, center(b))
	if e.prompt.kind != promptToGroup {
		t.Fatal("To group opened no question")
	}
	s, _ := e.promptSpec()
	rows := e.prompt.group.list.Rows()
	if len(rows) != 2 || rows[0].Text != "New (group 3)" || rows[1].Text != "Group 1: 2 units" || !strings.Contains(strings.Join(s.Notes, " "), "1 units of other players skipped") {
		t.Fatalf("the question offers %v, notes %q", rows, s.Notes)
	}
	ok2, _ := e.PromptButtonRect(0)
	chromeClick(e, center(ok2))
	if got := strings.Join(p.calls.lines, "|"); got != "SetUnitsGroup [{Unit 2}] 3" {
		t.Fatalf("New wrote %q", got)
	}
	if got := fmt.Sprint(e.GroupMembers()); got != "[2]" || !strings.Contains(e.selection.note, "skipped") {
		t.Fatalf("the new group is not chosen: %s, note %q", got, e.selection.note)
	}
	// An existing group: the second row.
	e.setSelection([]RecordRef{{"Unit", 2}})
	e.toGroupAsk()
	row, ok := e.PromptListRowRect(1)
	if !ok {
		t.Fatal("no row 1 in the question")
	}
	chromeClick(e, center(row))
	ok2, _ = e.PromptButtonRect(0)
	chromeClick(e, center(ok2))
	if got := p.calls.lines[len(p.calls.lines)-1]; got != "SetUnitsGroup [{Unit 2}] 1" {
		t.Fatalf("the existing group wrote %q", got)
	}
}

// Ctrl+1..9 puts the selection's units in that group; with a group of that
// number, only units of the group's player go in.
func TestMapEditorCtrlNumberPutsTheSelectionInAGroup(t *testing.T) {
	e, p := groupsEditor(t)
	e.setSelection([]RecordRef{{"Unit", 2}, {"Unit", 4}, {"Structure", 0}})
	_ = e.Key("togroup2")
	if got := strings.Join(p.calls.lines, "|"); got != "SetUnitsGroup [{Unit 4}] 2" {
		t.Fatalf("Ctrl+2 wrote %q", got)
	}
	if note := e.selection.note; !strings.Contains(note, "1 units of other players skipped") || !strings.Contains(note, "1 records that are not units skipped") {
		t.Fatalf("the note %q names no skip", note)
	}
	if got := fmt.Sprint(e.GroupMembers()); got != "[3 4]" {
		t.Fatalf("Ctrl+2 chose %s", got)
	}
	// Loss control: no unit selected writes nothing.
	e.setSelection([]RecordRef{{"Structure", 0}})
	_ = e.Key("togroup5")
	if len(p.calls.lines) != 1 {
		t.Fatal("Ctrl+5 with no unit wrote", p.calls.lines)
	}
}

// Remove from group and the row's cross clear the number; emptying a group a
// trigger names asks first and names the trigger, and Cancel writes nothing.
func TestMapEditorRemoveFromGroupWarnsBeforeEmptyingAUsedGroup(t *testing.T) {
	e, p := groupsEditor(t)
	chromeClick(e, barPoint(t, e, "groups"))
	row, _ := e.GroupsRowRect(2, 2, false)
	chromeClick(e, center(row))
	cross, ok := e.GroupsRowRect(2, 2, true)
	if !ok {
		t.Fatal("no Delete group on group 2")
	}
	chromeClick(e, center(cross))
	if got := strings.Join(p.calls.lines, "|"); got != "SetUnitsGroup [{Unit 3}] 0" || e.prompt.open() {
		t.Fatalf("the cross of an unused group wrote %q (prompt %v)", got, e.prompt.open())
	}
	// Taking one of two members out leaves group 1 a member: no question.
	e.setSelection([]RecordRef{{"Unit", 0}})
	b, _ := e.GroupsButtonRect(1)
	chromeClick(e, center(b))
	if len(p.calls.lines) != 2 || p.calls.lines[1] != "SetUnitsGroup [{Unit 0}] 0" || e.prompt.open() {
		t.Fatalf("Remove wrote %v (prompt %v)", p.calls.lines, e.prompt.open())
	}
	// The last member: the question names the trigger.
	row, _ = e.GroupsRowRect(1, 1, false)
	chromeClick(e, center(row))
	cross, _ = e.GroupsRowRect(1, 1, true)
	chromeClick(e, center(cross))
	if e.prompt.kind != promptGroupWarn || len(p.calls.lines) != 2 {
		t.Fatalf("emptying a used group asked nothing: prompt %d, calls %v", e.prompt.kind, p.calls.lines)
	}
	s, _ := e.promptSpec()
	if text := strings.Join(s.Notes, " "); !strings.Contains(text, "Group 1 will have no units") || !strings.Contains(text, "Hide them") {
		t.Fatalf("the warning reads %q", text)
	}
	cancel, _ := e.PromptButtonRect(1)
	chromeClick(e, center(cancel))
	if e.prompt.open() || len(p.calls.lines) != 2 {
		t.Fatal("Cancel wrote or stayed", p.calls.lines)
	}
	cross, _ = e.GroupsRowRect(1, 1, true)
	chromeClick(e, center(cross))
	goOn, _ := e.PromptButtonRect(0)
	chromeClick(e, center(goOn))
	if got := p.calls.lines[len(p.calls.lines)-1]; got != "SetUnitsGroup [{Unit 1}] 0" || len(p.calls.lines) != 3 {
		t.Fatalf("Empty it wrote %v", p.calls.lines)
	}
	// Moving a used group's last members into another group asks too.
	e2, p2 := groupsEditor(t)
	e2.setSelection([]RecordRef{{"Unit", 0}, {"Unit", 1}})
	_ = e2.Key("togroup4")
	if e2.prompt.kind != promptGroupWarn || len(p2.calls.lines) != 0 {
		t.Fatalf("moving group 1's members out asked nothing: %v", p2.calls.lines)
	}
}

// Delete on group 1, which a trigger names and units 0 and 1 hold (unit 0
// alone on the single route), asks first by the key, the selection popup's
// button and the selected record's own delete; Cancel writes nothing and
// Empty it deletes.
func TestMapEditorDeleteAsksBeforeEmptyingAUsedGroup(t *testing.T) {
	for _, route := range []string{"key", "button", "single"} {
		t.Run(route, func(t *testing.T) {
			e, p := groupsEditor(t)
			e.objectAt = image.Pt(400, 100)
			want := "DeleteRecords [{Unit 0} {Unit 1}]"
			if route == "single" {
				e.setSelection([]RecordRef{{"Unit", 1}})
				e.removeFromGroup()
				e.setSelection(nil)
				e.Select(0, false)
				p.calls.lines = nil
				want = "Delete Unit 0"
			} else {
				e.setSelection([]RecordRef{{"Unit", 0}, {"Unit", 1}})
			}
			press := func() {
				switch route {
				case "button":
					r, ok := e.SelectionButtonRect(2)
					if !ok {
						t.Fatal("missing Delete button")
					}
					chromeClick(e, center(r))
				default:
					if err := e.Key("delete"); err != nil {
						t.Fatal(err)
					}
				}
			}
			press()
			if e.prompt.kind != promptGroupWarn || len(p.calls.lines) != 0 {
				t.Fatalf("Delete emptied a used group without asking: prompt %d, calls %v", e.prompt.kind, p.calls.lines)
			}
			s, _ := e.promptSpec()
			if text := strings.Join(s.Notes, " "); !strings.Contains(text, "Group 1 will have no units") || !strings.Contains(text, "Hide them") {
				t.Fatalf("the warning reads %q", text)
			}
			cancel, _ := e.PromptButtonRect(1)
			chromeClick(e, center(cancel))
			if e.prompt.open() || len(p.calls.lines) != 0 {
				t.Fatalf("Cancel wrote or stayed: %v", p.calls.lines)
			}
			press()
			goOn, _ := e.PromptButtonRect(0)
			chromeClick(e, center(goOn))
			if len(p.calls.lines) != 1 || p.calls.lines[0] != want {
				t.Fatalf("Empty it wrote %v, want %q", p.calls.lines, want)
			}
		})
	}
}

// Remove from group on units of two players takes the group's player's units
// out and notes the others.
func TestMapEditorRemoveFromGroupSkipsOtherPlayersUnits(t *testing.T) {
	e, p := groupsEditor(t)
	e.setSelection([]RecordRef{{"Unit", 0}, {"Unit", 3}, {"Structure", 0}})
	e.removeFromGroup()
	if got := strings.Join(p.calls.lines, "|"); got != "SetUnitsGroup [{Unit 0}] 0" {
		t.Fatalf("Remove from group wrote %q", got)
	}
	if note := e.selection.note; !strings.Contains(note, "units of other players skipped") {
		t.Fatalf("the outcome notes %q", note)
	}
}
