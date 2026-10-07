package sim

import (
	"encoding/binary"
	"testing"
)

func TestCurrentDeathDefencePublishesExistingBasis(t *testing.T) {
	for _, present := range []bool{false, true} {
		w := mustWorld(t, 7, Bounds{Width: 4, Height: 4}, []Entity{{ID: 1, HP: 20, MaxHP: 20, Defence: 31}})
		if present {
			w.entities[0].ActorLoad = ActorLoad{Present: true, Source: SourceActor{Class: 1, Defence: [22]byte{31, 0, 9, 8}}}
		}
		w.entities[0].setCurrentHealth(0)
		w.clearFelled(0)
		check := func(want int32) {
			t.Helper()
			e := w.entities[0]
			if e.Defence != want || present && int16(binary.LittleEndian.Uint16(e.ActorLoad.Source.Defence[:])) != int16(want) || !present && e.ActorLoad.Source != (SourceActor{}) {
				t.Fatal("death/revival left stale or invented defence basis", e.Defence, e.ActorLoad.Source)
			}
			if present && (e.ActorLoad.Source.Defence[2] != 9 || e.ActorLoad.Source.Defence[3] != 8) {
				t.Fatal("defence publication changed another operand")
			}
		}
		check(15)
		w.entities[0].setCurrentHealth(5)
		w.restoreAfterHealthGain(0, 0)
		check(30)
		w.entities[0].HP, w.entities[0].Defence = -1, -2
		PrepareAuthoredBody(&w.entities[0])
		check(-1)
	}
}

func TestCurrentDeadMirrorChangesOnlyOnDeathMutation(t *testing.T) {
	w := deadWorld(t)
	if err := w.ImportOriginalDeadActors([]OriginalDeadActor{deadInput(1, 4, -237)}); err != nil {
		t.Fatal(err)
	}
	w.entities[0].HP = -238
	before, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var loaded World
	if err := loaded.UnmarshalBinary(before); err != nil {
		t.Fatal(err)
	}
	if loaded.originalDead[0].Source.State.HP != -237 {
		t.Fatal("LOAD rewrote historical mirror")
	}
	loaded.tick = decayPhase - 1
	loaded.decayPass()
	if loaded.originalDead[0].Source.State.HP != -237 {
		t.Fatal("non-death tick rewrote historical mirror")
	}
	loaded.tick = decayPhase
	loaded.decayPass()
	if loaded.originalDead[0].Source.State.HP != -239 || loaded.OriginalDeadActors()[0].Current.HP != -239 {
		t.Fatal("death mutation failed to publish current mirror")
	}
}
