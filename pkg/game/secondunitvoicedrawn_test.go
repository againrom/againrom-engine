package game

import (
	"slices"
	"testing"
	"time"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// TestSecondGamePlacedPersonDrawnAsHeroSpeaksAsOne checks that a placed
// person the client draws as a hero (word 21..23, R2-ENGINE-680) is
// hero-shaped for the picker and speaks the voice of his word.
func TestSecondGamePlacedPersonDrawnAsHeroSpeaksAsOne(t *testing.T) {
	mw := voiceFixture(t, armedMan, voicePerson{typeID: 1, dir: data.FigureDirWomanFighter, owner: sim.SelfSlot, hp: 10})
	secondVoiceMission(mw, nil, map[sim.EntityID]uint16{1: 100, 2: 23})
	mw.mission.drawnHeroes = map[sim.EntityID]int32{2: sim.HeroTypeID(false, true)}
	f, voices := voiceFront(mw, secondVoiceBank("characters/dina/select/1.wav", "characters/m_fighter/select/1.wav"))
	script := &scriptedDraw{}
	_, selection := f.runtimeAudio().unitReplies(mw, script.draw)
	script.set(0, 0)
	selection([]uint32{1, 2}, time.Unix(100, 0))
	if got := heard(voices, 0); !slices.Equal(got, []int16{1}) {
		t.Errorf("heard %v, want dina's select/1 from the person drawn as a hero", got)
	}
}
