package ui

import (
	"image"
	"testing"
)

func pickRace1115App(t *testing.T) (*App, *int) {
	t.Helper()
	a := newTestApp(t, appRows(0), okLoader(t))
	if err := a.OpenMission(okOpener(t)); err != nil {
		t.Fatal(err)
	}
	a.Layout(1280, 1024)
	v := a.flow.viewer
	v.SetFlat(true)
	entities := []MapEntity{
		{ID: 1, Cell: image.Pt(10, 9), FinePosition: true, FineX: 128, FineY: 128},
		{ID: 2, Cell: image.Pt(10, 10), FinePosition: true, FineX: 128, FineY: 64},
	}
	v.SetEntities(entities)
	ticks := new(int)
	a.flow.tick = func() {
		*ticks++
		entities[0].FineY = 128 + uint8(8*(*ticks/2))
		v.SetEntities(entities)
	}
	return a, ticks
}

func TestHeadlessSelect1115PrefersInteriorOverMovingOcclusionEdge(t *testing.T) {
	t.Run("exposed edge becomes lower ID on release", func(t *testing.T) {
		a, ticks := pickRace1115App(t)
		v := a.flow.viewer
		// Unit1 occupies [y288,y320); unit2 occupies [y312,y344).
		// y320 is the first unique scan row, then unit1 moves down one pixel
		// on the acting release. A point near unit2's middle remains visible.
		sx, sy := v.cam.WorldToScreen(320, 320)
		if got, ok := topAt(v.entities, v.entityPickRect, sx, sy); !ok || got != 2 {
			t.Fatalf("edge before press=%d,%t", got, ok)
		}
		wx, wy, err := v.frameToWindow(image.Pt(int(sx), int(sy)), "literal exposed edge")
		if err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessPointer("press", wx, wy); err != nil {
			t.Fatal(err)
		}
		if got, ok := topAt(v.entities, v.entityPickRect, sx, sy); !ok || got != 2 {
			t.Fatalf("edge after press=%d,%t", got, ok)
		}
		if err := a.HeadlessPointer("release", wx, wy); err != nil {
			t.Fatal(err)
		}
		if got, ok := v.SelectedUnit(); !ok || got != 1 || *ticks != 2 {
			t.Fatalf("production edge selected=%d,%t ticks=%d", got, ok, *ticks)
		}
	})
	t.Run("semantic selection chooses interior without stopping frames", func(t *testing.T) {
		a, ticks := pickRace1115App(t)
		if err := a.HeadlessSelectEntity(2); err != nil {
			t.Fatal(err)
		}
		if got, ok := a.flow.viewer.SelectedUnit(); !ok || got != 2 || *ticks != 2 {
			t.Fatalf("selection=%d,%t ticks=%d", got, ok, *ticks)
		}
	})
}
