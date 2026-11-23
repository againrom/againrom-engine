package game

import (
	"image"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The slope: altitude 8 per column and 3 per row, so a point's ground
// differs from its cell's mean corner height everywhere off the cell centre.
const groundW, groundH = 4, 4

func groundAltitudes() []uint8 {
	alts := make([]uint8, groundW*groundH)
	for r := 0; r < groundH; r++ {
		for c := 0; c < groundW; c++ {
			alts[r*groundW+c] = uint8(8*c + 3*r)
		}
	}
	return alts
}

// groundBlend is R2-ENGINE-1518's sample written out: corners A, B (x+1),
// C (y+1), D (both) of the fine point's cell, weights (v>>3)&31, each step
// divided by 32 toward zero.
func groundBlend(alt func(c, r int) int, x, y int) int {
	col, row, fx, fy := x>>8, y>>8, (x>>3)&31, (y>>3)&31
	a, b, c, d := alt(col, row), alt(col+1, row), alt(col, row+1), alt(col+1, row+1)
	p := a + (b-a)*fx/32
	q := c + (d-c)*fx/32
	return p + (q-p)*fy/32
}

// groundWorld is a sloped map with one carried record at fine (x, y) on a
// World of the second game when second is set, else of the first.
func groundWorld(t *testing.T, second bool, x, y int32) *mapWorld {
	t.Helper()
	var script *sim.Script
	if second {
		var err error
		if script, err = sim.NewROM2Script(nil, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	units, set := shotArchive(t)
	w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: groundW, Height: groundH}, sim.ModeCanonical, nil,
		[]sim.Entity{shotVictim(2, 3, 3)}, script, nil)
	if err != nil {
		t.Fatal(err)
	}
	v, err := ui.NewViewer("ground", terrain.Grid{Width: groundW, Height: groundH,
		Tiles: make([]uint16, groundW*groundH), Altitudes: groundAltitudes()}, &terrain.Tileset{})
	if err != nil {
		t.Fatal(err)
	}
	mw := newMapWorld(w, nil, units, v)
	mw.projectiles = set
	if err := mw.world.ImportOriginalProjectiles(sim.SavedProjectiles{FreeIndex: 2, IDs: []uint16{1},
		Items: []sim.SavedProjectile{{ID: 1, X: x, Y: y, Picture: shotPictureArrow, Action: 1, ActionSegments: 2, ActionX: 500, ActionY: 500}}}); err != nil {
		t.Fatal(err)
	}
	if err := mw.world.ImportOriginalWorldEffectDrivers(&sim.SavedWorldEffects{Projectiles: []sim.SavedProjectileDriver{{ID: 1, Phases: 1}}}); err != nil {
		t.Fatal(err)
	}
	set.Sheet(shotPictureArrow).CenterX, set.Sheet(shotPictureArrow).CenterY = 0, 0
	return mw
}

// groundAnchor is the record's painter anchor in the displaced view.
func groundAnchor(t *testing.T, mw *mapWorld) image.Point {
	t.Helper()
	v := mw.view
	v.SetFlat(false)
	v.SetFogReveal(true)
	draws := mw.savedProjectileDraws()
	if len(draws) != 1 {
		t.Fatalf("%d record draws, want 1", len(draws))
	}
	b := draws[0]
	v.SetSpellBolts(draws)
	cam := v.Camera()
	cam.X, cam.Y, cam.Zoom = 0, 0, 1
	cam.ViewW, cam.ViewH = 8192, 8192
	art, err := v.HeadlessArtDraws()
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range art {
		if d.Kind == "effect" && d.Effect == b.Sheet.Frame(b.Frame) {
			px, py := d.Geometry.Apply(0, 0)
			return image.Pt(int(px), int(py))
		}
	}
	t.Fatal("the record submitted no art")
	return image.Point{}
}

// A second-game effect record off its cell centre on rising ground stands at
// the ground blended at its own point (R2-ENGINE-1518); the first game's
// stands at its cell's mean corner height.
func TestSecondGameEffectStandsAtTheGroundBlendedAtItsPoint(t *testing.T) {
	const x, y = 1*256 + 40, 1*256 + 200
	proj := terrain.Project(groundAltitudes(), groundW, groundH)
	blend := groundBlend(proj.Altitude, x, y)
	mean := proj.AnchorHeight(1, 1)
	if blend == mean {
		t.Fatalf("fixture does not discriminate: blend %d equals the cell mean", blend)
	}
	for _, tc := range []struct {
		name   string
		second bool
		height int
	}{{"second game", true, blend}, {"first game", false, mean}} {
		t.Run(tc.name, func(t *testing.T) {
			got := groundAnchor(t, groundWorld(t, tc.second, x, y))
			want := image.Pt(x/8, y/8-tc.height-proj.MinV)
			if got != want {
				t.Errorf("record at fine (%d,%d) drawn at %v, want %v (height %d)", x, y, got, want, tc.height)
			}
		})
	}
}

// A second-game Lightning end subtracts the ground blended at its point
// (R2-ENGINE-1516, R2-ENGINE-1518); the first game's its cell's mean.
func TestSecondGamePathEndSubtractsTheGroundBlendedAtItsPoint(t *testing.T) {
	cell, launch := image.Pt(1, 1), image.Pt(-90, 60)
	pos := castOrigin(cell, launch)
	fine := pos.Add(image.Pt(128, 128))
	proj := terrain.Project(groundAltitudes(), groundW, groundH)
	ground := ui.GroundPixel(pos)
	for _, tc := range []struct {
		name   string
		second bool
		height int
	}{{"second game", true, groundBlend(proj.Altitude, fine.X, fine.Y)}, {"first game", false, proj.AnchorHeight(1, 1)}} {
		t.Run(tc.name, func(t *testing.T) {
			mw := groundWorld(t, tc.second, 300, 300)
			mw.view.SetFlat(false)
			x, y := mw.boltDisplayPoint(pos, cell)
			if int(x) != ground.X || int(y) != ground.Y-tc.height {
				t.Errorf("path end (%d,%d), want (%d,%d)", x, y, ground.X, ground.Y-tc.height)
			}
		})
	}
}
