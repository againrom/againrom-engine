package sim

import "testing"

func TestCurrentDecayHealthPublishesOnlyExistingBasis(t *testing.T) {
	for _, present := range []bool{false, true} {
		w := mustWorld(t, 7, Bounds{Width: 4, Height: 4}, []Entity{{ID: 1, HP: -20, MaxHP: 31, Decay: DecayBones}})
		if present {
			w.entities[0].ActorLoad = ActorLoad{Present: true, Source: SourceActor{Class: 1, Stats: [14]uint16{9, 8, 7, 6, 0, 0, 0, 300, 65516, 31}}}
		}
		w.tick = decayPhase
		w.decayPass()
		e := w.entities[0]
		if e.HP != -21 || present && e.ActorLoad.Source.Stats[8] != 65515 || !present && e.ActorLoad.Source != (SourceActor{}) {
			t.Fatal("native decay left a stale or invented source basis", present, e.ActorLoad.Source, e.HP)
		}
		if present && (e.ActorLoad.Source.Stats[0] != 9 || e.ActorLoad.Source.Stats[7] != 300 || e.ActorLoad.Source.Stats[9] != 31) {
			t.Fatal("health writer changed another source operand")
		}
		decaySessionHealth(&e)
		if e.HP != -22 || present && e.ActorLoad.Source.Stats[8] != 65514 {
			t.Fatal("native-domain session decay left stale health")
		}
		e.HP, e.CurrentProfileBasis = -32768, ProfileOriginalCurrent
		decaySessionHealth(&e)
		if e.HP != 32767 || present && e.ActorLoad.Source.Stats[8] != 32767 {
			t.Fatal("source-domain decay lost word arithmetic or its basis")
		}
		e.setCurrentHealth(noCorpseHP)
		if e.HP != noCorpseHP || present && e.ActorLoad.Source.Stats[8] != uint16(e.HP) || !present && e.ActorLoad.Source != (SourceActor{}) {
			t.Fatal("flying-body health changed basis presence or value")
		}
	}
}
