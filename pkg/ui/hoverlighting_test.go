package ui

// Hover selects shading zero only for the stored unit hit.

import (
	"image"
	"testing"

	"againrom/pkg/render/terrain"
)

// hoverEntityBrightness reads the production sprite submission's residual gain.
func hoverEntityBrightness(v *Viewer, id uint32) (float32, bool) {
	for _, s := range v.planeSprites() {
		if s.Inspection.Kind == InspectionUnit && s.Inspection.ID == id {
			return s.Brightness, true
		}
	}
	return 0, false
}

// giveEntitiesArt supplies drawable synthetic canvases.
func giveEntitiesArt(v *Viewer) {
	f := structureFrame()
	for i := range f.Pixels {
		f.Pixels[i].Opaque = true
	}
	art := &terrain.UnitClass{Width: terrain.CellSize, Height: terrain.CellSize,
		CenterX: terrain.CellSize / 2, CenterY: terrain.CellSize / 2, Frames: []*terrain.StaticFrame{f}}
	for i := range v.entities {
		v.entities[i].Art, v.entities[i].Frame = art, f
	}
}

func TestHoverBrightensOnlyTheHoveredUnitsSprite(t *testing.T) {
	a, v := inspectionFixture(t, image.Pt(1024, 768))
	giveEntitiesArt(v)
	inspectionHover(t, a, v, InspectionSubject{InspectionUnit, 7})
	v.refreshHoverLighting()

	got7, ok7 := hoverEntityBrightness(v, 7)
	if !ok7 {
		t.Fatal("hovered unit 7 submitted no Inspection-tagged sprite")
	}
	if got7 != 0 {
		t.Fatalf("hovered unit gain = %v, want no authored gain", got7)
	}
	foundZero := false
	for _, s := range v.planeSprites() {
		if s.Inspection == (InspectionSubject{InspectionUnit, 7}) {
			foundZero = s.ShadeZero
		}
		if s.Inspection == (InspectionSubject{InspectionUnit, 1}) && s.ShadeZero {
			t.Fatal("unhovered unit changed shading")
		}
	}
	if !foundZero {
		t.Fatal("hovered unit did not select shading zero")
	}

	got1, ok1 := hoverEntityBrightness(v, 1)
	if !ok1 {
		t.Fatal("unhovered unit 1 submitted no Inspection-tagged sprite")
	}
	if got1 != 0 {
		t.Fatalf("unhovered unit brightness = %v, want the normal-draw sentinel 0", got1)
	}
}

// TestHoverLightingUsesHoverInspectionNotTheSelectedFallback is DIV-1349's
// own boundary: Inspection() falls back to the selected primary when nothing
// is hovered (DIV-536); hoverSpriteFactor must not inherit that fallback, or
// a unit would stay lit forever once selected, with the cursor gone
// entirely.
func TestHoverLightingUsesHoverInspectionNotTheSelectedFallback(t *testing.T) {
	_, v := inspectionFixture(t, image.Pt(1024, 768))
	giveEntitiesArt(v)
	v.sel = selection{1} // unit 1 is selected; nothing is hovered
	if _, ok := v.hoverInspection(); ok {
		t.Fatal("fixture unexpectedly has a live hover; the boundary this test checks is untested")
	}
	v.refreshHoverLighting()

	got, ok := hoverEntityBrightness(v, 1)
	if !ok {
		t.Fatal("selected unit 1 submitted no Inspection-tagged sprite")
	}
	if got != 0 {
		t.Fatalf("selected-but-not-hovered unit brightness = %v, want the normal-draw sentinel 0 (Inspection()'s own selected fallback must not leak into lighting)", got)
	}
}

// TestHoverSpriteFactorRespectsDisableLighting mirrors spellSpriteFactor's
// own gate (overlay.go): every lighting behaviour this hotfix batch adds
// must stay off under the diagnostic.
func TestHoverSpriteFactorRespectsDisableLighting(t *testing.T) {
	a, v := inspectionFixture(t, image.Pt(1024, 768))
	giveEntitiesArt(v)
	inspectionHover(t, a, v, InspectionSubject{InspectionUnit, 7})
	v.refreshHoverLighting()
	v.SetGraphicsOptions(GraphicsOptions{DisableLighting: true})

	if got, engaged := v.hoverSpriteFactor(7); engaged || got != 1 {
		t.Fatalf("hoverSpriteFactor under DisableLighting = %v/%v, want 1/false", got, engaged)
	}
}
