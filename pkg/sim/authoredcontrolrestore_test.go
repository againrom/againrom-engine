package sim

import (
	"fmt"
	"testing"
)

func TestAuthoredControlRetainedCastsSurviveResumeRepair(t *testing.T) {
	for _, control := range []uint32{0, 1, 0x100} {
		for _, creature := range []bool{false, true} {
			t.Run(fmt.Sprintf("control-%x-creature-%t", control, creature), func(t *testing.T) {
				e := cbEnt(1, 3, 3)
				e.Owner = SelfSlot
				if creature {
					e.CreatureSpells = creatureCertainSlot(2)
				}
				w := cbWorld(t, 1, e)
				if err := w.RestoreCurrentPlayers([]SavedGroupPlayer{{ID: 91, Slot: SelfSlot}}, []PlayerParticipant{{PlayerID: 91, Value: control}}, true); err != nil {
					t.Fatal(err)
				}
				w.bookCasts = []bookCast{{Caster: 1, Target: 1, Spell: 2, Phase: bookCharging, Remaining: 4, Retained: true}}
				var cold World
				if err := cold.UnmarshalBinary(mustMarshal(t, w)); err != nil {
					t.Fatal(err)
				}
				want := 0
				if control == 0 && !creature {
					want = 1
				}
				for _, world := range []*World{w, &cold} {
					if got := world.RestoreOneShotPlayerCasts(); got != want {
						t.Fatalf("repaired %d casts, want %d", got, want)
					}
					if len(world.bookCasts) != 1 || world.bookCasts[0].Retained != (want == 0) {
						t.Fatalf("cast after repair: %+v", world.bookCasts)
					}
				}
				if cold.Hash() != w.Hash() {
					t.Fatal("cold repair changed continuation")
				}
			})
		}
	}
}

func TestManualCellApproachSurvivesResumeRepair(t *testing.T) {
	e := cbEnt(1, 3, 3)
	e.Owner, e.HasTarget, e.TargetX, e.TargetY = SelfSlot, true, 10, 10
	w := cbWorld(t, 1, e)
	w.bookCasts = []bookCast{{Caster: 1, Spell: 26, X: 10, Y: 10, AtCell: true, Phase: bookApproach, Retained: true}}
	var cold World
	if err := cold.UnmarshalBinary(mustMarshal(t, w)); err != nil {
		t.Fatal(err)
	}
	before := cold.Hash()
	if got := cold.RestoreOneShotPlayerCasts(); got != 0 || cold.Hash() != before {
		t.Fatal("resume discarded a live manual approach", got)
	}
}
