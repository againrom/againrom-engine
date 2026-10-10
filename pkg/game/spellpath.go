package game

import (
	"image"

	"againrom/pkg/data"
	"againrom/pkg/random"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// THE BOLT'S OWN FIGURE, AND THE TRAIL BEHIND A FLYING ONE.
//
// Pictures 34 (Lightning) and 36 (Prismatic Spray) draw a path, not a sprite
// at their own position. Every live driver call rebuilds the whole figure
// from the projectile display point to the target's (boltFigure); the object
// never moves and nothing is created at its end (MAGIC-BOLTGATE-069,
// MAGIC-BOLTSTILL-072, MAGIC-BOLTEND-074, MAGIC-275..281).
//
// The trail of pictures 10 and 12 is a queue of past positions
// (MAGIC-TRAIL-073), a different mechanism. Nothing here reaches pkg/sim: the
// figure is presentation, regenerated from the object's seed and age.

// boltRNG is the bolt and heal figures' stream: MSVC's rand recurrence
// (MAGIC-279) at the engine's own per-call seed (boltSeed, DIV-2681), so a
// replay draws the same figure.
type boltRNG struct{ state uint32 }

func (r *boltRNG) next() int {
	m := random.MSVC{State: r.state}
	v := m.Rand()
	r.state = m.State
	return int(v)
}

// boltUnitsPerPixel converts screen pixels into ShotScale units.
const boltUnitsPerPixel = ui.ShotScale / terrain.CellSize

// castOrigin is a cast object's start point: the from cell's centre plus its
// launch offset (castLaunch), in ShotScale units.
func castOrigin(from, launch image.Point) image.Point {
	return from.Mul(ui.ShotScale).Add(launch)
}

// castShotPoint interpolates a cast object from its launch point toward the
// target cell, so a flying picture, its trail and a path figure leave one
// point. A same-cell object (a burst, an area-paint cell, a Teleport object)
// stands at its launch point.
func castShotPoint(from, to image.Point, num, den int, launch image.Point) image.Point {
	origin := castOrigin(from, launch)
	if from == to {
		return origin
	}
	toPt := image.Point{X: to.X * ui.ShotScale, Y: to.Y * ui.ShotScale}
	return image.Point{
		X: origin.X + (toPt.X-origin.X)*num/den,
		Y: origin.Y + (toPt.Y-origin.Y)*num/den,
	}
}

// boltSeed is one driver call's stream seed: the observation's seed and the
// object's age. Each call reseeds, so the figure changes every tick and a
// replay repeats it (owner: the random source is ours, DIV-2681).
func boltSeed(b spellBolt) uint32 {
	m := random.MSVC{State: b.seed*2654435761 + uint32(b.age)*2246822519 + uint32(b.picture)}
	m.Rand()
	return m.State
}

// boltRamp is the phase for actionphase 1..13 (ANIM-BOLTRAMP-035); the World
// driver writes it into the record.
var boltRamp = [...]int{4, 3, 2, 1, 0, 1, 2, 1, 0, 1, 2, 3, 4}

// boltChainStride is picture 36's frame stride per link tag (ANIM-BOLTDRAW-034).
const boltChainStride = 5

// boltFrame and boltTag are the sheet frame and the stored tag of one link:
// picture 34 draws frame phase with tag 34; picture 36 draws phase+5*tag with
// the victim index modulo 7 (MAGIC-275, MAGIC-280).
func boltFrame(picture, phase, tag int) int {
	if picture == data.PicturePathSecond {
		return phase + boltChainStride*int(boltTag(picture, tag))
	}
	return phase
}

func boltTag(picture, tag int) uint8 {
	if picture == data.PicturePathSecond {
		return uint8(((tag % chainTagCount) + chainTagCount) % chainTagCount)
	}
	return data.PicturePathFirst
}

// pathDraws is one path object's call: one stamp per stored point, in list
// order, at the native display point (MAGIC-280). The object's own position
// is never drawn.
func (mw *mapWorld) pathDraws(b spellBolt) []ui.SpellBolt {
	sheet, frame, points := mw.pathFigure(b)
	out := make([]ui.SpellBolt, 0, len(points))
	for _, p := range points {
		out = append(out, ui.SpellBolt{
			Cell: b.from, To: b.to, Pos: p, Display: true, Sheet: sheet, Frame: frame, Owner: b.owner,
		})
	}
	return out
}

// pathFigure is one path object's sheet, frame and stored points this call,
// in native display pixels. It is empty when the sheet or the frame is
// absent. The spell light stamps the same points (objectLightStamps).
func (mw *mapWorld) pathFigure(b spellBolt) (*terrain.EffectSheet, int, []image.Point) {
	sheet := mw.projectiles.Sheet(b.picture)
	frame := boltFrame(b.picture, b.phase, b.tag)
	if sheet.Frame(frame) == nil {
		return nil, 0, nil
	}
	origin := castOrigin(b.from, b.launch)
	ax, ay := mw.boltDisplayPoint(origin, b.from)
	bx, by := mw.boltDisplayPoint(b.to.Mul(ui.ShotScale), b.to)
	rng := boltRNG{state: boltSeed(b)}
	return sheet, frame, boltImagePoints(boltFigure(ax, ay, bx, by, boltTag(b.picture, b.tag), rng.next))
}

// boltDisplayPoint is a cell-relative ShotScale point as a native display
// pixel: its ground pixel less the cell's terrain height, which is zero in
// the flat view. Both producer endpoints are such points (MAGIC-275).
func (mw *mapWorld) boltDisplayPoint(pos, cell image.Point) (int32, int32) {
	p := ui.GroundPixel(pos)
	height := 0
	if mw.view != nil {
		height = mw.view.DisplayHeight(cell)
	}
	return int32(p.X), int32(p.Y - height)
}

// boltImagePoints sign-extends the stored words, as the drawer reads them.
func boltImagePoints(points []boltPoint) []image.Point {
	out := make([]image.Point, len(points))
	for i, p := range points {
		out[i] = image.Pt(int(p.X), int(p.Y))
	}
	return out
}
