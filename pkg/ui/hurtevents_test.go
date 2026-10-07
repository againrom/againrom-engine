package ui

import (
	"slices"
	"testing"
	"time"
)

// hurtSlotOne is the unchanged-health cue slot of hurtClass.
const hurtSlotOne = 205

// hurtClass is a class's Sound array with every index named: swing, the
// no-damage cue, the two wound grunts and the fall.
func hurtClass() []int {
	return []int{soundSwingSlot, hurtSlotOne, soundHighSlot, soundLowSlot, soundDieSlot}
}

func hurtBank() voiceBank {
	b := voiceBankFor()
	b.stubBank[hurtSlotOne] = soundSample(hurtSlotOne)
	return b
}

// hurtDevices is a viewer with distinct effects and speech devices.
func hurtDevices(t *testing.T) (v *Viewer, effects, speech *recordingPlayer) {
	t.Helper()
	v = numeralViewer(t)
	effects, speech = &recordingPlayer{}, &recordingPlayer{}
	v.SetAudio(effects, hurtBank())
	v.SetSpeechAudio(speech)
	return v, effects, speech
}

func hurtUnit(id, owner uint32, hp int, voice string) MapEntity {
	e := soundUnit(id, owner, hp)
	e.Sound, e.Voice = hurtClass(), voice
	return e
}

// strike is e after the simulation reports one more unit strike that took no
// health.
func strike(v *Viewer, e *MapEntity) MapEntity {
	v.AppendDamageMessages([]DamageMessage{{Entity: *e, AfterHP: e.HP}})
	return *e
}

// A strike that took no health plays index 1 of the drawn class on the effects
// device for a bank-voiced and a class-voiced entity alike, with no gate and no
// timestamp, and never a bank leaf (ANIM-094, ANIM-125, ANIM-128).
func TestAStrikeThatTookNoHealthPlaysSoundOneOnTheEffectsDevice(t *testing.T) {
	for _, voice := range []string{"", "mf_hero"} {
		t.Run("voice "+voice, func(t *testing.T) {
			v, effects, speech := hurtDevices(t)
			u := hurtUnit(1, 1, 100, voice)
			push(v, at0, u)
			push(v, at0, strike(v, &u))
			push(v, at0.Add(time.Millisecond), strike(v, &u))
			if got := heard(effects); !slices.Equal(got, []string{"slot 205", "slot 205"}) {
				t.Errorf("effects device heard %v, want two plays of slot 205", got)
			}
			if got := heard(speech); len(got) != 0 {
				t.Errorf("speech device heard %v, want nothing", got)
			}
			// The strike stored no timestamp: a wound at once still sounds.
			push(v, at0.Add(2*time.Millisecond), blow(v, &u, 60))
			if got := heard(speech); len(got) != 1 {
				t.Errorf("a wound after two strikes heard %v, want one voice", got)
			}
		})
	}
}

// A strike on an entity's first frame voices nothing, like every hurt event.
func TestAStrikeOnTheFirstFrameIsSilent(t *testing.T) {
	v, effects, _ := hurtDevices(t)
	u := hurtUnit(1, 1, 100, "")
	v.AppendDamageMessages([]DamageMessage{{Entity: u, AfterHP: u.HP}, {Entity: u, AfterHP: u.HP}, {Entity: u, AfterHP: u.HP}})
	push(v, at0, u)
	if got := heard(effects); len(got) != 0 {
		t.Fatalf("first sighting heard %v", got)
	}
}

// Events 1 to 3 play on the speech device and never on the effects device,
// whether the entity is bank-voiced or class-voiced (ANIM-094, ANIM-128).
func TestWoundsAndFallsPlayOnTheSpeechDeviceAlone(t *testing.T) {
	for _, tc := range []struct {
		voice string
		want  []string
	}{
		{"mf_hero", []string{"mf_hero/easy.wav", "mf_hero/hard.wav", "mf_hero/die.wav"}},
		{"", []string{"slot 202", "slot 203", "slot 204"}},
	} {
		t.Run("voice "+tc.voice, func(t *testing.T) {
			v, effects, speech := hurtDevices(t)
			u := hurtUnit(1, 1, 100, tc.voice)
			push(v, at0, u)
			push(v, at0, blow(v, &u, 40))
			push(v, at0.Add(2*time.Second), blow(v, &u, 20))
			push(v, at0.Add(2*time.Second), blow(v, &u, -5))
			if got := heard(speech); !slices.Equal(got, tc.want) {
				t.Errorf("speech device heard %v, want %v", got, tc.want)
			}
			if got := heard(effects); len(got) != 0 {
				t.Errorf("effects device heard %v, want nothing", got)
			}
		})
	}
}

// Without a speech device a wound is silent and the strike cue still plays.
func TestAMissingSpeechDeviceSilencesWoundsOnly(t *testing.T) {
	v := numeralViewer(t)
	effects := &recordingPlayer{}
	v.SetAudio(effects, hurtBank())
	u := hurtUnit(1, 1, 100, "mf_hero")
	push(v, at0, u)
	push(v, at0, blow(v, &u, 60))
	push(v, at0.Add(time.Second), strike(v, &u))
	if got := heard(effects); !slices.Equal(got, []string{"slot 205"}) {
		t.Fatalf("effects device heard %v, want the strike cue alone", got)
	}
}

// A fallen body of the local player's, at -1 to -10, that loses a point with no
// blow plays event 2 through the shared gate; nobody else's body, and no body
// below that band, does (ANIM-126).
func TestAFallenBodyBleedsHardToItsOwnerOnly(t *testing.T) {
	bleed := func(u *MapEntity, hp int) MapEntity {
		u.HP, u.Life = hp, lifeAt(hp)
		return *u
	}
	for _, tc := range []struct {
		name   string
		owner  uint32
		from   int
		to     int
		voice  string
		wanted []string
	}{
		{"own body at -3", 0, -3, -4, "f_mage", []string{"f_mage/hard.wav"}},
		{"own body at -9 reaching -10", 0, -9, -10, "", []string{"slot 203"}},
		{"own body at -1", 0, -1, -2, "mf_hero", []string{"mf_hero/hard.wav"}},
		{"own body at -10", 0, -10, -11, "mf_hero", nil},
		{"own body at zero", 0, 0, -1, "mf_hero", nil},
		{"another player's body", 2, -3, -4, "mf_hero", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, effects, speech := hurtDevices(t)
			u := hurtUnit(1, tc.owner, tc.from, tc.voice)
			push(v, at0, u)
			push(v, at0, bleed(&u, tc.to))
			if got := heard(speech); !slices.Equal(got, tc.wanted) {
				t.Errorf("speech device heard %v, want %v", got, tc.wanted)
			}
			if got := heard(effects); len(got) != 0 {
				t.Errorf("effects device heard %v", got)
			}
		})
	}

	t.Run("the bleed shares the voice gate", func(t *testing.T) {
		v, _, speech := hurtDevices(t)
		u := hurtUnit(1, 0, -2, "mf_hero")
		push(v, at0, u)
		push(v, at0.Add(time.Second), bleed(&u, -3))
		push(v, at0.Add(2*time.Second), bleed(&u, -4))
		push(v, at0.Add(2600*time.Millisecond), bleed(&u, -5))
		push(v, at0.Add(4*time.Second), bleed(&u, -6))
		if got := heard(speech); len(got) != 2 {
			t.Errorf("heard %v, want the bleeds at 1.0 and 2.6 s", got)
		}
	})
}
