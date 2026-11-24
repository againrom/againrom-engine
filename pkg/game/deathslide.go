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
// unit's crossing at once and leaves the body on its logical cell, the cell it
// was entering once the crossing is made and the cell it was leaving before.
// The drawing keeps the slide to that cell's centre from where the unit was
// last drawn alive: a unit that is not alive after the tick starts a slide
// from that record, and every slide pays one tick and ends when none is owed
// (DIV-5128).
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

// rememberCrossing keeps the slide a living unit's body would make from where
// it is drawn, or forgets it when the unit stands or has just been relocated.
// Past the crossing the body goes on to the cell it entered; before it, the
// logical cell is still the one it leaves, and the body goes back to that
// cell's centre over the ticks the step has run, so it never jumps.
func (mw *mapWorld) rememberCrossing(e sim.Entity) {
	from := mw.prev[e.ID]
	step := drawnCell(e).Sub(from)
	if e.Transit == 0 || step == (image.Point{}) || step.X < -1 || step.X > 1 || step.Y < -1 || step.Y > 1 {
		delete(mw.crossing, e.ID)
		return
	}
	if mw.crossing == nil {
		mw.crossing = make(map[sim.EntityID]crossingMemo)
	}
	memo := crossingMemo{step: step, owed: int(e.Transit), span: mw.crossingSpan(e)}
	if image.Pt(int(e.X), int(e.Y)) == from {
		memo.step, memo.owed = step.Mul(-1), memo.span-int(e.Transit)
	}
	mw.crossing[e.ID] = memo
}
