package game

import (
	"image"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// dropShape checks that action node a is a drop zone on cell as the shipped
// maps write one: "Drop Location" with an X and a Y, named by exactly one
// trigger "Deploy Location" with no condition pair and only that action,
// firing once.
func dropShape(t *testing.T, sc alm.Script, a alm.ScriptNode, cell image.Point) {
	t.Helper()
	if a.Label != "Drop Location" || a.Opcode != mapload.ScriptConstant ||
		a.Type != [10]uint32{5, 6} || a.Value != [10]uint32{uint32(cell.X), uint32(cell.Y)} {
		t.Fatalf("the drop action reads %+v", a)
	}
	n := 0
	for _, tr := range sc.Triggers {
		if tr.Acts[0] != a.ID {
			continue
		}
		n++
		if tr.Name != "Deploy Location" || tr.Left != [3]uint32{} || tr.Right != [3]uint32{} ||
			tr.Acts != [4]uint32{a.ID} || tr.Once != 1 {
			t.Fatalf("the drop trigger reads %+v", tr)
		}
	}
	if n != 1 {
		t.Fatalf("%d triggers name drop action %d", n, a.ID)
	}
}

// drops are the map's drop actions in file order.
func drops(sc alm.Script) []alm.ScriptNode {
	var out []alm.ScriptNode
	for _, a := range sc.Actions {
		if a.Opcode == mapload.ScriptConstant {
			out = append(out, a)
		}
	}
	return out
}

// The drop tool adds a zone per click on an empty cell, selects a zone by a
// click, moves it by a drag and deletes the selected zone with Del, its
// trigger with it; D arms and ends the same tool.
func TestDropToolAddsMovesAndDeletesZones(t *testing.T) {
	e, _ := openGoalMap(t, emptyScriptMap(t, 40))
	e.Fit()
	_ = e.Key("drop")
	if on, _ := e.DropToolOn(); !on {
		t.Fatal("D did not arm the drop tool")
	}
	clickCell(e, image.Pt(5, 5))
	clickCell(e, image.Pt(30, 8))
	if on, sel := e.DropToolOn(); !on || sel != 1 {
		t.Fatalf("the tool after two zones: on %v, selected %d", on, sel)
	}
	sc := wizScript(t, e.Document().SourceBytes)
	d := drops(sc)
	if len(d) != 2 || len(sc.Triggers) != 2 {
		t.Fatalf("%d drop actions and %d triggers", len(d), len(sc.Triggers))
	}
	dropShape(t, sc, d[0], image.Pt(5, 5))
	dropShape(t, sc, d[1], image.Pt(30, 8))
	if got := e.Document().Drops; len(got) != 2 || got[0] != image.Pt(5, 5) || got[1] != image.Pt(30, 8) {
		t.Fatalf("the document's drops %v", got)
	}

	clickCell(e, image.Pt(5, 5))
	if _, sel := e.DropToolOn(); sel != 0 {
		t.Fatalf("a click on zone 1 selected %d", sel)
	}
	dragCell(e, image.Pt(5, 5), image.Pt(9, 20))
	sc = wizScript(t, e.Document().SourceBytes)
	if d = drops(sc); len(d) != 2 {
		t.Fatalf("the drag left %d zones", len(d))
	}
	dropShape(t, sc, d[0], image.Pt(9, 20))
	if err := e.Key("undo"); err != nil || wizScript(t, e.Document().SourceBytes).Actions[0].Value[0] != 5 {
		t.Fatal("one Undo did not take the move back", err)
	}
	_ = e.Key("redo")
	if on, _ := e.DropToolOn(); !on {
		_ = e.Key("drop") // Undo ends the tool as it ends every pick
	}

	clickCell(e, image.Pt(30, 8))
	_ = e.Key("delete")
	sc = wizScript(t, e.Document().SourceBytes)
	if d = drops(sc); len(d) != 1 || len(sc.Triggers) != 1 {
		t.Fatalf("Del left %d zones and %d triggers", len(d), len(sc.Triggers))
	}
	dropShape(t, sc, d[0], image.Pt(9, 20))
	if on, _ := e.DropToolOn(); !on {
		t.Fatal("Del ended the tool")
	}
	_ = e.Key("drop")
	if on, _ := e.DropToolOn(); on {
		t.Fatal("D did not end the tool")
	}
}

// The tool's square on a map with no unit of player 1 is the radius
// MISSION-SCATTER-053 gives for the hero alone: 5.
func TestDropToolSquareHasTheScatterRadius(t *testing.T) {
	e, _ := openGoalMap(t, emptyScriptMap(t, 40))
	if got := e.DropToolRadius(); got != 5 {
		t.Fatalf("radius %d, want 5", got)
	}
}

// Two zones: a mission start lands the hero on one of the two cells, and
// across seeds on each of them.
func TestTwoDropZonesLandTheHeroOnEither(t *testing.T) {
	e, _ := openGoalMap(t, emptyScriptMap(t, 40))
	e.Fit()
	_ = e.Key("drop")
	clickCell(e, image.Pt(12, 12))
	clickCell(e, image.Pt(28, 27))
	m, err := alm.Open(e.Document().SourceBytes)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[image.Point]int{}
	for seed := uint64(1); seed <= 24; seed++ {
		mm, err := alm.Open(e.Document().SourceBytes)
		if err != nil {
			t.Fatal(err)
		}
		ms, err := startMissionFromWith(mm, "scenario/10.alm", 10, nil, mapload.DifficultyNormal, []mapload.PartyMember{{Class: 1}}, sim.NewDraws(seed))
		if err != nil {
			t.Fatal(err)
		}
		c := image.Pt(int(ms.Start.Cells[0].X), int(ms.Start.Cells[0].Y))
		if c != image.Pt(12, 12) && c != image.Pt(28, 27) {
			t.Fatalf("seed %d lands the hero on %v, drop %v fallback %v", seed, c, ms.Start.Drop, ms.Start.Fallback)
		}
		seen[c]++
	}
	if len(seen) != 2 {
		t.Fatalf("24 seeds land the hero on %v only", seen)
	}
	if cells := mapload.DropCells(m); len(cells) != 2 {
		t.Fatalf("the game reader reads %d drop cells", len(cells))
	}
}

var _ ui.InspectionDropZones = (*mapInspectionEdits)(nil)
