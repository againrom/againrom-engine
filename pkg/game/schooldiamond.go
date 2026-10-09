package game

// schoolDiamondAnimation is presentation state only. TOWN-379 arms by writing
// step +1, without resetting phase or current. TOWN-380 advances once per
// school own-surface paint, not by the adjacent column's clock. Actual original
// display cadence and later request-reply effects remain Unknown (DIV-524).
// Frames stay in the startup art cache rather than being released (DIV-525).
type schoolDiamondAnimation struct {
	frame int
	step  int
	ready bool
}

func (a *schoolDiamondAnimation) arm() { a.step = 1 }

func (a *schoolDiamondAnimation) advance() {
	if a.step == 0 {
		return
	}
	a.frame += a.step
	if a.frame >= 8 {
		a.frame, a.step = 8, -1
	} else if a.frame == 0 {
		a.step = 0
	}
	// TOWN-381's common tail publishes frames[phase] even on completion.
	// The cached zero frame remains hidden by the zero-step paint gate.
	a.ready = true
}

// AdvanceTownSurfaceAnimation is called only by the town room compositor.
// Selection, hit-testing, app update ticks, and snapshot reads cannot advance
// the diamond. A school dialogue keeps its room background painting too.
func (t *townScreen) AdvanceTownSurfaceAnimation() {
	if t.inTavernInterior() {
		t.tavernPage().Advance()
		return
	}
	if t.room == roomSchool || t.room == roomTalk && t.dialogueBuilding == TownSchool {
		t.schoolDiamond.advance()
		t.advanceSchoolShine(t.townAnimationNow())
		// The legacy fallback below is degraded-install compatibility (no
		// complete tr family, so advanceSchoolTraining never owns the
		// column). It must not also become the route a live, complete
		// install's column keeps moving through while
		// schoolTrainingSuspended freezes everything else.
		if !t.advanceSchoolTraining() && !t.schoolTrainingSuspended() {
			t.schoolColumn.advance(t.townAnimationNow())
		}
	}
}
