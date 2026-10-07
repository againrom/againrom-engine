package ui

// Hover brightening (owner report 1, hotfix DIV-1349): hovering any unit
// brightens that unit's OWN sprite, as if it were lit, and no other unit's.

import (
	"image"
	"testing"

	"againrom/pkg/render/terrain"
)

// hoverEntityBrightness runs the full entity draw pass and returns the
// Brightness the given unit's own Inspection subject carries, mirroring
// TestSpellLightingAffectsOnlyTerrainAndActors' own reconstruction (never a
// second copy of drawPlaneSprite's own >0 && !=1 sentinel rule).
func hoverEntityBrightness(v *Viewer, id uint32) (float32, bool) {
	for _, s := range v.planeSprites() {
		if s.Inspection.Kind == InspectionUnit && s.Inspection.ID == id {
			return s.Brightness, true
		}
	}
	return 0, false
}

// giveEntitiesArt equips inspectionFixture's two bare entities (Selected id1,
// Visitor id7 — neither carries Art or Frame in that fixture, so
// terrain.UnitPlace refuses both and planeSprites' entity arm never submits
// them) with the minimal drawable UnitPlace accepts, structureFrame's own
// tile-sized frame reused as the sprite, matching
// TestInspectionUsesDrawDepthAndOpaquePixels' own precedent for this fixture.
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

// TestHoverBrightensOnlyTheHoveredUnitsSprite is the owner's own words as a
// test: hovering unit 7 brightens unit 7's OWN sprite and leaves unit 1 --
// sharing no cell with it, so a cell-keyed highlight could not be confused
// for this -- at the normal-draw sentinel (statics.go's own Brightness==0
// meaning "unchanged", drawPlaneSprite's own `>0 && !=1` gate).
func TestHoverBrightensOnlyTheHoveredUnitsSprite(t *testing.T) {
	a, v := inspectionFixture(t, image.Pt(1024, 768))
	giveEntitiesArt(v)
	inspectionHover(t, a, v, InspectionSubject{InspectionUnit, 7})
	v.refreshHoverLighting()

	got7, ok7 := hoverEntityBrightness(v, 7)
	if !ok7 {
		t.Fatal("hovered unit 7 submitted no Inspection-tagged sprite")
	}
	if got7 <= 1 {
		t.Fatalf("hovered unit brightness = %v, want strictly brighter than 1", got7)
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
