package mapload_test

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestMissionStartDropsAnOldTownPotionRecord(t *testing.T) {
	regeneration := sim.ActiveEffect{Target: 1, Kind: sim.EffectHealthRegeneration, Mode: sim.EffectDuration, Magnitude: 100, Remaining: 960}
	absorption := sim.ActiveEffect{Target: 1, Kind: sim.EffectAbsorption, Mode: sim.EffectDuration, Magnitude: 3, Remaining: 2}
	for _, tc := range []struct {
		name   string
		member func(applied bool) mapload.PartyMember
		effect sim.ActiveEffect
	}{
		{"native", func(bool) mapload.PartyMember { return nativeCarryParty()[0] }, regeneration},
		{"native history", func(applied bool) mapload.PartyMember {
			p := nativeCarryParty()[0]
			raw := uint16(0x1234)
			if applied {
				raw += 3
			}
			b := [64]byte{}
			binary.LittleEndian.PutUint16(b[44:], raw)
			p.Carry = &mapload.Carry{NativeHistory: &mapload.NativeCarryHistory{Basis: (sim.NativeActorBasis{}).WithModifier(b), Class: sim.NativeClass{Present: true, Fighter: true}}}
			return p
		}, absorption},
		{"original Human", func(applied bool) mapload.PartyMember {
			h := data.HumanState{Body: 10, Reaction: 10, Mind: 10, Spirit: 10, Health: 20, HealthMax: 20, Speed: 40, Capacity: 200, Fighter: true, TypeID: 33, HasOwner: true, ManaReservePercent: 95}
			source := mapload.HumanSourceActor(h, 2)
			if applied {
				binary.LittleEndian.PutUint16(source.Modifier[10:], binary.LittleEndian.Uint16(source.Modifier[10:])+100)
			}
			return mapload.PartyMember{ID: "original", Class: 33, Hero: data.Hero{Body: 10, Reaction: 10, Mind: 10, Spirit: 10},
				Carry: &mapload.Carry{LiveLoad: &sim.ActorLoadSnapshot{Inventory: sim.ActorLoad{Present: true, ContainerPresent: true, Source: source}, Capacity: 200, Speed: 40}},
				Saved: &mapload.Saved{HP: 20, MaxHP: 20}}
		}, regeneration},
	} {
		for _, scripted := range []bool{false, true} {
			name := tc.name
			if scripted {
				name += " scripted"
			}
			t.Run(name, func(t *testing.T) {
				control := tc.member(false)
				old := tc.member(true)
				effect := tc.effect
				old.PotionEffect = &effect
				m := carryMap(t)
				start := func(p mapload.PartyMember) (*sim.World, mapload.Start) {
					if !scripted {
						return mustStart(t, m, []mapload.PartyMember{p})
					}
					w, st, err := mapload.StartMissionScripted(m, nil, mapload.DifficultyNormal, []mapload.PartyMember{p}, &sim.Script{})
					if err != nil {
						t.Fatal(err)
					}
					return w, st
				}
				cw, cst := start(control)
				w, st := start(old)
				for tick := 0; tick < 2; tick++ {
					if len(w.ActiveEffects()) != 0 {
						t.Fatalf("tick %d: old town record reached the mission: %+v", tick, w.ActiveEffects())
					}
					want, got := entityOf(t, cw, cst.IDs[0]), entityOf(t, w, st.IDs[0])
					if got.HealthRegeneration != want.HealthRegeneration || got.ManaRegeneration != want.ManaRegeneration || got.Absorption != want.Absorption ||
						got.HP != want.HP || got.Mana != want.Mana || got.NativeBasis != want.NativeBasis || got.ActorLoad.Source.Modifier != want.ActorLoad.Source.Modifier {
						t.Fatalf("tick %d: actor differs from one that never drank: got %+v want %+v", tick, got, want)
					}
					sim.Step(cw, nil)
					sim.Step(w, nil)
				}
			})
		}
	}
}
