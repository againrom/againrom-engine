package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/reg"
	"againrom/pkg/ui"
)

// Mission 20, a unit selected, the pointer on three installed structure classes.
func TestReleaseHoverCursorOverStructureClasses(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	a := f.App("structure hover")
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpener(20)); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	live := f.live
	if err := a.HeadlessSelectEntity(uint32(live.mission.ids[0])); err != nil {
		t.Fatal(err)
	}
	for i := range live.fog.visible {
		live.fog.visible[i], live.fog.explored[i] = 1, 1
	}
	live.push()
	regBytes, err := f.Archives.Containers.ReadFile("graphics/structures/structures.reg")
	if err != nil {
		t.Fatal(err)
	}
	r, err := reg.Parse(regBytes)
	if err != nil {
		t.Fatal(err)
	}
	classes, err := data.LoadStructureClasses(r)
	if err != nil {
		t.Fatal(err)
	}
	pick := func(match func(*data.StructureClass) bool) (ui.InspectionSubject, int, int) {
		for _, s := range live.world.Structures() {
			c, ok := classes.ByID(int32(live.mission.state.Map.Objects[s.ID].Kind))
			if ok && s.MaxHealth != 0 && match(c) {
				return ui.InspectionSubject{Kind: ui.InspectionStructure, ID: uint32(s.ID)}, int(s.Col), int(s.Row)
			}
		}
		t.Fatal("no structure of the wanted class in mission 20")
		return ui.InspectionSubject{}, 0, 0
	}
	for _, tc := range []struct {
		name  string
		match func(*data.StructureClass) bool
		want  string
	}{
		{"indestructible unusable", func(c *data.StructureClass) bool { return c.Indestructible != 0 && c.Usable == 0 }, "move"},
		{"destructible", func(c *data.StructureClass) bool { return c.Indestructible == 0 && c.Usable == 0 }, "select"},
		{"usable", func(c *data.StructureClass) bool { return c.Usable != 0 }, "town"},
	} {
		ref, col, row := pick(tc.match)
		inspectionCentre(live, col, row)
		releaseHoverInspection(t, a, live, ref)
		got, ok := a.HeadlessMapCursor()
		if !ok || got != tc.want {
			t.Errorf("%s (structure %d): cursor %q,%v, want %q", tc.name, ref.ID, got, ok, tc.want)
		}
	}
}
