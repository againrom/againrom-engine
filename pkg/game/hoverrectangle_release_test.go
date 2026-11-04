package game

import (
	"image"
	"math"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// Installed first missions, both games and roots. Only presentation fog is opened.
func TestReleaseMissionHoverRectangleReaders(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	a := f.App("mission hover rectangles")
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpener(f.NewGameMission())); err != nil {
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
	m := live.mission.state.Map
	proj := terrain.Project(m.Altitudes, m.Width, m.Height)
	cam := live.view.Camera()
	cam.SetZoom(1)
	before, pending := live.world.Hash(), len(live.pending)
	frameRect := func(r image.Rectangle) image.Rectangle {
		x, y := cam.WorldToScreen(float64(r.Min.X), float64(r.Min.Y))
		return image.Rect(int(math.Round(x)), int(math.Round(y)), int(math.Round(x))+r.Dx(), int(math.Round(y))+r.Dy())
	}
	checkEdge := func(ref ui.InspectionSubject, bounds image.Rectangle) bool {
		t.Helper()
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			point := image.Pt(bounds.Max.X, y)
			wx, wy, ok := live.view.Placement().FrameToWindow(point)
			if !ok || point.X < 24 || point.Y < 24 || point.X >= cam.ViewW-24 || point.Y >= cam.ViewH-24 {
				continue
			}
			if err := a.HeadlessPointer("ctrl-hover", wx, wy); err != nil {
				t.Fatal(err)
			}
			if got, ok := live.view.Inspection(); !ok || got != ref {
				continue
			}
			panel, ok := live.view.InspectionPanel()
			if !ok || panel.Kind != ref.Kind || panel.ID != ref.ID {
				t.Fatalf("edge %v: panel=%v:%d/%v", point, panel.Kind, panel.ID, ok)
			}
			name, ok := a.HeadlessMapCursor()
			if !ok || name != "attack" {
				t.Fatalf("edge %v: cursor=%q/%v", point, name, ok)
			}
			_, at, err := a.HeadlessMissionCursor()
			if err != nil || at.Add(image.Pt(3, 3)) != point {
				t.Fatalf("edge %v: hotspot at %v: %v", point, at, err)
			}
			return true
		}
		return false
	}
	unitOK := false
	for _, e := range live.entityDraws() {
		if e.Art == nil || e.Frame == nil || e.Untargetable || e.Step != (image.Point{}) || e.Art.Selection.Empty() {
			continue
		}
		inspectionCentre(live, e.Cell.X, e.Cell.Y)
		lift, origin := 0, 0
		if live.view.Mode() == ui.ModeDisplaced {
			lift, origin = proj.AnchorHeight(e.Cell.X, e.Cell.Y), proj.MinV
		}
		offset := image.Pt(e.Cell.X*32+16-e.Art.CenterX, e.Cell.Y*32+16-e.Art.CenterY-lift-origin)
		offset = offset.Add(image.Pt(max(0, e.TokenSize-1)*16, max(0, e.TokenSize-1)*16)).Add(e.DamageJolt)
		if e.FinePosition {
			offset = offset.Add(image.Pt((int(e.FineX)-128)*32/256, (int(e.FineY)-128)*32/256))
		}
		if !checkEdge(ui.InspectionSubject{Kind: ui.InspectionUnit, ID: e.ID}, frameRect(e.Art.Selection.Canon().Add(offset))) {
			continue
		}
		draws, err := live.view.HeadlessArtDraws()
		if err != nil {
			t.Fatal(err)
		}
		// The frame placement differs from the registered canvas rectangle.
		dx, dy := cam.WorldToScreen(float64(offset.X+e.Art.Width/2-e.Frame.Width/2), float64(offset.Y+e.Art.Height/2-e.Frame.Height/2))
		found := false
		for _, d := range draws {
			x, y := d.Geometry.Apply(0, 0)
			if d.Frame == e.Frame && x == dx && y == dy && d.Kind == "sprite" {
				found = true
				if d.Shading != 0 || d.ColorScale.R() != 1 {
					t.Fatalf("hovered unit %d: shading=%d gain=%v", e.ID, d.Shading, d.ColorScale.R())
				}
			}
		}
		if !found {
			t.Fatal("hovered unit has no production sprite draw")
		}
		unitOK = true
		break
	}
	if !unitOK {
		t.Fatal("no unit rectangle has an exposed box edge in the first mission")
	}
	structureOK := false
	for _, s := range live.world.Structures() {
		c := live.view.StructureClass(s.Kind)
		if c == nil || c.Selection.Empty() || c.Indestructible && !c.Usable || s.MaxHealth == 0 && s.UseAmount == 0 {
			continue
		}
		col, row := int(s.Col), int(s.Row)
		inspectionCentre(live, col, row)
		lift, origin := 0, 0
		if live.view.Mode() == ui.ModeDisplaced {
			lift, origin = proj.AnchorHeight(col, row), proj.MinV
		}
		offset := image.Pt(col*32, (row+c.TileHeight-c.FullHeight)*32-lift-origin)
		if checkEdge(ui.InspectionSubject{Kind: ui.InspectionStructure, ID: uint32(s.ID)}, frameRect(c.Selection.Canon().Add(offset))) {
			structureOK = true
			break
		}
	}
	if !structureOK {
		t.Fatal("no admitted structure rectangle has an exposed box edge in the first mission")
	}
	if live.world.Hash() != before || len(live.pending) != pending {
		t.Fatal("hover changed world state or queued a command")
	}
}
