package game

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// statSaveFields is the part of one Human's SAVE record a timed stat effect
// touches: the live word, the modifier word the original un-applies at expiry,
// and the effect record's own key, magnitude and remaining time.
type statSaveFields struct {
	Live, Modifier       int32
	Magnitude, Remaining int32
	Children             int
}

type castRoute int

const (
	castTarget castRoute = iota
	castSelf
)

type statCase struct {
	name      string
	spell     uint16
	effect    sim.EffectKind
	key       uint32
	modOffset int
	scale     int32
	cast      castRoute
	live      func(doc *sav.DocumentData, a *sav.DocumentRecordData) int32
}

func statDefence(index int) func(*sav.DocumentData, *sav.DocumentRecordData) int32 {
	return func(_ *sav.DocumentData, a *sav.DocumentRecordData) int32 {
		raw, err := savedActorRaw(a, "UBE", 22)
		if err != nil {
			panic(err)
		}
		return int32(int16(binary.LittleEndian.Uint16(raw[index:])))
	}
}

var statCases = []statCase{
	{"Protection from Fire", 5, sim.EffectProtectionFire, 21, 48, 1, castTarget, statDefence(6)},
	{"Shield", 18, sim.EffectAbsorption, 16, 44, 1, castSelf, statDefence(2)},
}

func statSaveRead(t *testing.T, doc sav.DocumentData, tc statCase, reaction int) statSaveFields {
	t.Helper()
	var out statSaveFields
	found := 0
	for i := range doc.Objects {
		a := &doc.Objects[i]
		if a.Class != "Human" {
			continue
		}
		if v, _ := savedStructureValue(a, "Reaction"); int(v) != reaction {
			continue
		}
		found++
		block, err := savedActorRaw(a, "UD4", 64)
		if err != nil {
			t.Fatal(err)
		}
		out.Live = tc.live(&doc, a)
		out.Modifier = int32(int16(binary.LittleEndian.Uint16(block[tc.modOffset:])))
		refs, _ := savedObjectRefs(a, "Effects")
		for _, r := range refs {
			child := &doc.Objects[r-1]
			if kind, _ := savedStructureValue(child, "E3C"); kind != tc.key {
				continue
			}
			out.Children++
			operand, _ := savedStructureValue(child, "E40")
			out.Magnitude, out.Remaining = int32(int16(operand)), int32(operand>>16)
		}
	}
	if found != 1 {
		t.Fatalf("Human records with Reaction %d: %d, want one", reaction, found)
	}
	return out
}

func statActive(f *FrontEnd, tc statCase) (sim.ActiveEffect, bool) {
	for _, e := range f.live.world.ActiveEffects() {
		if e.Spell == tc.spell && e.Kind == tc.effect {
			return e, true
		}
	}
	return sim.ActiveEffect{}, false
}

func statLive(t *testing.T, f *FrontEnd, id sim.EntityID, tc statCase) int32 {
	e := releaseEntity(t, f.live, id)
	switch tc.effect {
	case sim.EffectAbsorption:
		return e.Absorption
	}
	return e.Protection[tc.effect-sim.EffectProtectionFire]
}

func statParty() []mapload.PartyMember {
	hero := data.Hero{Body: 60, Reaction: 60, Mind: 40, Spirit: 100}
	for i := range hero.Skill {
		hero.Skill[i] = 45
	}
	return []mapload.PartyMember{{ID: "hero", PlayerCharacter: true, StartingHero: true, Mage: true,
		Profile: data.Profile{HealthColumn: true, ManaColumn: true}, Hero: hero, KnownSpells: 0x1ffffffe,
		Saved: &mapload.Saved{Cell: mapload.Cell{X: 53, Y: 54}, HP: 1000, MaxHP: 1000, Mana: 1000, MaxMana: 1000}},
		{ID: "companion", Profile: data.Profile{HealthColumn: true}, Hero: data.Hero{Body: 26, Reaction: 15},
			Saved: &mapload.Saved{Cell: mapload.Cell{X: 54, Y: 55}, HP: 1000, MaxHP: 1000}}}
}

// SAVE under a timed Shield or Protection effect writes the modifier
// word the original un-applies at expiry; cold LOAD applies the effect once,
// and the SAVE after expiry holds the base value with a zero modifier word.
func TestReleaseTimedStatEffectSavesKeepModifierWordThroughColdLoadAndExpiry(t *testing.T) {
	for _, tc := range statCases {
		t.Run(tc.name, func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			if err := f.App("stat save").OpenMission(f.MissionOpenerWith(101, statParty())); err != nil {
				t.Fatal(err)
			}
			caster, target := f.live.mission.ids[0], f.live.mission.ids[1]
			bases := map[sim.EntityID]int32{caster: statLive(t, f, caster, tc), target: statLive(t, f, target, tc)}
			victim := releaseEntity(t, f.live, target)
			switch tc.cast {
			case castSelf:
				f.live.attackOrCast(uint32(caster), uint32(caster), uint32(tc.spell), int(victim.X), int(victim.Y), false)
			default:
				f.live.attackOrCast(uint32(caster), uint32(target), uint32(tc.spell), int(victim.X), int(victim.Y), false)
			}
			var effect sim.ActiveEffect
			for tick := 0; ; tick++ {
				if tick > 256 {
					t.Fatal("effect never attached", f.live.world.BookSpellRefusal(caster, target, uint32(tc.spell)))
				}
				f.live.tick()
				if e, ok := statActive(f, tc); ok {
					effect = e
					break
				}
			}
			target = effect.Target
			base := bases[target]
			reaction := releaseEntity(t, f.live, target).Reaction
			mag := effect.Magnitude
			boosted := statLive(t, f, target, tc)
			t.Logf("%s: base %d boosted %d magnitude %d remaining %d", tc.name, base, boosted, mag, effect.Remaining)
			if boosted != base+mag {
				t.Fatalf("live %d, want base %d + %d", boosted, base, mag)
			}
			store := SaveStore{Dir: t.TempDir()}
			name, doc := deadPatrolSave(t, f, store)
			saved := statSaveRead(t, doc, tc, int(reaction))
			t.Logf("SAVE under effect: live %d modifier %d child magnitude %d remaining %d children %d", saved.Live, saved.Modifier, saved.Magnitude, saved.Remaining, saved.Children)
			if saved.Live != boosted || saved.Magnitude != mag || saved.Children != 1 {
				t.Fatalf("effect state not written: %+v", saved)
			}
			if saved.Modifier != mag*tc.scale {
				t.Fatalf("modifier word %d, want the applied magnitude %d; the original un-applies only that word at expiry and its next derive refolds it", saved.Modifier, mag*tc.scale)
			}

			cold := loadLocalLegacySave(t, store, name)
			loaded, ok := statActive(cold, tc)
			if !ok {
				t.Fatal("cold LOAD lost the effect")
			}
			if got := statLive(t, cold, loaded.Target, tc); got != boosted {
				t.Fatalf("cold LOAD live %d, want %d (effect applied once)", got, boosted)
			}
			_, again := deadPatrolSave(t, cold, SaveStore{Dir: t.TempDir()})
			if r := statSaveRead(t, again, tc, int(reaction)); r.Modifier != saved.Modifier || r.Live != boosted || r.Children != 1 {
				t.Fatalf("second SAVE: %+v, want modifier %d live %d", r, saved.Modifier, boosted)
			}
			for tick := 0; ; tick++ {
				if tick > 3000 {
					t.Fatal("effect never expired")
				}
				f.live.tick()
				cold.live.tick()
				_, a := statActive(f, tc)
				_, b := statActive(cold, tc)
				if a != b {
					t.Fatal("live and loaded effect expired on different ticks", tick)
				}
				if !a {
					break
				}
			}
			if got := statLive(t, f, target, tc); got != base {
				t.Fatalf("live after expiry %d, want base %d", got, base)
			}
			if got := statLive(t, cold, loaded.Target, tc); got != base {
				t.Fatalf("loaded after expiry %d, want base %d", got, base)
			}
			_, after := deadPatrolSave(t, cold, SaveStore{Dir: t.TempDir()})
			final := statSaveRead(t, after, tc, int(reaction))
			t.Logf("SAVE after expiry: live %d modifier %d children %d", final.Live, final.Modifier, final.Children)
			if final.Live != base || final.Modifier != 0 || final.Children != 0 {
				t.Fatalf("post-expiry SAVE: %+v, want live %d modifier 0", final, base)
			}
		})
	}
}
