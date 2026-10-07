package terrain

import (
	"image"
	"image/color"
)

// The effect layer: the projectile and burst art a cast puts on the map, and
// the pure selections that choose one frame of it.
//
// Everything here is either a decoded picture or an integer function of a
// descriptor. Nothing opens an entry, builds a texture or reads a clock, so
// the whole of the selection is reachable from a test with no archive, no
// window and no world.

// EffectFrame is one frame of a projectile sheet: a Width x Height grid of
// PREMULTIPLIED RGBA in row-major order, holding exactly Width*Height pixels.
//
// IT IS NOT A StaticFrame AND MUST NOT BECOME ONE. A static object's pixel
// is a palette index plus one boolean, because a sheet in that format has no
// coverage of its own — a pixel is drawn or it is a hole. The sheets a
// cast reaches are in the format that carries a four-bit coverage per pixel,
// and the translucent glow that coverage produces is the whole appearance of
// a fire bolt and a heal. Resolving one into an index and a boolean discards
// it.
//
// The colours are resolved ONCE, by the tier that may read a sprite stream, and
// this tier neither holds a palette nor walks one. That is the statics loader's
// own division and it is what keeps a second, disagreeing blend out of the tree.
type EffectFrame struct {
	Width  int
	Height int
	Pixels []color.RGBA
}

// RGBA is the frame as an image the window tier can upload: the same
// premultiplied bytes, in Go's own layout, on a canvas of the frame's size.
//
// A frame of no area answers a 1x1 transparent image rather than an empty one,
// because the engine refuses an image with no pixels and a caller that has to
// test for that would be testing for it at every draw. A caller that wants to
// know whether there is anything to draw asks the frame's own size.
func (f *EffectFrame) RGBA() *image.RGBA {
	if f == nil || f.Width <= 0 || f.Height <= 0 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	pic := image.NewRGBA(image.Rect(0, 0, f.Width, f.Height))
	for i, c := range f.Pixels {
		o := i * 4
		if o+3 >= len(pic.Pix) {
			break
		}
		pic.Pix[o], pic.Pix[o+1], pic.Pix[o+2], pic.Pix[o+3] = c.R, c.G, c.B, c.A
	}
	return pic
}

// EffectClock names which of the three phase clocks a sheet runs.
//
// The default is one frame per two ticks and it is what 49 of the 51 reachable
// pictures take. The other two are per-picture overrides in the engine's own
// switch, and they are carried as a field on the sheet rather than resolved from
// a picture id here: this tier has no picture ids, and the loader that reads the
// registry is the one place that already knows one.
type EffectClock int

const (
	// EffectClockHalf is phase = ((age+1) / 2) % Phases.
	EffectClockHalf EffectClock = iota
	// EffectClockDirect is phase = age, with no modulus — the engine's
	// counter less one, which is this age.
	EffectClockDirect
	// EffectClockRaw is phase = age + 1, with no modulus — the engine's
	// counter itself.
	EffectClockRaw
)

// EffectSheet is one projectiles.reg row as this layer draws it: the frames, the
// three scalars the frame index is computed from, and the centring halves.
//
// CenterX and CenterY ARE NOT THE ART'S SIZE. They are the registry's own
// Width and Height halved, and on the shipped rows the registry disagrees
// with the sheet: one row states 16x16 against a 12x12 frame. The draw
// subtracts them from the object's point, so a disagreement SHIFTS the
// sprite and never clips or scales it.
//
// Flip is the registry's own bit and it is a HALVING OF THE SHEET rather than a
// property of the art: a sheet that carries it stores nine facings and the draw
// mirrors the other seven. Exactly two of the spell-reachable rows carry it.
type EffectSheet struct {
	Frames         []*EffectFrame
	Phases         int
	RotationPhases int
	Flip           bool
	Clock          EffectClock
	CenterX        int
	CenterY        int
}

// Frame is the sheet's frame at i, or nil for an index the sheet does not hold.
func (s *EffectSheet) Frame(i int) *EffectFrame {
	if s == nil || i < 0 || i >= len(s.Frames) {
		return nil
	}
	return s.Frames[i]
}

// EffectSet is the loaded projectile art, keyed by PICTURE ID — the id a spell
// computes and the id the registry's own record array is indexed by, so a lookup
// here is the engine's own subscript.
type EffectSet struct {
	Pictures map[int]bool
	Sheets   map[int]*EffectSheet

	// Smoke is the two trail sheets, and they are NOT keyed by a picture id
	// because they have none: projectiles.reg does not name them, and the
	// engine loads them under a constructed path into a two-slot global of
	// its own (`REG-PROJ-086`, `ANIM-PROJ-026`). Slot 0 is the trail behind
	// picture 10 and slot 1 the trail behind picture 12.
	Smoke [2]*EffectSheet
}

// SmokeSheet is one trail sheet, or nil for an install whose entry was absent
// or would not decode. Slot 0 belongs to picture 10 and slot 1 to picture 12;
// any other index answers nil.
func (s *EffectSet) SmokeSheet(slot int) *EffectSheet {
	if s == nil || slot < 0 || slot >= len(s.Smoke) {
		return nil
	}
	return s.Smoke[slot]
}

// Sheet answers the art a picture id names, or nil. A picture with no sheet
// draws nothing, which is the normal outcome for 21 of the 28 spell ids.
func (s *EffectSet) Sheet(picture int) *EffectSheet {
	if s == nil {
		return nil
	}
	return s.Sheets[picture]
}

func (s *EffectSet) HasPicture(picture int) bool {
	if s == nil {
		return false
	}
	if s.Pictures != nil {
		return s.Pictures[picture]
	}
	return s.Sheet(picture) != nil
}

// The four tangent boundaries of a sixteen-way split, scaled by effectTanScale:
// tan(11.25), tan(33.75), tan(56.25) and tan(78.75) degrees. They divide the
// quarter turn into four 22.5-degree sectors either side of each axis.
const (
	effectTanScale   = 10000
	effectTan11Deg25 = 1989
	effectTan33Deg75 = 6682
	effectTan56Deg25 = 14966
	effectTan78Deg75 = 50273
)

// effectQuarter is how many 22.5-degree steps a vector stands from the `along`
// axis toward the `across` axis: 0 on the along axis and 4 on the across one.
// Both arguments are non-negative magnitudes.
//
// along == 0 answers 4 through the last arm, with no division and no special
// case: every comparison is against zero and every one of them fails.
func effectQuarter(along, across int) int {
	switch {
	case across*effectTanScale < along*effectTan11Deg25:
		return 0
	case across*effectTanScale < along*effectTan33Deg75:
		return 1
	case across*effectTanScale < along*effectTan56Deg25:
		return 2
	case across*effectTanScale < along*effectTan78Deg75:
		return 3
	}
	return 4
}

// EffectFacing is the SHEET's sixteen-way facing for a flight running (dx,
// dy) in screen axes — x east, y south.
//
// 0 IS SOUTH and the wheel runs clockwise: S, SSW, SW, WSW, W, WNW, NW, NNW, N,
// NNE, NE, ENE, E, ESE, SE, SSE. That is not a choice made here. It is the
// eight-way sheet ordering this tree already carries — S 0, SW 1, W 2, NW 3, N 4,
// NE 5, E 6, SE 7 — doubled, and it is the space the sheet's own fold and frame
// index are defined in: a sheet carrying the halving bit stores facings 0 to 8
// and mirrors 9 to 15 onto 7 to 1, which is `8 - octant` doubled and is the same
// fold the unit layer applies at eight.
//
// THE SECTOR BOUNDARIES ARE OURS. The engine derives a projectile's direction
// from the two coordinates in a routine that is not published, so what is
// decoded is the space this answers in and not the split that reaches it. The
// split is an even sixteen, in integers.
//
// A ZERO DELTA ANSWERS SOUTH rather than refusing. A projectile standing on its
// own target — every burst, and a cast at zero range — has no direction, and the
// facing it is drawn at is read by nothing: a burst sheet states one rotation
// phase, so its frame is the phase alone.
func EffectFacing(dx, dy int) int {
	north, east := -dy, dx
	switch {
	case north == 0 && east == 0:
		return 0
	case north >= 0 && east >= 0: // the quarter from north round to east
		return 8 + effectQuarter(north, east)
	case east >= 0: // from south round to east, running backwards
		return -effectQuarter(-north, east) & 0xf
	case north < 0: // from south round to west
		return effectQuarter(-north, -east)
	default: // from north round to west, running backwards
		return 8 - effectQuarter(north, -east)
	}
}

// wallFirePhases and wallFireBias are `ANIM-WALLFIREFRAME-033`'s two `.text`
// immediates: arm 1 of the per-cell overlay draw replaces the record's phase
// count with 5 (at L02928) and biases the remainder by 3 (at L02929). So a burning cell shows frames 3, 4, 5, 6 and 7 of
// the 11-frame `firewall` sheet and never the three birth frames or the three
// fade frames. At one frame per two ticks that is a 10-tick cycle, against 22
// for the whole sheet.
//
// THEY ARE AN ENGINE LIMIT (G2): the five frames a burning cell can show do not
// move when `projectiles.reg` is edited.
const (
	wallFireSpell  = 3
	wallFirePhases = 5
	wallFireBias   = 3
)

// OverlayFrame is the frame one RETAINED per-cell area overlay draws at this
// cell on this tick (`ANIM-WALLFIREFRAME-033`, `MAGIC-OVERLAYART-051`).
//
// IT IS NOT EffectPhase AND MUST NOT BECOME ONE. A projectile object has a life
// and an age, and its phase is a function of that age. A cloud's cell has
// neither: what the cloud put on the client is a per-cell bitmask carrying no
// object, no counter and no phase (`MAGIC-OVERLAY-050`), and the map draw
// re-reads the mask every frame. So the phase here is a function of the map
// view's own free-running counter and of the cell, never of when the cell
// started burning — which is why a burning cell cannot cycle a birth and a fade
// however long it burns.
//
// The law is `abs(counter/2 + spatial) mod phases`, with `wall_of_fire` alone
// overriding phases to 5 and adding 3; the other three arms take the record's
// own `Phases`. The spatial term decorrelates neighbouring cells, so a wall
// looks like fire rather than like one sprite blinking in unison.
//
// THE SPATIAL TERM IS THE CELL'S OWN COORDINATES HERE. The claim states it as
// `screenX*screenY` and does not say in what units those are measured. Taking
// the cell keeps the picture stable under scrolling and zoom, which a screen
// term would not be — disclosed at `DIV-078`.
//
// ok is false where the sheet states no positive phase count, EffectPhase's own
// answer to the same.
func OverlayFrame(spell, phases, counter, cellX, cellY int) (int, bool) {
	if spell == wallFireSpell {
		phases = wallFirePhases
	}
	if phases <= 0 {
		return 0, false
	}
	v := counter/2 + cellX*cellY
	if v < 0 {
		v = -v
	}
	frame := v % phases
	if spell == wallFireSpell {
		frame += wallFireBias
	}
	return frame, true
}

// EffectPhase is the sheet's phase at age ticks of the object's own life,
// and whether the sheet has one at all.
//
// age COUNTS DRAWN TICKS FROM 0. The engine's counter is incremented before the
// phase is read, so its first drawn tick is counter 1 and every form below is
// its own form with that offset folded in. The direct clock is the check on that
// alignment: its picture lives 21 ticks and its sheet holds 21 phases, and the
// two cover each other exactly only here.
//
// A sheet with no positive phase count has NO PHASE and says so, which is the
// registry's own default for a row that states none. The two clocks with no
// modulus can and do answer past the phase count — the frame selection is what
// refuses that, by the sheet's real frame count rather than by this scalar.
//
// A negative age answers phase 0 on the half clock and its own value on the
// other two; no input divides by zero or panics, and equal inputs give equal
// answers.
func EffectPhase(clock EffectClock, age, phases int) (int, bool) {
	if phases <= 0 {
		return 0, false
	}
	switch clock {
	case EffectClockDirect:
		return age, true
	case EffectClockRaw:
		return age + 1, true
	}
	p := (age + 1) / 2 % phases
	if p < 0 {
		p += phases
	}
	return p, true
}

// SelectEffectFrame is the frame one sheet draws at a facing and a phase,
// whether it draws reflected, and whether there is a frame at all.
//
// Two forms, and the sheet chooses between them: a sheet stating ONE rotation
// phase has no per-direction block at all and its frame is the phase alone;
// every other sheet lays its frames out as Phases per facing, and the index is
// `Phases * facing + phase`.
//
// THE FOLD IS THE HALVING BIT'S. A sheet carrying it stores nine facings and the
// draw mirrors 9 to 15 onto 7 to 1, which is the same fold the unit layer
// applies to a nine-facing layout at eight. A sheet without it takes the facing
// as given.
//
// THE THIRD RETURN IS THE POINT. The unit selector answers a refused index
// as frame 0 unmirrored, because a unit that draws nothing is worse than a
// unit drawing its first frame. Here the opposite holds: the engine's own
// draw refuses a picture past its array and an empty slot by drawing
// nothing, and a spell that draws nothing is the normal outcome for most of
// the book. So every refusal — a nil sheet, a sheet with no frames, a
// non-positive phase count and an index outside the frames — reaches the
// caller (spec AC-10).
//
// No input panics and no input divides.
func SelectEffectFrame(s *EffectSheet, facing, phase int) (frame int, mirror bool, ok bool) {
	if s == nil || len(s.Frames) == 0 || s.Phases <= 0 {
		return 0, false, false
	}
	if s.RotationPhases == 1 {
		frame = phase
	} else {
		f := facing & 0xf
		if s.Flip && f > 8 {
			f, mirror = 16-f, true
		}
		frame = s.Phases*f + phase
	}
	if frame < 0 || frame >= len(s.Frames) {
		return 0, false, false
	}
	return frame, mirror, true
}
