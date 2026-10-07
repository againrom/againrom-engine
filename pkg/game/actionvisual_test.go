package game

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"againrom/pkg/sim"
)

func visualIdentityWorld(t *testing.T, ids []sim.EntityID, floor sim.EntityID) *sim.World {
	t.Helper()
	var entities []sim.Entity
	for i, id := range ids {
		entities = append(entities, sim.Entity{ID: id, X: int32(i + 1), Y: 1, HP: 10, MaxHP: 10})
	}
	w, err := sim.NewWorld(1, sim.Bounds{Width: 32, Height: 32}, sim.ModeCanonical, nil, entities)
	if err != nil {
		t.Fatal(err)
	}
	w.ReserveEntityIDs([]sim.EntityID{floor})
	return w
}

func TestCurrentVisualIdentityKeepsFutureBirthsRemovedIDsAndPointZero(t *testing.T) {
	native := &mapWorld{world: visualIdentityWorld(t, []sim.EntityID{81, 84}, 84)}
	rows, next := native.currentVisualIdentities()
	if next != 85 || native.visualCastSeed(84, 81, 6, false) != 1238302146 {
		t.Fatal("uninterrupted seed vocabulary changed")
	}
	for i := range rows {
		rows[i].Entity -= 14
	}
	cold := &mapWorld{world: visualIdentityWorld(t, []sim.EntityID{67, 70}, 70)}
	cold.restoreVisualIdentities(rows, next)
	if cold.visualCastSeed(70, 67, 6, false) != 1238302146 {
		t.Fatal("remint changed new Heal particle seed")
	}
	// Six allocated IDs include actors already removed before a visual is made.
	// The authoritative allocator floor, not the surviving population size,
	// carries these gaps into the visual namespace.
	cold.world = visualIdentityWorld(t, []sim.EntityID{67, 70, 77}, 80)
	before := cold.world.Hash()
	preview, n := cold.currentVisualIdentities()
	if n != 95 || cold.visualNext != 85 || cold.world.Hash() != before {
		t.Fatal("SAVE preview mutated or lost allocation gaps")
	}
	if cold.visualCastSeed(70, 77, 6, false) != castSeed(84, 91, 6) {
		t.Fatal("new actor used compacted local ID")
	}
	if cold.visualActorLabel(77)%chainTagCount != 91%chainTagCount {
		t.Fatal("weapon path tag lost stable identity")
	}
	// A point's Target=0 must not bind actor zero after another compact LOAD.
	for i := range preview {
		switch preview[i].Entity {
		case 67:
			preview[i].Entity = 0
		case 70:
			preview[i].Entity = 5
		case 77:
			preview[i].Entity = 7
		}
	}
	second := &mapWorld{world: visualIdentityWorld(t, []sim.EntityID{0, 5, 7}, 7)}
	second.restoreVisualIdentities(preview, n)
	if second.visualCastSeed(5, 0, 6, false) != castSeed(84, 81, 6) || second.visualCastSeed(5, 0, 2, true) != castSeed(84, 0, 2) {
		t.Fatal("actor zero and point endpoint conflated")
	}
	second.world = visualIdentityWorld(t, []sim.EntityID{0, 5, 11}, 11)
	if second.visualCastSeed(5, 11, 6, false) != castSeed(84, 98, 6) {
		t.Fatal("second cut lost future visual allocator")
	}
}

func TestCurrentVisualContinuationKeepsBirthDelayPathAndExpiry(t *testing.T) {
	for _, c := range []struct {
		spell  uint16
		weapon bool
	}{{2, false}, {6, false}, {11, false}, {13, false}, {14, false}, {14, true}, {26, false}} {
		spell, name := c.spell, fmt.Sprint(c.spell)
		if c.weapon {
			name += "weapon"
		}
		t.Run(name, func(t *testing.T) {
			mw := spellSoundFireBallWorld(t, 100)
			if spell == 26 {
				mw = spellSoundTeleportWorld(t)
			}
			// The normal immutable observer builds all records; the test never
			// fabricates a bolt, particle cohort, animation counter or lifetime.
			ev := sim.CastEvent{Caster: 1, Target: 1, Spell: spell, Owner: 7, TargetOwner: 7, FromX: 8, FromY: 8, ToX: 12, ToY: 8}
			if c.weapon {
				// A staff's spray: its set is the one live across a SAVE.
				ev.Weapon, ev.Victims = true, []sim.CellPoint{{X: 12, Y: 8}, {X: 10, Y: 9}, {X: 9, Y: 6}}
			}
			mw.observeCasts([]sim.CastEvent{ev})
			if n := len(spPictureBolts(mw, 36)); c.weapon && n != 3 {
				t.Fatalf("the staff spray spawned %d figures, want 3", n)
			}
			for range 2 {
				mw.advanceBolts()
				mw.advanceHealBursts()
				mw.advanceCastRuns()
			}
			var saved SnapshotResidue
			mw.actionVisuals(&saved)
			wire, err := json.Marshal(saved)
			if err != nil {
				t.Fatal(err)
			}
			var back SnapshotResidue
			if err = json.Unmarshal(wire, &back); err != nil {
				t.Fatal(err)
			}
			cold := spellSoundFireBallWorld(t, 100)
			cold.restoreActionVisuals(back.SpellBolts, back.HealBursts)
			cold.restoreCastRuns(back.CastRuns)
			for tick := 0; tick < 100; tick++ {
				var a, b SnapshotResidue
				mw.actionVisuals(&a)
				cold.actionVisuals(&b)
				if !reflect.DeepEqual(a.SpellBolts, b.SpellBolts) || !reflect.DeepEqual(a.HealBursts, b.HealBursts) || !reflect.DeepEqual(a.CastRuns, b.CastRuns) || !reflect.DeepEqual(mw.boltDraws(mw.world.Entities()), cold.boltDraws(cold.world.Entities())) {
					t.Fatalf("spell%d visual cut differs at%d", spell, tick)
				}
				mw.advanceBolts()
				mw.advanceHealBursts()
				mw.advanceCastRuns()
				cold.advanceBolts()
				cold.advanceHealBursts()
				cold.advanceCastRuns()
			}
			if len(cold.bolts)+len(cold.healBursts)+len(cold.castRun) != 0 {
				t.Fatal("visual did not expire")
			}
		})
	}
}
