package ui

import (
	"image"
	"image/color"
	"strconv"
	"time"

	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
)

// DamageNumeralLife is how long a figure lives: 1000 ms of WALL CLOCK
// (ANIM-NUM-020, High). A record at exactly the bound is still drawn — the
// original's own comparison is "at most", not "less than".
//
// IT IS NOT A TICK COUNT, and the difference is observable. The life is
// independent of the game's cadence, of the player's pause and of a popup
// standing over the map, while the DRIFT below is paced by the tick — so at a
// slow speed a figure covers a shorter distance in the same second rather than
// hanging about for longer. That consequence is the decode's, not ours.
const DamageNumeralLife = time.Second

// The drift, per tick of the ambient animation clock (ANIM-NUM-020, High): two
// up, one sideways. The vertical step is unconditional; the horizontal one takes
// the ownership sign below, so a figure travels DIAGONALLY.
const (
	damageNumeralRise  = 2
	damageNumeralDrift = 1
)

// The offset a record is born at, relative to the struck unit's own drawn
// position: sideways by the ownership sign, and upward (ANIM-NUM-020, High).
//
// A DECODED TERM HAS NO COUNTERPART HERE. The original scales both by the
// struck unit's own `vt+0x20()`, a virtual that is not decoded, so nothing
// in this tree can compute what it returns. This build takes that scalar as
// ONE, which is what makes the two numbers below exactly 16 and 48; a
// plausible substitute chosen silently would be indistinguishable from a
// decoded number, so it is named here instead. If the virtual is later
// decoded to return anything else, these two constants are what change.
const (
	damageNumeralOffsetX = 16
	damageNumeralOffsetY = 48
)

// DamageNumeralColors is the palette a figure takes BY THE STRUCK UNIT'S OWNER.
//
// What is decoded is the RULE — the colour table is indexed by the victim's
// owning player and by nothing else, and the constructor's colour argument is a
// literal zero at its one call site, so no caller can override it
// (ANIM-NUM-020, ANIM-NUM-021, High). The VALUES are ours: the original's table
// is thirty-two bytes per player index and has not been decoded.
//
// It is indexed by OWNER and not by the mine/not-mine flag the drift uses, and
// the difference is not cosmetic: collapsed to two entries, a four-participant
// map would draw two colours where the original draws four. Index 0 is "owned by
// nobody", which is the seam's own zero, and it is a colour of its own rather
// than a share of some player's.
//
// An owner past the end folds by remainder, so no value can index out of range
// and the totality property needs no guard of its own.
var DamageNumeralColors = [8]color.RGBA{
	{R: 0xe0, G: 0xe0, B: 0xe0, A: 0xff}, // 0 — owned by nobody
	{R: 0xff, G: 0xd8, B: 0x40, A: 0xff},
	{R: 0xff, G: 0x50, B: 0x40, A: 0xff},
	{R: 0x60, G: 0xa0, B: 0xff, A: 0xff},
	{R: 0x60, G: 0xe0, B: 0x60, A: 0xff},
	{R: 0xe0, G: 0x60, B: 0xe0, A: 0xff},
	{R: 0x40, G: 0xe0, B: 0xe0, A: 0xff},
	{R: 0xff, G: 0x98, B: 0x30, A: 0xff},
}

// damageNumeral follows its victim and merges only at the same raw tick.
type damageNumeral struct {
	victim uint32
	tick   uint64
	damage int
	born   time.Time
	off    image.Point
	drift  int
	colour color.RGBA
	pic    *image.RGBA
	blit   *image.RGBA
	calls  []text.DrawCall
}

// numeralsOn reports whether the display is showing. It is stored INVERTED on
// the viewer so that SHOWN is the zero value: the display defaults to on
// (ANIM-NUM-020 — the map view's constructor sets the flag), and a default that
// is the zero value is one no struct literal can get wrong.
func (v *Viewer) numeralsOn() bool { return !v.numeralsHidden }

// DamageNumerals reports whether the display is on and how many figures are
// live, mirroring EntityMarkers' shape and reason: the tier that owns the world
// needs to see that a blow produced a figure without opening a window, and a
// viewer silently making none is otherwise indistinguishable from one drawing
// them.
func (v *Viewer) DamageNumerals() (on bool, live int) { return v.numeralsOn(), len(v.numerals) }

// ToggleDamageNumerals flips the display and reports the new state.
//
// It is a flip and not a show: the display is on from the moment a map opens, so
// this key exists for the frame the player wants clean.
//
// IT GATES CREATION AND NOT DRAWING, which is what the original's own flag
// does — the arm tests it before building the record at all. So switching
// the display on shows nothing until the next blow, rather than revealing
// figures that had been accruing invisibly. Records already made are left
// alone: this only stops new ones.
func (v *Viewer) ToggleDamageNumerals() bool {
	v.numeralsHidden = !v.numeralsHidden
	return v.numeralsOn()
}

// receiveDamageNumeral consumes a causal message, never a snapshot decrease.
// ANIM-071: merging changes the amount without refreshing birth or position.
func (v *Viewer) receiveDamageNumeral(e soundEntity, tick uint64, damage int, now time.Time) {
	if damage <= 0 || !v.numeralsOn() || !v.fogGateEntity(e.Owner, e.Cell.X, e.Cell.Y) {
		return
	}
	for i := range v.numerals {
		n := &v.numerals[i]
		if n.victim == e.ID && n.tick == tick {
			n.damage += damage
			n.pic, n.blit, n.calls = glyphPicture(v.textSmoothingEnabled && v.font != nil, func() *image.RGBA {
				return v.composeNumeral(n.damage, n.colour)
			})
			return
		}
	}
	drift := damageNumeralDrift
	if e.Owner != v.localOwner {
		drift = -damageNumeralDrift
	}
	colour := DamageNumeralColors[int(e.Owner%uint32(len(DamageNumeralColors)))]
	pic, blit, calls := glyphPicture(v.textSmoothingEnabled && v.font != nil, func() *image.RGBA {
		return v.composeNumeral(damage, colour)
	})
	age := 0
	if v.lightClock >= tick {
		age = int(v.lightClock - tick)
	}
	v.numerals = append(v.numerals, damageNumeral{
		victim: e.ID, tick: tick, damage: damage, born: now,
		off:   image.Pt(drift*(damageNumeralOffsetX+age), -damageNumeralOffsetY-age*damageNumeralRise),
		drift: drift, colour: colour, pic: pic, blit: blit, calls: calls,
	})
}

// stepNumerals advances flight and lifetime before consuming the client stream.
func (v *Viewer) stepNumerals(now time.Time) {
	// Include the first resumed span: its previous sample was still paused.
	if !v.numeralLast.IsZero() && (v.playerPaused || v.numeralPaused) {
		if elapsed := now.Sub(v.numeralLast); elapsed > 0 {
			for i := range v.numerals {
				v.numerals[i].born = v.numerals[i].born.Add(elapsed)
			}
		}
	}
	v.numeralLast, v.numeralPaused = now, v.playerPaused
	v.driftNumerals()
	v.expireNumerals(now)
	v.stepSound(now)
}

// driftNumerals moves every live figure by however many ANIMATION TICKS have
// passed since the last call.
//
// IT IS DRIVEN BY THE COUNTER'S DELTA AND NOT BY THE FRAME. The original steps
// its records once per paced logic tick, and this tree already has that clock:
// the ambient counter the water phase, the animated objects and the animated
// structures are all selected from, whose period the cadence seam sets from the
// speed setting. So the numeral needs no clock of its own and cannot come to
// disagree with the rest of the ambient picture. Driven by the frame instead,
// the drift would be a function of the frame rate; driven by the world's tick,
// it would need a value this package may not name.
//
// A delta rather than a per-frame step is what makes several ticks inside one
// frame move a record several times, which is what the original does — and the
// subtraction is over the counter's own unsigned width, so its wrap is a delta
// like any other rather than an enormous jump.
//
// The horizontal sign is the record's OWN, fixed at its birth by ownership. It
// is not recomputed from the current local participant, so a figure already in
// flight keeps travelling the way it set off.
func (v *Viewer) driftNumerals() {
	count := v.anim.Count()
	steps := int(count - v.numeralTick)
	v.numeralTick = count
	if steps == 0 {
		return
	}
	for i := range v.numerals {
		v.numerals[i].off.X += steps * v.numerals[i].drift
		v.numerals[i].off.Y -= steps * damageNumeralRise
	}
}

// expireNumerals drops every figure whose WALL-CLOCK life is spent.
//
// The bound is inclusive: a record at exactly the life is still live, which is
// the original's own comparison. It is tested against the instant the frame's
// step was given rather than inside the draw — this tree separates a pure
// decision from a draw that only uploads and blits, and the draw takes no clock
// — and both run once per frame, so no frame differs.
//
// A now BEFORE a record's birth yields a negative elapsed and keeps the record.
// That is deliberate: a clock that went backwards must not reap a figure it
// cannot have aged, and the record is reaped as soon as time passes it again.
//
// It compacts in place and blanks the tail, so a dropped record's picture is not
// held alive by the slice's spare capacity.
func (v *Viewer) expireNumerals(now time.Time) {
	kept := v.numerals[:0]
	for _, n := range v.numerals {
		if now.Sub(n.born) <= DamageNumeralLife {
			kept = append(kept, n)
		}
	}
	for i := len(kept); i < len(v.numerals); i++ {
		v.numerals[i] = damageNumeral{}
	}
	v.numerals = kept
}

// DamageNumeralShadow is how far the shadow sits behind the face, in pixels
// down and right.
//
// THE SHAPE IS DECODED AND THE NUMBER IS OURS. The original's draw issues the
// same text virtual TWICE at offset positions — a shadow and a face
// (ANIM-NUM-020, Medium: the glyph blit was identified by its call site and not
// followed into the font, so the offsets it uses are not decoded). One pixel is
// this build's choice, and it is the only thing here that a decode of that
// routine would move.
const DamageNumeralShadow = 1

// DamageNumeralShadowColor is what the shadow is drawn in. Ours, for the reason
// the offset above is: the second issue is decoded, its colour is not. Opaque
// black, so a figure stays legible over bright terrain and over its own unit's
// art alike.
var DamageNumeralShadowColor = color.RGBA{A: 0xff}

// composeNumeral is the figure as a picture: the shadow, then the face over
// it, in ONE image.
//
// One image rather than two draws keeps the blit count at one per figure and
// keeps the whole of what is drawn reachable from a pure method — two draws
// would put the shadow's offset in the draw path, where no assertion can see it.
//
// A viewer holding no font composes nothing and returns nil, which is the
// ordinary state of the standalone developer viewer and of every viewer built
// before a font existed: the record still lives, drifts and expires, and simply
// draws nothing.
func (v *Viewer) composeNumeral(damage int, c color.RGBA) *image.RGBA {
	if v.font == nil {
		return nil
	}
	s := strconv.Itoa(damage)
	w, h := v.font.Measure(s)
	if w <= 0 || h <= 0 {
		return nil
	}
	img := image.NewRGBA(image.Rect(0, 0, w+DamageNumeralShadow, h+DamageNumeralShadow))
	v.font.Draw(img, s, DamageNumeralShadow, DamageNumeralShadow, DamageNumeralShadowColor)
	v.font.Draw(img, s, 0, 0, c)
	return img
}

// numeralPlacement is one figure ready to reach the screen: the picture and the
// window position its top-left corner goes at.
type numeralPlacement struct {
	Pic *image.RGBA
	At  image.Point
	// Blit is Pic without its glyphs and Calls those glyphs, when the text
	// overlay draws them.
	Blit  *image.RGBA
	Calls []text.DrawCall
}

// numeralPlacements is every live figure the current frame can place, in the
// order the records were made.
//
// IT IS PURE AND NEEDS NO WINDOW, which is the shape every other box in this
// package has and for the same reason: a position computed inside the draw is a
// position no assertion can see.
//
// The anchor is the STRUCK UNIT'S OWN drawn position, taken through the very
// transform the health bar takes — the cell footprint, that entity's own sub-tick
// displacement, the displaced mode's per-cell relief lift, the camera and the
// view cull. So a figure carries the relief its unit carries, follows a walking
// unit, and is culled with it. A second copy of that arithmetic here is exactly
// what would let a figure drift off the unit it belongs to on a slope.
//
// THE OFFSET GOES THROUGH THE CAMERA AND THE GLYPH DOES NOT. The record's offset
// is added in cell-local pixels before the transform, so it scales with the zoom
// and a figure stays the same distance out of its unit at any zoom; the picture
// is then blitted at native size, because a numeral is something to read and one
// that shrank with the camera would stop being one. The original has a single
// resolution and no zoom, so this question does not arise there and neither
// choice reproduces it.
//
// A record whose unit is NOT IN THIS FRAME'S ENTITIES places nothing and is not
// destroyed: it expires on its own clock like every other. That is one rule and
// not two, and it is also what the original does with the record it declines to
// draw.
func (v *Viewer) numeralPlacements() []numeralPlacement {
	if len(v.numerals) == 0 {
		return nil
	}
	var out []numeralPlacement
	for _, n := range v.numerals {
		if n.pic == nil {
			continue
		}
		e, ok := v.entityByID(n.victim)
		if !ok {
			continue
		}
		// A FIGURE OVER A UNIT THAT MAY NOT BE SEEN PLACES NOTHING (hotfix), and
		// is not destroyed: it expires on its own clock like every other, exactly
		// as a record whose unit left the frame does. One rule, applied to a
		// second reason a unit is not on screen.
		//
		// This is the half of the numeral's fog gate that catches a victim
		// struck in the light who walks into the dark while his figure is
		// still rising; receiveDamageNumeral carries the other half, over creation.
		if !v.fogGateEntity(e.Owner, e.Cell.X, e.Cell.Y) {
			continue
		}
		foot, in := terrain.CellFootprint(e.Cell.X, e.Cell.Y, v.grid.Width, v.grid.Height, terrain.CellSize)
		if !in {
			continue
		}
		// A ONE-PIXEL ARM at the anchor, so what comes back is a position and
		// not a box: the picture's own size is the picture's, and giving the arm
		// the glyph's dimensions would scale them by the zoom on the way through.
		anchor := image.Pt(foot.Min.X+terrain.CellSize/2, foot.Max.Y).
			Add(n.off).Add(v.entityShift(e))
		r, vis := v.placeArm(e.Cell, image.Rectangle{Min: anchor, Max: anchor.Add(image.Pt(1, 1))})
		if !vis {
			continue
		}
		out = append(out, numeralPlacement{Pic: n.pic, At: image.Pt(int(r.X), int(r.Y)), Blit: n.blit, Calls: n.calls})
	}
	return out
}

// entityByID is this frame's entry for one id, or false when the frame holds
// none. It is a linear walk over the snapshot rather than an index kept beside
// it: the snapshot is rebuilt every tick and an index would be a second
// structure to rebuild with it, for a list this package already walks per frame
// several times over.
func (v *Viewer) entityByID(id uint32) (MapEntity, bool) {
	for _, e := range v.entities {
		if e.ID == id {
			return e, true
		}
	}
	return MapEntity{}, false
}

// DamageNumeralState is a read-only record for headless event witnesses.
type DamageNumeralState struct {
	Victim uint32
	Tick   uint64
	Amount int
	Offset image.Point
}

// DamageNumeralStates copies the live records without their glyph storage.
func (v *Viewer) DamageNumeralStates() []DamageNumeralState {
	out := make([]DamageNumeralState, len(v.numerals))
	for i, n := range v.numerals {
		out[i] = DamageNumeralState{Victim: n.victim, Tick: n.tick, Amount: n.damage, Offset: n.off}
	}
	return out
}
