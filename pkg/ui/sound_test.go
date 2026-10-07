package ui

import (
	"slices"
	"strconv"
	"testing"
	"time"

	"againrom/pkg/audio"
)

// A landed blow's sound (0126). Every test below drives the viewer's own step
// through push, the exact helper numeral_test.go already built for the
// numeral's own tests: SetEntities, then the viewer's step at a hand-made
// instant. A recording audio.Player stands in for real audio hardware, so no
// window, no clock and no sound archive is needed to say what a blow leaves
// behind — the same reason a recording device stands in for real hardware in
// sounddev.go's own doc comment, one file over.

// recordedPlay is one call this test's recording audio.Player observed.
type recordedPlay struct {
	Sample    audio.Sample
	Placement audio.Placement
}

// recordingPlayer is the audio.Player every test below hands SetAudio. It
// makes no sound; it records every play in the order Play was called, so a
// test can assert exactly how many plays a push made and, through the
// Sample each one carries, which slot it resolved to.
type recordingPlayer struct {
	plays []recordedPlay
}

func (r *recordingPlayer) Play(s audio.Sample, p audio.Placement) {
	r.plays = append(r.plays, recordedPlay{Sample: s, Placement: p})
}

func (r *recordingPlayer) RequestSample(s audio.Sample, request audio.Request) audio.Voice {
	r.Play(s, request.Placement)
	return nil
}

type stubBank map[int]audio.Sample

func (b stubBank) Sample(slot int) (audio.Sample, bool) {
	s, ok := b[slot]
	return s, ok
}

// The three archive slot numbers soundBankFor answers for, distinct from
// each other and from the numeral tests' own ids and cells so a play can be
// told apart by the one PCM value soundSample gives it.
const (
	soundSwingSlot = 101
	soundHighSlot  = 202
	soundLowSlot   = 203
	soundDieSlot   = 204
)

// soundSample is a one-frame Sample carrying its own slot number as its only
// PCM value, so a test can identify which slot a recordingPlayer's play
// resolved to by reading the sample it was handed, without this file
// needing its own copy of a bank lookup.
func soundSample(slot int) audio.Sample {
	return audio.Sample{Rate: audio.DeviceRate, PCM: []int16{int16(slot)}}
}

// soundBankFor is the stub bank every test below shares: the two wound
// grunts a class's Sound array can name, plus the swing slot — unused by
// this task's own ACs (AC-14 is T3's), kept here so soundClass's array is
// complete against spec.md's Terms table rather than answering only the two
// indices this task happens to exercise.
func soundBankFor() stubBank {
	return stubBank{
		soundSwingSlot: soundSample(soundSwingSlot),
		soundHighSlot:  soundSample(soundHighSlot),
		soundLowSlot:   soundSample(soundLowSlot),
		soundDieSlot:   soundSample(soundDieSlot),
	}
}

// soundClass is a class's Sound array in spec.md's Terms order: the swing,
// the unchanged-health grunt (index 1, out of scope — left zero), the
// at-or-above-half grunt, the below-half grunt.
func soundClass() []int {
	return []int{soundSwingSlot, 0, soundHighSlot, soundLowSlot}
}

// soundUnit is numeralUnit's own entity (numeral_test.go), carrying the
// class's Sound array too — the one field numeralUnit's own callers, testing
// a different instrument, have no use for.
func soundUnit(id, owner uint32, hp int) MapEntity {
	e := numeralUnit(id, owner, hp)
	e.Sound = soundClass()
	e.Life = lifeAt(hp)
	return e
}

// lifeAt is the life a pushed health shows, as the game tier translates it.
func lifeAt(hp int) uint8 {
	switch {
	case hp > 0:
		return LifeAlive
	case hp == 0:
		return LifeDowned
	}
	return LifeDead
}

// blow is e after the simulation reports one more blow that leaves it at hp.
func blow(v *Viewer, e *MapEntity, hp int) MapEntity {
	v.AppendDamageMessages([]DamageMessage{{Entity: *e, AfterHP: hp}})
	e.HP, e.Life = hp, lifeAt(hp)
	return *e
}

// AC-1: a viewer holding a recording device and a class with slots, struck
// from full health to 60% of maximum, plays slot index 2 exactly once.
func TestFR1PlaysTheAtOrAboveHalfGrunt(t *testing.T) {
	v := numeralViewer(t)
	rec := &recordingPlayer{}
	v.SetAudio(rec, soundBankFor())
	v.SetSpeechAudio(rec)

	u := soundUnit(1, 1, 100)
	push(v, at0, u)               // first sighting: plays nothing
	push(v, at0, blow(v, &u, 60)) // struck at 100, at or above half

	if len(rec.plays) != 1 {
		t.Fatalf("plays = %d, want exactly 1", len(rec.plays))
	}
	if got := rec.plays[0].Sample.PCM[0]; got != soundHighSlot {
		t.Errorf("played slot %d, want %d — the at-or-above-half grunt (index 2)", got, soundHighSlot)
	}
}

// AC-2: the band is the health held BEFORE the blow (ANIM-095). Struck at 40
// of 100 it plays slot index 3; struck at 100 down to 40 it still plays
// index 2.
func TestFR1PlaysTheBelowHalfGrunt(t *testing.T) {
	for _, tc := range []struct {
		name         string
		before, then int
		want         int16
	}{
		{"struck below half", 40, 30, soundLowSlot},
		{"struck from full to below half", 100, 40, soundHighSlot},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := numeralViewer(t)
			rec := &recordingPlayer{}
			v.SetAudio(rec, soundBankFor())
			v.SetSpeechAudio(rec)

			u := soundUnit(1, 1, tc.before)
			push(v, at0, u)
			push(v, at0, blow(v, &u, tc.then))

			if len(rec.plays) != 1 {
				t.Fatalf("plays = %d, want exactly 1", len(rec.plays))
			}
			if got := rec.plays[0].Sample.PCM[0]; got != tc.want {
				t.Errorf("played slot %d, want %d", got, tc.want)
			}
		})
	}
}

// AC-3: health pushed unchanged, higher, or lower with no blow behind it
// plays nothing.
func TestFR1AHealthThatDidNotFallPlaysNothing(t *testing.T) {
	for _, tc := range []struct {
		name string
		then int
	}{
		{"unchanged", 40},
		{"risen", 70},
		{"lowered by no blow", 30},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := numeralViewer(t)
			rec := &recordingPlayer{}
			v.SetAudio(rec, soundBankFor())
			v.SetSpeechAudio(rec)

			push(v, at0, soundUnit(1, 1, 40))
			push(v, at0, soundUnit(1, 1, tc.then))

			if len(rec.plays) != 0 {
				t.Errorf("a health that %s made %d play(s), want 0", tc.name, len(rec.plays))
			}
		})
	}
}

// AC-4: a blow on an entity holding -10 plays nothing; one holding -9 plays
// (GruntFloor, on the stored health).
func TestFR2RefusesAtTheFloorAndPlaysOneAboveIt(t *testing.T) {
	for _, tc := range []struct {
		name      string
		hp        int
		wantPlays int
	}{
		{"struck at the floor, -10, plays nothing", -10, 0},
		{"struck one above the floor, -9, plays", -9, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := numeralViewer(t)
			rec := &recordingPlayer{}
			v.SetAudio(rec, soundBankFor())
			v.SetSpeechAudio(rec)

			u := soundUnit(1, 1, tc.hp)
			push(v, at0, u)
			push(v, at0, blow(v, &u, tc.hp-5))

			if len(rec.plays) != tc.wantPlays {
				t.Errorf("plays = %d, want %d", len(rec.plays), tc.wantPlays)
			}
		})
	}
}

// THE OWNER'S REPORT (hotfix): «поправить звук когда
// враг получает урон от истлевания — тут
// не должно быть звука». The decay ladder walks a corpse
// from 0 downward, and every step from -1 to -9 clears GruntFloor's own -10,
// so a body grunted its way down after it was already dead.
//
// THE COUNT IS EXACTLY ONE AND EXACTLY NINE ZEROES, not "fewer plays". The
// killing blow lands on a living victim, so it MUST still grunt; the nine
// decay steps after it must make no sound at all.
//
// EVERY PUSH IS 2000ms PAST THE ONE BEFORE, well clear of GruntThrottle's
// 1500ms, and that spacing is load-bearing rather than incidental: pushed at
// one instant the throttle alone would suppress the nine and this test would
// pass against a tree that voiced decay. A decay step carries no new blow,
// and that is what MUST be the reason.
func TestADecayingCorpseGruntsOnceForTheKillingBlowAndNotForTheLadder(t *testing.T) {
	v := numeralViewer(t)
	rec := &recordingPlayer{}
	v.SetAudio(rec, soundBankFor())
	v.SetSpeechAudio(rec)

	const gap = 2000 * time.Millisecond
	now := at0
	u := soundUnit(1, 1, 5)
	push(v, now, u) // alive, first sighting: plays nothing

	now = now.Add(gap)
	push(v, now, blow(v, &u, -1)) // THE KILLING BLOW: 5 -> -1, must grunt
	if len(rec.plays) != 1 {
		t.Fatalf("the killing blow made %d play(s), want exactly 1", len(rec.plays))
	}
	if got := rec.plays[0].Sample.PCM[0]; got != soundLowSlot {
		t.Errorf("the killing blow played slot %d, want %d — struck below half of 100", got, soundLowSlot)
	}

	// The ladder: -2 through -9, every rung above GruntFloor's -10 and every
	// one of them a strict decrease with no blow behind it.
	for hp := -2; hp >= -9; hp-- {
		now = now.Add(gap)
		u.HP, u.Life = hp, lifeAt(hp)
		push(v, now, u)
		if len(rec.plays) != 1 {
			t.Fatalf("the decay step to %d brought the total to %d play(s), want it still at 1",
				hp, len(rec.plays))
		}
	}
	if len(rec.plays) != 1 {
		t.Errorf("plays = %d over a killing blow and nine rungs of decay, want exactly 1", len(rec.plays))
	}
}

// AC-5: two wounds 100ms apart play once; two 1600ms apart play twice
// (GruntThrottle). The clock already reaches push/step, so this needs no
// seam of its own.
func TestFR3ThrottlesGruntsPerVictim(t *testing.T) {
	for _, tc := range []struct {
		name      string
		gap       time.Duration
		wantPlays int
	}{
		{"100ms apart throttles to one grunt", 100 * time.Millisecond, 1},
		{"1600ms apart plays both", 1600 * time.Millisecond, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := numeralViewer(t)
			rec := &recordingPlayer{}
			v.SetAudio(rec, soundBankFor())
			v.SetSpeechAudio(rec)

			u := soundUnit(1, 1, 100)
			push(v, at0, u)
			push(v, at0, blow(v, &u, 90))             // first wound: plays
			push(v, at0.Add(tc.gap), blow(v, &u, 80)) // second wound, tc.gap later

			if len(rec.plays) != tc.wantPlays {
				t.Errorf("plays = %d, want %d", len(rec.plays), tc.wantPlays)
			}
		})
	}
}

// AC-6: an entity present in the first push at low health plays nothing on
// that push — arriving is not a wound, whatever blows it arrives with.
func TestFR4TheFirstSightingPlaysNothing(t *testing.T) {
	v := numeralViewer(t)
	rec := &recordingPlayer{}
	v.SetAudio(rec, soundBankFor())
	v.SetSpeechAudio(rec)

	u := soundUnit(1, 1, 10) // low health, but the FIRST push
	v.AppendDamageMessages([]DamageMessage{{Entity: u, AfterHP: u.HP}})
	push(v, at0, u)

	if len(rec.plays) != 0 {
		t.Errorf("plays = %d, want 0 — arriving is not a wound", len(rec.plays))
	}
}

// AC-7: a class whose slot array is empty, shorter than the index, or zero
// at the index plays nothing, and the push does not fail.
func TestAC7AShortOrZeroClassArrayPlaysNothingAndDoesNotFail(t *testing.T) {
	for _, tc := range []struct {
		name  string
		sound []int
	}{
		{"empty array", nil},
		{"shorter than the index", []int{soundSwingSlot}},
		{"zero at the index", []int{soundSwingSlot, 0, 0, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := numeralViewer(t)
			rec := &recordingPlayer{}
			v.SetAudio(rec, soundBankFor())
			v.SetSpeechAudio(rec)

			e := numeralUnit(1, 1, 100)
			e.Sound = tc.sound
			push(v, at0, e)
			push(v, at0, blow(v, &e, 30))

			if len(rec.plays) != 0 {
				t.Errorf("plays = %d, want 0", len(rec.plays))
			}
		})
	}
}

// AC-8: with the numeral display toggled off, AC-1 still plays — the two
// instruments are independent.
func TestFR6SoundIsNotGatedByTheNumeralToggle(t *testing.T) {
	v := numeralViewer(t)
	rec := &recordingPlayer{}
	v.SetAudio(rec, soundBankFor())
	v.SetSpeechAudio(rec)
	v.ToggleDamageNumerals()

	u := soundUnit(1, 1, 100)
	push(v, at0, u)
	push(v, at0, blow(v, &u, 60))

	if len(rec.plays) != 1 {
		t.Errorf("plays = %d, want 1 — sound is not the numeral's switch", len(rec.plays))
	}
	if on, _ := v.DamageNumerals(); on {
		t.Fatalf("test setup: the numeral display is still on")
	}
}

// AC-11: a nil device and a nil bank both leave every push silent and
// failing nothing. Neither branch asserts a play count — there is no
// recorder to ask — so the property under test is that push does not panic.
func TestAC11ANilDeviceAndANilBankAreSilentAndFailNothing(t *testing.T) {
	t.Run("SetAudio never called", func(t *testing.T) {
		v := numeralViewer(t)
		u := soundUnit(1, 1, 100)
		push(v, at0, u)
		push(v, at0, blow(v, &u, -1)) // a real wound and fall, if anything were listening
	})
	t.Run("SetAudio(nil, nil)", func(t *testing.T) {
		v := numeralViewer(t)
		v.SetAudio(nil, nil)
		u := soundUnit(1, 1, 100)
		u.Voice = "mf_hero"
		push(v, at0, u)
		push(v, at0, blow(v, &u, -1))
	})
}

func TestHurtEvent(t *testing.T) {
	for _, tc := range []struct {
		name          string
		stored, maxHP int
		want          int
		wantPlays     bool
	}{
		{"below half of an ordinary maximum", 40, 100, hurtHard, true},
		{"exactly half is NOT below half", 50, 100, hurtEasy, true},
		{"at or above half", 60, 100, hurtEasy, true},
		{"half of an odd maximum truncates", 49, 99, hurtEasy, true},
		{"at the floor plays nothing", -10, 100, 0, false},
		{"below the floor plays nothing", -30, 100, 0, false},
		{"one above the floor is below half", -9, 100, hurtHard, true},
		{"a zero maximum halves to zero", -5, 0, hurtHard, true},
		{"a zero maximum at zero is not below half", 0, 0, hurtEasy, true},
		{"a negative maximum halves toward zero", -5, -21, hurtEasy, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			k, ok := hurtEvent(tc.stored, tc.maxHP)
			if ok != tc.wantPlays {
				t.Fatalf("ok = %v, want %v", ok, tc.wantPlays)
			}
			if ok && k != tc.want {
				t.Errorf("event = %d, want %d", k, tc.want)
			}
		})
	}
}

// voiceBank is soundBankFor with a human voice bank: every recording carries
// its own name.
type voiceBank struct {
	stubBank
	names map[string]int16
}

func (b voiceBank) VoiceSample(name string) (audio.Sample, bool) {
	marker, ok := b.names[name]
	if !ok {
		return audio.Sample{}, false
	}
	return audio.Sample{Rate: audio.DeviceRate, PCM: []int16{marker}}, true
}

// voiceNames are the recordings voiceBankFor answers, by marker.
var voiceNames = map[int16]string{
	11: "mf_hero/easy.wav", 12: "mf_hero/hard.wav", 13: "mf_hero/die.wav",
	21: "f_mage/easy.wav", 22: "f_mage/hard.wav", 23: "f_mage/die.wav",
}

func voiceBankFor() voiceBank {
	b := voiceBank{stubBank: soundBankFor(), names: map[string]int16{}}
	for marker, name := range voiceNames {
		b.names[name] = marker
	}
	return b
}

// heard names every recording rec played, in order: a bank leaf by its name,
// a class slot by its number.
func heard(rec *recordingPlayer) []string {
	var out []string
	for _, p := range rec.plays {
		m := p.Sample.PCM[0]
		if name, ok := voiceNames[m]; ok {
			out = append(out, name)
		} else {
			out = append(out, "slot "+strconv.Itoa(int(m)))
		}
	}
	return out
}

// A bank-voiced entity's wounds play its bank's easy and hard leaves by the
// stored health, and never its drawn class's Sound array (ANIM-096).
func TestABankVoicedWoundPlaysTheBanksLeafNotTheClassSlot(t *testing.T) {
	v := numeralViewer(t)
	rec := &recordingPlayer{}
	v.SetAudio(rec, voiceBankFor())
	v.SetSpeechAudio(rec)

	u := soundUnit(1, 1, 100)
	u.Voice = "mf_hero"
	push(v, at0, u)
	push(v, at0, blow(v, &u, 60))                            // stored 100
	push(v, at0.Add(1600*time.Millisecond), blow(v, &u, 45)) // stored 60
	push(v, at0.Add(3200*time.Millisecond), blow(v, &u, 20)) // stored 45

	if got := heard(rec); !slices.Equal(got, []string{"mf_hero/easy.wav", "mf_hero/easy.wav", "mf_hero/hard.wav"}) {
		t.Fatalf("heard %v, want easy, easy, hard from mf_hero", got)
	}
}

// The fall plays the die leaf with no gate and stores no timestamp, and a
// blow on the fallen body above the floor is still a wound (ANIM-095). A body
// first seen fallen, or fallen again with no revival, plays no second die.
func TestAFallPlaysTheDieLeafUngatedAndUnstamped(t *testing.T) {
	v := numeralViewer(t)
	rec := &recordingPlayer{}
	v.SetAudio(rec, voiceBankFor())
	v.SetSpeechAudio(rec)

	u := soundUnit(1, 1, 100)
	u.Voice = "f_mage"
	push(v, at0, u)
	push(v, at0, blow(v, &u, 60))                             // easy, stamps at0
	push(v, at0.Add(1000*time.Millisecond), blow(v, &u, -2))  // gated wound; the die plays
	push(v, at0.Add(1600*time.Millisecond), blow(v, &u, -6))  // 1600 past the stamp: hard
	push(v, at0.Add(4000*time.Millisecond), blow(v, &u, -10)) // stored -6: hard again
	push(v, at0.Add(6000*time.Millisecond), blow(v, &u, -14)) // stored -10: nothing

	if got := heard(rec); !slices.Equal(got, []string{"f_mage/easy.wav", "f_mage/die.wav", "f_mage/hard.wav", "f_mage/hard.wav"}) {
		t.Fatalf("heard %v, want easy, die, hard, hard from f_mage", got)
	}

	fresh := numeralViewer(t)
	rec = &recordingPlayer{}
	fresh.SetAudio(rec, voiceBankFor())
	fresh.SetSpeechAudio(rec)
	down := soundUnit(2, 1, -3)
	down.Voice = "f_mage"
	push(fresh, at0, down)
	down.HP, down.Life = -4, lifeAt(-4)
	push(fresh, at0.Add(2*time.Second), down)
	if len(rec.plays) != 0 {
		t.Fatalf("a body first seen fallen voiced %v", heard(rec))
	}
}

// A class-voiced fall plays index 4 of its drawn class's Sound array with no
// gate and stores no timestamp, as the bank-voiced fall plays die (ANIM-094,
// ANIM-095). A class holding no index 4 falls silent.
func TestAClassVoicedFallPlaysSoundFourUngatedAndUnstamped(t *testing.T) {
	v := numeralViewer(t)
	rec := &recordingPlayer{}
	v.SetAudio(rec, soundBankFor())
	v.SetSpeechAudio(rec)

	u := soundUnit(1, 1, 100)
	u.Sound = append(u.Sound, soundDieSlot)
	push(v, at0, u)
	push(v, at0, blow(v, &u, 60))                             // Sound[2], stamps at0
	push(v, at0.Add(1000*time.Millisecond), blow(v, &u, -2))  // gated wound; the fall plays Sound[4]
	push(v, at0.Add(1600*time.Millisecond), blow(v, &u, -6))  // 1600 past the stamp: Sound[3]
	push(v, at0.Add(4000*time.Millisecond), blow(v, &u, -10)) // stored -6: Sound[3] again
	push(v, at0.Add(6000*time.Millisecond), blow(v, &u, -14)) // stored -10: nothing

	want := []int16{soundHighSlot, soundDieSlot, soundLowSlot, soundLowSlot}
	var got []int16
	for _, p := range rec.plays {
		got = append(got, p.Sample.PCM[0])
	}
	if !slices.Equal(got, want) {
		t.Fatalf("played slots %v, want %v", got, want)
	}

	silent := numeralViewer(t)
	rec = &recordingPlayer{}
	silent.SetAudio(rec, soundBankFor())
	silent.SetSpeechAudio(rec)
	short := soundUnit(2, 1, 5)
	push(silent, at0, short)
	short.HP, short.Life = -1, LifeDead // a fall by no blow, in a class with no index 4
	push(silent, at0, short)
	if len(rec.plays) != 0 {
		t.Fatalf("a class with no Sound[4] played %v", heard(rec))
	}

	fresh := numeralViewer(t)
	rec = &recordingPlayer{}
	fresh.SetAudio(rec, soundBankFor())
	fresh.SetSpeechAudio(rec)
	down := soundUnit(3, 1, -3)
	down.Sound = append(down.Sound, soundDieSlot)
	push(fresh, at0, down)
	down.HP, down.Life = -4, lifeAt(-4)
	push(fresh, at0.Add(2*time.Second), down)
	if len(rec.plays) != 0 {
		t.Fatalf("a body first seen fallen played %v", heard(rec))
	}
}

// A bank-less SoundBank leaves a bank-voiced entity silent.
func TestABanklessDeviceLeavesABankVoicedEntitySilent(t *testing.T) {
	v := numeralViewer(t)
	rec := &recordingPlayer{}
	v.SetAudio(rec, soundBankFor())
	v.SetSpeechAudio(rec)
	u := soundUnit(1, 1, 100)
	u.Voice = "mf_hero"
	push(v, at0, u)
	push(v, at0, blow(v, &u, -1))
	if len(rec.plays) != 0 {
		t.Fatalf("a bank-less device played %v", heard(rec))
	}
}

// Wounds and replies store one timestamp per drawable (ANIM-094): a reply
// holds a wound off for 1500 ms, and a wound holds a reply off for its own
// 3000 ms.
func TestWoundsAndRepliesShareOneVoiceTimestamp(t *testing.T) {
	const reply = 3 * time.Second
	v := numeralViewer(t)
	rec := &recordingPlayer{}
	v.SetAudio(rec, voiceBankFor())
	v.SetSpeechAudio(rec)

	u := soundUnit(1, 1, 100)
	u.Voice = "mf_hero"
	push(v, at0, u)
	if !v.ClaimVoice(1, at0, reply) {
		t.Fatal("the first reply was refused")
	}
	push(v, at0.Add(1400*time.Millisecond), blow(v, &u, 90))
	if len(rec.plays) != 0 {
		t.Fatalf("a wound 1400 ms after a reply voiced %v", heard(rec))
	}
	push(v, at0.Add(1500*time.Millisecond), blow(v, &u, 80))
	if got := heard(rec); !slices.Equal(got, []string{"mf_hero/easy.wav"}) {
		t.Fatalf("a wound 1500 ms after a reply voiced %v, want the easy leaf", got)
	}
	if v.ClaimVoice(1, at0.Add(4499*time.Millisecond), reply) {
		t.Fatal("a reply 2999 ms after a wound was admitted")
	}
	if !v.ClaimVoice(1, at0.Add(4500*time.Millisecond), reply) {
		t.Fatal("a reply 3000 ms after a wound was refused")
	}
	if !v.ClaimVoice(2, at0, reply) {
		t.Fatal("another drawable's reply shared the timestamp")
	}
}
