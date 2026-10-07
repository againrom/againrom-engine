package game

import (
	"image"
	"reflect"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Owner-reported mission-10 bridge/start-pad regression. The literal window
// pixels hit opaque decoration art in the failing build after inspectionCentre;
// they do not ask the inspection picker to choose an expected empty point.
// Only presentation fog is opened. Installs and simulation state stay unchanged.
func TestReleaseInspectionSkipsDecorations(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	a := f.App("decorative hover")
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	live := f.live
	if err := a.HeadlessSelectEntity(uint32(live.mission.ids[0])); err != nil {
		t.Fatal(err)
	}
	selected, _ := live.view.SelectedUnit()
	for i := range live.fog.visible {
		live.fog.visible[i], live.fog.explored[i] = 1, 1
	}
	live.push()
	hash, inventory, pending := live.world.Hash(), live.invSubject, len(live.pending)
	for _, tc := range []struct {
		name        string
		id          sim.StructureID
		kind        uint32
		cell, point image.Point
	}{
		{"bridge", 5, 38, image.Pt(42, 25), image.Pt(592, 368)},
		{"hanging decoration", 11, 36, image.Pt(42, 34), image.Pt(481, 301)},
		{"start pad", 14, 48, image.Pt(16, 65), image.Pt(320, 338)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := live.world.Structures()[tc.id]
			if s.ID != tc.id || live.mission.state.Map.Objects[tc.id].Kind != tc.kind ||
				image.Pt(int(s.Col), int(s.Row)) != tc.cell || s.Field42 != 0 || s.MaxHealth != 0 {
				t.Fatalf("installed decoration changed: %+v", s)
			}
			inspectionCentre(live, tc.cell.X, tc.cell.Y)
			if err := a.HeadlessPointer("hover", tc.point.X, tc.point.Y); err != nil {
				t.Fatal(err)
			}
			panel, ok := live.view.InspectionPanel()
			if !ok || panel.Kind != ui.InspectionUnit || panel.ID != selected {
				t.Errorf("decoration replaced selected unit's card: kind=%d id=%d HP=%d/%d", panel.Kind, panel.ID, panel.HP, panel.MaxHP)
			}
			if _, _, err := live.view.InspectionPoint(ui.InspectionSubject{Kind: ui.InspectionStructure, ID: uint32(tc.id)}); err == nil {
				t.Error("decoration still has an inspectable pixel")
			}
		})
	}
	// Towers stay inspectable, as does the indestructible well with real HP.
	for _, id := range []sim.StructureID{0, 10, 15, 16, 17} {
		s := live.world.Structures()[id]
		inspectionCentre(live, int(s.Col), int(s.Row))
		releaseHoverInspection(t, a, live, ui.InspectionSubject{Kind: ui.InspectionStructure, ID: uint32(id)})
		panel, ok := live.view.InspectionPanel()
		if !ok || panel.MaxHP != int(s.MaxHealth) || panel.HP != int(s.Field42) || panel.MaxHP == 0 {
			t.Fatalf("building %d lost its live card: %+v", id, panel)
		}
	}
	if live.world.Hash() != hash || len(live.pending) != pending || !reflect.DeepEqual(live.invSubject, inventory) {
		t.Fatal("hover changed simulation, commands or inventory")
	}
	t.Log("mission10: bridge, hanging decoration and start pad suppressed; four towers and well remain inspectable")
}
