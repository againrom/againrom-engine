package ui

import (
	"errors"
	"slices"
	"testing"
	"time"
)

func TestSessionEntryVoicesOnlyStageOneOncePerCorpse(t *testing.T) {
	v, effects, speech := hurtDevices(t)
	var entities []MapEntity
	for stage := uint8(0); stage <= 5; stage++ {
		e := hurtUnit(uint32(stage)+1, 0, -15, "mf_hero")
		e.CorpseStage = stage
		if stage == 0 {
			e.HP, e.Life = 100, LifeAlive
		}
		entities = append(entities, e)
	}
	second := hurtUnit(9, 0, 0, "f_mage")
	second.CorpseStage = 1
	entities = append(entities, second)
	v.SetEntities(entities)
	v.beginSoundEntry()
	for range 3 {
		push(v, at0, entities...)
	}
	want := []string{"mf_hero/die.wav", "f_mage/die.wav"}
	if got := heard(speech); !slices.Equal(got, want) || len(effects.plays) != 0 {
		t.Fatal("entry selected health or replayed a cue", got, heard(effects))
	}
	if !v.ClaimVoice(2, at0, GruntThrottle) {
		t.Fatal("entry die stamped the voice gate")
	}
	v.beginSoundEntry()
	push(v, at0, entities...)
	if got := heard(speech); !slices.Equal(got, append(want, want...)) {
		t.Fatal("an explicit new entry at the same IDs and time did not replay once", got)
	}
}

func TestSessionEntryCancellationAndFailedLoadKeepSoundMemory(t *testing.T) {
	v, _, speech := hurtDevices(t)
	body := hurtUnit(1, 0, -15, "mf_hero")
	body.CorpseStage = 1
	v.SetEntities([]MapEntity{body})
	a := newTestApp(t, appRows(0), okLoader(t))
	open := func() (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
		return v, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil
	}
	if err := a.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	push(v, at0, body)
	a.SetSaveSeams(nil, func() []SaveEntry { return []SaveEntry{{Name: "failed.sav", Label: "failed"}} },
		func(string) (MapOpener, bool, error) {
			return func() (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
				return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, errors.New("broken save")
			}, false, nil
		})
	for _, fail := range []bool{false, true} {
		a.flow.openLoad(ScreenMap)
		if fail {
			a.flow.chooseLoad()
			if a.Screen() != ScreenLoad || a.HeadlessMessage() == "" {
				t.Fatal("failed LOAD did not stay on its chooser", a.Screen(), a.HeadlessMessage())
			}
		}
		a.flow.closeLoad()
		push(v, at0, body)
		if a.flow.viewer != v || len(speech.plays) != 1 {
			t.Fatal("cancelled or failed LOAD replaced the session sound memory", heard(speech))
		}
	}
}

func TestEntryKeepsItsStageProjectionBeforeTheFirstTick(t *testing.T) {
	v, _, speech := hurtDevices(t)
	body := hurtUnit(1, 0, -15, "mf_hero")
	body.CorpseStage = 1
	v.SetEntities([]MapEntity{body})
	v.beginSoundEntry()
	body.CorpseStage = 2
	push(v, at0, body)
	push(v, at0.Add(time.Millisecond), body)
	if got := heard(speech); !slices.Equal(got, []string{"mf_hero/die.wav"}) {
		t.Fatal("the first tick replaced the entry stage", got)
	}
}

func TestAStageOneSnapshotWithoutSessionEntryStaysSilent(t *testing.T) {
	v, _, speech := hurtDevices(t)
	body := hurtUnit(1, 0, -15, "mf_hero")
	body.CorpseStage = 1
	push(v, at0, body)
	if len(speech.plays) != 0 {
		t.Fatal("a snapshot invented a session entry", heard(speech))
	}
}
