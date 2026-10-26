package ui

import (
	"image"
	"testing"
)

// A unit's stored cell is the top-left of its n by n footprint, and its body
// stands on the footprint's centre: 16 pixels right and down per extra cell
// (MOVE-087, ANIM-TELEPORT-167). The sprite, its rim, its square and its bars
// all take that one offset; a one-cell unit takes none.
func TestALargeUnitIsDrawnOnItsFootprintCentre(t *testing.T) {
	for _, n := range []int{1, 2, 3} {
		art := entityArtA()
		art.TileSize = n
		ents := []MapEntity{
			{ID: 1, Cell: shiftSpriteCell, Art: art, Frame: art.Frames[0], Life: LifeAlive, HP: 80, MaxHP: 100, TokenSize: n},
			{ID: 2, Cell: shiftSquareCell, Life: LifeAlive, HP: 55, MaxHP: 100, TokenSize: n},
		}
		want := image.Pt((n-1)*16, (n-1)*16)
		for _, relief := range []bool{false, true} {
			v := shiftViewer(t, relief, ents, 0, 0)
			for _, e := range ents {
				shiftAssert(t, v, shiftProjection(), relief, e, "side "+string(rune('0'+n)), want)
			}
			sprites, _, _ := v.entityLayer()
			ground := image.Pt(shiftSpriteCell.X*32+16, shiftSpriteCell.Y*32+16).Add(want)
			if !relief && (len(sprites) != 1 || sprites[0].Ground() != ground) {
				t.Errorf("side %d: the body's ground point is %v, want %v", n, sprites[0].Ground(), ground)
			}
		}
	}
}

// A unit at rest on the saved sub-cell centre (0x80,0x80) with side 2 is drawn
// where the original's own drawable centre stands: the corner its four cells
// share. This is the ogre of the owner's report, stored at (17,105).
func TestARestingTwoCellUnitStandsOnItsCellCorner(t *testing.T) {
	art := entityArtA()
	art.TileSize = 2
	e := MapEntity{ID: 1, Cell: image.Pt(3, 3), Art: art, Frame: art.Frames[0], Life: LifeAlive,
		HP: 1, MaxHP: 1, TokenSize: 2, FinePosition: true, FineX: 0x80, FineY: 0x80}
	v := shiftViewer(t, false, []MapEntity{e}, 0, 0)
	sprites, _, _ := v.entityLayer()
	if len(sprites) != 1 || sprites[0].Ground() != image.Pt(4*32, 4*32) {
		t.Fatalf("the resting ogre's ground point is %v, want the shared corner (128,128)", sprites)
	}
}

// The pointer picks a large unit where its body is drawn: a tap on its
// footprint centre takes it, and a tap in the far corner of its stored cell,
// where nothing of the body stands, does not. A one-cell unit is picked on its
// own cell as before.
func TestATapTakesALargeUnitWhereItIsDrawn(t *testing.T) {
	const col, row = 1, 1
	top := liftedCellY(col, row)
	centre := image.Pt(col*32+32, top+32)
	corner := image.Pt(col*32+2, top+2)
	for _, tc := range []struct {
		side             int
		centreHit, inTop bool
	}{
		{1, false, true},
		{2, true, false},
		{3, true, false},
	} {
		for _, p := range []struct {
			at   image.Point
			want bool
		}{{centre, tc.centreHit}, {corner, tc.inTop}} {
			v := hitViewer(t, []MapEntity{{ID: 7, Cell: image.Pt(col, row), Life: LifeAlive, TokenSize: tc.side}})
			hitTapAt(v, p.at.X, p.at.Y)
			if got := len(v.sel) == 1 && v.sel[0] == 7; got != p.want {
				t.Errorf("side %d, tap at %v: selected %v, want hit=%v", tc.side, p.at, v.sel, p.want)
			}
			r, ok := v.entityPickRect(v.entities[0])
			wantRect, wantOK := v.cellScreenRect(image.Pt(col, row), footprintCentre(tc.side))
			if ok != wantOK || r != wantRect {
				t.Errorf("side %d: pick rect %v, want the rim's own rect %v", tc.side, r, wantRect)
			}
		}
	}
}
