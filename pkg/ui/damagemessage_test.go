package ui

import (
	"slices"
	"testing"
	"time"

	"againrom/pkg/audio"
)

func TestDamageMessageUsesClientStoredHealthAndEqualityBeforeFloor(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		client, server, after int
		effects, speech       []string
	}{
		{"client differs from server", 40, 90, 90, nil, []string{"mf_hero/hard.wav"}},
		{"equality at floor", -10, -9, -10, []string{"slot 205"}, nil},
		{"below floor changed", -10, -9, -11, nil, nil},
		{"signed word equality", 65526, 0, 65526, []string{"slot 205"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, effects, speech := hurtDevices(t)
			u := hurtUnit(1, 1, tc.client, "mf_hero")
			push(v, at0, u)
			u.HP = tc.server
			v.AppendDamageMessages([]DamageMessage{{Entity: u, AfterHP: tc.after}})
			v.stepSound(at0)
			if !slices.Equal(heard(effects), tc.effects) || !slices.Equal(heard(speech), tc.speech) {
				t.Fatalf("effects=%v speech=%v", heard(effects), heard(speech))
			}
			if got := v.voices[1].hp; got != int(int16(tc.after)) {
				t.Fatalf("stored=%d want %d", got, int(int16(tc.after)))
			}
		})
	}
}

type hurtProbeBank struct {
	voiceBank
	probe func()
}

func (b hurtProbeBank) VoiceSample(name string) (audio.Sample, bool) {
	b.probe()
	return b.voiceBank.VoiceSample(name)
}

type hurtRefusingPlayer struct{ probe func() }

func (p hurtRefusingPlayer) Play(audio.Sample, audio.Placement) { panic("typed request required") }
func (p hurtRefusingPlayer) RequestSample(audio.Sample, audio.Request) audio.Voice {
	p.probe()
	return nil
}

func TestDamageMessagePublishesStampBeforeResolutionAndAdmission(t *testing.T) {
	v, _, _ := hurtDevices(t)
	u := hurtUnit(1, 1, 40, "mf_hero")
	push(v, at0, u)
	var resolved, refused bool
	probe := func() {
		t.Helper()
		m := v.voices[1]
		if !m.at.Equal(at0) || m.hp != 40 {
			t.Fatalf("hook sees stamp=%v HP=%d; stamp must precede hook, HP store must follow", m.at, m.hp)
		}
		if v.ClaimVoice(1, at0, 3*time.Second) {
			t.Fatal("reply admitted inside wound hook")
		}
	}
	v.SetAudio(&recordingPlayer{}, hurtProbeBank{voiceBank: hurtBank(), probe: func() { probe(); resolved = true }})
	v.SetSpeechAudio(hurtRefusingPlayer{probe: func() { probe(); refused = true }})
	v.AppendDamageMessages([]DamageMessage{{Entity: u, AfterHP: 30}})
	v.stepSound(at0)
	if !resolved || !refused || v.voices[1].hp != 30 || !v.voices[1].at.Equal(at0) {
		t.Fatal("refusal lost source/admission hook or message health")
	}
}

func TestDamageMessageQueueAppendsAcrossCatchUpAndDrainsOnce(t *testing.T) {
	v, effects, speech := hurtDevices(t)
	u := hurtUnit(1, 1, 100, "mf_hero")
	push(v, at0, u)
	v.AppendDamageMessages([]DamageMessage{{Entity: u, AfterHP: 100}, {Entity: u, AfterHP: 40}, {Entity: u, AfterHP: 40}})
	u.HP = 40
	v.SetEntities([]MapEntity{u})
	v.SetEntities([]MapEntity{u})
	v.AppendDamageMessages([]DamageMessage{{Entity: u, AfterHP: 20}, {Entity: u, AfterHP: 20}})
	u.HP = 20
	v.SetEntities([]MapEntity{u})
	v.step(Input{}, at0.Add(time.Millisecond))
	if got := heard(effects); !slices.Equal(got, []string{"slot 205", "slot 205", "slot 205"}) {
		t.Fatalf("queued zero messages=%v", got)
	}
	if got := heard(speech); !slices.Equal(got, []string{"mf_hero/easy.wav"}) {
		t.Fatalf("ordered wound/throttle=%v", got)
	}
	if v.voices[1].hp != 20 || len(v.soundMessages) != 0 {
		t.Fatal("queue did not store final health and drain")
	}
	push(v, at0.Add(time.Second), u)
	if len(effects.plays) != 3 || len(speech.plays) != 1 {
		t.Fatal("repeated snapshot replayed causal messages")
	}
}

func TestDamageMessageClaimsBeforeMissingSourceAndFogAndAdvancesHealth(t *testing.T) {
	for _, fog := range []bool{false, true} {
		v, _, speech := hurtDevices(t)
		u := hurtUnit(1, 2, 100, "mf_hero")
		push(v, at0, u)
		if fog {
			v.SetFog(make([]uint8, numeralGrid*numeralGrid), numeralGrid, numeralGrid)
		} else {
			v.soundBank = nil
		}
		v.AppendDamageMessages([]DamageMessage{{Entity: u, AfterHP: 40}})
		v.stepSound(at0)
		if len(speech.plays) != 0 || v.voices[1].hp != 40 || !v.voices[1].at.Equal(at0) {
			t.Fatal("suppression lost the claimed timestamp or client health")
		}
		v.SetFog(nil, 0, 0)
		v.soundBank = hurtBank()
		v.AppendDamageMessages([]DamageMessage{{Entity: u, AfterHP: 30}})
		v.stepSound(at0.Add(time.Second))
		if len(speech.plays) != 0 || v.voices[1].hp != 30 {
			t.Fatal("throttle did not advance client health silently")
		}
		v.AppendDamageMessages([]DamageMessage{{Entity: u, AfterHP: 20}})
		v.stepSound(at0.Add(2 * time.Second))
		if !slices.Equal(heard(speech), []string{"mf_hero/hard.wav"}) {
			t.Fatalf("next wound used stale client health: %v", heard(speech))
		}
	}
}

func TestCatchUpSnapshotsKeepEveryFallAndNoReplay(t *testing.T) {
	v, _, speech := hurtDevices(t)
	a, b := hurtUnit(1, 1, 100, "mf_hero"), hurtUnit(2, 1, 100, "f_mage")
	push(v, at0, a, b)
	a.HP, a.Life = -1, LifeDead
	v.SetEntities([]MapEntity{a, b})
	b.HP, b.Life = -1, LifeDead
	v.SetEntities([]MapEntity{a, b})
	v.SetEntities([]MapEntity{a, b})
	v.stepSound(at0)
	if got := heard(speech); !slices.Equal(got, []string{"mf_hero/die.wav", "f_mage/die.wav"}) {
		t.Fatalf("falls=%v", got)
	}
	v.stepSound(at0.Add(time.Millisecond))
	if len(speech.plays) != 2 {
		t.Fatal("fall replay")
	}
}
