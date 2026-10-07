package ui

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/terrain"
)

// Where a spell's own art is placed on the map.

// uiSheet is a synthetic effect sheet: n frames of w x h, with the registry's
// centring halves stated. No archive is opened and no pixel is decoded.
func uiSheet(n, w, h, cx, cy int) *terrain.EffectSheet {
	frames := make([]*terrain.EffectFrame, n)
	for i := range frames {
		frames[i] = &terrain.EffectFrame{Width: w, Height: h, Pixels: make([]color.RGBA, w*h)}
	}
	return &terrain.EffectSheet{Frames: frames, Phases: n, RotationPhases: 1, CenterX: cx, CenterY: cy}
}

func TestASpellSpriteIsCentredByTheRegistrysOwnHalves(t *testing.T) {
	t.Parallel()

	v := commandViewer(t)
	// A 12x12 frame under a registry stating 16x16, which is `healing`'s own
	// shipped disagreement.
	sheet := uiSheet(4, 12, 12, 8, 8)
	v.SetSpellBolts([]SpellBolt{{
		Cell: image.Pt(2, 2), Pos: image.Pt(2*ShotScale, 2*ShotScale), Sheet: sheet, Frame: 1,
	}})

	placed := v.spellArtPlacements()
	if len(placed) != 1 {
		t.Fatalf("one bolt placed %d rectangles, want one", len(placed))
	}
	if placed[0].Frame != sheet.Frames[1] {
		t.Errorf("the placement carries a frame that is not the one the seam named")
	}
	// The world rectangle before the camera is 12x12; the placement scales it
	// by the camera's zoom and nothing else.
	if got, want := placed[0].W, 12*v.cam.Zoom; got != want {
		t.Errorf("the placed rectangle is %v wide, want the frame's 12 at zoom (%v)", got, want)
	}
	if got, want := placed[0].H, 12*v.cam.Zoom; got != want {
		t.Errorf("the placed rectangle is %v tall, want the frame's 12 at zoom (%v)", got, want)
	}

	// Moving the halves moves the sprite by exactly that much on screen, so the
	// centring is what places it and nothing downstream re-anchors.
	shifted := uiSheet(4, 12, 12, 0, 0)
	v.SetSpellBolts([]SpellBolt{{
		Cell: image.Pt(2, 2), Pos: image.Pt(2*ShotScale, 2*ShotScale), Sheet: shifted, Frame: 1,
	}})
	moved := v.spellArtPlacements()
	if len(moved) != 1 {
		t.Fatalf("the shifted bolt placed %d rectangles, want one", len(moved))
	}
	if got, want := moved[0].X-placed[0].X, 8*v.cam.Zoom; got != want {
		t.Errorf("dropping the 8-pixel half moved the sprite by %v, want %v", got, want)
	}
}

// TestASpellSpriteIsRefusedWhereTheSheetHasNoSuchFrame is AC-10 read at the
// seam: a nil sheet, an index the sheet does not hold and a frame of no area
// each place nothing, and none of them is an error.
func TestASpellSpriteIsRefusedWhereTheSheetHasNoSuchFrame(t *testing.T) {
	t.Parallel()

	empty := uiSheet(1, 0, 0, 0, 0)
	for _, tc := range []struct {
		name string
		bolt SpellBolt
	}{
		{"no sheet at all", SpellBolt{Cell: image.Pt(2, 2)}},
		{"an index past the sheet", SpellBolt{Cell: image.Pt(2, 2), Sheet: uiSheet(2, 8, 8, 4, 4), Frame: 9}},
		{"a negative index", SpellBolt{Cell: image.Pt(2, 2), Sheet: uiSheet(2, 8, 8, 4, 4), Frame: -1}},
		{"a frame of no area", SpellBolt{Cell: image.Pt(2, 2), Sheet: empty}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := commandViewer(t)
			v.SetSpellBolts([]SpellBolt{tc.bolt})
			if got := v.spellArtPlacements(); len(got) != 0 {
				t.Errorf("placed %d rectangles, want none", len(got))
			}
		})
	}
}

// TestAViewerWithNoSpellArtPlacesNothing is the rule that keeps every frame
// of this build before a spell was cast composing exactly the frame it
// composed before, and building no texture.
func TestAViewerWithNoSpellArtPlacesNothing(t *testing.T) {
	t.Parallel()

	v := commandViewer(t)
	if got := v.spellArtPlacements(); got != nil {
		t.Errorf("a viewer holding no spell objects placed %d rectangles, want none", len(got))
	}
	if got := v.SpellBolts(); got != 0 {
		t.Errorf("SpellBolts = %d, want 0", got)
	}
	if v.effectImages != nil {
		t.Error("a viewer that has never drawn holds a texture cache")
	}
}

// TestSpellArtKeepsTheOrderItWasHandedIn is the ordering the tier above relies
// on: the placements are a function of the objects and not of a map walk, so
// what is drawn last is what was pushed last.
func TestSpellArtKeepsTheOrderItWasHandedIn(t *testing.T) {
	t.Parallel()

	v := commandViewer(t)
	a, b := uiSheet(1, 8, 8, 4, 4), uiSheet(1, 6, 6, 3, 3)
	v.SetSpellBolts([]SpellBolt{
		{Cell: image.Pt(2, 2), Pos: image.Pt(2*ShotScale, 2*ShotScale), Sheet: a},
		{Cell: image.Pt(3, 3), Pos: image.Pt(3*ShotScale, 3*ShotScale), Sheet: b},
	})
	placed := v.spellArtPlacements()
	if len(placed) != 2 {
		t.Fatalf("two bolts placed %d rectangles, want two", len(placed))
	}
	if placed[0].Frame != a.Frames[0] || placed[1].Frame != b.Frames[0] {
		t.Error("the placements came back in an order the seam did not hand over")
	}
}

func TestAMirroredFacingIsCarriedToThePlacement(t *testing.T) {
	t.Parallel()

	v := commandViewer(t)
	sheet := uiSheet(4, 12, 12, 6, 6)
	v.SetSpellBolts([]SpellBolt{
		{Cell: image.Pt(2, 2), Pos: image.Pt(2*ShotScale, 2*ShotScale), Sheet: sheet, Frame: 2},
		{Cell: image.Pt(2, 2), Pos: image.Pt(2*ShotScale, 2*ShotScale), Sheet: sheet, Frame: 2, Mirror: true},
	})
	placed := v.spellArtPlacements()
	if len(placed) != 2 {
		t.Fatalf("two bolts placed %d rectangles, want two", len(placed))
	}
	if placed[0].Mirror || !placed[1].Mirror {
		t.Errorf("the mirror bits placed as %v and %v, want false then true",
			placed[0].Mirror, placed[1].Mirror)
	}
	if placed[0].screenRect != placed[1].screenRect {
		t.Errorf("mirroring moved the rectangle: %+v against %+v",
			placed[0].screenRect, placed[1].screenRect)
	}
}

func TestHealSpritesUseSheetFramesThroughTheActualPlacement(t *testing.T) {
	t.Parallel()

	v := commandViewer(t)
	sheet := uiSheet(8, 12, 12, 8, 8)
	v.SetHealSprites([]HealSprite{
		{Cell: image.Pt(2, 2), Pos: image.Pt(2*ShotScale, 2*ShotScale), Sheet: sheet, Frame: 1},
		{Cell: image.Pt(2, 2), Pos: image.Pt(2*ShotScale+20, 2*ShotScale-30), Sheet: sheet, Frame: 5},
	})
	placed := v.healArtPlacements()
	if len(placed) != 2 {
		t.Fatalf("two shipped Heal sprites placed %d frames, want two", len(placed))
	}
	if placed[0].Frame != sheet.Frames[1] || placed[1].Frame != sheet.Frames[5] {
		t.Fatal("the target-local consumer substituted a procedural glyph for the decoded frames")
	}
	if placed[1].Y >= placed[0].Y {
		t.Errorf("the second rising instance is not above the first: %v >= %v", placed[1].Y, placed[0].Y)
	}
	if v.HealSprites() != 2 {
		t.Errorf("HealSprites = %d, want 2", v.HealSprites())
	}
}

// ------------------------------------------------- the two ends' own relief

func TestABoltsFarEndTakesTheTargetCellsOwnRelief(t *testing.T) {
	v := identityViewer(t, cliffGrid(), cliffW*terrain.CellSize, cliffCanvasH)
	if v.Mode() != ModeDisplaced {
		t.Fatalf("Mode() = %v, want displaced — the fixture must select displaced or this test says nothing", v.Mode())
	}
	cam := v.Camera()
	proj := cliffProjection()
	from, to := image.Pt(1, 1), image.Pt(0, 3)
	if proj.AnchorHeight(from.X, from.Y) == proj.AnchorHeight(to.X, to.Y) {
		t.Fatalf("fixture drift: cells %v and %v are at the same height, so the test cannot separate them", from, to)
	}

	sheet := uiSheet(1, 8, 8, 4, 4)
	near := image.Pt(from.X*ShotScale, from.Y*ShotScale)
	far := image.Pt(to.X*ShotScale, to.Y*ShotScale)
	mid := image.Pt((near.X+far.X)/2, (near.Y+far.Y)/2)
	v.SetSpellBolts([]SpellBolt{
		{Cell: from, To: to, Pos: near, Sheet: sheet},
		{Cell: from, To: to, Pos: mid, Sheet: sheet},
		{Cell: from, To: to, Pos: far, Sheet: sheet},
	})

	placed := v.spellArtPlacements()
	if len(placed) != 3 {
		t.Fatalf("three stamps placed %d rectangles, want three — the view must reach all of them", len(placed))
	}

	// The oracle: each end's own world point, shifted by that end's own
	// -AnchorHeight-MinV, through the camera. It reads proj, never v.
	wantY := func(pos image.Point, cell image.Point) float64 {
		_, py := EffectGroundPoint(pos, sheet)
		dy := -proj.AnchorHeight(cell.X, cell.Y) - proj.MinV
		_, sy := cam.WorldToScreen(0, float64(py+dy))
		return sy
	}
	if got, want := placed[0].Y, wantY(near, from)+float64(spellHandLift); got != want {
		t.Errorf("the near end is at %v, want the caster cell's own relief plus the hand lift, %v", got, want)
	}
	if got, want := placed[2].Y, wantY(far, to); got != want {
		t.Errorf("the far end is at %v, want the target cell's own %v", got, want)
	}
	// And it is NOT what the whole figure used to take: the caster's lift at
	// the far end is the defect itself, stated so the test fails if it returns.
	if bad := wantY(far, from); placed[2].Y == bad {
		t.Errorf("the far end is lifted by the CASTER's cell (%v) — that is the reported defect", bad)
	}
	// The stamps between the two ends interpolate: the middle one is strictly
	// between, so a figure crossing a slope is tilted rather than shifted.
	lo, hi := placed[0].Y, placed[2].Y
	if lo > hi {
		lo, hi = hi, lo
	}
	if placed[1].Y <= lo || placed[1].Y >= hi {
		t.Errorf("the middle stamp is at %v, want strictly between the two ends %v and %v", placed[1].Y, lo, hi)
	}
}

// TestAStationaryObjectTakesOneLift is the other half: an object whose two
// cells are equal — every burst, every overlay cell — is lifted exactly as a
// mark on that cell is, and the interpolation is not entered.
func TestAStationaryObjectTakesOneLift(t *testing.T) {
	v := identityViewer(t, cliffGrid(), cliffW*terrain.CellSize, cliffCanvasH)
	cam := v.Camera()
	proj := cliffProjection()
	cell := image.Pt(1, 1)

	sheet := uiSheet(1, 8, 8, 4, 4)
	pos := image.Pt(cell.X*ShotScale, cell.Y*ShotScale)
	v.SetSpellBolts([]SpellBolt{{Cell: cell, To: cell, Pos: pos, Sheet: sheet}})

	placed := v.spellArtPlacements()
	if len(placed) != 1 {
		t.Fatalf("one burst placed %d rectangles, want one", len(placed))
	}
	_, py := EffectGroundPoint(pos, sheet)
	_, want := cam.WorldToScreen(0, float64(py-proj.AnchorHeight(cell.X, cell.Y)-proj.MinV))
	if placed[0].Y != want {
		t.Errorf("a burst on cell %v is at %v, want its own cell's %v", cell, placed[0].Y, want)
	}
}

// ---------------------------------------------- the departure's own hand height

func TestABoltsDepartureCarriesHandHeightDecayingToTheFarEnd(t *testing.T) {
	v := identityViewer(t, cliffGrid(), cliffW*terrain.CellSize, cliffCanvasH)
	if v.Mode() != ModeDisplaced {
		t.Fatalf("Mode() = %v, want displaced — the fixture must select displaced or this test says nothing", v.Mode())
	}

	from, to := image.Pt(1, 1), image.Pt(0, 3)
	sheet := uiSheet(1, 8, 8, 0, 0)
	near := image.Pt(from.X*ShotScale, from.Y*ShotScale) // (256, 256)
	far := image.Pt(to.X*ShotScale, to.Y*ShotScale)      // (0, 768)
	mid := image.Pt((near.X+far.X)/2, (near.Y+far.Y)/2)  // (128, 512), num/den = 1/2 exactly
	v.SetSpellBolts([]SpellBolt{
		{Cell: from, To: to, Pos: near, Sheet: sheet},
		{Cell: from, To: to, Pos: mid, Sheet: sheet},
		{Cell: from, To: to, Pos: far, Sheet: sheet},
	})

	placed := v.spellArtPlacements()
	if len(placed) != 3 {
		t.Fatalf("three stamps placed %d rectangles, want three", len(placed))
	}
	// near: py = 256*32/256+16 = 48; lift = cellLift(1,1) + spellHandLift = -32-16 = -48; screen = 0.
	if got, want := placed[0].Y, 0.0; got != want {
		t.Errorf("near end at %v, want %v (48 py, -48 lift: -32 terrain, -16 full hand)", got, want)
	}
	// mid: py = 512*32/256+16 = 80; terrainLift = 127*163840/327680 = 63;
	// hand = -16*163840/327680 = -8; lift = -32+63-8 = 23; screen = 103.
	if got, want := placed[1].Y, 103.0; got != want {
		t.Errorf("mid stamp at %v, want %v (80 py, 23 lift: half hand carried)", got, want)
	}
	// far: py = 768*32/256+16 = 112; lift = cellLift(0,3) = 95, no hand term; screen = 207.
	if got, want := placed[2].Y, 207.0; got != want {
		t.Errorf("far end at %v, want %v (112 py, 95 lift, no hand term)", got, want)
	}
}

// TestABoltsHandHeightAppliesInFlatModeToo is the mode question the same
// hotfix leaves explicit in spellBoltLift's own comment: the hand height is a
// property of the caster, not of the terrain, so it is not gated on
// v.Mode() == ModeDisplaced the way the terrain interpolation is. cellLift
// already returns 0 for both ends in flat mode, so only the hand term's own
// contribution differs from zero here.
func TestABoltsHandHeightAppliesInFlatModeToo(t *testing.T) {
	v := identityViewer(t, cliffGrid(), cliffW*terrain.CellSize, cliffCanvasH)
	v.SetFlat(true)
	if v.Mode() != ModeFlat {
		t.Fatalf("SetFlat(true): Mode() = %v, want flat", v.Mode())
	}
	// The flat world (cliffFlatH, 128) is shorter than this view (cliffCanvasH,
	// 223), so SetFlat re-centres the camera vertically; it is no longer the
	// identity identityViewer checked before the mode switch. cam.WorldToScreen
	// is the same independent oracle TestABoltsFarEndTakesTheTargetCellsOwnRelief
	// already uses, so the recentring is read from the camera rather than
	// assumed, and only the WORLD-space py+lift figures below are literals.
	cam := v.Camera()

	from, to := image.Pt(1, 1), image.Pt(0, 3)
	sheet := uiSheet(1, 8, 8, 0, 0)
	near := image.Pt(from.X*ShotScale, from.Y*ShotScale)
	far := image.Pt(to.X*ShotScale, to.Y*ShotScale)
	mid := image.Pt((near.X+far.X)/2, (near.Y+far.Y)/2)
	v.SetSpellBolts([]SpellBolt{
		{Cell: from, To: to, Pos: near, Sheet: sheet},
		{Cell: from, To: to, Pos: mid, Sheet: sheet},
		{Cell: from, To: to, Pos: far, Sheet: sheet},
	})

	placed := v.spellArtPlacements()
	if len(placed) != 3 {
		t.Fatalf("three stamps placed %d rectangles, want three", len(placed))
	}
	// near: py=48, lift = 0 (flat) + a LITERAL -16 hand lift = -16; world Y = 32.
	_, wantNear := cam.WorldToScreen(0, 32)
	if got := placed[0].Y; got != wantNear {
		t.Errorf("flat-mode near end at %v, want %v — the hand term must survive with no terrain lift", got, wantNear)
	}
	// mid: py=80, terrainLift=0 (both cellLifts 0 in flat), a LITERAL -8 half
	// hand lift; world Y = 72.
	_, wantMid := cam.WorldToScreen(0, 72)
	if got := placed[1].Y; got != wantMid {
		t.Errorf("flat-mode mid stamp at %v, want %v", got, wantMid)
	}
	// far: py=112, lift=0, no hand term; world Y = 112.
	_, wantFar := cam.WorldToScreen(0, 112)
	if got := placed[2].Y; got != wantFar {
		t.Errorf("flat-mode far end at %v, want %v — the far end carries no hand term in either mode", got, wantFar)
	}
}
