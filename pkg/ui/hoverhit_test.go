package ui

import (
	"bytes"
	"image"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
)

func hoverHitFixture(t *testing.T, second bool) (*App, *Viewer) {
	t.Helper()
	a, v := inspectionFixture(t, image.Pt(1024, 768))
	v.SetMissionScreen(MissionScreen{PickUsableInExploredGround: second})
	v.SetHoverProfile(KindRectangleHover)
	giveEntitiesArt(v)
	for i := range v.entities {
		c := *v.entities[i].Art
		c.Selection = image.Rect(0, 0, 48, 48)
		v.entities[i].Art = &c
	}
	v.structureInfo[7].Selection = image.Rect(0, 0, 48, 48)
	v.entities[1].Hostile = true
	return a, v
}

func TestHoverKindRectangleBoxAndRegistration(t *testing.T) {
	for _, second := range []bool{false, true} {
		a, v := hoverHitFixture(t, second)
		for i := range v.entities[1].Frame.Pixels {
			v.entities[1].Frame.Pixels[i].Opaque = false
		}
		for _, p := range []image.Point{{300, 212}, {304, 212}} {
			if err := a.HeadlessPointer("hover", p.X, p.Y); err != nil {
				t.Fatal(err)
			}
			if got, ok := v.hoverInspection(); !ok || got != (InspectionSubject{InspectionUnit, 7}) {
				t.Errorf("second=%v pointer=%v: hit=%v/%v", second, p, got, ok)
			}
		}
		if err := a.HeadlessPointer("hover", 305, 212); err != nil {
			t.Fatal(err)
		}
		if _, ok := v.hoverInspection(); ok {
			t.Fatal("two pixels beyond the last rectangle pixel hit")
		}
		v.entities[0].Cell = v.entities[1].Cell
		if err := a.HeadlessPointer("hover", 270, 212); err != nil {
			t.Fatal(err)
		}
		if got, ok := v.hoverInspection(); !ok || got.ID != 7 {
			t.Fatalf("last registered unit lost: %v/%v", got, ok)
		}
	}
}

func TestHoverReadersAndClickShareTheRectangle(t *testing.T) {
	for _, second := range []bool{false, true} {
		for _, category := range []terrain.UnitCategory{terrain.UnitOrdinary, terrain.UnitAir} {
			a, v := hoverHitFixture(t, second)
			v.entities[1].DrawCategory = category
			if err := a.HeadlessPointer("hover", 304, 212); err != nil {
				t.Fatal(err)
			}
			v.refreshHoverLighting()
			if name, _ := v.missionHoverCursor(); name != "attack" {
				t.Errorf("cursor=%q", name)
			}
			if p, ok := v.InspectionPanel(); !ok || p.Kind != InspectionUnit || p.ID != 7 {
				t.Errorf("panel=%v/%v", p, ok)
			}
			if _, lit := v.hoverSpriteFactor(7); !lit {
				t.Error("cursor's unit has no highlight")
			}
			if _, lit := v.hoverSpriteFactor(1); lit {
				t.Error("unhovered unit highlighted")
			}
			if err := a.HeadlessPointer("press", 304, 212); err != nil {
				t.Fatal(err)
			}
			ords, ok := v.command(appInput{CursorX: 304, CursorY: 212, PrimaryReleased: true})
			if !ok || len(ords) != 1 || ords[0].kind != orderKindAttack || ords[0].victim != 7 {
				t.Errorf("second=%v category=%v click=%+v/%v", second, category, ords, ok)
			}
		}
	}
}

func TestHoverFinishingBodyAndStructureReaders(t *testing.T) {
	for _, second := range []bool{false, true} {
		for _, flat := range []bool{false, true} {
			a, v := hoverHitFixture(t, second)
			v.entities[1].Life = LifeDead
			v.entities[1].HP = -5
			if err := a.HeadlessPointer("hover", 304, 212); err != nil {
				t.Fatal(err)
			}
			v.refreshHoverLighting()
			p, ok := v.InspectionPanel()
			if name, _ := v.missionHoverCursor(); name != "attack" || !ok || p.ID != 7 || p.Kind != InspectionUnit {
				t.Errorf("finishing body cursor=%q panel=%v/%v", name, p, ok)
			}
			if _, lit := v.hoverSpriteFactor(7); !lit {
				t.Error("finishing body not highlighted")
			}
			v.structureInfo[7].Flat = flat
			if err := a.HeadlessPointer("hover", 240, 212); err != nil {
				t.Fatal(err)
			}
			v.refreshHoverLighting()
			p, ok = v.InspectionPanel()
			if name, _ := v.missionHoverCursor(); name != "select" || !ok || p.ID != 7 || p.Kind != InspectionStructure {
				t.Errorf("flat=%v structure cursor=%q panel=%v/%v", flat, name, p, ok)
			}
			if _, lit := v.hoverSpriteFactor(7); lit {
				t.Error("structure ID highlighted overlapping unit ID")
			}
		}
	}
}

func TestHoverUsesZeroShadingPixels(t *testing.T) {
	a, v := hoverHitFixture(t, false)
	v.sun = terrain.SunAt(0, true)
	if err := a.HeadlessPointer("hover", 270, 212); err != nil {
		t.Fatal(err)
	}
	v.refreshHoverLighting()
	draws, err := v.HeadlessArtDraws()
	if err != nil {
		t.Fatal(err)
	}
	want := v.entities[1].Frame.RGBALit(v.lightTint(), 0)
	found := false
	for _, d := range draws {
		x, _ := d.Geometry.Apply(0, 0)
		if d.Kind == "sprite" && x == 256 {
			found = true
			if !bytes.Equal(d.Pixels.Pix, want.Pix) || d.ColorScale.R() != 1 {
				t.Error("hover did not submit shading-zero pixels without a gain")
			}
		}
	}
	if !found {
		t.Fatal("no hovered sprite submission")
	}
}

func TestAttackFallbackReadsItsRegisteredHotspot(t *testing.T) {
	for _, hot := range []image.Point{{3, 3}, {7, 9}} {
		_, v := hoverHitFixture(t, false)
		pic := image.NewRGBA(image.Rect(0, 0, 32, 32))
		mgr := NewCursorManager()
		mgr.SetRegistry(NewCursorRegistry([]CursorSlot{
			{Name: "default", Frames: []*image.RGBA{pic}, FrameCount: 1},
			{Name: "attack", Frames: []*image.RGBA{pic}, FrameCount: 1, Hotspot: hot},
		}))
		mgr.SetCursor("default")
		v.SetCursorManager(mgr)
		v.SetAttackPointer(pic)
		v.cursorX, v.cursorY, v.hasCursor, v.armed = 200, 200, true, true
		_, at, ok := v.attackPointerPresent()
		if !ok || at.Add(hot) != image.Pt(200, 200) {
			t.Errorf("registered hot=%v at=%v/%v", hot, at, ok)
		}
		v.advanceCursorManager(time.Unix(0, 0))
	}
	for _, spots := range [][]image.Point{{{5, 5}, {3, 3}, {15, 15}}, {{4, 4}, {3, 3}, {2, 3}}} {
		_, v := hoverHitFixture(t, false)
		pic := image.NewRGBA(image.Rect(0, 0, 32, 32))
		mgr := NewCursorManager()
		names := []string{"default", "attack", "move"}
		var slots []CursorSlot
		for i, name := range names {
			slots = append(slots, CursorSlot{Name: name, Hotspot: spots[i], Frames: []*image.RGBA{pic}, FrameCount: 1})
		}
		mgr.SetRegistry(NewCursorRegistry(slots))
		v.SetCursorManager(mgr)
		v.cursorX, v.cursorY, v.hasCursor = 200, 200, true
		for i, name := range names {
			mgr.SetCursor(name)
			_, at, ok := v.mapCursorPresent()
			if !ok || at.Add(spots[i]) != image.Pt(200, 200) {
				t.Errorf("%s moved pointer: %v/%v", name, at, ok)
			}
		}
	}
}
