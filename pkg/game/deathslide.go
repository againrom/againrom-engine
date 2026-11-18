package game

import (
	"image"

	"againrom/pkg/sim"
)

// crossingMemo is the part of a step a unit still owes: the cell offset it is
// drawn sliding down, the ticks left to pay and the crossing's length.
type crossingMemo struct {
	step       image.Point
	owed, span int
}

// advanceDeathSlides runs after each tick. The simulation drops a felled
// unit's crossing at once and leaves the body on the cell it was entering, so
// the drawing keeps what the unit owed when it was last seen alive: a unit
// that is not alive after the tick starts a slide from that record, and every
// slide pays one tick of its crossing and ends when none is owed.
func (mw *mapWorld) advanceDeathSlides() {
	for _, e := range mw.world.EntityView() {
		if e.Alive() {
			delete(mw.deathSlide, e.ID)
			mw.rememberCrossing(e)
			continue
		}
		slide, sliding := mw.deathSlide[e.ID]
		if !sliding {
			slide, sliding = mw.crossing[e.ID]
			delete(mw.crossing, e.ID)
		}
		if !sliding {
			continue
		}
		slide.owed--
		if slide.owed <= 0 {
			delete(mw.deathSlide, e.ID)
			continue
		}
		if mw.deathSlide == nil {
			mw.deathSlide = make(map[sim.EntityID]crossingMemo)
		}
		mw.deathSlide[e.ID] = slide
	}
}

// rememberCrossing keeps the step a living unit is in the middle of, or
// forgets it when the unit stands or has just been relocated.
func (mw *mapWorld) rememberCrossing(e sim.Entity) {
	step := image.Pt(int(e.X), int(e.Y)).Sub(mw.prev[e.ID])
	if e.Transit == 0 || step == (image.Point{}) || step.X < -1 || step.X > 1 || step.Y < -1 || step.Y > 1 {
		delete(mw.crossing, e.ID)
		return
	}
	if mw.crossing == nil {
		mw.crossing = make(map[sim.EntityID]crossingMemo)
	}
	mw.crossing[e.ID] = crossingMemo{step: step, owed: int(e.Transit), span: mw.crossingSpan(e)}
}
