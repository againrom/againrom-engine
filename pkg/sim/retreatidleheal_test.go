package sim

import "testing"

// A mage under Retreat runs the ordinary acquisition when no living hostile
// stands in the flee radius, so it heals its wounded team mate; with a living
// hostile in the radius it keeps walking away and casts nothing (AI-RETREAT-273).
func TestARetreatingMageHealsOnlyWhenNoLivingHostileIsNear(t *testing.T) {
	for _, tc := range []struct {
		name     string
		second   bool
		hostileX int32 // 0: no hostile
		wantHeal bool
	}{
		{"clear radius", false, 0, true},
		{"clear radius, second game", true, 0, true},
		{"hostile in the radius", false, 4, false},
		{"hostile in the radius, second game", true, 4, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			caster := acCaster(1, 2, 2, 50, 1<<6, 0)
			caster.Owner = SelfSlot
			caster.ScanRange = 5
			mate := spEnt(2, 3, 2)
			mate.Owner, mate.HP = SelfSlot, 40
			ents := []Entity{caster, mate}
			if tc.hostileX != 0 {
				foe := spEnt(3, tc.hostileX, 2)
				foe.Owner = 2
				ents = append(ents, foe)
			}
			w := hlWorld(t, 61, engRel(t, [3]uint32{SelfSlot, 2, relationHostile}, [3]uint32{2, SelfSlot, relationHostile}),
				[]SpellRule{hlHeal()}, ents...)
			if tc.second {
				w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
			}
			Step(w, []Command{GroupRetreat(1, SelfSlot, 0)})
			if w.entities[0].ActorState != actorStateRetreat {
				t.Fatalf("state %d, want Retreat", w.entities[0].ActorState)
			}
			for n := 0; n < 200; n++ {
				Step(w, nil)
			}
			healed := spAt(t, w, 2).HP > 40
			if healed != tc.wantHeal {
				t.Fatalf("team mate health %d, healed %t, want %t", spAt(t, w, 2).HP, healed, tc.wantHeal)
			}
		})
	}
}
