package game

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"

	"againrom/pkg/sim"
)

type potionSaveCase struct {
	name      string
	effect    sim.EffectKind
	key       uint32
	modOffset int
}

var potionSaveCases = []potionSaveCase{
	{"Potion Health Regeneration", sim.EffectHealthRegeneration, 8, 10},
	{"Potion Mana Regeneration", sim.EffectManaRegeneration, 11, 14},
}

func potionRegen(t *testing.T, f *FrontEnd, id sim.EntityID, c potionSaveCase) int32 {
	e := releaseEntity(t, f.live, id)
	if c.effect == sim.EffectManaRegeneration {
		return e.ManaRegeneration
	}
	return e.HealthRegeneration
}

func potionActive(f *FrontEnd) (sim.ActiveEffect, bool) {
	for _, e := range f.live.world.ActiveEffects() {
		if e.Spell == 0 {
			return e, true
		}
	}
	return sim.ActiveEffect{}, false
}

func TestReleaseCarriedPotionMissionSaveKeepsModifierWordThroughColdLoadAndExpiry(t *testing.T) {
	for _, tc := range potionSaveCases {
		t.Run(tc.name, func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			party := statParty()
			if err := f.App("potion save").OpenMission(f.MissionOpenerWith(101, party)); err != nil {
				t.Fatal(err)
			}
			id := f.live.mission.ids[0]
			base := potionRegen(t, f, id, tc)
			mapload.BindSourceDerive(f.live.world)
			if !f.live.world.RestorePotionEffect(id, sim.ActiveEffect{Kind: tc.effect, Mode: 1, Magnitude: 100, Remaining: 960}) {
				t.Fatal("potion restore refused")
			}
			boosted := potionRegen(t, f, id, tc)
			if _, ok := potionActive(f); !ok || boosted != base+100 {
				t.Fatal("mission entry did not apply the carried potion", boosted, f.live.world.ActiveEffects())
			}
			for range 10 {
				f.live.tick()
			}
			reaction := releaseEntity(t, f.live, id).Reaction
			read := func(doc sav.DocumentData) (modifier int32, children int, remaining int32) {
				for i := range doc.Objects {
					a := &doc.Objects[i]
					if a.Class != "Human" {
						continue
					}
					if v, _ := savedStructureValue(a, "Reaction"); int(v) != int(reaction) {
						continue
					}
					block, err := savedActorRaw(a, "UD4", 64)
					if err != nil {
						t.Fatal(err)
					}
					modifier = int32(int16(binary.LittleEndian.Uint16(block[tc.modOffset:])))
					refs, _ := savedObjectRefs(a, "Effects")
					for _, r := range refs {
						child := &doc.Objects[r-1]
						if kind, _ := savedStructureValue(child, "E3C"); kind == tc.key {
							children++
							operand, _ := savedStructureValue(child, "E40")
							remaining = int32(operand >> 16)
						}
					}
				}
				return
			}
			store := SaveStore{Dir: t.TempDir()}
			name, doc := deadPatrolSave(t, f, store)
			modifier, children, remaining := read(doc)
			if modifier != 100 || children != 1 || remaining != 950 {
				t.Fatalf("SAVE under the potion: modifier %d children %d remaining %d, want 100/1/950", modifier, children, remaining)
			}
			cold := loadLocalLegacySave(t, store, name)
			loaded, ok := potionActive(cold)
			if !ok {
				t.Fatal("cold LOAD lost the potion")
			}
			if got := potionRegen(t, cold, loaded.Target, tc); got != boosted {
				t.Fatalf("cold LOAD regeneration %d, want %d", got, boosted)
			}
			for tick := 0; ; tick++ {
				if tick > 1200 {
					t.Fatal("potion never expired")
				}
				// The mission's hostiles reach the hero before the potion runs
				// out, and a fallen hero's effects stop counting down. Both
				// worlds take the same heal on the same tick, so they stay alike.
				if e := releaseEntity(t, f.live, id); e.HP > 0 && e.HP < e.MaxHP/2 {
					if err := f.live.world.HeadlessHeal(id); err != nil {
						t.Fatal(err)
					}
					if err := cold.live.world.HeadlessHeal(loaded.Target); err != nil {
						t.Fatal(err)
					}
				}
				f.live.tick()
				cold.live.tick()
				_, a := potionActive(f)
				_, b := potionActive(cold)
				if a != b {
					t.Fatal("live and loaded potion expired on different ticks", tick)
				}
				if !a {
					break
				}
			}
			if potionRegen(t, f, id, tc) != base || potionRegen(t, cold, loaded.Target, tc) != base {
				t.Fatal("regeneration after expiry differs from base", base)
			}
			_, after := deadPatrolSave(t, cold, SaveStore{Dir: t.TempDir()})
			if m, c, _ := read(after); m != 0 || c != 0 {
				t.Fatalf("post-expiry SAVE modifier %d children %d, want zero", m, c)
			}
		})
	}
}
