package ui

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/terrain"
)

// markSheet is a one-frame effect sheet with a known size and centring pair, so
// every literal below is the position expression's own arithmetic and not the
// art's.
func markSheet() *terrain.EffectSheet {
	f := &terrain.EffectFrame{Width: 8, Height: 8,
		Pixels: make([]color.RGBA, 64)}
	return &terrain.EffectSheet{Frames: []*terrain.EffectFrame{f}, Phases: 1, CenterX: 4, CenterY: 4}
}

// TestTheTwoMarkPassesSplitAroundTheActorSprite — MAGIC-MARK-059: the unit draw
// walks the record array twice, once before the actor's own sprite drawing only
// records with `depth > 0` and once after drawing only the rest. The band this
// package builds must therefore carry a positive-depth mark BEFORE the entity's
// sprite and a non-positive one AFTER it, in one ordered list.
func TestTheTwoMarkPassesSplitAroundTheActorSprite(t *testing.T) {
	v := identityViewer(t, cliffGrid(), cliffW*terrain.CellSize, cliffCanvasH)
	sheet := markSheet()
	e := withFrame(image.Pt(0, 1), entityArtA())
	e.Marks = []UnitMark{
		{Sheet: sheet, Mark: terrain.EffectMark{DX: 1, DY: -20, Depth: 5}},  // behind
		{Sheet: sheet, Mark: terrain.EffectMark{DX: 2, DY: -20, Depth: 0}},  // in front
		{Sheet: sheet, Mark: terrain.EffectMark{DX: 3, DY: -20, Depth: -7}}, // in front
	}
	v.SetEntities([]MapEntity{e})

	band := v.planeSprites()
	if len(band) != 4 {
		t.Fatalf("the band holds %d entries, want 4 — one back mark, the sprite, two front marks\ngot %+v", len(band), band)
	}
	if band[0].Effect == nil || band[0].Frame != nil {
		t.Errorf("band[0] = %+v, want the depth>0 mark before the sprite", band[0])
	}
	if band[1].Frame == nil || band[1].Effect != nil {
		t.Errorf("band[1] = %+v, want the actor's own sprite", band[1])
	}
	for i := 2; i < 4; i++ {
		if band[i].Effect == nil || band[i].Frame != nil {
			t.Errorf("band[%d] = %+v, want a depth<=0 mark after the sprite", i, band[i])
		}
	}

	// And the position is the claim's: the actor's own cell anchor, plus dx and
	// dy, minus depth, minus the sheet's two centring halves — over the SAME
	// relief lift the actor's own sprite takes, read off the projection here
	// rather than restated, so the mark and the actor cannot part on a slope.
	anchorX := 0*terrain.CellSize + terrain.CellSize/2
	anchorY := 1*terrain.CellSize + terrain.CellSize/2 - v.proj.AnchorHeight(0, 1) - v.proj.MinV
	want := screenRect{X: float64(anchorX + 1 - 4), Y: float64(anchorY - 20 - 5 - 4), W: 8, H: 8}
	if band[0].screenRect != want {
		t.Errorf("the back mark landed at %+v, want %+v", band[0].screenRect, want)
	}
}

// TestAnEntityWithNoMarksDrawsExactlyWhatItDidBefore — the band an unmarked
// entity produces is the one it produced before this story, so no frame of a
// mission without a marking effect changes.
func TestAnEntityWithNoMarksDrawsExactlyWhatItDidBefore(t *testing.T) {
	v := identityViewer(t, cliffGrid(), cliffW*terrain.CellSize, cliffCanvasH)
	v.SetEntities([]MapEntity{withFrame(image.Pt(0, 1), entityArtA())})
	band := v.planeSprites()
	if len(band) != 1 || band[0].Effect != nil {
		t.Fatalf("an unmarked entity produced %+v, want its sprite alone", band)
	}
}

// TestAMarkWhoseSheetCannotAnswerDrawsNothing — a nil sheet and a phase the
// sheet does not hold are the two refusals a spell object already takes, and a
// mark takes them the same way rather than drawing a placeholder.
func TestAMarkWhoseSheetCannotAnswerDrawsNothing(t *testing.T) {
	v := identityViewer(t, cliffGrid(), cliffW*terrain.CellSize, cliffCanvasH)
	e := withFrame(image.Pt(0, 1), entityArtA())
	e.Marks = []UnitMark{
		{Mark: terrain.EffectMark{Depth: 5}},
		{Sheet: markSheet(), Mark: terrain.EffectMark{Depth: 5, Phase: 9}},
	}
	v.SetEntities([]MapEntity{e})
	if band := v.planeSprites(); len(band) != 1 || band[0].Effect != nil {
		t.Fatalf("two unresolvable marks produced %+v, want the sprite alone", band)
	}
}
