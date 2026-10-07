package game

import (
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// hurtVoiceSplit runs one hurtVoiceFight with its own recorder for the speech
// device, so a play shows which device it took, and returns the effects and
// speech plays.
func hurtVoiceSplit(t *testing.T, hero sim.Entity, fig figureID, damage int32, frames int) (mw *mapWorld, effects, speech *hurtVoiceRecorder) {
	t.Helper()
	var a *ui.App
	mw, a, effects = hurtVoiceFight(t, hero, fig, damage, 8)
	speech = &hurtVoiceRecorder{}
	mw.view.SetSpeechAudio(speech)
	for effects.frame = 0; effects.frame < frames; effects.frame++ {
		speech.frame = effects.frame
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	return mw, effects, speech
}

// A hero's wounds sound on the speech device from his bank, and what he wields
// changes none of it: Danas as the archer class and as the fighter class play
// mf_hero leaves and nothing from either class's array (ANIM-094, ANIM-096,
// ANIM-128). The effects device holds no wound.
func TestAHerosWoundsPlayFromHisBankOnTheSpeechDeviceWhateverHeWields(t *testing.T) {
	for _, class := range []int32{14, 2} {
		hero := sim.Entity{ID: 0, X: 4, Y: 4, Owner: sim.SelfSlot, TypeID: sim.HeroTypeID(false, false),
			Class: class, Humanoid: true, HP: 100, MaxHP: 100}
		mw, effects, speech := hurtVoiceSplit(t, hero, figureID{Dir: data.FigureDirManFighter, Hero: true}, 7, 150)
		if e, _ := mw.world.Entity(0); e.HP >= 100 {
			t.Fatalf("class %d: the fixture landed no blow", class)
		}
		if len(speech.plays) == 0 {
			t.Fatalf("class %d: no wound reached the speech device", class)
		}
		for _, p := range speech.plays {
			if p.src != "mf_hero/easy.wav" && p.src != "mf_hero/hard.wav" {
				t.Errorf("class %d: speech device played %s", class, p.src)
			}
		}
		if len(effects.plays) != 0 {
			t.Errorf("class %d: effects device played %v", class, effects.plays)
		}
	}
}

// A unit strike that takes no health plays index 1 of the drawn class on the
// effects device, once per strike, and no wound: that is event 0 (ANIM-094,
// ANIM-125). The drawn class is the one cue weapon changes (ANIM-096).
func TestAStrikeThatTakesNoHealthPlaysTheDrawnClassCue(t *testing.T) {
	for _, tc := range []struct {
		class int32
		slot  string
	}{{14, "class slot 211"}, {2, "class slot 212"}} {
		hero := sim.Entity{ID: 0, X: 4, Y: 4, Owner: sim.SelfSlot, TypeID: sim.HeroTypeID(false, false),
			Class: tc.class, Humanoid: true, HP: 100, MaxHP: 100}
		mw, effects, speech := hurtVoiceSplit(t, hero, figureID{Dir: data.FigureDirManFighter, Hero: true}, 0, 150)
		strikes := int(mw.strikes[0])
		if strikes < 3 {
			t.Fatalf("class %d: the fixture reported %d zero-damage strikes", tc.class, strikes)
		}
		var got []string
		for _, p := range effects.plays {
			got = append(got, p.src)
		}
		want := slices.Repeat([]string{tc.slot}, strikes)
		if !slices.Equal(got, want) {
			t.Errorf("class %d: effects device played %v, want %d of %s", tc.class, got, strikes, tc.slot)
		}
		if len(speech.plays) != 0 {
			t.Errorf("class %d: speech device played %v", tc.class, speech.plays)
		}
		if e, _ := mw.world.Entity(0); e.HP != 100 || mw.blows[0] != 0 {
			t.Errorf("class %d: health %d, %d blows; no blow should have landed", tc.class, e.HP, mw.blows[0])
		}
	}
}
