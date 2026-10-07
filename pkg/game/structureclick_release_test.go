package game

import (
	"slices"
	"testing"

	"againrom/pkg/ui"
)

// Mission 20, the hero selected: a pointer click on Well 1, an indestructible
// structure with no use, is an ordinary move order and keeps the selection.
func TestReleaseClickOverIndestructibleStructureMoves(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	a := f.App("structure click")
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpener(20)); err != nil {
		t.Fatal(err)
	}
	live := f.live
	entryStep1084(a)
	id := live.mission.ids[0]
	for i := range live.fog.visible {
		live.fog.visible[i], live.fog.explored[i] = 1, 1
	}
	live.push()
	well := live.world.Structures()[1]
	ref := ui.InspectionSubject{Kind: ui.InspectionStructure, ID: uint32(well.ID)}
	// Centre on the well and read the point afresh before every pointer
	// frame: the viewer scrolls toward the selected unit on the frames between.
	cursor := func(act string) string {
		t.Helper()
		// The cursor manager shows a frame's choice on the next frame.
		for range 2 {
			inspectionCentre(live, int(well.Col), int(well.Row))
			x, y, err := live.view.InspectionPoint(ref)
			if err != nil {
				t.Fatal(err)
			}
			if err := a.HeadlessPointer(act, x, y); err != nil {
				t.Fatal(err)
			}
		}
		got, _ := a.HeadlessMapCursor()
		return got
	}
	dist := func() int {
		e, _ := live.entity(id)
		dx, dy := int(e.X)-int(well.Col), int(e.Y)-int(well.Row)
		return max(dx, -dx) + max(dy, -dy)
	}

	// No selection: the structure is skipped, so the pointer is the default.
	if got := cursor("hover"); got != "default" {
		t.Errorf("nothing selected: cursor %q, want default", got)
	}
	// Paused, the camera and the pointer stay where each frame puts them.
	if err := a.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	hero, _ := live.entity(id)
	inspectionCentre(live, int(hero.X), int(hero.Y))
	if err := a.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatal(err)
	}
	inspectionCentre(live, int(well.Col), int(well.Row))
	live.push()
	// The selection shows the command panel, which shrinks the map view over
	// a few frames; hover until the camera is steady.
	for range 10 {
		cursor("hover")
	}
	if got := cursor("ctrl-hover"); got != "swarm" {
		t.Errorf("Ctrl held: cursor %q, want swarm", got)
	}
	if got := cursor("hover"); got != "move" {
		t.Errorf("unit selected: cursor %q, want move", got)
	}
	before, pending := dist(), len(live.pending)
	cursor("press")
	cursor("release")
	if sel := a.HeadlessSelection(); !slices.Equal(sel, []uint32{uint32(id)}) {
		t.Errorf("selection after the click %v, want [%d]", sel, id)
	}
	if len(live.pending) != pending+1 {
		t.Fatalf("pending orders %d, want one move order added to %d", len(live.pending), pending)
	}
	// The hover frames ran paused; the order is followed on a running world.
	if err := a.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	for range 400 {
		entryStep1084(a)
	}
	e, _ := live.entity(id)
	col, row := int(well.Col), int(well.Row)
	tx, ty := int(e.TargetX), int(e.TargetY)
	if !e.HasTarget || tx < col-1 || tx > col+int(well.Width) || ty < row-1 || ty > row+int(well.Height) {
		t.Errorf("unit target %d,%d (has %v), want a cell at or beside the well %d,%d", tx, ty, e.HasTarget, col, row)
	}
	if after := dist(); after > before-20 {
		t.Errorf("distance to the well %d then %d after 400 steps, want the unit walking toward it", before, after)
	}
}
