package game

import (
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/town"
)

func (t *townScreen) schoolPage() *town.Scene { return t.roomPage("school") }

func (t *townScreen) schoolColumn() *town.Target {
	return t.schoolPage().Actor("column").(*town.Target)
}

func (t *townScreen) schoolTraining() *town.Training {
	return t.schoolPage().Actor("training").(*town.Training)
}

func (t *townScreen) schoolDiamond() *town.Bounce {
	return t.schoolPage().Actor("diamond").(*town.Bounce)
}

func (t *townScreen) schoolShine() *town.Cycle {
	return t.schoolPage().Actor("shine").(*town.Cycle)
}

func (t *townScreen) townAnimationNow() time.Time {
	if now := t.draws.animationClock(); now != nil {
		return now()
	}
	return time.Now()
}

// schoolMemberClass is the shown member's class: fighter 0, mage 1, -1 with
// no member.
func (t *townScreen) schoolMemberClass() int {
	party, member := t.shopParty(), t.shopMemberIndex()
	if member < 0 || member >= len(party) {
		return -1
	}
	if party[member].Mage {
		return schoolMageClass
	}
	return schoolFighterClass
}

func (t *townScreen) inSchoolTraining() bool {
	return t != nil && (t.room == roomSchool || t.room == roomTalk && t.dialogueBuilding == TownSchool)
}

// enterSchoolPage enters the school page; without a session it only returns
// the page to one never entered.
func (t *townScreen) enterSchoolPage() {
	if t.sess == nil {
		t.schoolPage().Reset()
		return
	}
	t.schoolPage().Enter()
}

func (t *townScreen) leaveSchoolTraining() {
	t.schoolPage().Reset()
	t.schoolSounds.Stop()
	if t.soundRoom() != roomSchool && t.audioRoom == roomSchool {
		t.destroyRoomAudio()
	}
}

// SchoolTrainingActive is Againrom's explicit delivery policy for focus,
// menu and cutscene pauses. A resume rebases the shared gate, both idle
// clocks and the shine, so hidden wall time can never catch up animation
// frames (DIV-872).
func (t *townScreen) SchoolTrainingActive(active bool) {
	if t == nil || t.sess == nil {
		return
	}
	t.schoolPage().SetActive(active, t.inSchoolTraining())
}

// retargetSchoolColumn raises the class change on the school page when the
// shown member's class changes inside the school.
func (t *townScreen) retargetSchoolColumn(previousMage, mage bool) {
	if t.room != roomSchool || previousMage == mage {
		return
	}
	t.schoolPage().Event("class-change", schoolClassForMage(previousMage), schoolClassForMage(mage))
}

func schoolClassForMage(mage bool) int {
	if mage {
		return schoolMageClass
	}
	return schoolFighterClass
}

func (t *townScreen) schoolTrainingBusy() bool {
	return t != nil && t.schoolTraining().Busy(t.schoolPage())
}

// schoolColumnShown answers the column frame the scene draws and whether it
// draws one.
func (t *townScreen) schoolColumnShown() (int, bool) {
	col := t.schoolColumn()
	if !col.Ready {
		return col.Frame, false
	}
	art := t.in.TownSchoolArt.Value()
	if art == nil {
		return col.Frame, false
	}
	frames := art.Scene["column"]
	return col.Frame, col.Frame >= 0 && col.Frame < len(frames) && frames[col.Frame] != nil
}

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
	class := t.schoolMemberClass()
	if class < 0 || t.schoolTrainingBusy() {
		return -1
	}
	col := t.schoolColumn()
	if !col.Ready {
		return class
	}
	for c := schoolFighterClass; c < schoolClassCount; c++ {
		if col.Frame == col.Endpoint(c) {
			return c
		}
	}
	return -1
}

// AdvanceTownSurfaceAnimation is called only by the town room compositor.
// Selection, hit-testing, app update ticks, and snapshot reads cannot advance
// a room page. A school dialogue keeps its room background painting too.
func (t *townScreen) AdvanceTownSurfaceAnimation() {
	if t.inTavernInterior() {
		t.tavernPage().Advance()
		return
	}
	if t.inSchoolTraining() {
		t.schoolPage().Advance()
	}
}
