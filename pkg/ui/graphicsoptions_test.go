package ui

import (
	"image"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestGraphics1190ShadowGatePreservesBodies(t *testing.T) {
	v := shadowViewer(t)
	before := v.planeSprites()
	if len(v.shadowDraws()) == 0 || len(before) == 0 {
		t.Fatal("empty render witness")
	}
	v.SetGraphicsOptions(GraphicsOptions{HideShadows: true})
	if len(v.shadowDraws()) != 0 || !reflect.DeepEqual(before, v.planeSprites()) {
		t.Fatal("shadow switch changed bodies or left shadows")
	}
	v.SetGraphicsOptions(GraphicsOptions{})
	if len(v.shadowDraws()) == 0 {
		t.Fatal("shadows did not return")
	}
}

func TestGraphics1190DynamicLightKeepsDayNight(t *testing.T) {
	v := overlayViewer(t, 8, 8, 256, 256)
	v.SetTimeFlow(true)
	sun := v.Sun()
	cell := image.Pt(3, 3)
	v.SetSpellLighting([]SpellLightCell{{Cell: cell, Terrain: 3, Sprite: 2}})
	if v.cornerScales(3, 3) != ([4]float32{3, 3, 3, 3}) {
		t.Fatal("missing dynamic light")
	}
	v.SetGraphicsOptions(GraphicsOptions{DisableLighting: true})
	if v.cornerScales(3, 3) != v.cornerShading(3, 3) || v.spellSpriteFactor(cell) != 1 || v.Sun() != sun || !v.TimeFlow() {
		t.Fatal("dynamic switch changed day/night or retained light")
	}
	v.SetGraphicsOptions(GraphicsOptions{})
	if v.cornerScales(3, 3) != ([4]float32{3, 3, 3, 3}) {
		t.Fatal("live light did not return")
	}
}

func TestGraphics1190ObjectAnimationKeepsSharedClock(t *testing.T) {
	v := newObjectAnimViewer(t, objectAnimViewerGrid(), terrain.AnimGateTiles)
	now := driveObjectAnim(v, time.Unix(0, 0), 4)
	changed := false
	for _, p := range v.staticPlacements() {
		if p.Frame != p.Class.Frames[0] {
			changed = true
		}
	}
	if !changed {
		t.Fatal("fixture never selected a later frame")
	}
	v.SetGraphicsOptions(GraphicsOptions{StaticObjects: true})
	before := v.AnimationCounter()
	driveObjectAnim(v, now, 5)
	if v.AnimationCounter() <= before || !v.graphics.DisableLighting {
		t.Fatal("stopped unrelated animation or lost lighting coupling")
	}
	for _, p := range v.staticPlacements() {
		if p.Frame != p.Class.Frames[0] {
			t.Fatal("scenery did not select initial frame")
		}
	}
}

func TestGraphics1190ObjectAnimationWithRuinedStructure(t *testing.T) {
	frames := []*terrain.StaticFrame{structureFrame(), structureFrame(), structureFrame()}
	set := new(terrain.StructureSet)
	set.Classes[1] = &terrain.StructureClass{
		TileWidth: 1, TileHeight: 1, FullHeight: 1,
		Frames: frames, Timeline: []int{0, 1}, Rank: []int{0}, Live: 1,
	}
	g := terrain.Grid{
		Width: 8, Height: 8, Tiles: make([]uint16, 64), Altitudes: make([]uint8, 64),
		Structures: []terrain.StructureRecord{
			{ID: 7, X: 2 << 8, Y: 3 << 8, Key: 1},
			{ID: 8, X: 4 << 8, Y: 3 << 8, Key: 1},
		},
	}
	v := newStructureViewer(t, g, set, true)
	v.SetStructures([]MapStructure{{ID: 7, Health: 0, MaxHealth: 10}, {ID: 8, Health: 10, MaxHealth: 10}})
	now := time.Unix(0, 0)
	for _, enabled := range []bool{false, true} {
		v.SetGraphicsOptions(GraphicsOptions{StaticObjects: !enabled})
		seen := make(map[*terrain.StaticFrame]bool)
		before := v.AnimationCounter()
		for tick := 0; tick < 4; tick++ {
			now = driveObjectAnim(v, now, 1)
			places := v.structurePlacements()
			if len(places) != 2 {
				t.Fatalf("missing structures: %d", len(places))
			}
			for _, p := range places {
				if p.StructureID == 7 && p.Frame != frames[2] {
					t.Fatal("animation preference changed ruin art")
				}
				if p.StructureID == 8 {
					seen[p.Frame] = true
					if !enabled && p.Frame != frames[0] {
						t.Fatal("intact building animates while Object Animations is OFF")
					}
				}
			}
		}
		if v.AnimationCounter() <= before {
			t.Fatal("structure preference stopped shared clock")
		}
		if enabled && (!seen[frames[0]] || !seen[frames[1]] || len(seen) != 2) {
			t.Fatal("intact building did not resume animation")
		}
	}
}

type graphicsAlphaRecorder []float32

func (r *graphicsAlphaRecorder) DrawImage(_ *ebiten.Image, op *ebiten.DrawImageOptions) {
	*r = append(*r, op.ColorScale.A())
}

func TestGraphics1190SmoothingDrawsPairedSackBoundary(t *testing.T) {
	v := sackIdentityViewer(t)
	base, boundary := sackFrame(8, 8), sackFrame(10, 9)
	v.SetSackFrames([]*terrain.StaticFrame{base})
	v.SetSackBoundaries([]*terrain.StaticFrame{boundary})
	v.SetSacks([]MapSack{{Cell: image.Pt(2, 2)}})
	if got := v.planeSprites(); len(got) != 1 || got[0].Frame != base {
		t.Fatal("default base changed")
	}
	v.SetGraphicsOptions(GraphicsOptions{Smoothing: true})
	got := v.planeSprites()
	if len(got) != 2 || got[0].Frame != base || got[1].Frame != boundary || !got[1].Translucent || got[0].X != got[1].X || got[0].Y != got[1].Y {
		t.Fatal("boundary pairing, order or origin changed")
	}
	var drawn graphicsAlphaRecorder
	v.drawPlane(&drawn)
	if !reflect.DeepEqual(drawn, graphicsAlphaRecorder{1, 0.5}) {
		t.Fatal("boundary did not reach half-alpha draw", drawn)
	}
	v.SetSackBoundaries(nil)
	if len(v.planeSprites()) != 1 {
		t.Fatal("missing boundary lost ordinary sack")
	}
}
