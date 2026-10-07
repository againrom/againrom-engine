package game

import (
	"time"

	"againrom/pkg/audio"
)

const schoolColumnInterval = 83 * time.Millisecond
const schoolRotateSound = "town/school/rotate.wav"

// schoolColumnAnimation is presentation only. TOWN-147 supplies the single
// step and strict clock gate. Immediate target changes, including reversal
// from the displayed frame, are owner policy (DIV-814), not decoded lifecycle.
type schoolColumnAnimation struct {
	frame, target int
	ready         bool
	last          time.Time
}

func schoolColumnEndpoint(mage bool) int {
	if mage {
		return 15
	}
	return 0
}

func (a *schoolColumnAnimation) reset(mage bool) {
	frame := schoolColumnEndpoint(mage)
	*a = schoolColumnAnimation{frame: frame, target: frame, ready: true}
}

func (a *schoolColumnAnimation) retarget(mage bool, now time.Time) bool {
	target := schoolColumnEndpoint(mage)
	if a.target == target {
		return false
	}
	a.target = target
	a.last = now
	return a.frame != target
}

func (a *schoolColumnAnimation) advance(now time.Time) {
	if !a.ready || a.frame == a.target || now.Sub(a.last) <= schoolColumnInterval {
		return
	}
	a.advanceStep()
	// Never catch up missed frames: the original advances at most once/paint.
	a.last = now
}

func (a *schoolColumnAnimation) advanceStep() {
	if !a.ready || a.frame == a.target {
		return
	}
	if a.frame < a.target {
		a.frame++
	} else {
		a.frame--
	}
}

func (t *townScreen) townAnimationNow() time.Time {
	if now := t.draws.animationClock(); now != nil {
		return now()
	}
	return time.Now()
}

func (t *townScreen) resetSchoolColumn() {
	t.schoolColumn = schoolColumnAnimation{}
	party, member := t.shopParty(), t.shopMemberIndex()
	if member >= 0 && member < len(party) {
		t.schoolColumn.reset(party[member].Mage)
	}
}

func (t *townScreen) retargetSchoolColumn(previousMage, mage bool) {
	if t.room != roomSchool || previousMage == mage {
		return
	}
	if !t.schoolColumn.ready {
		t.schoolColumn.reset(previousMage)
	}
	now := t.townAnimationNow()
	if t.armSchoolTrainingTransition(schoolClassForMage(mage), now, true) {
		return
	}
	if !t.schoolColumnComplete() {
		t.schoolColumn.reset(mage)
		return
	}
	if t.schoolColumn.retarget(mage, now) {
		t.requestSchoolRotateSound()
	}
}

func (t *townScreen) requestSchoolRotateSound() { t.requestSchoolSound(schoolRotateSound) }

func (t *townScreen) requestSchoolSound(name string) {
	if t == nil || t.sess == nil || t.sound.soundDevice() == nil {
		return
	}
	if sample, ok := t.in.SoundBank.namedSample(name); ok {
		audio.Dispatch(t.roomSoundPlayer(audio.EffectsChannel), sample,
			audio.FixedRequest("school-training", name, audio.EffectsChannel, 128, false,
				audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}))
	}
}

// schoolPanelClass is the displayed endpoint, not the newly selected class.
// Older synthetic front ends without entry state retain their static panel.
func (t *townScreen) schoolPanelClass() int {
	party, member := t.shopParty(), t.shopMemberIndex()
	if member < 0 || member >= len(party) {
		return -1
	}
	if t.schoolTrainingBusy() {
		return -1
	}
	if !t.schoolColumn.ready {
		return schoolColumnEndpoint(party[member].Mage) / 15
	}
	switch t.schoolColumn.frame {
	case 0:
		return 0
	case 15:
		return 1
	default:
		return -1
	}
}
