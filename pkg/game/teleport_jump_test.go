package game

import (
	"fmt"
	"image"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func jumpDrawPosition(t *testing.T, v *ui.Viewer, cell image.Point, phase int) image.Point {
	t.Helper()
	v.SetPhase(phase, 62000)
	v.Camera().ViewW, v.Camera().ViewH = 400, 300
	v.Camera().SetZoom(1)
	inspectionCentre(&mapWorld{view: v}, cell.X, cell.Y)
	draws, err := v.HeadlessArtDraws()
	if err != nil {
		t.Fatal(err)
	}
	for _, draw := range draws {
		if draw.Kind == "sprite" {
			x, y := draw.Geometry.Apply(0, 0)
			return image.Pt(int(x), int(y))
		}
	}
	t.Fatal("the production art pass submitted no sprite")
	return image.Point{}
}

func TestCellDisplacementDrawsOnlyAdjacentSteps(t *testing.T) {
	for _, delta := range []image.Point{
		{}, image.Pt(1, 0), image.Pt(-1, 0), image.Pt(0, 1), image.Pt(0, -1),
		image.Pt(1, 1), image.Pt(-1, 1), image.Pt(1, -1), image.Pt(-1, -1),
		image.Pt(2, 0), image.Pt(-2, 0), image.Pt(0, 2), image.Pt(0, -2),
		image.Pt(7, -5), image.Pt(-9, 3),
	} {
		for _, idle := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d,%d/idle=%t", delta.X, delta.Y, idle), func(t *testing.T) {
				cell := image.Pt(20, 20)
				m := worldFixtureMap()
				v := worldFixtureViewer(t, m)
				art := odoWalker(odoAlignedTrack(), idle)
				w, err := sim.NewWorld(1, sim.Bounds{Width: int32(m.Width), Height: int32(m.Height)},
					sim.ModeCanonical, nil, []sim.Entity{{ID: 1, X: 20, Y: 20, Class: 1, Owner: sim.SelfSlot, HP: 100, MaxHP: 100}})
				if err != nil {
					t.Fatal(err)
				}
				mw := newMapWorld(w, nil, &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{1: art}}, v)
				mw.prev[1] = cell.Sub(delta)
				mw.walk[1] = walkClock{dist: 32}
				before := w.Hash()
				mw.advanceOdometers()
				draw := mw.entityDraws()[0]
				v.SetEntities([]ui.MapEntity{draw})
				adjacent := delta.X >= -1 && delta.X <= 1 && delta.Y >= -1 && delta.Y <= 1
				wantStep, wantDistance := image.Point{}, 0
				if idle {
					wantDistance = 32
				}
				if adjacent && delta != (image.Point{}) {
					wantStep = delta
					wantDistance = 32 + terrain.WalkAdvance(delta.X, delta.Y, 1, 0)
				}
				if draw.Cell != cell || draw.Step != wantStep || mw.drawnMoving[1] != (wantStep != (image.Point{})) {
					t.Errorf("cell=%v step=%v moving=%t; want %v %v %t", draw.Cell, draw.Step, mw.drawnMoving[1], cell, wantStep, wantStep != (image.Point{}))
				}
				if mw.walk[1].dist != wantDistance {
					t.Errorf("walk distance=%d, want %d", mw.walk[1].dist, wantDistance)
				}
				end := jumpDrawPosition(t, v, cell, 62000)
				projection := terrain.Project(m.Altitudes, m.Width, m.Height)
				from := cell.Sub(wantStep)
				rise := projection.AnchorHeight(from.X, from.Y) - projection.AnchorHeight(cell.X, cell.Y)
				for _, phase := range []int{-1000, 0, 15500, 31000, 62000, 63000} {
					left := 62000 - min(max(phase, 0), 62000)
					want := image.Pt(-wantStep.X*terrain.CellSize*left/62000,
						-wantStep.Y*terrain.CellSize*left/62000-rise*left/62000)
					if got := jumpDrawPosition(t, v, cell, phase).Sub(end); got != want {
						t.Errorf("phase %d: production sprite shift=%v, want %v", phase, got, want)
					}
				}
				if w.Hash() != before {
					t.Fatal("view rule changed the World hash")
				}
			})
		}
	}
}

func TestFirstSeenEntityHasNoDrawnStep(t *testing.T) {
	mw := spellSoundTeleportWorld(t)
	mw.prev = make(map[sim.EntityID]image.Point)
	mw.advanceOdometers()
	if draw := mw.entityDraws()[0]; draw.Step != (image.Point{}) || mw.drawnMoving[1] {
		t.Fatalf("first seen entity step=%v moving=%t", draw.Step, mw.drawnMoving[1])
	}
}

func TestMapPlacementDrawsAtDestination(t *testing.T) {
	mw := spellSoundTeleportWorld(t)
	mw.recordCells()
	if err := mw.world.HeadlessPlace(1, 12, 9); err != nil {
		t.Fatal(err)
	}
	placed, _ := mw.world.Entity(1)
	destination := image.Pt(int(placed.X), int(placed.Y))
	if destination.Sub(image.Pt(8, 8)).X <= 1 {
		t.Fatal("placement did not relocate more than one cell", destination)
	}
	before := mw.world.Hash()
	mw.advanceOdometers()
	mw.push()
	draw := teleportJumpPushed(t, mw, 1)
	if draw.Cell != destination || draw.Step != (image.Point{}) || mw.drawnMoving[1] || mw.walk[1].dist != 0 {
		t.Fatalf("placement cell=%v step=%v moving=%t distance=%d", draw.Cell, draw.Step, mw.drawnMoving[1], mw.walk[1].dist)
	}
	if mw.world.Hash() != before {
		t.Fatal("placement projection changed the World hash")
	}
}
